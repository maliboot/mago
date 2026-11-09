package user

import (
	"time"

	"github.com/maliboot/mago"
)

type Model struct {
	mago.AbstractDataObject

	Email             string     `gorm:"column:email;not null;uniqueIndex:uk_email"`    // 邮箱
	PasswordHash      string     `gorm:"column:password_hash;not null"`                 // 密码（bcrypt，永不回显）
	Name              string     `gorm:"column:name;not null"`                          // 名称
	Role              int8       `gorm:"column:role;not null"`                          // 身份（管理员/员工）；无 default tag——RoleAdmin=0 是 Go 零值，带 default 会被 GORM 跳过写入落到列默认1(employee)，引导admin降权bug
	IsActive          int8       `gorm:"column:is_active;not null"`                     // 业务状态（启用/停用）；无 default tag——停用=0 是 Go 零值，带 default:1 会让 Create 时显式传 false 被跳过落 1（同类 Role bug）
	AlertEmailEnabled int8       `gorm:"column:alert_email_enabled;not null;default:0"` // 是否接收预警邮件【开/关，默认关】
	WorkCountry       string     `gorm:"column:work_country;not null"`                  // 工作国家（B53①/FR-090 必输）
	WorkTimezone      string     `gorm:"column:work_timezone;not null"`                 // 工作时区（IANA；FR-091 不参与判定）
	LastLoginAt       *time.Time `gorm:"column:last_login_at"`                          // 最后登录时间
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime"`              // 创建时间
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime"`              // 最后更新时间
	IsDeleted         int8       `gorm:"column:is_deleted;not null;default:0"`          // 软删（数据层机制，独立于启用/停用）
}

// TableName 表名单数纪律。
func (Model) TableName() string { return "user" }

func (Model) DatabaseName() string {
	return "mali"
}
