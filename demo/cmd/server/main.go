package main

import (
	"context"
	"flag"
	"strings"

	"tests/conf"
	"tests/conf/server"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/maliboot/mago"
	"github.com/maliboot/mago/autoloadx"
	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/emqx"
	"github.com/maliboot/mago/helper"
	"github.com/maliboot/mago/queuex"
)

var (
	flagConf string
)

func init() {
	flag.StringVar(&flagConf, "f", "./conf.yml", "config path, eg: -f conf.yml")
}

// NewApp 汇总已构造的 Server engine，返回可运行的 App。
// 形参只允许配置、Server engine 与 conf.Handler；禁止出现 adapter 内实例（*UserController / *UserMcp …），
// 那些由 wire 装配进 conf.Handler（conf.InitHandler），本函数只做「绑定 + 汇总」。
func NewApp(c *config.Conf, hs *mago.Http, mqtt *emqx.MQTT, mcpSrv *server.McpServer, queue queuex.Server, handler *conf.Handler) *mago.App {
	// adapter 入口（HTTP 路由 + MCP 协议入口）绑定到 engine
	handler.Bind(hs, mcpSrv, c)

	servers := make([]mago.Server, 0, 2)
	servers = append(servers, hs)
	if mqtt != nil { // nil = 本次不启用
		servers = append(servers, mqtt)
	}
	if queue != nil {
		servers = append(servers, queue)
	}

	return mago.New(c.AppName, servers)
}

// main 只负责进程生命周期：解析参数、加载配置、取 handler 图、造 Server engine、启动。
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

	// handler 图：wire 生成，缺依赖或类型冲突在生成期就报错
	handler, err := conf.InitHandler(c)
	if err != nil {
		hlog.Fatalf("依赖装配失败: %v", err)
	}

	// Server engine：HTTP / MQTT / MCP（MCP 引擎把各上下文的工具声明装进去）
	hs := server.NewHttpServer(c)
	mqttSrv := server.NewMQTTServer(c)
	mcpSrv := server.NewMcpServer(c, handler.McpRegistrars()...)
	queueSrv := server.NewQueueServer(c, handler.TaskRegistrars()...)

	// os.runtime.log
	if err := autoloadx.NewOSRuntime(c, nil, func(ctx context.Context) {}).Monitor(); err != nil {
		hlog.Fatalf("OSRuntimeErr: %+v", err)
	}

	// start
	if err := NewApp(c, hs, mqttSrv, mcpSrv, queueSrv, handler).Run(); err != nil {
		if strings.Contains(err.Error(), "terminated") {
			return
		}
		hlog.Fatal(err)
	}
}
