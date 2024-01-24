package mago

import (
	"github.com/cloudwego/hertz/pkg/app"
)

type AbstractAdapter struct {
}

func (a AbstractAdapter) Response(ctx *app.RequestContext, vo interface{}) {
	resp := NewResponse(ctx)
	if vo == nil {
		resp.Success(nil)
		return
	}

	resVO, ok := vo.(ViewObjectTemplate)
	if !ok {
		resp.Success(vo)
		return
	}

	err := resVO.Error()
	if err != nil {
		resp.FailureError(err)
		return
	}
	if resVO.Zero() {
		resp.Success(nil)
		return
	}
	resp.Success(resVO.Body())
}
