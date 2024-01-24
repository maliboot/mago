package mago

// Error 普通错误
type Error interface {
	Code() ErrorCode
	HttpCode() int
	Msg() string
	WithMsg(string) Error
	Error() string
}

type errorContext struct {
	code     ErrorCode
	httpCode int
	msg      string
}

func NewError(code ErrorCode) Error {
	ins := &errorContext{}
	ins.code = code
	ins.httpCode = code.HttpCode()
	ins.msg = code.String()
	return ins
}

func (e *errorContext) Code() ErrorCode {
	return e.code
}

func (e *errorContext) HttpCode() int {
	return e.httpCode
}

func (e *errorContext) WithMsg(msg string) Error {
	e.msg = msg
	return e
}

func (e *errorContext) Msg() string {
	return e.msg
}

func (e *errorContext) Error() string {
	return e.msg
}
