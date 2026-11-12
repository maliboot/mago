package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func Print() app.HandlerFunc {
	return func(c context.Context, ctx *app.RequestContext) {
		hlog.Debugf("print middleware")
		ctx.Next(c)
	}
}
