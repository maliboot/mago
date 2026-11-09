package autoloadx

import (
	"github.com/cloudwego/hertz/pkg/app/server/binding"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/route/param"
)

type XBinder struct {
	defaultBinder binding.Binder
}

func NewXBinder() *XBinder {
	return &XBinder{
		defaultBinder: binding.DefaultBinder(),
	}
}

func (x *XBinder) Name() string {
	return "mali"
}

func (x *XBinder) Bind(request *protocol.Request, i interface{}, params param.Params) error {
	return x.defaultBinder.Bind(request, i, params)
}

func (x *XBinder) Validate(request *protocol.Request, i interface{}) error {
	return x.defaultBinder.Validate(request, i)
}

func (x *XBinder) BindQuery(request *protocol.Request, i interface{}) error {
	return x.defaultBinder.BindQuery(request, i)
}

func (x *XBinder) BindHeader(request *protocol.Request, i interface{}) error {
	return x.defaultBinder.BindHeader(request, i)
}

func (x *XBinder) BindPath(request *protocol.Request, i interface{}, params param.Params) error {
	return x.defaultBinder.BindPath(request, i, params)
}

func (x *XBinder) BindForm(request *protocol.Request, i interface{}) error {
	return x.defaultBinder.BindForm(request, i)
}

func (x *XBinder) BindJSON(request *protocol.Request, i interface{}) error {
	return x.defaultBinder.BindJSON(request, i)
}

func (x *XBinder) BindProtobuf(request *protocol.Request, i interface{}) error {
	return x.defaultBinder.BindProtobuf(request, i)
}
