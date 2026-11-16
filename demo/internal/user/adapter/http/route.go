package http

import "github.com/cloudwego/hertz/pkg/app"

// Route 描述本控制器的一条路由，集中声明便于契约测试校验路径形状。
// 单独成文件，保持「一个文件一个核心 struct」（README §18.1）。
type Route struct {
	// Method 是 HTTP 方法。
	Method string
	// Path 是相对上下文分组（/api/v1/user）的路径。
	Path string
	// Authed 为真时挂在需要 JWT 的分组上。
	Authed bool
	// Handler 是处理函数。
	Handler app.HandlerFunc
}
