package dto

import "github.com/maliboot/mago"

// EmailCodeLoginRequest 是邮箱 + 验证码登录入参。
// 与账号密码登录分成两个入口：两种登录方式的凭证字段不同，塞进同一个结构会逼前端传空字段。
type EmailCodeLoginRequest struct {
	mago.AbstractDTO
	Email string `json:"email" jsonschema:"登录邮箱"`
	Code  string `json:"code" jsonschema:"邮箱收到的 6 位验证码"`
}
