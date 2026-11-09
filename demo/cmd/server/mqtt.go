package main

import (
	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/emqx"
)

func NewMQTTServer(c *config.Conf) *emqx.MQTT {
	mqtt := emqx.NewMQTT(c.MQTT)
	return mqtt
}
