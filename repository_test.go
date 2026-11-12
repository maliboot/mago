package mago

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/logx"
	"gorm.io/gorm"
)

// TestMain 初始化日志：GORM logger 依赖 logx 的包级实例，不初始化在建表时会 panic。
func TestMain(m *testing.M) {
	logDir, err := os.MkdirTemp("", "mago-tx-test-")
	if err != nil {
		panic(err)
	}

	logx.NewLogRus("test", logDir)

	code := m.Run()
	_ = os.RemoveAll(logDir)
	os.Exit(code)
}

// txUser 是事务测试用的 DO：实现 DataObject 的最小集合。
type txUser struct {
	ID   int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Name string `gorm:"column:name"`
}

func (txUser) DatabaseName() string { return "mali" }
func (txUser) PrimaryKey() string   { return "id" }
func (d txUser) PrimaryValue() uint { return uint(d.ID) }
func (txUser) TableName() string    { return "tx_user" }

// txNote 是第二个仓储的 DO，用于验证事务跨仓储共享。
type txNote struct {
	ID   int64  `gorm:"column:id;primaryKey;autoIncrement"`
	Note string `gorm:"column:note"`
}

func (txNote) DatabaseName() string { return "mali" }
func (txNote) PrimaryKey() string   { return "id" }
func (d txNote) PrimaryValue() uint { return uint(d.ID) }
func (txNote) TableName() string    { return "tx_note" }

// newTxConf 构造指向独立内存库的配置，避免测试互相污染。
func newTxConf(t *testing.T) *config.Conf {
	t.Helper()

	dsn := fmt.Sprintf("file:tx_%d?mode=memory&cache=shared", time.Now().UnixNano())

	return &config.Conf{
		AppName: "tx-test",
		Databases: map[string]*config.DataBaseConf{
			"mali": {Type: "sqlite", Dsn: dsn, SingularTable: true},
		},
	}
}

// TestTxFromCtx 校验 ctx 取值与回退规则，不依赖数据库。
func TestTxFromCtx(t *testing.T) {
	fallback := &gorm.DB{}
	tx := &gorm.DB{}

	if got := TxFromCtx(context.Background(), fallback); got != fallback {
		t.Fatalf("ctx 无事务时应回退默认连接，实际 %p", got)
	}
	if got := TxFromCtx(WithTx(context.Background(), tx), fallback); got != tx {
		t.Fatalf("ctx 带事务时应优先返回事务句柄，实际 %p", got)
	}
	if got := TxFromCtx(WithTx(context.Background(), nil), fallback); got != fallback {
		t.Fatalf("ctx 里事务为 nil 时应回退默认连接，实际 %p", got)
	}
}

// TestWithTransactionCommits 校验提交后数据可见，且非事务方法仍走默认连接。
func TestWithTransactionCommits(t *testing.T) {
	conf := newTxConf(t)
	repo := &AbstractRepository[txUser]{Conf: conf}
	ctx := context.Background()

	if err := repo.DB(txUser{}).AutoMigrate(&txUser{}); err != nil {
		t.Fatalf("建表失败：%v", err)
	}

	if err := repo.WithTransaction(ctx, func(ctx context.Context) error {
		return repo.TxDB(ctx, txUser{}).Create(&txUser{Name: "张三"}).Error
	}); err != nil {
		t.Fatalf("事务应提交成功：%v", err)
	}

	var got txUser
	if err := repo.TxDB(ctx, txUser{}).First(&got).Error; err != nil {
		t.Fatalf("提交后应能查到数据：%v", err)
	}
	if got.Name != "张三" {
		t.Fatalf("数据内容不符：%+v", got)
	}

	// 老的非 ctx 便捷方法仍走默认连接，事务提交后同样可见
	if found := repo.Find(int(got.ID)); found == nil || found.Name != "张三" {
		t.Fatalf("Find 应查到已提交数据，实际 %+v", found)
	}
}

// TestWithTransactionRollsBack 校验 fn 返回错误时整体回滚，且错误原样返回。
func TestWithTransactionRollsBack(t *testing.T) {
	conf := newTxConf(t)
	repo := &AbstractRepository[txUser]{Conf: conf}
	ctx := context.Background()

	if err := repo.DB(txUser{}).AutoMigrate(&txUser{}); err != nil {
		t.Fatalf("建表失败：%v", err)
	}

	sentinel := errors.New("业务校验失败")
	err := repo.WithTransaction(ctx, func(ctx context.Context) error {
		if err := repo.TxDB(ctx, txUser{}).Create(&txUser{Name: "李四"}).Error; err != nil {
			return err
		}

		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("应返回原错误 %v，实际 %v", sentinel, err)
	}

	var count int64
	if err := repo.TxDB(ctx, txUser{}).Model(&txUser{}).Count(&count).Error; err != nil {
		t.Fatalf("计数失败：%v", err)
	}
	if count != 0 {
		t.Fatalf("回滚后应为 0 条，实际 %d", count)
	}
}

// TestWithTransactionSharesAcrossRepositories 校验不同仓储在同一 ctx 下共用同一事务。
func TestWithTransactionSharesAcrossRepositories(t *testing.T) {
	conf := newTxConf(t)
	users := &AbstractRepository[txUser]{Conf: conf}
	notes := &AbstractRepository[txNote]{Conf: conf}
	ctx := context.Background()

	if err := users.DB(txUser{}).AutoMigrate(&txUser{}); err != nil {
		t.Fatalf("建表失败：%v", err)
	}
	if err := notes.DB(txNote{}).AutoMigrate(&txNote{}); err != nil {
		t.Fatalf("建表失败：%v", err)
	}

	sentinel := errors.New("模拟后续步骤失败")
	err := users.WithTransaction(ctx, func(ctx context.Context) error {
		if err := users.TxDB(ctx, txUser{}).Create(&txUser{Name: "王五"}).Error; err != nil {
			return err
		}

		// 用另一个仓储的句柄查用户表：能查到未提交数据，说明拿到的是同一个事务
		var inside int64
		if err := notes.TxDB(ctx, txNote{}).Table("tx_user").Count(&inside).Error; err != nil {
			return err
		}
		if inside != 1 {
			return fmt.Errorf("事务内应可见未提交数据，实际 %d 条", inside)
		}

		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("应返回原错误 %v，实际 %v", sentinel, err)
	}

	var after int64
	if err := notes.TxDB(ctx, txNote{}).Table("tx_user").Count(&after).Error; err != nil {
		t.Fatalf("计数失败：%v", err)
	}
	if after != 0 {
		t.Fatalf("回滚后应为 0 条，实际 %d", after)
	}
}
