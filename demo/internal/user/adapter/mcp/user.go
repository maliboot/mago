package mcp

import (
	"context"

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
)

// UserMcp 是用户上下文的 MCP 入口：只声明本上下文的工具，与 HTTP 入口共用同一个应用服务。
// 协议引擎（sdk.Server、Streamable HTTP handler、/mcp 挂载）在 conf/server/mcp.go，本层不建引擎。
// 入参复用 client/dto，出参复用 client/vo，本层不定义任何业务结构。
type UserMcp struct {
	service application.UserService
}

// 非单例：入口适配器，由组合根装配一次。
func NewUserMcp(service application.UserService) *UserMcp {
	return &UserMcp{service: service}
}

// RegisterTools 把本上下文的工具注册到协议引擎上；由 server.NewMcpServer 在装配时调用。
func (m *UserMcp) RegisterTools(srv *sdk.Server) {
	sdk.AddTool(srv, &sdk.Tool{
		Name:        ToolLogin,
		Description: "用邮箱和密码登录，返回访问令牌。",
	}, m.login)

	sdk.AddTool(srv, &sdk.Tool{
		Name:        ToolRegister,
		Description: "注册新账号。",
	}, m.register)

	sdk.AddTool(srv, &sdk.Tool{
		Name:        ToolMe,
		Description: "按用户 ID（userId）查询用户信息。",
	}, m.me)

	sdk.AddTool(srv, &sdk.Tool{
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
