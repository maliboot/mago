package mcp

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maliboot/mago"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"tests/internal/user/client/dto"
	"tests/internal/user/client/enum"
	"tests/internal/user/client/vo"
)

// fakeService 是 application.Service 的测试替身，避免依赖数据库与配置。
type fakeService struct{}

func (fakeService) Login(_ context.Context, req *dto.UserAuthRequest) *mago.VO[vo.UserTokenVO] {
	if req.Password != "secret" {
		return mago.NewVO[vo.UserTokenVO](
			mago.NewError(mago.ErrAuthLoginFailed).WithMsg("用户不存在或密码错误"),
			nil,
		)
	}

	return mago.NewVO[vo.UserTokenVO](nil, vo.NewUserTokenVO("token-1", "2026-11-10 12:00:00"))
}

func (fakeService) Register(context.Context, *dto.UserAuthRequest) *mago.VO[mago.EmptyVO] {
	return mago.NewVO[mago.EmptyVO](nil, nil)
}

func (fakeService) Me(context.Context, uint) *mago.VO[vo.UserVO] {
	return mago.NewVO[vo.UserVO](nil, vo.NewUserVO("a@b.com", enum.UserStatusActive))
}

// connectMemory 用官方 SDK 的内存传输建立一对已初始化的会话。
func connectMemory(t *testing.T) *sdk.ClientSession {
	t.Helper()

	ctx := context.Background()
	serverTransport, clientTransport := sdk.NewInMemoryTransports()

	srv := NewUserMcp(fakeService{})
	serverSession, err := srv.Server().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("服务端连接失败：%v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	client := sdk.NewClient(&sdk.Implementation{Name: "user-mcp-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("客户端连接失败：%v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	return session
}

// TestListTools 校验通过协议取到的工具清单与入参 schema。
func TestListTools(t *testing.T) {
	session := connectMemory(t)

	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("tools/list 失败：%v", err)
	}
	if len(res.Tools) != 3 {
		t.Fatalf("期望 3 个工具，实际 %d", len(res.Tools))
	}

	want := map[string][]string{
		ToolLogin:    {"email", "password"},
		ToolRegister: {"email", "password"},
		ToolMe:       {"userId"},
	}
	for _, tool := range res.Tools {
		required, ok := want[tool.Name]
		if !ok {
			t.Fatalf("未预期的工具：%s", tool.Name)
		}
		if tool.Description == "" {
			t.Errorf("%s 缺少描述", tool.Name)
		}
		schema, ok := tool.InputSchema.(map[string]any)
		if !ok {
			t.Fatalf("%s 的 inputSchema 结构异常：%T", tool.Name, tool.InputSchema)
		}
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("%s 的 properties 结构异常", tool.Name)
		}
		for _, field := range required {
			if _, ok := props[field]; !ok {
				t.Errorf("%s 的 schema 缺少字段 %s", tool.Name, field)
			}
		}
	}
}

// TestCallToolLogin 校验登录工具的成功返回（结构化输出）。
func TestCallToolLogin(t *testing.T) {
	session := connectMemory(t)

	res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      ToolLogin,
		Arguments: map[string]any{"email": "a@b.com", "password": "secret"},
	})
	if err != nil {
		t.Fatalf("tools/call 失败：%v", err)
	}
	if res.IsError {
		t.Fatalf("不应是错误结果：%+v", res.Content)
	}

	payload, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("结构化输出无法序列化：%v", err)
	}
	if !strings.Contains(string(payload), "token-1") {
		t.Fatalf("结构化输出缺少令牌：%s", payload)
	}
}

// TestCallToolBusinessError 校验业务错误经 SDK 透出（domain 给出的错误码文案）。
func TestCallToolBusinessError(t *testing.T) {
	session := connectMemory(t)

	res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      ToolLogin,
		Arguments: map[string]any{"email": "a@b.com", "password": "bad"},
	})
	if err != nil {
		// 协议层直接报错也算透出，只要文案可见。
		if !strings.Contains(err.Error(), "用户不存在或密码错误") {
			t.Fatalf("错误信息不符：%v", err)
		}

		return
	}

	if !res.IsError {
		t.Fatalf("期望 isError=true，实际 %+v", res)
	}
	if text := textOf(t, res); !strings.Contains(text, "用户不存在或密码错误") {
		t.Fatalf("错误文案不符：%s", text)
	}
}

// TestCallToolUnknown 校验未知工具报错。
func TestCallToolUnknown(t *testing.T) {
	session := connectMemory(t)

	res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      "user_delete",
		Arguments: map[string]any{},
	})
	if err != nil {
		return
	}
	if !res.IsError {
		t.Fatalf("期望未知工具返回 isError，实际 %+v", res)
	}
}

// textOf 汇总工具结果里的文本内容，便于断言。
func textOf(t *testing.T, res *sdk.CallToolResult) string {
	t.Helper()

	var b strings.Builder
	for _, item := range res.Content {
		if text, ok := item.(*sdk.TextContent); ok {
			b.WriteString(text.Text)
		}
	}

	return b.String()
}

// TestStreamableHTTP 走真实 HTTP：httptest 起服务 + 官方 Streamable HTTP 客户端。
func TestStreamableHTTP(t *testing.T) {
	ts := httptest.NewServer(NewUserMcp(fakeService{}).Handler())
	t.Cleanup(ts.Close)

	client := sdk.NewClient(&sdk.Implementation{Name: "user-mcp-http-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), &sdk.StreamableClientTransport{
		Endpoint:             ts.URL,
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatalf("HTTP 连接失败：%v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	res, err := session.CallTool(context.Background(), &sdk.CallToolParams{
		Name:      ToolMe,
		Arguments: map[string]any{"userId": 1},
	})
	if err != nil {
		t.Fatalf("HTTP 调用失败：%v", err)
	}

	payload, _ := json.Marshal(res.StructuredContent)
	if !strings.Contains(string(payload), "a@b.com") {
		t.Fatalf("结构化输出缺少邮箱：%s", payload)
	}
}
