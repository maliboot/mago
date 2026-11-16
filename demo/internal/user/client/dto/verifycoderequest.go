package dto

import "github.com/maliboot/mago"

// VerifyCodeRequest 是发送验证码的入参。
type VerifyCodeRequest struct {
	mago.AbstractDTO
	Email string `json:"email" jsonschema:"接收验证码的邮箱"`
}
