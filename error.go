package mago

import "strings"

// Error 普通错误
type Error interface {
	Code() ErrorCode
	HttpCode() int
	Msg() string
	WithMsg(string) Error
	SetTemplate(name string, value string) Error
	Error() string
}

type errorContext struct {
	code     ErrorCode
	httpCode int
	msg      string
	template map[string]string
}

func NewError(code ErrorCode) Error {
	ins := &errorContext{}
	ins.code = code
	ins.httpCode = code.HttpCode()
	ins.msg = code.String()
	ins.template = make(map[string]string)
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

func (e *errorContext) SetTemplate(name string, value string) Error {
	e.template[name] = value
	return e
}

func (e *errorContext) Msg() string {
	if len(e.template) > 0 {
		newMsg := ""
		for k, v := range e.template {
			newMsg = strings.ReplaceAll(e.msg, k, v)
		}
		return newMsg
	}
	return e.msg
}

func (e *errorContext) Error() string {
	return e.Msg()
}
