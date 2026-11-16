package vo

import "github.com/maliboot/mago"

// UserTokenVO 是登录成功返回的令牌视图。
type UserTokenVO struct {
	mago.AbstractViewObject
	Token  string `json:"token"`
	Expire string `json:"expire"`
}

// NewUserTokenVO 构造令牌视图，转换逻辑挂在 VO 上，调用方只传值。
func NewUserTokenVO(token, expire string) *UserTokenVO {
	return &UserTokenVO{
		Token:  token,
		Expire: expire,
	}
}
