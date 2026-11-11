package application

import (
	"testing"

	"tests/internal/user/client/api"
)

// 跨上下文契约的唯一保障点：api.UserService 在本仓库还没有调用方，
// 组合根也校验不到，所以放在测试里断言，接口方法集一漂移 go test 就会失败。
var _ api.UserService = (*userService)(nil)

func TestImplementsClientAPI(t *testing.T) {
	// 断言在包初始化时已生效，这里只需保证测试文件被编译执行。
	if (*userService)(nil) != nil {
		t.Fatal("service 零值应恒为 nil")
	}
}
