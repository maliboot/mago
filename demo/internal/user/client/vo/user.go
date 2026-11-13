package vo

import (
	"github.com/maliboot/mago"

	"tests/internal/user/client/enum"
)

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

// LoginSubjectVO 是登录主体：令牌载荷需要的身份与业务字段，不含令牌本身。
// 令牌由入口层的 JWT 中间件签发，application 只负责校验凭证（§12.5）。
type LoginSubjectVO struct {
	mago.AbstractViewObject
	UserID       uint   `json:"userId"`
	Email        string `json:"email"`
	TokenVersion int    `json:"tokenVersion"`
}

// NewLoginSubjectVO 构造登录主体；TokenVersion 用于「改密 / 停用后旧令牌失效」。
func NewLoginSubjectVO(userID uint, email string, tokenVersion int) *LoginSubjectVO {
	return &LoginSubjectVO{
		UserID:       userID,
		Email:        email,
		TokenVersion: tokenVersion,
	}
}

// UserVO 是当前用户视图，也是跨上下文可见的用户数据。
type UserVO struct {
	mago.AbstractViewObject
	Email  string          `json:"email"`
	Status enum.UserStatus `json:"status"`
}

// NewUserVO 构造用户视图，领域标记由调用方先转成对外枚举。
func NewUserVO(email string, status enum.UserStatus) *UserVO {
	return &UserVO{
		Email:  email,
		Status: status,
	}
}
