package mago

import "testing"

func Test_errorContext_Msg(t *testing.T) {
	tests := []struct {
		name   string
		code   ErrorCode
		msg    string
		tplKey string
		tplVal string
		want   string
	}{
		{
			name:   "test-error-code",
			code:   ErrServerError,
			msg:    ErrServerError.String() + "[id=%id%]",
			tplKey: "%id%",
			tplVal: "325",
			want:   "服务器异常[id=325]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &errorContext{
				code:     tt.code,
				msg:      tt.msg,
				httpCode: tt.code.HttpCode(),
			}
			e.template.Store(tt.tplKey, tt.tplVal)

			if got := e.Msg(); got != tt.want {
				t.Errorf("Msg() = %v, want %v", got, tt.want)
			}
		})
	}
}
