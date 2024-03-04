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

func TestToSnakeJson(t *testing.T) {
	type args struct {
		lowerCamelJson []byte
	}
	camelJson := `
{
  "data": {
    "msgList": [
      {
        "msgData": {
          "content": "我是群发-文本类型"
        },
        "type": 0
      },
      {
        "msgData": {
          "title": "我是群发title",
          "iconUrl": "https://wework.qpic.cn/wwpic3az/59411_H0x4p2RNQHmdDjh_1709274772/0",
          "linkUrl": "https://www.baidu.com",
          "desc": "我是群发desc"
        },
        "type": 13
      }
    ],
    "send_type": 0,
    "toIdList": [
      "7881303048045815",
      "7881302321987664",
      "7881300452930187"
    ]
  },
  "syncKey": "1709541456928768300",
  "type": 5400
}
`
	snakeJson := `{"data":{"msg_list":[{"msg_data":{"content":"我是群发-文本类型"},"type":0
      },{"msg_data":{"title":"我是群发title","icon_url":"https://wework.qpic.cn/wwpic3az/59411_H0x4p2RNQHmdDjh_1709274772/0","link_url":"https://www.baidu.com","desc":"我是群发desc"},"type":13
      }],"send_type":0,"to_id_list":["7881303048045815","7881302321987664","7881300452930187"]},"sync_key":"1709541456928768300","type":5400
}`
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "toSnakeJson",
			args: args{lowerCamelJson: []byte(camelJson)},
			want: []byte(snakeJson),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToSnakeJson(tt.args.lowerCamelJson)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToSnakeJson() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToSnakeJson() got = %v, want %v", got, tt.want)
			}
		})
	}
}
