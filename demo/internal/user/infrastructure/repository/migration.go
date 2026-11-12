package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/maliboot/mago"
	"github.com/maliboot/mago/config"

	"tests/internal/user/domain"
	"tests/internal/user/infrastructure/do/db"
)

// 种子用户的固定信息：迁移只保证「存在」，不覆盖已修改过的账号。
const (
	// SeedUserEmail 是默认用户的登录邮箱。
	SeedUserEmail = "zhangsan@example.com"
	// SeedUserName 是默认用户的名称。
	SeedUserName = "张三"
	// SeedUserPassword 是默认用户的初始密码，仅用于演示环境，上线前必须修改。
	SeedUserPassword = "Aa123456"
)

// MigrationResult 描述一次迁移的结果。
type MigrationResult struct {
	// TableReady 表示用户表已就绪。
	TableReady bool
	// SeedCreated 表示本次新建了种子用户（false 代表已存在、跳过）。
	SeedCreated bool
	// SeedEmail 是种子用户邮箱。
	SeedEmail string
}

// Migrator 执行用户上下文的数据迁移：建表 + 默认种子用户，幂等可重复执行。
// 只做数据层的建成与补齐，不承载业务规则；入口脚本在 adapter/script/seeduser。
type Migrator struct {
	users  *User
	hasher *Password
}

// NewMigrator 由启动入口或脚本注入配置，内部装配用户 repository 与散列实现。
// 非单例：迁移脚本每次执行时装配，跑完即结束。
func NewMigrator(conf *config.Conf) *Migrator {
	return &Migrator{
		users:  NewUser(conf),
		hasher: NewPassword(),
	}
}

// Run 执行迁移：先保证表结构存在，再补齐默认用户「张三」。
func (m *Migrator) Run(ctx context.Context) (*MigrationResult, error) {
	var do db.User
	if err := m.users.DB(do).WithContext(ctx).AutoMigrate(&db.User{}); err != nil {
		return nil, fmt.Errorf("migration: 建表失败: %w", err)
	}

	result := &MigrationResult{TableReady: true, SeedEmail: SeedUserEmail}

	_, err := m.users.GetByEmail(ctx, SeedUserEmail)
	switch {
	case err == nil:
		// 已存在，保持原样，避免每次启动覆盖用户改过的密码
		return result, nil
	case !isRecordNotFound(err):
		return nil, err
	}

	hash, err := m.hasher.Hash(ctx, SeedUserPassword)
	if err != nil {
		return nil, fmt.Errorf("migration: 生成初始密码失败: %w", err)
	}

	user, err := domain.NewUser(SeedUserEmail, hash, SeedUserName)
	if err != nil {
		return nil, err
	}

	if err = m.users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("migration: 创建默认用户失败: %w", err)
	}

	result.SeedCreated = true

	return result, nil
}

// isRecordNotFound 判断错误是否为「记录不存在」。
func isRecordNotFound(err error) bool {
	var e mago.Error

	return errors.As(err, &e) && e.Code() == mago.ErrUriNotFound
}
