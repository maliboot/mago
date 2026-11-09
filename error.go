package mago

import (
	"strings"
	"sync"
)

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
	template sync.Map
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

func (e *errorContext) SetTemplate(name string, value string) Error {
	e.template.Store(name, value)
	return e
}

func (e *errorContext) Msg() string {
	retMsg := e.msg
	e.template.Range(func(k, v any) bool {
		retMsg = strings.ReplaceAll(retMsg, k.(string), v.(string))
		return true
	})
	return retMsg
}

func (e *errorContext) Error() string {
	return e.Msg()
}
