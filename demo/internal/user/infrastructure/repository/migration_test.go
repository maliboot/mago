package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/logx"

	"tests/internal/user/infrastructure/do/db"
)

// TestMain 初始化日志：GORM 的 logger 依赖 logx 的包级实例，不初始化会在建表时 panic。
func TestMain(m *testing.M) {
	logDir, err := os.MkdirTemp("", "mago-repo-test-")
	if err != nil {
		panic(err)
	}

	logx.NewLogRus("test", logDir)

	code := m.Run()
	_ = os.RemoveAll(logDir)
	os.Exit(code)
}

// newMigratorConf 构造指向独立内存库的配置，避免测试互相污染。
func newMigratorConf(t *testing.T) *config.Conf {
	t.Helper()

	dsn := fmt.Sprintf("file:seed_%d?mode=memory&cache=shared", time.Now().UnixNano())

	return &config.Conf{
		AppName: "migration-test",
		Databases: map[string]*config.DataBaseConf{
			"mali": {Type: "sqlite", Dsn: dsn, SingularTable: true},
		},
	}
}

func TestMigratorSeedsZhangSan(t *testing.T) {
	conf := newMigratorConf(t)
	migrator := NewMigrator(conf)
	ctx := context.Background()

	result, err := migrator.Run(ctx)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !result.TableReady || !result.SeedCreated {
		t.Fatalf("首次迁移应建表并新建种子用户，实际 %+v", result)
	}
	if result.SeedEmail != SeedUserEmail {
		t.Fatalf("种子邮箱 = %q", result.SeedEmail)
	}

	// 落库校验：名称、邮箱、启用状态、密码可校验
	var do db.User
	if err = migrator.users.DB(do).WithContext(ctx).Where("email = ?", SeedUserEmail).First(&do).Error; err != nil {
		t.Fatalf("查询种子用户失败: %v", err)
	}
	if do.Name != SeedUserName {
		t.Fatalf("名称 = %q, 期望 %q", do.Name, SeedUserName)
	}
	if do.IsActive != 1 {
		t.Fatalf("IsActive = %d, 期望 1", do.IsActive)
	}
	if !migrator.hasher.Verify(ctx, SeedUserPassword, do.PasswordHash) {
		t.Fatal("初始密码校验失败")
	}
}

func TestMigratorIsIdempotent(t *testing.T) {
	conf := newMigratorConf(t)
	migrator := NewMigrator(conf)
	ctx := context.Background()

	if _, err := migrator.Run(ctx); err != nil {
		t.Fatalf("首次 Run: %v", err)
	}

	// 用户改过密码后重跑迁移，不应覆盖
	var do db.User
	if err := migrator.users.DB(do).WithContext(ctx).Where("email = ?", SeedUserEmail).First(&do).Error; err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if err := migrator.users.DB(do).WithContext(ctx).Model(&db.User{}).
		Where("id = ?", do.ID).Update("password_hash", "changed-by-user").Error; err != nil {
		t.Fatalf("改密码失败: %v", err)
	}

	result, err := NewMigrator(conf).Run(ctx)
	if err != nil {
		t.Fatalf("二次 Run: %v", err)
	}
	if result.SeedCreated {
		t.Fatal("种子用户已存在时不应重复创建")
	}

	var count int64
	if err = migrator.users.DB(do).WithContext(ctx).Model(&db.User{}).
		Where("email = ?", SeedUserEmail).Count(&count).Error; err != nil {
		t.Fatalf("计数失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("种子用户数量 = %d, 期望 1", count)
	}

	if err = migrator.users.DB(do).WithContext(ctx).Where("id = ?", do.ID).First(&do).Error; err != nil {
		t.Fatalf("复查失败: %v", err)
	}
	if do.PasswordHash != "changed-by-user" {
		t.Fatalf("迁移覆盖了用户改过的密码: %q", do.PasswordHash)
	}
}
