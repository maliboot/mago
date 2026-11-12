package application

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/maliboot/mago"
	"github.com/maliboot/mago/dispatcher"
	"github.com/maliboot/mago/logx"

	"tests/internal/user/client/constant"
	"tests/internal/user/client/dto"
	"tests/internal/user/domain"
)

// TestMain 初始化日志：publishRegistered 的降级告警走 logx 包级实例，
// 生产环境由 config.Bootstrap 初始化，测试里要自己来一次。
func TestMain(m *testing.M) {
	logDir, err := os.MkdirTemp("", "mago-app-test-")
	if err != nil {
		panic(err)
	}

	logx.NewLogRus("test", logDir)

	code := m.Run()
	_ = os.RemoveAll(logDir)
	os.Exit(code)
}

// fakeTaskQueue 记录入队任务，避免用例依赖真实 redis。
type fakeTaskQueue struct {
	tasks []*dispatcher.Task
	err   error
}

func (f *fakeTaskQueue) Enqueue(_ context.Context, task *dispatcher.Task) error {
	f.tasks = append(f.tasks, task)

	return f.err
}

// fakeUserGateway 只实现 Register 路径用到的方法。
type fakeUserGateway struct {
	existing *domain.User
	created  *domain.User
}

func (f *fakeUserGateway) GetByEmail(_ context.Context, _ string) (*domain.User, error) {
	if f.existing != nil {
		return f.existing, nil
	}

	return nil, mago.NewError(mago.ErrUriNotFound).WithMsg("用户不存在")
}

func (f *fakeUserGateway) Create(_ context.Context, user *domain.User) error {
	user.ID = 42
	f.created = user

	return nil
}

func (f *fakeUserGateway) Update(context.Context, *domain.User) error { return nil }

// fakeHasher 固定返回同一个散列，不跑 bcrypt。
type fakeHasher struct{}

func (fakeHasher) Hash(context.Context, string) (string, error) { return "hash", nil }

func (fakeHasher) Verify(context.Context, string, string) bool { return true }

// TestRegisterEnqueuesUserRegistered 校验注册成功后把事件投给队列（任务名与载荷契约）。
func TestRegisterEnqueuesUserRegistered(t *testing.T) {
	queue := &fakeTaskQueue{}
	gateway := &fakeUserGateway{}
	service := &userService{users: gateway, hasher: fakeHasher{}, tasks: queue}

	res := service.Register(context.Background(), &dto.UserAuthRequest{Email: " New@Example.com ", Password: "secret123"})
	if err := res.Error(); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if gateway.created == nil {
		t.Fatal("应落库一个用户")
	}
	if len(queue.tasks) != 1 {
		t.Fatalf("应入队 1 个任务，实际 %d", len(queue.tasks))
	}

	task := queue.tasks[0]
	if task.Name != constant.TaskUserRegistered {
		t.Fatalf("任务名 = %q, 期望 %q", task.Name, constant.TaskUserRegistered)
	}

	var evt dto.UserRegisteredEvent
	if err := json.Unmarshal(task.Payload, &evt); err != nil {
		t.Fatalf("载荷解析失败: %v（载荷=%s）", err, task.Payload)
	}
	if evt.UserID != 42 {
		t.Fatalf("事件 userID = %d, 期望自增回填的 42", evt.UserID)
	}
	if evt.Email != "new@example.com" {
		t.Fatalf("事件 email = %q, 期望规范化后的 new@example.com", evt.Email)
	}
	if evt.At.Time.IsZero() {
		t.Fatal("事件 at 不应为零值")
	}
}

// TestRegisterSucceedsWhenQueueFails 校验队列不可用时注册仍然成功（只降级告警）。
func TestRegisterSucceedsWhenQueueFails(t *testing.T) {
	queue := &fakeTaskQueue{err: errors.New("redis down")}
	gateway := &fakeUserGateway{}
	service := &userService{users: gateway, hasher: fakeHasher{}, tasks: queue}

	res := service.Register(context.Background(), &dto.UserAuthRequest{Email: "a@b.com", Password: "secret123"})
	if err := res.Error(); err != nil {
		t.Fatalf("队列失败不应影响注册: %v", err)
	}
	if gateway.created == nil {
		t.Fatal("用户仍应落库")
	}
	if len(queue.tasks) != 1 {
		t.Fatalf("应尝试入队 1 次，实际 %d", len(queue.tasks))
	}
}

// TestRegisterWithoutQueue 校验未注入任务端口时注册照常工作（demo 关闭队列的场景）。
func TestRegisterWithoutQueue(t *testing.T) {
	gateway := &fakeUserGateway{}
	service := &userService{users: gateway, hasher: fakeHasher{}}

	if err := service.Register(context.Background(), &dto.UserAuthRequest{Email: "a@b.com", Password: "secret123"}).Error(); err != nil {
		t.Fatalf("无队列时注册应成功: %v", err)
	}
	if gateway.created == nil {
		t.Fatal("用户应落库")
	}
}
