package dto

import "github.com/maliboot/mago"

// UserAuthRequest 是登录、注册、当前用户三个用例共用的入参。
// dto 只放入参，出参统一放 client/vo。
type UserAuthRequest struct {
	mago.AbstractDTO
	Email          string `json:"email,omitempty" jsonschema:"登录 / 注册邮箱；user_me 不需要"`
	Password       string `json:"password,omitempty" jsonschema:"登录 / 注册密码；user_me 不需要"`
	Ext            string `json:"ext,omitempty" jsonschema:"扩展字段，可选"`
	IsExtendExpire bool   `json:"extendExpire,omitempty" jsonschema:"是否延长令牌有效期，可选"`
}

// EmailCodeLoginRequest 是邮箱 + 验证码登录入参。
// 与账号密码登录分成两个入口：两种登录方式的凭证字段不同，塞进同一个结构会逼前端传空字段。
type EmailCodeLoginRequest struct {
	mago.AbstractDTO
	Email string `json:"email" jsonschema:"登录邮箱"`
	Code  string `json:"code" jsonschema:"邮箱收到的 6 位验证码"`
}

// VerifyCodeRequest 是发送验证码的入参。
type VerifyCodeRequest struct {
	mago.AbstractDTO
	Email string `json:"email" jsonschema:"接收验证码的邮箱"`
}
