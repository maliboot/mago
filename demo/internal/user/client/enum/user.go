package enum

// UserStatus 是对外暴露的用户状态枚举，与领域内的 IsActive 标记一一对应。
// 直接用字面量，JSON 与工具 schema 都按 string 暴露，不需要再做格式化。
type UserStatus string

const (
	// UserStatusDisabled 已停用
	UserStatusDisabled UserStatus = "disabled"
	// UserStatusActive 启用中
	UserStatusActive UserStatus = "active"
)

// FromActive 把领域内的启用标记转换为对外枚举。
func FromActive(active bool) UserStatus {
	if active {
		return UserStatusActive
	}

	return UserStatusDisabled
}
