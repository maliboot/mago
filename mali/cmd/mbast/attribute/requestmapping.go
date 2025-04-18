package attribute

import (
	"strings"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type RequestMapping struct {
	Path        string
	Methods     []string
	Auth        string
	Middlewares []string
}

func (r *RequestMapping) Name() string {
	return "RequestMapping"
}

func (r *RequestMapping) FQN() string {
	return "RequestMapping"
}

func (r *RequestMapping) InitArgs(args map[string]string) Attribute {
	if path, ok := args["0"]; ok {
		r.Path = path
		if methods, ok := args["1"]; ok {
			r.Methods = r.formatMethodsDoc(methods)
		} else {
			hlog.Errorf("注解[RequestMapping]参数异常, Path:[%s], args:%s", r.Path, args)
			return r
		}
		if auth, ok := args["2"]; ok {
			r.Auth = auth
		}
		if middlewares, ok := args["3"]; ok {
			r.Middlewares = formatMiddlewaresDoc(middlewares)
		}

		return r
	}

	if path, ok := args["path"]; ok {
		r.Path = path
	}
	if methods, ok := args["methods"]; ok {
		r.Methods = r.formatMethodsDoc(methods)
	}
	if auth, ok := args["auth"]; ok {
		r.Auth = auth
	}
	if middlewares, ok := args["middlewares"]; ok {
		r.Middlewares = formatMiddlewaresDoc(middlewares)
	}
	return r
}

func (r *RequestMapping) formatMethodsDoc(methodsDoc string) []string {
	return strings.Split(strings.ToUpper(methodsDoc), ",")
}

func (r *RequestMapping) GetPathByPrefix(prefix string) string {
	if len(r.Path) > 0 && r.Path[0] == '/' {
		return r.Path
	}

	return prefix + "/" + r.Path
}
