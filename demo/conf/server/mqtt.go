package server

import (
	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/emqx"
)

// NewMQTTServer 返回 MQTT Server；未配置或未启用时返回 nil，由组合根跳过启动（nil = 不启用）。
func NewMQTTServer(c *config.Conf) *emqx.MQTT {
	if c.MQTT == nil || !c.MQTT.Enable {
		return nil
	}

	mqtt := emqx.NewMQTT(c.MQTT)
	return mqtt
}
