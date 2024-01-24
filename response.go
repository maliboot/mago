package mago

import (
	"errors"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Response 统一响应格式
type Response interface {
	// Success 成功响应
	Success(data interface{})
	// Failure 错误响应-默认错误信息
	Failure(code ErrorCode)
	// FailureMsg 错误响应-自定义错误信息
	FailureMsg(code ErrorCode, msg string)
	// FailureError 错误响应-根据Error
	FailureError(err error)
}

type response struct {
	ctx *app.RequestContext
}

func NewResponse(ctx *app.RequestContext) Response {
	return &response{ctx}
}

// Success 成功响应
func (r *response) Success(data interface{}) {
	if data == nil {
		data = make([]int, 0)
	}
	r.ctx.JSON(consts.StatusOK, utils.H{
		"code": ErrNone,
		"msg":  "成功",
		"data": data,
	})
}

// Failure 错误响应-默认错误信息
func (r *response) Failure(code ErrorCode) {
	r.failure(code, "")
}

// FailureMsg 错误响应-自定义错误信息
func (r *response) FailureMsg(code ErrorCode, msg string) {
	r.failure(code, msg)
}

// FailureError 错误响应-根据Error
func (r *response) FailureError(e error) {
	var err Error
	ok := errors.As(e, &err)
	if ok {
		r.FailureMsg(err.Code(), err.Msg())
		return
	}
	hlog.Error(e)
	r.Failure(ErrServerError)
}

func (r *response) failure(code ErrorCode, msg string) {
	err := NewError(code)
	if msg != "" {
		err = err.WithMsg(msg)
	}
	r.ctx.JSON(err.HttpCode(), utils.H{
		"code": int(err.Code()),
		"data": make([]int, 0),
		"msg":  err.Msg(),
	})
}
