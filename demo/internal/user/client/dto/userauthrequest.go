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
