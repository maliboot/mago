package mago

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"gorm.io/gorm/schema"
)

type DateTime struct {
	time.Time
}

func (t *DateTime) MarshalJSON() ([]byte, error) {
	if !t.Time.IsZero() {
		return json.Marshal(fmt.Sprintf("%s", t.Format("2006-01-02 15:04:05")))
	}
	return json.Marshal(nil)
}

func (t *DateTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	if data[0] == '"' {
		data = data[1:]
	}
	dataLen := len(data)
	if data[dataLen-1] == '"' {
		data = data[:dataLen-1]
	}
	myTime, err := time.ParseInLocation("2006-01-02 15:04:05", string(data), time.Now().Location())
	if err != nil {
		return err
	}

	t.Time = myTime
	return nil
}

func (t *DateTime) Scan(_ context.Context, _ *schema.Field, _ reflect.Value, fieldValue interface{}) error {
	if fieldValue == nil {
		return nil
	}
	value, ok := fieldValue.(time.Time)
	if ok {
		*t = DateTime{Time: value}
		return nil
	}
	return fmt.Errorf("can not convert %v to timestamp", fieldValue)
}

func (t *DateTime) Value(_ context.Context, _ *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	var zeroTime time.Time
	if t.Time.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return t.Time, nil
}
