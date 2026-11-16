package vo

import (
	"github.com/maliboot/mago"

	"tests/internal/user/client/enum"
)

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
