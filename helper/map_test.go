package helper

import (
	"reflect"
	"testing"
)

func TestMapGet(t *testing.T) {
	type args struct {
		data map[string]interface{}
		key  string
	}
	type testCase[T any] struct {
		name string
		args args
		want T
	}
	d := map[string]interface{}{
		"id":   1,
		"name": "stone",
	}
	tests := []testCase[any]{
		{
			name: "mapGet.str",
			args: args{
				data: d,
				key:  "name",
			},
			want: "stone",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MapGet[string](tt.args.data, tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapGet() = %v, want %v", got, tt.want)
			}
		})
	}
}
