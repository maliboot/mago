package main

import (
	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/copool"
	"github.com/maliboot/mago/emqx"
	"github.com/maliboot/mago/queuex"
)

func NewCoPool(c *config.Conf, queueSrv queuex.Server, mqtt *emqx.MQTT) *copool.CoPool {
	ins := copool.NewCoPool(c, queueSrv, mqtt)
	return ins
}
