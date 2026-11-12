package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/hibiken/asynq"

	eghelper "github.com/maliboot/mago/helper"
	"github.com/maliboot/mago/queuex"
	"github.com/maliboot/mago/queuex/dbq"
	"github.com/maliboot/mago/queuex/logq"
	"github.com/maliboot/mago/queuex/redisq"
)

type QueueConf struct {
	Default string `yaml:"default"`
	Sqlite  struct {
	} `yaml:"sqlite"`
	Mysql struct {
		Dsn string `yaml:"dsn"`
	} `yaml:"mysql"`
	Redis struct {
		Dsn string `yaml:"dsn"`
	} `yaml:"redis"`
	Concurrency        int    `yaml:"concurrency"`
	CacheDir           string `yaml:"cache_dir"`
	AggregatorMaxSize  int    `yaml:"aggregator_max_size"`
	AggregatorMaxDelay int    `yaml:"aggregator_max_delay"`
	Ctx                any

	srv queuex.Server
	clt queuex.Client
}

func (q *QueueConf) initialize() error {
	if q.srv != nil {
		return nil
	}

	if q.Default == "" {
		q.Default = "sqlite"
	}
	if q.Concurrency == 0 {
		q.Concurrency = runtime.NumCPU() * 3 // 不超过4倍
	}
	aggregatorEnable := true
	if q.AggregatorMaxSize <= 0 || q.AggregatorMaxDelay <= 0 {
		aggregatorEnable = false
	}
	aggregatorMaxDelay := time.Duration(q.AggregatorMaxDelay) * time.Millisecond

	exist := eghelper.FileExists(q.CacheDir)
	if !exist {
		err := os.MkdirAll(q.CacheDir, 0750)
		if err != nil {
			logq.Log.Fatalf("QueueCacheDir_0[%s]Err: %v", q.CacheDir, err)
		}
	}

	switch q.Default {
	case "sqlite":
		absPath, err := filepath.Abs(q.CacheDir)
		if err != nil {
			return fmt.Errorf("QueueConf.Sqlite获取绝对路径[%s]失败: %v", q.CacheDir, err)
		}
		q.srv = dbq.NewServer(absPath, q.Concurrency, q.AggregatorMaxSize, aggregatorMaxDelay)
		q.clt = dbq.NewClient(absPath, aggregatorEnable)
		break
	case "mysql":
		break
	case "redis":
		redisOpt, err := asynq.ParseRedisURI(q.Redis.Dsn)
		if err != nil {
			return fmt.Errorf("queue type %s dsn err", q.Default)
		}
		q.srv = redisq.NewServer(redisOpt, q.CacheDir, q.Concurrency, q.AggregatorMaxSize, aggregatorMaxDelay)
		q.clt = redisq.NewClient(redisOpt, aggregatorEnable)
		break
	default:
		return fmt.Errorf("queue type %s not support", q.Default)
	}

	return nil
}

// GetClient 返回队列客户端；队列服务尚未初始化时返回错误。
// 业务侧（如异步任务发布）用它做降级判断，不要用 Must 版本把进程打死。
func (q *QueueConf) GetClient() (queuex.Client, error) {
	if q.clt == nil {
		return nil, errors.New("队列服务未启动，不可以使用队列客户端")
	}

	return q.clt, nil
}

func (q *QueueConf) MustGetClient() queuex.Client {
	clt, err := q.GetClient()
	if err != nil {
		hlog.Fatal(err)
	}

	return clt
}

func (q *QueueConf) MustGetServer() queuex.Server {
	if q.srv == nil {
		err := q.initialize()
		if err != nil {
			hlog.Fatal(err)
		}
	}
	return q.srv
}
