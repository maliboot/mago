package http

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"tests/internal/user/client/api"
)

// modulePrefix 是本上下文对外路由的模块前缀（§7.5）。
const modulePrefix = "/api/v1/user"

// resourcePath 匹配资源路由：/users、/users/:id、/users/:id/account。
var resourcePath = regexp.MustCompile(`^/api/v1/user/[a-z][a-zA-Z]*(/:id)?(/[a-z][a-zA-Z]*)?$`)

// TestRoutesFollowContract 校验：每条路由要么是资源形状，要么已登记为动作类接口。
func TestRoutesFollowContract(t *testing.T) {
	routes := NewUserController(nil, nil).Routes()
	if len(routes) == 0 {
		t.Fatal("控制器未声明任何路由")
	}

	for _, r := range routes {
		full := modulePrefix + r.Path

		if _, ok := actionOf(r.Method, full); ok {
			continue
		}

		if !resourcePath.MatchString(full) {
			t.Errorf("路由 %s %s 不符合 /api/v1/user/<资源复数>[/{id}][/<子资源>]，也没有在 client/api.Actions 登记", r.Method, full)
		}
		if strings.ContainsAny(r.Path, "-_") {
			t.Errorf("路由 %s 出现 kebab/snake 片段，资源名必须小驼峰复数", full)
		}
		if strings.Contains(r.Path, ":id") && strings.Count(r.Path, ":id") > 1 {
			t.Errorf("路由 %s 嵌套层级过深", full)
		}
	}
}

// TestActionsAreRegistered 校验：登记的动作都能对上真实路由，避免契约与实现漂移。
func TestActionsAreRegistered(t *testing.T) {
	routes := NewUserController(nil, nil).Routes()

	for _, action := range api.Actions {
		found := false
		for _, r := range routes {
			if r.Method == action.Method && modulePrefix+r.Path == action.Path {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("动作契约 %s %s 未在控制器注册", action.Method, action.Path)
		}
	}
}

// TestActionContractShape 校验动作契约必填字段，避免只写路径不写语义。
func TestActionContractShape(t *testing.T) {
	if len(api.Actions) == 0 {
		t.Fatal("client/api.Actions 为空")
	}

	for _, action := range api.Actions {
		if action.Method != http.MethodPost && action.Method != http.MethodPatch {
			t.Errorf("动作 %s 的方法 %s 不在允许范围", action.Name, action.Method)
		}
		if action.Name == "" || action.Purpose == "" || action.Request == "" {
			t.Errorf("动作 %s 缺少 Name/Purpose/Request 描述", action.Path)
		}
		if len(action.Errors) == 0 {
			t.Errorf("动作 %s 未登记错误码", action.Path)
		}
	}
}

// actionOf 判断某方法+路径是否登记为动作类接口。
func actionOf(method, full string) (api.ActionContract, bool) {
	for _, action := range api.Actions {
		if action.Method == method && action.Path == full {
			return action, true
		}
	}

	return api.ActionContract{}, false
}
