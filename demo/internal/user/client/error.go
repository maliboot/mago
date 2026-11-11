package client

import "github.com/maliboot/mago"

// 用户上下文对框架全局错误码的扩展，统一放在 client 包：本上下文与调用方共用。
// 错误码沿用框架规则：前三位是 HTTP 状态码，后三位是上下文内业务编号。
// domain 层不得引用本包，只能使用框架内置错误码。
const (
	// ErrUserDisabled 账号已停用（认证类，HTTP 401）
	ErrUserDisabled mago.ErrorCode = 401501
	// ErrEmailTaken 邮箱已被注册（参数类，HTTP 422）
	ErrEmailTaken mago.ErrorCode = 422101
)

func init() {
	mago.InitErrorCodeMsg(ErrUserDisabled, "账号已停用")
	mago.InitErrorCodeMsg(ErrEmailTaken, "邮箱已被注册")
}
