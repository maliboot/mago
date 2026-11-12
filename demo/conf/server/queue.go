package server

import (
	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/queuex"
)

// NewQueueServer 构造队列 Server。注意：它会真正初始化队列客户端（sqlite/redis），
// 且 copool.NewCoPool 依赖同一份客户端，所以「队列 + 协程池」必须成对启用。
// demo 目前不启用队列，故两者都未加入 conf/wire.go 的依赖图；启用时一起加进 wire.Build。
func NewQueueServer(c *config.Conf) queuex.Server {
	if c.Queue == nil {
		return nil
	}

	return c.Queue.MustGetServer()
}
