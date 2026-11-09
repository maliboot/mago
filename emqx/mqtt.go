package emqx

import (
	"context"
	"fmt"
	"github.com/maliboot/mago/config"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTT struct {
	c *config.MQTT

	client     mqtt.Client
	msgHandler mqtt.MessageHandler
	status     bool
}

func NewMQTT(c *config.MQTT) *MQTT {
	if !c.Enable {
		return nil
	}

	//mqtt.DEBUG = NewMQTTLog(hlog.LevelDebug)
	mqtt.ERROR = NewMQTTLog(hlog.LevelError)
	mqtt.WARN = NewMQTTLog(hlog.LevelWarn)
	mqtt.CRITICAL = NewMQTTLog(hlog.LevelFatal)
	return &MQTT{c: c}
}

func (m *MQTT) SetMsgHandler(handler mqtt.MessageHandler) {
	m.msgHandler = handler
}

func (m *MQTT) GetStatus() bool {
	return m.status
}

func (m *MQTT) Start(_ context.Context) error {
	opts := mqtt.NewClientOptions().AddBroker(fmt.Sprintf("%s:%d", m.c.Ip, m.c.Port)).SetClientID(fmt.Sprintf("%s_%d", "emqx_sdk", time.Now().Unix()))

	opts.SetKeepAlive(60 * time.Second)
	// 设置消息回调处理函数
	if m.msgHandler != nil {
		opts.SetDefaultPublishHandler(m.msgHandler)
	}
	//opts.SetPingTimeout(1 * time.Second)
	opts.SetUsername(m.c.Username)
	opts.SetPassword(m.c.Password)

	c := mqtt.NewClient(opts)
	token := c.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}

	m.client = c
	m.status = true
	return nil
}

func (m *MQTT) Stop(_ context.Context) error {
	// 断开连接
	time.Sleep(time.Second * 3)
	if m.client != nil {
		m.client.Disconnect(250)
	}
	return nil
}

// Subscribe 订阅主题
func (m *MQTT) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) error {
	if token := m.client.Subscribe(topic, qos, callback); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// Unsubscribe 取消订阅
func (m *MQTT) Unsubscribe(topic string) error {
	if token := m.client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

// Publish 发布消息
func (m *MQTT) Publish(topic string, qos byte, retained bool, data interface{}) bool {
	return m.RawPublish(topic, qos, retained, data)
}

// RawPublish 发布原生消息
func (m *MQTT) RawPublish(topic string, qos byte, retained bool, payload interface{}) bool {
	token := m.client.Publish(topic, qos, retained, payload)
	if token.Error() != nil {
		hlog.Errorf("mqtt.publish失败.0,topic:%s, payload:%s, err:%+v", topic, payload, token.Error())
		return false
	}
	res := token.WaitTimeout(time.Second * 5)
	if !res {
		hlog.Errorf("mqtt.publish失败.1,topic:%s, payload:%s, err:%+v", topic, payload, token.Error())
	}
	return res
}
