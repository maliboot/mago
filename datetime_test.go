package mago

import (
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// dateTimeModel 是验证 DateTime 与 GORM 协作的最小模型。
// 字段故意不叫 CreatedAt/UpdatedAt：GORM 会按字段名自动填充这两个名字的时间字段，
// 这里要验证的是「手动赋值 / 零值写 NULL」的行为。
type dateTimeModel struct {
	ID          uint
	Name        string
	PublishedAt DateTime
	RemovedAt   DateTime
}

// autoTimeModel 复刻 dataobject.go 里的自动时间字段写法。
type autoTimeModel struct {
	ID        uint
	Name      string
	CreatedAt DateTime `gorm:"autoCreateTime;type:timestamp null;comment:创建时间"`
	UpdatedAt DateTime `gorm:"autoUpdateTime;type:timestamp null;comment:更新时间"`
}

// newSQLiteDB 起一个内存库，避免依赖外部数据库。
func newSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		// 与 config.DataBaseConf 一致：时间截断到秒，避免内存值与库值差一秒
		NowFunc: func() time.Time { return time.Now().Truncate(time.Second) },
	})
	if err != nil {
		t.Fatalf("打开内存库失败: %v", err)
	}

	if err = db.AutoMigrate(&dateTimeModel{}, &autoTimeModel{}); err != nil {
		t.Fatalf("建表失败: %v", err)
	}

	return db
}

// TestDateTimeGormRoundTrip 校验 DateTime 能通过 GORM 写入并读回。
func TestDateTimeGormRoundTrip(t *testing.T) {
	db := newSQLiteDB(t)

	now := time.Date(2026, 11, 11, 10, 20, 30, 0, time.Local)
	want := dateTimeModel{Name: "with-time", PublishedAt: DateTime{Time: now}}

	if err := db.Create(&want).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}

	var got dateTimeModel
	if err := db.First(&got, want.ID).Error; err != nil {
		t.Fatalf("First: %v", err)
	}

	if !got.PublishedAt.Time.Equal(now) {
		t.Fatalf("读回时间 = %v, 期望 %v", got.PublishedAt.Time, now)
	}
	if !got.RemovedAt.Time.IsZero() {
		t.Fatalf("零值字段应读回零值，实际 %v", got.RemovedAt.Time)
	}
}

// TestDateTimeZeroWritesNull 校验零值写库为 NULL，而不是零时间。
func TestDateTimeZeroWritesNull(t *testing.T) {
	db := newSQLiteDB(t)

	row := dateTimeModel{Name: "zero-time"}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}

	var isNull int
	if err := db.Raw("SELECT published_at IS NULL FROM date_time_models WHERE id = ?", row.ID).Scan(&isNull).Error; err != nil {
		t.Fatalf("查询 NULL 失败: %v", err)
	}

	if isNull != 1 {
		t.Fatal("零值 DateTime 应写入 NULL")
	}
}

// TestDateTimeAutoCreate 校验 autoCreateTime / autoUpdateTime 仍能写入。
func TestDateTimeAutoCreate(t *testing.T) {
	db := newSQLiteDB(t)

	row := autoTimeModel{Name: "auto"}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}
	if row.CreatedAt.Time.IsZero() {
		t.Fatal("autoCreateTime 未写入")
	}

	first := row.CreatedAt.Time
	if err := db.Model(&row).Update("name", "auto-updated").Error; err != nil {
		t.Fatalf("Update: %v", err)
	}

	var got autoTimeModel
	if err := db.First(&got, row.ID).Error; err != nil {
		t.Fatalf("First: %v", err)
	}
	if !got.CreatedAt.Time.Equal(first) {
		t.Fatalf("CreatedAt 被改动: %v → %v", first, got.CreatedAt.Time)
	}
	if got.UpdatedAt.Time.IsZero() {
		t.Fatal("autoUpdateTime 未写入")
	}
}

// TestDateTimeColumnType 校验迁移出来的列仍是时间列。
func TestDateTimeColumnType(t *testing.T) {
	db := newSQLiteDB(t)

	var ddl string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE name = 'date_time_models'").Scan(&ddl).Error; err != nil {
		t.Fatalf("读取表结构失败: %v", err)
	}
	if !strings.Contains(strings.ToLower(ddl), "datetime") && !strings.Contains(strings.ToLower(ddl), "timestamp") {
		t.Fatalf("created_at 列类型不是时间列: %s", ddl)
	}
}

// TestDateTimeJSON 确认序列化格式未变：非零输出 "2006-01-02 15:04:05"，零值输出 null。
func TestDateTimeJSON(t *testing.T) {
	nonZero := DateTime{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.Local)}
	bs, err := nonZero.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(bs) != `"2026-01-02 03:04:05"` {
		t.Fatalf("序列化 = %s", bs)
	}

	var back DateTime
	if err = back.UnmarshalJSON(bs); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if !back.Time.Equal(nonZero.Time) {
		t.Fatalf("反序列化 = %v, 期望 %v", back.Time, nonZero.Time)
	}

	zeroVal := DateTime{}
	zero, err := zeroVal.MarshalJSON()
	if err != nil {
		t.Fatalf("零值 MarshalJSON: %v", err)
	}
	if string(zero) != "null" {
		t.Fatalf("零值序列化 = %s, 期望 null", zero)
	}
}
