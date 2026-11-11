package domain

import (
	"strings"
	"time"

	"github.com/maliboot/mago"
)

// User 是用户聚合根，只表达业务状态与规则，不依赖数据库、HTTP 等技术实现。
// 字段导出便于各层显式映射；状态变更只走领域方法，不在外部直接赋值。
type User struct {
	ID           uint
	Email        string
	PasswordHash string
	Name         string
	IsActive     bool
	LastLoginAt  *time.Time
}

// NewUser 创建新用户，邮箱做规范化并校验必填项。
// 领域层只用标准库与框架的全局错误码，不自定义错误类型或错误变量。
func NewUser(email, passwordHash, name string) (*User, error) {
	email = NormalizeEmail(email)
	if email == "" {
		return nil, mago.NewError(mago.ErrInvalidParams).WithMsg("邮箱不能为空")
	}
	if passwordHash == "" {
		return nil, mago.NewError(mago.ErrInvalidParams).WithMsg("密码不能为空")
	}

	return &User{
		Email:        email,
		PasswordHash: passwordHash,
		Name:         strings.TrimSpace(name),
		IsActive:     true,
	}, nil
}

// RestoreUser 由 repository 从持久化数据恢复聚合，不做业务校验。
func RestoreUser(
	id uint,
	email, passwordHash, name string,
	isActive bool,
	lastLoginAt *time.Time,
) *User {
	return &User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
		IsActive:     isActive,
		LastLoginAt:  lastLoginAt,
	}
}

// NormalizeEmail 统一邮箱写法，避免大小写与空白造成重复账号。
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// CanLogin 判断账号当前是否允许登录。
func (u *User) CanLogin() error {
	if !u.IsActive {
		return mago.NewError(mago.ErrAuthFailed).WithMsg("账号已停用")
	}

	return nil
}

// MarkLoggedIn 记录一次成功登录，时间由调用方注入，便于测试。
func (u *User) MarkLoggedIn(at time.Time) error {
	if err := u.CanLogin(); err != nil {
		return err
	}

	u.LastLoginAt = &at
	return nil
}
