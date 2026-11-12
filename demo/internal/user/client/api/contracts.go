package api

import (
	"net/http"

	"github.com/maliboot/mago"
)

// ActionContract 登记一个无法用资源表达的写操作（动作类接口）。
// 只有登记在 Actions 里的动作路径才允许存在，契约测试会校验（见 adapter/http 的路由契约测试）。
type ActionContract struct {
	// Method 是 HTTP 方法。
	Method string
	// Path 是完整路径，含 /api/v1/user 前缀。
	Path string
	// Name 是动作名，与用例方法对应。
	Name string
	// Purpose 说明动作用途，供评审与文档生成使用。
	Purpose string
	// Request 是入参类型（client/dto 中的类型名）。
	Request string
	// Errors 是可能返回的错误码。
	Errors []mago.ErrorCode
}

// Actions 是用户上下文允许的动作类接口清单。
// 新增动作路径前先在此登记，未登记的动作路径视为违反 §7.5。
var Actions = []ActionContract{
	{
		Method:  http.MethodPost,
		Path:    "/api/v1/user/users/verify-code",
		Name:    "sendVerifyCode",
		Purpose: "向邮箱发送 6 位登录验证码",
		Request: "dto.VerifyCodeRequest",
		Errors:  []mago.ErrorCode{mago.ErrInvalidParams, mago.ErrServerError},
	},
}
