package vo

import "github.com/maliboot/mago"

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
