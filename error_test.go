package mago

import "testing"

func Test_errorContext_Msg(t *testing.T) {
	type fields struct {
		errorCode ErrorCode
		template  map[string]string
		msg       string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "test-error-code",
			fields: fields{errorCode: ErrServerError, template: map[string]string{"%id%": "325"}, msg: ErrServerError.String() + "[id=%id%]"},
			want:   "服务器异常[id=325]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &errorContext{
				code:     tt.fields.errorCode,
				msg:      tt.fields.msg,
				httpCode: tt.fields.errorCode.HttpCode(),
				template: tt.fields.template,
			}
			if got := e.Msg(); got != tt.want {
				t.Errorf("Msg() = %v, want %v", got, tt.want)
			}
		})
	}
}
