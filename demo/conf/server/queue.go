package server

import (
	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/queuex"
)

// TaskRegistrar 由各上下文的 adapter/mq 实现：只声明本上下文消费的任务，不关心队列实现。
// 接口在使用方（server）定义，adapter 反向实现，避免 adapter 依赖本包。
type TaskRegistrar interface {
	RegisterTasks(srv queuex.Server)
}

// NewQueueServer 构造并初始化队列服务，然后把各上下文登记的任务注册上去。
// 队列实现由配置决定（README §12.3：优先 redis）；未配置队列时返回 nil，组合根跳过启动。
// 注意：MustGetServer 会同时初始化客户端，repository.Queue 之后才能取到它；
// copool.NewCoPool 也依赖同一份客户端，两者要成对启用。
func NewQueueServer(c *config.Conf, registrars ...TaskRegistrar) queuex.Server {
	if c.Queue == nil || c.Queue.Default == "" {
		return nil
	}

	srv := c.Queue.MustGetServer()

	for _, registrar := range registrars {
		if registrar == nil {
			continue
		}

		registrar.RegisterTasks(srv)
	}

	return srv
}
