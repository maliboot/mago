package mq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/maliboot/mago/dispatcher"
	"github.com/maliboot/mago/queuex"

	"tests/internal/user/client/constant"
	"tests/internal/user/client/dto"
)

// UserConsumer 是用户上下文的 MQ 消费入口：只做「任务 → 用例」的协议转换，不写业务规则。
// 与 HTTP、MCP 入口并列，共用同一个 application 服务；任务登记由 server.NewQueueServer 统一完成。
type UserConsumer struct{}

// 非单例：入口适配器，由组合根装配一次。
func NewUserConsumer() *UserConsumer {
	return &UserConsumer{}
}

// RegisterTasks 把本上下文消费的任务登记到队列服务上（实现 server.TaskRegistrar）。
func (c *UserConsumer) RegisterTasks(srv queuex.Server) {
	srv.RegisterHandleFunc(constant.TaskUserRegistered, c.handleUserRegistered)
}

// handleUserRegistered 消费「用户已注册」事件：demo 只落日志，真实项目在这里发欢迎邮件、发券、建档案。
func (c *UserConsumer) handleUserRegistered(_ context.Context, task *dispatcher.Task) error {
	var evt dto.UserRegisteredEvent
	if err := json.Unmarshal(task.Payload, &evt); err != nil {
		return fmt.Errorf("用户注册事件解析失败: %w", err)
	}

	hlog.Infof("[queue] 已消费 %s: userID=%d email=%s at=%s",
		task.Name, evt.UserID, evt.Email, evt.At.Format("2006-01-02 15:04:05"))

	return nil
}
