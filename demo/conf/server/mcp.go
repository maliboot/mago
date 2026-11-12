package server

import (
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/maliboot/mago/config"
)

// serverVersion 是 MCP 引擎的版本号，出现在 initialize 握手的 serverInfo 里。
const serverVersion = "1.0.0"

// ToolRegistrar 由各上下文的 adapter/mcp 实现：只声明本上下文的工具，不关心协议引擎。
// 接口在使用方（server）定义，adapter 反过来实现，避免 adapter 依赖本包。
type ToolRegistrar interface {
	RegisterTools(srv *sdk.Server)
}

// McpServer 是 MCP 协议引擎：官方 go-sdk 的 Server + Streamable HTTP handler。
// 它是服务端基础设施，与 hertz engine 并列，由组合根构造并挂到同一台 HTTP 服务上（路径 /mcp）。
// 各上下文的工具声明留在 adapter/mcp，本类型只负责装配与传输。
type McpServer struct {
	srv     *sdk.Server
	handler http.Handler
}

// NewMcpServer 构造协议引擎，并把各上下文声明的工具注册进去。
// 引擎元信息取自应用配置，工具清单来自入参 registrar，本函数不写任何工具定义。
func NewMcpServer(c *config.Conf, registrars ...ToolRegistrar) *McpServer {
	srv := sdk.NewServer(&sdk.Implementation{Name: c.AppName + "-mcp", Version: serverVersion}, nil)

	for _, registrar := range registrars {
		if registrar == nil {
			continue
		}

		registrar.RegisterTools(srv)
	}

	return &McpServer{
		srv: srv,
		handler: sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server {
			return srv
		}, nil),
	}
}

// Handler 返回 Streamable HTTP 处理器，由 conf.Handler.Bind 挂到 HTTP engine 的 /mcp 路径。
func (m *McpServer) Handler() http.Handler {
	return m.handler
}

// Server 暴露底层 SDK server，便于测试用内存传输直连。
func (m *McpServer) Server() *sdk.Server {
	return m.srv
}
