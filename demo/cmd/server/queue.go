package main

import (
	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/queuex"
)

func NewQueueServer(c *config.Conf) queuex.Server {
	return c.Queue.MustGetServer()
}
