//go:build dblive

// 真实 MySQL 联调：只在显式指定 tag 时运行。
//
//	go test -tags dblive -run TestTxLive -v -timeout 5m .
//	go test -tags dblive -run TestDateTimeLive -v -timeout 5m .
//
// 连接参数从仓库根 runtime/db.txt（KEY=VALUE）读取，代码里不落任何口令。
// 只操作带 mago_live_ 前缀的临时表，用例结束 DROP，不碰业务表。
package mago

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maliboot/mago/config"
	"gorm.io/gorm"
)

// liveDBConfPath 是凭据文件相对包目录的位置（包在仓库根）。
const liveDBConfPath = "runtime/db.txt"

// liveDBConf 是联调所需的连接参数。
type liveDBConf struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

// loadLiveDBConf 解析 KEY=VALUE 形式的配置文件。
func loadLiveDBConf(t *testing.T) liveDBConf {
	t.Helper()

	bs, err := os.ReadFile(liveDBConfPath)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", liveDBConfPath, err)
	}

	values := make(map[string]string)
	for _, line := range strings.Split(string(bs), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	port, err := strconv.Atoi(values["DB_PORT"])
	if err != nil {
		t.Fatalf("DB_PORT 解析失败: %v", err)
	}

	conf := liveDBConf{
		Host:     values["DB_HOST"],
		Port:     port,
		User:     values["DB_USER"],
		Password: values["DB_PASSWORD"],
		Name:     values["DB_NAME"],
	}
	if conf.Host == "" || conf.User == "" || conf.Name == "" {
		t.Fatalf("%s 缺少 DB_HOST/DB_USER/DB_NAME", liveDBConfPath)
	}

	return conf
}

// newLiveDBConf 构造指向真实 MySQL 的配置；超时写进 DSN，避免服务端抖動时用例挂死。
func newLiveDBConf(t *testing.T) *config.Conf {
	t.Helper()

	c := loadLiveDBConf(t)
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=Local&charset=utf8mb4&timeout=10s&readTimeout=20s&writeTimeout=20s",
		c.User, c.Password, c.Host, c.Port, c.Name,
	)

	return &config.Conf{
		AppName: "mago-live-db",
		Databases: map[string]*config.DataBaseConf{
			"mali": {Type: "mysql", Dsn: dsn, SingularTable: true},
		},
	}
}

// openLiveDB 打开连接并做一次探活。
func openLiveDB(t *testing.T) *gorm.DB {
	t.Helper()

	dc := newLiveDBConf(t).Databases["mali"]
	db, err := dc.GetDB()
	if err != nil {
		t.Fatalf("连接 MySQL 失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		t.Fatalf("MySQL 探活失败: %v", err)
	}

	return db
}

// resetLiveTable 建一张干净的临时表，并在用例结束（或失败）时删除。
func resetLiveTable(t *testing.T, db *gorm.DB, model any) {
	t.Helper()

	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		t.Fatalf("解析表名失败: %v", err)
	}
	table := stmt.Schema.Table

	t.Cleanup(func() {
		if err := db.Exec("DROP TABLE IF EXISTS `" + table + "`").Error; err != nil {
			t.Logf("清理临时表 %s 失败: %v", table, err)
		}
	})

	if err := db.Exec("DROP TABLE IF EXISTS `" + table + "`").Error; err != nil {
		t.Fatalf("删除旧临时表 %s 失败: %v", table, err)
	}
	if err := db.AutoMigrate(model); err != nil {
		t.Fatalf("建表 %s 失败: %v", table, err)
	}
}

// liveTxUser / liveTxNote 是事务联调用的 DO，表名带 mago_live_ 前缀。
type liveTxUser struct {
	ID   int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Name string `gorm:"column:name;size:64"`
}

func (liveTxUser) DatabaseName() string { return "mali" }
func (liveTxUser) PrimaryKey() string   { return "id" }
func (d liveTxUser) PrimaryValue() uint { return uint(d.ID) }
func (liveTxUser) TableName() string    { return "mago_live_tx_user" }

type liveTxNote struct {
	ID   int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Note string `gorm:"column:note;size:64"`
}

func (liveTxNote) DatabaseName() string { return "mali" }
func (liveTxNote) PrimaryKey() string   { return "id" }
func (d liveTxNote) PrimaryValue() uint { return uint(d.ID) }
func (liveTxNote) TableName() string    { return "mago_live_tx_note" }

// TestTxLive 在真实 MySQL 上验证 AbstractRepository 的事务能力。
func TestTxLive(t *testing.T) {
	db := openLiveDB(t)
	conf := newLiveDBConf(t)
	users := &AbstractRepository[liveTxUser]{Conf: conf}
	notes := &AbstractRepository[liveTxNote]{Conf: conf}
	ctx := context.Background()

	resetLiveTable(t, db, &liveTxUser{})
	resetLiveTable(t, db, &liveTxNote{})

	t.Run("Commit", func(t *testing.T) {
		if err := users.WithTransaction(ctx, func(ctx context.Context) error {
			return users.TxDB(ctx, liveTxUser{}).Create(&liveTxUser{Name: "live-commit"}).Error
		}); err != nil {
			t.Fatalf("事务应提交成功: %v", err)
		}

		var got liveTxUser
		if err := users.TxDB(ctx, liveTxUser{}).Where("name = ?", "live-commit").First(&got).Error; err != nil {
			t.Fatalf("提交后应能查到数据: %v", err)
		}
		if got.ID == 0 {
			t.Fatalf("自增主键应回填: %+v", got)
		}
	})

	t.Run("Rollback", func(t *testing.T) {
		sentinel := errors.New("模拟业务失败")
		err := users.WithTransaction(ctx, func(ctx context.Context) error {
			if err := users.TxDB(ctx, liveTxUser{}).Create(&liveTxUser{Name: "live-rollback"}).Error; err != nil {
				return err
			}

			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Fatalf("应原样返回业务错误 %v，实际 %v", sentinel, err)
		}

		var count int64
		if err := users.TxDB(ctx, liveTxUser{}).Model(&liveTxUser{}).Where("name = ?", "live-rollback").Count(&count).Error; err != nil {
			t.Fatalf("计数失败: %v", err)
		}
		if count != 0 {
			t.Fatalf("回滚后应为 0 条，实际 %d", count)
		}
	})

	t.Run("SharedAcrossRepositories", func(t *testing.T) {
		sentinel := errors.New("模拟后续步骤失败")
		err := users.WithTransaction(ctx, func(ctx context.Context) error {
			if err := users.TxDB(ctx, liveTxUser{}).Create(&liveTxUser{Name: "live-shared"}).Error; err != nil {
				return err
			}

			// 另一个仓储用同一个 ctx：能看到未提交数据说明共用同一事务
			var inside int64
			if err := notes.TxDB(ctx, liveTxNote{}).Table("mago_live_tx_user").Where("name = ?", "live-shared").Count(&inside).Error; err != nil {
				return err
			}
			if inside != 1 {
				return fmt.Errorf("事务内应可见未提交数据，实际 %d 条", inside)
			}

			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Fatalf("应原样返回业务错误 %v，实际 %v", sentinel, err)
		}

		var after int64
		if err := notes.TxDB(ctx, liveTxNote{}).Table("mago_live_tx_user").Where("name = ?", "live-shared").Count(&after).Error; err != nil {
			t.Fatalf("计数失败: %v", err)
		}
		if after != 0 {
			t.Fatalf("回滚后应为 0 条，实际 %d", after)
		}
	})

	t.Run("FallbackOutsideTx", func(t *testing.T) {
		// 不在事务里：TxDB 应回退到默认连接，写入立即生效
		if err := users.TxDB(ctx, liveTxUser{}).Create(&liveTxUser{Name: "live-autocommit"}).Error; err != nil {
			t.Fatalf("自动提交写入失败: %v", err)
		}

		var count int64
		if err := users.TxDB(context.Background(), liveTxUser{}).Model(&liveTxUser{}).Where("name = ?", "live-autocommit").Count(&count).Error; err != nil {
			t.Fatalf("计数失败: %v", err)
		}
		if count != 1 {
			t.Fatalf("非事务写入应立即生效，实际 %d 条", count)
		}
	})
}

// liveDateTimeModel 复刻 dataobject.go 的时间字段写法，验证 MySQL 上的列类型与往返。
type liveDateTimeModel struct {
	ID          uint     `gorm:"column:id;primaryKey;autoIncrement"`
	Name        string   `gorm:"column:name;size:64"`
	PublishedAt DateTime `gorm:"column:published_at"`
	CreatedAt   DateTime `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   DateTime `gorm:"column:updated_at;autoUpdateTime"`
}

func (liveDateTimeModel) TableName() string { return "mago_live_datetime" }

// TestDateTimeLive 在真实 MySQL 上验证 DateTime 的 GORM 兼容修复（v1.31 的 Valuer/Scanner 路径）。
func TestDateTimeLive(t *testing.T) {
	db := openLiveDB(t)
	resetLiveTable(t, db, &liveDateTimeModel{})

	t.Run("ColumnType", func(t *testing.T) {
		var cols []struct {
			ColumnName string `gorm:"column:column_name"`
			DataType   string `gorm:"column:data_type"`
			IsNullable string `gorm:"column:is_nullable"`
		}
		if err := db.Raw(`SELECT column_name, data_type, is_nullable FROM information_schema.columns
			WHERE table_schema = DATABASE() AND table_name = ? ORDER BY ordinal_position`, "mago_live_datetime").Scan(&cols).Error; err != nil {
			t.Fatalf("查询列信息失败: %v", err)
		}
		if len(cols) != 5 {
			t.Fatalf("期望 5 列，实际 %d：%+v", len(cols), cols)
		}

		for _, col := range cols {
			switch col.ColumnName {
			case "published_at", "created_at", "updated_at":
				if col.DataType != "datetime" {
					t.Errorf("%s 列类型应为 datetime，实际 %s（GormDBDataType 未生效）", col.ColumnName, col.DataType)
				}
				if col.IsNullable != "YES" {
					t.Errorf("%s 应允许 NULL，实际 %s（零值语义会丢）", col.ColumnName, col.IsNullable)
				}
			}
		}
	})

	t.Run("ZeroWritesNull", func(t *testing.T) {
		row := liveDateTimeModel{Name: "live-zero"}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("Create: %v", err)
		}

		var isNull int
		if err := db.Raw("SELECT published_at IS NULL FROM mago_live_datetime WHERE id = ?", row.ID).Scan(&isNull).Error; err != nil {
			t.Fatalf("查询 NULL 失败: %v", err)
		}
		if isNull != 1 {
			t.Fatal("DateTime 零值应写成 NULL（Value 返回 nil）")
		}

		var got liveDateTimeModel
		if err := db.First(&got, row.ID).Error; err != nil {
			t.Fatalf("First: %v", err)
		}
		if !got.PublishedAt.Time.IsZero() {
			t.Fatalf("NULL 应扫描回零值，实际 %v", got.PublishedAt.Time)
		}
	})

	t.Run("RoundTrip", func(t *testing.T) {
		want := time.Date(2026, 11, 12, 9, 8, 7, 0, time.Local)
		row := liveDateTimeModel{Name: "live-roundtrip", PublishedAt: DateTime{Time: want}}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("Create: %v", err)
		}

		var got liveDateTimeModel
		if err := db.First(&got, row.ID).Error; err != nil {
			t.Fatalf("First: %v", err)
		}
		if !got.PublishedAt.Time.Equal(want) {
			t.Fatalf("时间往返不一致：want %v, got %v", want, got.PublishedAt.Time)
		}

		// 按秒比较，避免 MySQL datetime 无小数秒带来的误判
		if got.PublishedAt.Format("2006-01-02 15:04:05") != "2026-11-12 09:08:07" {
			t.Fatalf("格式化结果不符：%s", got.PublishedAt.Format("2006-01-02 15:04:05"))
		}
	})

	t.Run("AutoTimestamps", func(t *testing.T) {
		row := liveDateTimeModel{Name: "live-auto"}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("Create: %v", err)
		}

		// 以库里的值为准：内存值由 GORM 的 NowFunc 给出，可能带小数秒，库值截断到秒
		var created liveDateTimeModel
		if err := db.First(&created, row.ID).Error; err != nil {
			t.Fatalf("First: %v", err)
		}
		if created.CreatedAt.Time.IsZero() || created.UpdatedAt.Time.IsZero() {
			t.Fatalf("autoCreateTime/autoUpdateTime 未写入：%+v", created)
		}

		time.Sleep(1100 * time.Millisecond) // MySQL datetime 精确到秒，等一秒才能看出更新

		if err := db.Model(&liveDateTimeModel{}).Where("id = ?", row.ID).Update("name", "live-auto-updated").Error; err != nil {
			t.Fatalf("Update: %v", err)
		}

		var got liveDateTimeModel
		if err := db.First(&got, row.ID).Error; err != nil {
			t.Fatalf("First: %v", err)
		}
		if !got.CreatedAt.Time.Equal(created.CreatedAt.Time) {
			t.Fatalf("updated 时不应改动 created_at：want %v, got %v", created.CreatedAt.Time, got.CreatedAt.Time)
		}
		if !got.UpdatedAt.Time.After(created.CreatedAt.Time) {
			t.Fatalf("updated_at 应随更新前进：created %v, updated %v", created.CreatedAt.Time, got.UpdatedAt.Time)
		}
	})
}
