package vo

import "github.com/maliboot/mago"

// MigrationResultVO 描述一次迁移的结果。
// 跨层共用 struct：infrastructure/repository 产出、adapter/script 消费，
// 按 README §18.7 优先落 client/vo，让共用方只依赖契约包而非某层内部包。
type MigrationResultVO struct {
	mago.AbstractViewObject
	// TableReady 表示用户表已就绪。
	TableReady bool `json:"tableReady"`
	// SeedCreated 表示本次新建了种子用户（false 代表已存在、跳过）。
	SeedCreated bool `json:"seedCreated"`
	// SeedEmail 是种子用户邮箱。
	SeedEmail string `json:"seedEmail"`
}
