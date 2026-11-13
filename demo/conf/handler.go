package conf

import (
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/maliboot/mago"
	"github.com/maliboot/mago/config"

	"tests/conf/server"
	userhttp "tests/internal/user/adapter/http"
	usermcp "tests/internal/user/adapter/mcp"
	usermq "tests/internal/user/adapter/mq"
)

// Handler 汇总所有上下文的入口 handler（HTTP controller + MCP 工具声明 + MQ 任务声明），
// 由 wire 装配后注入组合根。它只持有 adapter 实例：不构造协议引擎、不启动、不写业务。
type Handler struct {
	user    *userhttp.UserController
	jwtAuth *userhttp.JWTAuth
	mcp     *usermcp.UserMcp
	mq      *usermq.UserConsumer
}

// Bind 接收 server engine，把各上下文的路由与协议入口挂上去；新增上下文只改这里。
func (h *Handler) Bind(hs *mago.Http, mcpSrv *server.McpServer, c *config.Conf) {
	// JWT 中间件的业务扩展由本上下文提供：身份键、动态作废与登录成功响应（§12.5）。
	// 必须在取 MiddlewareFunc() 之前替换，否则会挂到 autoloadx 的占位 Tenant 实现上。
	jwtConf := c.Server.Http.JWT["frontend"]
	jwtConf.ResetUserConfig(h.jwtAuth)

	v1 := hs.Group("/api/v1/user")
	authed := v1.Group("", jwtConf.MiddlewareFunc())

	h.user.Router(v1, authed)

	// MCP 是协议入口：独立路径 /mcp，不套资源路由；引擎与 handler 由 conf/server 提供
	hs.Any("/mcp", adaptor.HertzHandler(mcpSrv.Handler()))
}

// McpRegistrars 返回各上下文的 MCP 工具声明，供 server.NewMcpServer 装配协议引擎。
// 协议引擎在 conf/server/mcp.go，工具定义留在各上下文 adapter/mcp。
func (h *Handler) McpRegistrars() []server.ToolRegistrar {
	return []server.ToolRegistrar{h.mcp}
}

// TaskRegistrars 返回各上下文的 MQ 任务声明，供 server.NewQueueServer 装配队列服务。
// 队列实现在 conf/server/queue.go，任务处理器留在各上下文 adapter/mq。
func (h *Handler) TaskRegistrars() []server.TaskRegistrar {
	return []server.TaskRegistrar{h.mq}
}
