package main

import (
	"context"
	"flag"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/maliboot/mago"
	"github.com/maliboot/mago/autoloadx"
	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/copool"
	"github.com/maliboot/mago/emqx"
	"github.com/maliboot/mago/helper"
	"github.com/maliboot/mago/queuex"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

var (
	flagConf string
)

func init() {
	flag.StringVar(&flagConf, "f", "./conf.yml", "config path, eg: -f conf.yml")
}

func NewApp(
	c *config.Conf,
	hs *mago.Http,
	mqtt *emqx.MQTT,
	queueSrv queuex.Server,
	gocoPool *copool.CoPool,
	mqttHandler mqtt.MessageHandler,
) *mago.App {
	servers := make([]mago.Server, 0)

	// copool
	if gocoPool != nil {
		servers = append(servers, gocoPool)
	}

	// http
	servers = append(servers, hs)

	// mqtt
	if mqtt != nil {
		mqtt.SetMsgHandler(mqttHandler)
		servers = append(servers, mqtt)
	}

	// os.runtime.log
	err := autoloadx.NewOSRuntime(c, gocoPool, func(ctx context.Context) {
		return
	}).Monitor()
	if err != nil {
		hlog.Errorf("OSRuntimeErr: %+v", err)
		return nil
	}

	// queue
	if queueSrv != nil {
		servers = append(servers, queueSrv)
	}

	// app
	return mago.New(c.AppName, servers)
}

func main() {
	defer helper.PanicRecover("项目初始化")
	flag.Parse()

	// 配置
	c := config.NewConf(config.WithConfFile(flagConf))
	if err := c.Bootstrap(false); err != nil {
		panic(err)
	}

	// uuid
	// ServerMachineId
	if c.Server.ServerMachineId == "" {
		c.Server.ServerMachineId = helper.GetSdkServerId("fetish")
	}

	// start
	if err := NewApp(c, NewHttpServer(c), nil, nil, nil, nil).Run(); err != nil {
		if strings.Contains(err.Error(), "terminated") {
			return
		}
		hlog.Fatal(err)
	}
}
