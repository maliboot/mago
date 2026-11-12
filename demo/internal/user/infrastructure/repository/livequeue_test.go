//go:build queuelive

// 真实 Redis 联调：只在显式指定 tag 时运行。
//
//	cd demo/internal/user/infrastructure/repository && go test -tags queuelive -run TestQueueLive -v -timeout 2m
//
// 连接参数从仓库根 runtime/db.txt（KEY=VALUE）的 REDIS_URI 读取，代码里不落任何口令；
// REDIS_URI 可能是 tcp://host:port/db?auth=xxx（其他组件在用的写法），会转成 asynq 认的 redis:// 形式。
package repository

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maliboot/mago"
	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/dispatcher"

	"tests/internal/user/client/constant"
	"tests/internal/user/client/dto"
)

// liveDBConfPath 是凭据文件相对包目录的位置（包在 demo/internal/user/infrastructure/repository）。
const liveDBConfPath = "../../../../../runtime/db.txt"

// liveRedisURIEnv 允许不改凭据文件就把联调指向另一个实例，例如本地 podman redis：
//
//	MAGO_LIVE_REDIS_URI=redis://127.0.0.1:6379/0 go test -tags queuelive -run TestQueueLive -v .
const liveRedisURIEnv = "MAGO_LIVE_REDIS_URI"

// loadLiveRedisURI 读取 runtime/db.txt 的 REDIS_URI（可用 MAGO_LIVE_REDIS_URI 覆盖），并转成 asynq 支持的 URI。
// asynq 只认 redis://、rediss://、redis-socket://、redis-sentinel://；
// 文件里若是 tcp://host:port/db?auth=pass，这里补成 redis://:pass@host:port/db。
func loadLiveRedisURI(t *testing.T) string {
	t.Helper()

	if override := strings.TrimSpace(os.Getenv(liveRedisURIEnv)); override != "" {
		return normalizeRedisURI(t, override)
	}

	bs, err := os.ReadFile(liveDBConfPath)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", liveDBConfPath, err)
	}

	raw := ""
	for _, line := range strings.Split(string(bs), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(key) == "REDIS_URI" {
			raw = strings.TrimSpace(value)
		}
	}
	if raw == "" {
		t.Fatalf("%s 缺少 REDIS_URI", liveDBConfPath)
	}

	return normalizeRedisURI(t, raw)
}

// normalizeRedisURI 把 tcp://host:port/db?auth=pass 转成 asynq 认的 redis://:pass@host:port/db。
func normalizeRedisURI(t *testing.T, raw string) string {
	t.Helper()

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("REDIS_URI 解析失败: %v", err)
	}

	switch u.Scheme {
	case "redis", "rediss":
		return raw
	case "tcp":
		auth := u.Query().Get("auth")
		uri := "redis://"
		if auth != "" {
			uri += url.UserPassword("", auth).String() + "@"
		}

		return uri + u.Host + u.Path
	default:
		t.Fatalf("不支持的 REDIS_URI scheme: %q", u.Scheme)

		return ""
	}
}

// liveQueueConf 构造指向真实 Redis 的队列配置。
func liveQueueConf(t *testing.T) *config.Conf {
	t.Helper()

	qc := &config.QueueConf{
		Default:     "redis",
		Concurrency: 2,
		CacheDir:    t.TempDir(),
	}
	qc.Redis.Dsn = loadLiveRedisURI(t)

	return &config.Conf{
		AppName: "mago-live-queue",
		Queue:   qc,
	}
}

// isOldRedisErr 识别 asynq 在低版本 Redis 上的 Lua 报错。
func isOldRedisErr(err error) bool {
	msg := err.Error()

	return strings.Contains(msg, "redis eval error") || strings.Contains(msg, "Error running script")
}

// TestQueueLive 在真实 Redis 上验证「发布 → 消费」闭环（走 demo 的 repository.Queue 与 adapter 契约）。
func TestQueueLive(t *testing.T) {
	conf := liveQueueConf(t)

	srv := conf.Queue.MustGetServer()
	if srv == nil {
		t.Fatal("队列服务未初始化")
	}

	consumed := make(chan dto.UserRegisteredEvent, 1)
	srv.RegisterHandleFunc(constant.TaskUserRegistered, func(_ context.Context, task *dispatcher.Task) error {
		var evt dto.UserRegisteredEvent
		if err := json.Unmarshal(task.Payload, &evt); err != nil {
			return err
		}
		consumed <- evt

		return nil
	})

	startErr := make(chan error, 1)
	go func() {
		if err := srv.Start(context.Background()); err != nil {
			startErr <- err
		}
	}()
	t.Cleanup(func() { _ = srv.Stop(context.Background()) })

	// Start 内部固定等 3s（留时间给其它 server 注册 handler），这里等它进入消费循环
	time.Sleep(4 * time.Second)

	select {
	case err := <-startErr:
		t.Fatalf("队列启动失败（检查 REDIS_URI 与网络）: %v", err)
	default:
	}

	want := dto.UserRegisteredEvent{
		UserID: 9999,
		Email:  "live-queue@example.com",
		At:     mago.DateTime{Time: time.Now().Truncate(time.Second)},
	}
	payload, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("载荷序列化失败: %v", err)
	}

	pub := NewQueue(conf)
	task := dispatcher.NewTask(
		constant.TaskUserRegistered, nil, payload,
		dispatcher.TaskID("mago-live-"+strconv.FormatInt(time.Now().UnixNano(), 10)),
	)
	if err := pub.Enqueue(context.Background(), task); err != nil {
		if isOldRedisErr(err) {
			t.Skipf("联调 Redis 版本过低（asynq 需要 Redis ≥ 4.0，实测账上实例 3.2.12）：%v\n"+
				"可改用本地新版 Redis：podman run -d -p 6379:6379 redis:7-alpine，"+
				"并设置 %s=redis://127.0.0.1:6379/0", err, liveRedisURIEnv)
		}

		t.Fatalf("入队失败: %v", err)
	}

	select {
	case got := <-consumed:
		if got.UserID != want.UserID || got.Email != want.Email {
			t.Fatalf("消费到的载荷不符：want %+v, got %+v", want, got)
		}
		if !got.At.Time.Equal(want.At.Time) {
			t.Fatalf("时间字段往返不一致：want %v, got %v", want.At.Time, got.At.Time)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("15s 内没有消费到任务（检查 REDIS_URI 与 worker 是否启动）")
	}
}
