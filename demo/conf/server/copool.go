package server

import (
	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/copool"
	"github.com/maliboot/mago/emqx"
	"github.com/maliboot/mago/queuex"
)

// NewCoPool 构造协程池。注意：内部会调 conf.Queue.MustGetClient()，队列未初始化时直接 Fatal，
// 所以必须与 NewQueueServer 成对启用；demo 目前两者都未加入 conf/wire.go 的依赖图。
func NewCoPool(c *config.Conf, queueSrv queuex.Server, mqtt *emqx.MQTT) *copool.CoPool {
	ins := copool.NewCoPool(c, queueSrv, mqtt)
	return ins
}
