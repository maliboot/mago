package helper

import (
	"reflect"
	"testing"
)

func TestToLowerCamelJson(t *testing.T) {
	type args struct {
		snakeJson []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "ToLowerCamelJson-array",
			args: args{snakeJson: []byte(`["123", "my_name", {"my_age":{"my_id":1}}]`)},
			want: []byte(`["123","my_name",{"myAge":{"myId":1}}]`),
		},
		{
			name: "ToLowerCamelJson-object",
			args: args{snakeJson: []byte(`{"my_first":{"my_id":1}, "my_second": "haha"}`)},
			want: []byte(`{"myFirst":{"myId":1},"mySecond":"haha"}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToLowerCamelJson(tt.args.snakeJson)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToLowerCamelJson() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToLowerCamelJson() got = %v, want %v", got, tt.want)
			}
		})
	}
}
