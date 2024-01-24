package mago

import (
	"fmt"
	"strconv"
)

// ErrorCode 错误码
type ErrorCode int

const (
	// ErrNone 成功
	ErrNone ErrorCode = 200
	// ErrTokenInvalid token失效
	ErrTokenInvalid ErrorCode = 401000
	// ErrAuthLoginFailed 用户或密码错误
	ErrAuthLoginFailed ErrorCode = 401100
	// ErrAuthTokenInvalid 非法token
	ErrAuthTokenInvalid ErrorCode = 401200
	// ErrAuthSessionExpired token过期
	ErrAuthSessionExpired ErrorCode = 401300
	// ErrAuthUnauthorized 未认证,没有token
	ErrAuthUnauthorized ErrorCode = 401400
	// ErrAuthFailed 认证失败
	ErrAuthFailed ErrorCode = 401500
	// ErrAccessDenied 没有权限
	ErrAccessDenied ErrorCode = 403100
	// ErrAccessRefuse 拒绝客户端请求
	ErrAccessRefuse ErrorCode = 403200
	// ErrNoRepetitionOperation 禁止重复操作
	ErrNoRepetitionOperation ErrorCode = 403400
	// ErrBadRequest 客户端错误
	ErrBadRequest ErrorCode = 400100
	// ErrUriNotFound 资源未找到
	ErrUriNotFound ErrorCode = 404100
	// ErrInvalidParams 非法的参数
	ErrInvalidParams ErrorCode = 422100
	// ErrServerError 服务器异常
	ErrServerError ErrorCode = 500100
)

var errMsg = map[ErrorCode]string{
	ErrNone:                  "成功",
	ErrTokenInvalid:          "token失效",
	ErrAuthLoginFailed:       "用户或密码错误",
	ErrAuthTokenInvalid:      "非法token",
	ErrAuthSessionExpired:    "token过期",
	ErrAuthUnauthorized:      "未认证,没有token",
	ErrAuthFailed:            "认证失败",
	ErrAccessDenied:          "没有权限",
	ErrAccessRefuse:          "拒绝客户端请求",
	ErrNoRepetitionOperation: "禁止重复操作",
	ErrBadRequest:            "客户端错误",
	ErrUriNotFound:           "资源未找到",
	ErrInvalidParams:         "非法的参数",
	ErrServerError:           "服务器异常",
}

func (e ErrorCode) String() string {
	key, _ := strconv.Atoi(fmt.Sprintf("%d", e))
	newKey := ErrorCode(key)
	return errMsg[newKey]
}

func (e ErrorCode) Int() int {
	return int(e)
}

func (e ErrorCode) HttpCode() int {
	strCode := fmt.Sprintf("%d", e)
	strHttpCode := strCode[0:3]

	httpCode, _ := strconv.Atoi(strHttpCode)
	return httpCode
}
