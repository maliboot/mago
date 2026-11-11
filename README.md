# mago

Go 后端框架：COLA 5.0 分层架构，路由用 hertz、ORM 用 gorm、模型上下文协议用官方 `modelcontextprotocol/go-sdk`。仓库内附示例工程 `demo/`，可直接当作新项目的骨架。

## 文档入口

| 想做的事 | 看这里 |
| --- | --- |
| 架构规范：目录结构、分层职责、依赖方向、命名、错误处理、CQRS、测试 | [demo/README.md](demo/README.md) |
| 给 AI 编码助手的速查索引（硬约束 + 任务落点） | [demo/llms.txt](demo/llms.txt) |

`demo/README.md` 是架构规范的唯一来源；`demo/llms.txt` 由它派生，只做索引与速查，冲突以 README 为准。

## 仓库结构

- `demo/` —— 示例工程。`internal/user` 是一个完整的 Bounded Context：`adapter/http`、`adapter/mcp`、`application`、`domain`、`infrastructure`、`client`。
- 其余文件 —— 框架本体（HTTP、仓储、错误码、DTO/VO、日志、队列、EMQX 等）。

## 校验

```bash
cd demo
go build ./... && go vet ./... && go test -count=1 ./...
```