package mago

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// DateTime 是数据库与 JSON 共用的时间类型：
// 库内映射时间列，JSON 输出 "2006-01-02 15:04:05"，零值输出 null。
//
// GORM v1.31 起只认标准 driver.Valuer / sql.Scanner（旧的 schema.ValuerInterface、
// schema.ScannerInterface 已从 gorm.io/gorm/schema 移除），因此本类型实现这两个标准接口。
type DateTime struct {
	time.Time
}

// dateTimeLayouts 是 Scan 时尝试解析的时间格式，按从精确到宽松排列。
var dateTimeLayouts = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02",
}

// MarshalJSON 用值接收者：json.Marshal 对「非指针字段」（如 dto 里的 At DateTime）不会取地址，
// 指针接收者的套件方法不会被看到，序列化会退化成 time.Time 的 RFC3339（带纳秒），
// 而消费端按 "2006-01-02 15:04:05" 解析就会失败。
func (t DateTime) MarshalJSON() ([]byte, error) {
	if !t.Time.IsZero() {
		return json.Marshal(fmt.Sprintf("%s", t.Format("2006-01-02 15:04:05")))
	}

	return json.Marshal(nil)
}

func (t *DateTime) UnmarshalJSON(data []byte) error {
	raw := strings.Trim(string(data), `"`)
	if raw == "" || raw == "null" {
		return nil
	}

	myTime, err := time.ParseInLocation("2006-01-02 15:04:05", raw, time.Now().Location())
	if err != nil {
		return err
	}

	t.Time = myTime

	return nil
}

// GormDataType 声明 GORM 字段类型，避免 GORM 依据 driver.Valuer 的返回值反推成 time，
// 从而绕过本类型的 Value/Scan（零值写 NULL、字符串解析等语义会丢）。
//
// 必须返回 "time"（schema.Time）而不是 "datetime"：GORM 用这个值判断 autoCreateTime /
// autoUpdateTime 该写时间还是 unix 秒，返回非 Time 会让它把整数塞进时间列（MySQL 报 1292）。
// 真实列类型由 GormDBDataType 控制，仍是 datetime。
func (DateTime) GormDataType() string {
	return "time"
}

// GormDBDataType 按方言给出列类型，保证迁移出来的列仍是时间列。
func (DateTime) GormDBDataType(db *gorm.DB, _ *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql":
		return "datetime"
	case "postgres":
		return "timestamptz"
	case "sqlserver":
		return "datetime2"
	case "sqlite":
		return "datetime"
	default:
		return "datetime"
	}
}

// Value 实现 driver.Valuer：零值写 NULL，其余写 time.Time。
// 写入前截断到秒：datetime / timestamp 列只有秒精度，不截断时 MySQL 会四舍五入
// （如 12:00:00.504 存成 12:00:01），内存值与库值就会差一秒。
func (t DateTime) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}

	return t.Time.Truncate(time.Second), nil
}

// Scan 实现 sql.Scanner，兼容 time.Time、字符串与字节切片（MySQL 驱动两种都可能出现）。
func (t *DateTime) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		t.Time = time.Time{}
		return nil
	case time.Time:
		t.Time = v
		return nil
	case []byte:
		return t.parse(string(v))
	case string:
		return t.parse(v)
	default:
		return fmt.Errorf("mago: 无法把 %T 扫描为 DateTime", value)
	}
}

// parse 按预置格式解析字符串时间。
func (t *DateTime) parse(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "0000-00-00 00:00:00" {
		t.Time = time.Time{}
		return nil
	}

	for _, layout := range dateTimeLayouts {
		if parsed, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			t.Time = parsed
			return nil
		}
	}

	return fmt.Errorf("mago: 无法解析时间 %q", raw)
}
