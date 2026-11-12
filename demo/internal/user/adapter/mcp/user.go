package mcp

import (
	"context"
	"net/http"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"tests/internal/user/application"
	"tests/internal/user/client/dto"
	"tests/internal/user/client/vo"
)

// 工具名按「上下文_动作」命名，避免与其他上下文的工具冲突。
const (
	ToolLogin    = "user_login"
	ToolRegister = "user_register"
	ToolMe       = "user_me"
	ToolSendCode = "user_send_verify_code"

	serverName    = "mago-user-mcp"
	serverVersion = "1.0.0"
)

// UserMcp 是用户上下文的 MCP 入口：用官方 Go SDK 注册工具，与 HTTP 入口共用同一个应用服务。
// 入参复用 client/dto，出参复用 client/vo，本层不定义任何业务结构。
type UserMcp struct {
	service application.UserService
	server  *sdk.Server
}

// 非单例：入口适配器，由组合根装配一次。
func NewUserMcp(service application.UserService) *UserMcp {
	m := &UserMcp{service: service}
	m.server = sdk.NewServer(&sdk.Implementation{Name: serverName, Version: serverVersion}, nil)
	m.registerTools()

	return m
}

// Handler 返回 MCP 的 Streamable HTTP 处理器，由 cmd 挂到现有 HTTP 服务上。
func (m *UserMcp) Handler() http.Handler {
	return sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return m.server }, nil)
}

// Server 暴露底层 SDK server，便于测试用内存传输直连。
func (m *UserMcp) Server() *sdk.Server {
	return m.server
}

func (m *UserMcp) registerTools() {
	sdk.AddTool(m.server, &sdk.Tool{
		Name:        ToolLogin,
		Description: "用邮箱和密码登录，返回访问令牌。",
	}, m.login)

	sdk.AddTool(m.server, &sdk.Tool{
		Name:        ToolRegister,
		Description: "注册新账号。",
	}, m.register)

	sdk.AddTool(m.server, &sdk.Tool{
		Name:        ToolMe,
		Description: "按用户 ID（userId）查询用户信息。",
	}, m.me)

	sdk.AddTool(m.server, &sdk.Tool{
		Name:        ToolSendCode,
		Description: "向邮箱发送 6 位登录验证码。",
	}, m.sendVerifyCode)
}

// login 复用 HTTP 入口的入参与出参，只做协议映射。
func (m *UserMcp) login(ctx context.Context, _ *sdk.CallToolRequest, in dto.UserAuthRequest) (*sdk.CallToolResult, *vo.UserTokenVO, error) {
	res := m.service.Login(ctx, &in)
	if err := res.Error(); err != nil {
		return nil, nil, err
	}

	return nil, res.Data, nil
}

// register 成功时没有业务返回值，用空结构体占位。
func (m *UserMcp) register(ctx context.Context, _ *sdk.CallToolRequest, in dto.UserAuthRequest) (*sdk.CallToolResult, struct{}, error) {
	res := m.service.Register(ctx, &in)

	return nil, struct{}{}, res.Error()
}

func (m *UserMcp) me(ctx context.Context, _ *sdk.CallToolRequest, in dto.UserAuthRequest) (*sdk.CallToolResult, *vo.UserVO, error) {
	res := m.service.Me(ctx, in.UserId)
	if err := res.Error(); err != nil {
		return nil, nil, err
	}

	return nil, res.Data, nil
}

// sendVerifyCode 复用 HTTP 入口的入参，成功时没有业务返回值。
func (m *UserMcp) sendVerifyCode(ctx context.Context, _ *sdk.CallToolRequest, in dto.VerifyCodeRequest) (*sdk.CallToolResult, struct{}, error) {
	res := m.service.SendVerifyCode(ctx, &in)

	return nil, struct{}{}, res.Error()
}
