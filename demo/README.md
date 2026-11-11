# Go 后端架构规范

- **文档版本：** 1.0.0
- **适用范围：** Go 后端项目
- **架构模式：** 分层架构（adapter / application / domain / infrastructure）+ 模块化单体 + DDD
- **核心原则：** 业务边界清晰、依赖方向明确、遵循 Go 语言习惯、避免过度设计

> 本文件是架构规范的唯一来源。AI 编码助手可先读同目录 [llms.txt](llms.txt)（速查索引，约束条目与本文一一对应）；仓库入口见 [../README.md](../README.md)。

---

## 1. 架构总览

本项目采用 COLA 5.0 分层架构（参考 [alibaba/COLA v5.0.0](https://github.com/alibaba/COLA/tree/v5.0.0)）：

**单 Module + 四层：adapter / application / domain / infrastructure**

具体约定：

1. 一个 Go Module 就是一个部署单元，不按层拆 Module。
2. 入口适配器统一放 `adapter/`，出站实现统一放 `infrastructure/`，两类适配器不混放。
3. 代码按层组织，一层一个 Go Package：`adapter`、`application`、`domain`、`infrastructure`。
4. 业务模块按限界上下文隔离，每个上下文内部重复相同的四层包结构。
5. `cmd/` 只放启动入口与依赖装配，是唯一允许同时引用四层的地方。
6. `adapter` 只做协议转换与参数校验，调用 `application` 暴露的服务接口。
7. `application` 编排用例，依赖 `domain` 的聚合、领域服务与 Gateway 接口。
8. `domain` 定义聚合、值对象、领域服务、领域错误和出站端口（Gateway 接口），不依赖任何外层。
9. `infrastructure` 实现 `domain` 定义的 Gateway，承载数据库、缓存、消息队列、外部 API 等技术细节。
10. 依赖方向单向向内：`adapter → application → domain ← infrastructure`；不引入 DI 框架，统一由启动入口手动装配。

分层约束靠两条腿落地：Package 边界与架构测试（见 17.1）。

包命名、接口命名、错误处理必须符合 Go 习惯（见第 3 章、第 16 章）。

### 1.1 核心架构关系

```text
                 外部请求（HTTP / MCP / MQ / 定时任务）
                                |
                                v
            +---------------------------------------+
            |  adapter   入口适配器                  |
            |  Controller / MQ Consumer / Job        |
            +-------------------+-------------------+
                                | 调用
                                v
            +---------------------------------------+
            |  application   用例编排                |
            |  Service 接口 + 实现、dto               |
            +-------------------+-------------------+
                                | 调用聚合 / 领域服务
                                v
            +---------------------------------------+
            |  domain   领域层                       |
            |  Aggregate / ValueObject /            |
            |  DomainService / Gateway 接口 / 错误   |
            +-------------------+-------------------+
                                ^ 实现 Gateway 接口
                                |
            +-------------------+-------------------+
            |  infrastructure   技术实现             |
            |  DB / Redis / MQ / 外部 API 客户端      |
            +---------------------------------------+

            cmd/ 为组合根：构造四层实现并注入，不写业务逻辑。
```

上图表达的是依赖方向，不要求每个限界上下文都创建完全相同的目录。

---

## 2. 项目目录结构

推荐采用以下目录结构（以 `user` 上下文为例，其余上下文同构）：

```text
demo/
├── go.mod
├── conf.yml
├── conf-production.yml
├── README.md                       # 架构规范（唯一文档入口）
├── llms.txt                        # 给 AI 的速查索引（由 README 派生，不替代）
│
├── cmd/
│   └── server/
│       ├── main.go          # 启动入口：读配置、装配四层、启动应用
│       ├── http.go          # HTTP Server 装配
│       ├── queue.go         # 队列装配（按需）
│       ├── mqtt.go          # MQTT 装配（按需）
│       └── copool.go        # 协程池装配（按需）
│
└── internal/
    ├── user/                       # 限界上下文：用户
    │   ├── adapter/                # 入口适配器，按协议分子目录
    │   │   ├── http/
    │   │   │   └── user.go         # HTTP Controller
    │   │   ├── mcp/
    │   │   │   └── user.go         # MCP 工具注册（官方 go-sdk）
    │   │   └── script/
    │   │       └── user_migration_20261110.sql   # 数据迁移 / 修数脚本
    │   │
    │   ├── client/                 # 跨模块契约：本上下文产出，其他上下文只依赖这里
    │   │   ├── api/
    │   │   │   └── user.go         # 对外 Service 接口
    │   │   ├── dto/
    │   │   │   └── user.go         # Request / Response / Query DTO
    │   │   ├── vo/
    │   │   │   └── user.go         # 对外视图对象（按需）
    │   │   ├── enum/
    │   │   │   └── user.go         # 对外枚举（按需）
    │   │   └── error.go            # 本上下文的错误码扩展
    │   │
    │   ├── application/            # 用例编排
    │   │   └── user.go             # Service 接口 + 实现 + 构造函数
    │   │
    │   ├── domain/                 # 领域层
    │   │   ├── user.go             # Aggregate / Entity / ValueObject
    │   │   ├── gateway.go          # 出站端口（接口）
    │   │   └── token.go            # TokenIssuer 等能力端口
    │   │
    │   ├── infrastructure/         # 技术实现
    │   │   ├── repository/         # 该 DO 唯一的 repository：写走 Gateway，读直查列
    │   │   │   └── user.go         # type User struct
    │   │   ├── do/                 # 数据对象，按介质分子包
    │   │   │   └── db/
    │   │   │       └── user.go     # type User struct（表结构）
    │   │   ├── convertor/          # DO ↔ 领域对象转换
    │   │   │   └── user.go         # UserToEntity / UserToDO
    │   │   └── util/               # 层内辅助：类型转换、散列、令牌签发等
    │   │       ├── bool.go
    │   │       ├── password.go
    │   │       └── token.go
    │   │
    │   └── (其余子领域按需新建，如 address/，结构与本上下文同构)
    │
    ├── catalog/                    # 其余上下文：货品目录
    ├── inventory/                  # 其余上下文：库存
    └── finance/                    # 其余上下文：财务
```

### 2.1 目录职责

| 目录 | 职责 |
|---|---|
| `cmd/` | 启动入口与依赖装配（组合根） |
| `internal/<ctx>/adapter/` | 入口适配器，按协议分子目录（`http/`、`mcp/`、`mq/`、`job/`），文件按业务对象命名 |
| `internal/<ctx>/client/` | 跨模块契约：`api/` 接口、`dto/`、`vo/`、`enum/`、`error.go` 错误码 |
| `internal/<ctx>/application/` | 用例编排、实现 `client/api` 的接口 |
| `internal/<ctx>/domain/` | 聚合、实体、值对象、领域服务、领域错误、Gateway 接口 |
| `internal/<ctx>/infrastructure/` | `repository/` 一个 DO 一个 repository（读写都在这里）、`do/` 数据对象、`convertor/`、`util/` |
| `internal/<ctx>/adapter/script/` | 数据迁移与修数脚本，按 `上下文_说明_日期.sql` 命名 |
| `conf.yml` | 运行配置 |
| `README.md` | 架构规范，唯一文档入口（不另建 docs/ 目录） |
| `llms.txt` | AI Agent 速查索引，指向 README.md 对应章节（第 18 章为完整约束） |

### 2.2 业务模块按 Bounded Context 组织

一级目录应代表业务限界上下文，而不是技术类型。

例如：

- `user`：用户与账号上下文
- `catalog`：货品目录上下文
- `inventory`：库存上下文
- `logistics`：物流上下文
- `customs`：报关清关上下文
- `finance`：财务上下文
- `workflow`：工作流上下文

每个上下文内部按四层组织，层可以按实际复杂度裁剪，不同上下文允许不同。

**不允许为了目录统一，给每个业务模块机械创建没有实际意义的空目录。**
**普通数据查询按轻量级CQRS原则，不强制走domain层，如搜索分页、详情查询**

例如，一个暂时没有复杂领域规则的模块可以只有：

```text
internal/
└── catalog/
    ├── client/
    │   └── dto/
    ├── adapter/
    ├── application/
    └── infrastructure/
```

但一旦出现业务不变量、状态流转或跨实体规则，必须补上 `domain` 承载，不得塞进 `application`。

### 2.3 目录约束

1. 入口适配器统一放 `adapter/`，出站实现统一放 `infrastructure/`，不在层内再分方向子目录。
2. 不创建全局技术分类目录（`handlers/`、`services/`、`repositories/`、`models/`、`dto/` 这类跨上下文的目录）。
3. 跨上下文复用的技术能力使用独立 Package 或框架能力（`config`、`logx`、`httpx`、`queuex` 等），不放进任一业务上下文。
4. 不建根目录 `docs/`：架构规范就是根目录 `README.md`，ADR 直接写进正文对应章节。
5. 不建根目录 `migrations/`：数据迁移与修数一律做成 `adapter/script/` 下的脚本，随上下文走，可回放、可评审。

### 2.4 infrastructure 分层与文件命名

`infrastructure/` 固定两类内容，都按业务对象建文件，不写 `repository.go`、`model.go` 这类泛名。

| 子目录 | 内容 | 命名示例 |
|---|---|---|
| `repository/` | domain 端口（Gateway）的实现 | `repository/user.go`，内含 `type User struct` |
| `repository/`（读方法） | 读路径挂在同一个 repository 上，只查展示列并组装 VO | `repository/user.go` 中的 `ByID` |
| `do/` | 数据对象（DataObject），按介质分子包 `db/`、`redis/`、`mq/`、`http/` | `do/db/user.go`，内含 `type User struct` |
| `convertor/` | DO 与领域聚合的双向转换 | `convertor/user.go`，内含 `UserToEntity` / `UserToDO` |
| `util/` | 层内辅助：类型转换、散列、令牌签发、键拼接等 | `util/bool.go`、`util/password.go`、`util/token.go` |

约定：

1. 文件名 = 业务对象名（小写下划线或单词），如 `user.go`、`user_address.go`，一个对象一个文件。
2. 同一业务对象在不同层使用各自的类型，靠包名区分（`domain.User`、`repository.User`、`db.User`），转换函数写在 `convertor/` 内。
3. `do/` 下的每个介质子包只描述该介质的结构：数据库表、缓存键值、消息体、HTTP 报文，不写业务规则。
4. 一个业务对象需要多种介质时并列建文件，如 `do/db/user.go` 与 `do/redis/user.go`，互不引用。
5. 同样的命名规则适用于其他层：`client/api/user.go`、`client/dto/user.go`、`application/user.go`、`domain/user.go`、`adapter/http/user.go`、`infrastructure/repository/user.go`。
6. 所有 DTO、VO、枚举、错误码、对外接口一律放 `client/`，`application` 与 `adapter` 不得再建自己的 dto 目录。

---

## 3. Go Package 设计规范

这是本项目最重要的 Go 工程约束之一。

### 3.1 Package 是基本的封装和依赖边界

Go 语言中，同一个目录下的普通 Go 源文件属于同一个 Package。

例如：

```text
internal/purchase/
├── application/
│   ├── order.go
│   └── supplier.go
└── client/dto/
    ├── order.go
    └── supplier.go
```

这些文件可以统一声明：

```go
package application
```

它们共同组成一个 Go Package。

因此：

- 不要为每个 Go 文件创建独立目录。
- 不要按「一个类型一个目录」的方式设计目录。
- 不要为了形式上的分层而创建大量只有一个文件的 Package。
- 不要把 Package 理解成单个类型。
- 应当根据业务概念、封装职责、依赖关系决定是否拆分 Package。

### 3.2 什么情况下应该拆分 Package？

满足以下情况时，可以考虑拆分：

1. 两部分代码具有不同的依赖约束。
2. 两部分代码需要独立控制对外暴露的 API。
3. 两部分代码职责明显不同。
4. 拆分后能够阻止不合理的依赖。
5. 代码已经形成相对独立的业务概念。

例如：

```text
purchase/
├── domain/
│   ├── order/
│   └── supplier/
│
├── application/
│   └── order.go
│
├── adapter/
│   └── order.go
│
└── infrastructure/
    ├── repository/
    │   └── order.go
    ├── convertor/
    │   └── order.go
    └── do/db/
        └── order.go
```

这里不同目录代表不同 Package，能够形成实际的编译级依赖边界。

### 3.3 不要为了分层而滥用 Package

以下结构不推荐：

```text
purchase/
├── domain/
│   └── order/
│       ├── entity/
│       ├── valueobject/
│       ├── aggregate/
│       ├── repository/
│       └── service/
```

如果这些目录没有独立的封装意义，就没有必要拆分。

可以直接使用：

```text
purchase/
└── domain/
    └── order/
        ├── order.go
        ├── item.go
        ├── status.go
        ├── repository.go
        └── errors.go
```

一个 Package 内可以包含多个文件及多个相关类型。

### 3.4 不使用体系化命名后缀

避免：

```text
OrderServiceImpl
OrderManagerImpl
OrderRepositoryImpl
OrderFactoryImpl
```

推荐：

```go
type OrderService struct {
    repo OrderRepository
}
```

如果是具体的 MySQL Repository 实现：

```go
type MySQLOrderRepository struct {
    db *gorm.DB
}
```

Go 的命名应该简洁、明确，优先表达职责，不需要机械添加 `Impl` 后缀。

---

## 4. Domain 领域层

Domain 负责表达业务领域本身。

包括：

- Entity
- Aggregate
- Value Object
- Domain Service
- Domain Event
- Domain Error
- 领域行为和业务不变量

### 4.1 Domain 不允许依赖基础设施技术

Domain 不允许直接依赖：

- Gin
- GORM
- Redis
- Kafka
- MySQL Driver
- HTTP Client
- 具体的消息队列 SDK

Domain 应当尽量依赖 Go 标准库和纯领域代码。

例外：允许引用框架 `mago` 的全局错误码（`mago.ErrorCode`、`mago.NewError`）表达领域错误；不得引用 `mago` 的 repository、http、queue 等运行期组件。

### 4.2 Entity 示例

```go
package order

import (
    "errors"
    "time"
)

type Status string

const (
    StatusPending  Status = "pending"
    StatusApproved Status = "approved"
    StatusRejected Status = "rejected"
)

// Order 是订单聚合根。字段导出便于各层映射，状态变更只走领域方法。
type Order struct {
    ID        string
    Status    Status
    CreatedAt time.Time
}

func NewOrder(id string) (*Order, error) {
    if id == "" {
        return nil, errors.New("order id is required")
    }

    return &Order{
        ID:        id,
        Status:    StatusPending,
        CreatedAt: time.Now(),
    }, nil
}

// Approve 只允许待审批状态流转，业务不变量由领域方法守护。
func (o *Order) Approve() error {
    if o.Status != StatusPending {
        return errors.New("only pending orders can be approved")
    }

    o.Status = StatusApproved
    return nil
}
```

示例仅展示领域行为，不代表所有实体都必须采用相同的封装方式。

### 4.3 领域模型的设计原则

1. 业务规则应放在真正拥有该规则的领域模型中。
2. 不要把所有业务逻辑都放进 Application Service。
3. 不要创建只有字段、没有行为且没有领域意义的复杂 Entity。
4. 不要为了 DDD 而强行把简单 CRUD 设计成复杂 Aggregate。
5. Aggregate 应根据业务一致性边界设计，而不是根据数据库表的关联关系设计。
6. 跨 Aggregate 操作应明确事务和一致性策略。

### 4.4 字段可见性

聚合、实体、值对象的字段一律导出（首字母大写），不写 getter / setter。

理由：

1. 跨层映射、序列化、日志、调试都要读字段，getter 只是样板，Go 没有 Java 那样的封装文化。
2. 不变量由构造函数（`NewUser`）与领域方法（`CanLogin`、`MarkLoggedIn`）保证，不靠字段私有化模拟。
3. 减少代码量：一个 6 字段的聚合少写 6 个 getter，AI 与人工都少一层噪声。

对应的约束：

1. 外部不得直接给聚合字段赋值来改变业务状态，状态变更一律走领域方法。
2. 恢复数据用 `RestoreXxx` 构造函数，与 `NewXxx` 区分「新建要校验、恢复不校验」。
3. 需要强约束的小值对象（如金额 `Money`、时间区间）可以保留私有字段 + 构造函数，保证自身始终有效。
4. 导出字段不等于可以导出内部集合，切片/映射字段仍以只读方式使用，修改通过方法完成。

---

## 5. Application 应用层

Application 负责执行应用用例。

例如：

- 创建采购订单
- 审批采购订单
- 取消采购订单
- 提交付款申请
- 创建出库单
- 查询订单列表

Application 的主要职责：

1. 接收用例输入。
2. 加载所需的领域对象。
3. 调用领域行为。
4. 协调多个领域对象或领域服务。
5. 调用 Gateway 等出站端口。
6. 控制事务边界。
7. 返回用例结果。

Application 不应该承载本应属于 Domain 的核心业务规则。

### 5.1 Service 接口与实现

应用层不按用例建类，而是按业务能力组织应用服务：

```text
application/
└── user.go        # Service 接口 + 实现 + 构造函数（入参出参用 client/dto、client/vo）

client/
├── api/user.go    # 对外接口
├── dto/user.go    # 入参
├── vo/user.go     # 出参视图
├── enum/user.go   # 对外枚举
└── error.go       # 错误码扩展
```

示例：

```go
package application

import (
    "context"

    "project/internal/purchase/domain/order"
)

// OrderService 暴露采购订单的用例。
type OrderService interface {
    ApproveOrder(ctx context.Context, id string) error
}

type orderService struct {
    orders order.Gateway
}

func NewOrderService(orders order.Gateway) OrderService {
    return &orderService{orders: orders}
}

func (s *orderService) ApproveOrder(ctx context.Context, id string) error {
    o, err := s.orders.GetByID(ctx, id)
    if err != nil {
        return err
    }

    if err := o.Approve(); err != nil {
        return err
    }

    return s.orders.Save(ctx, o)
}
```

这里的 Application 负责协调用例，`Approve()` 负责领域状态约束。

约束：

1. 接口按业务命名并导出（如 `UserService`），实现不导出（如 `userService`），adapter 只能看到接口。
2. 用例多到一个 Service 明显过载时，按读写或子能力拆成 `QueryService`、`CommandService`，而不是退回「一个用例一个 Package」。
3. 入参用 `client/dto`，出参用 `client/vo`，不把 domain 聚合直接暴露给 adapter。
4. 领域内的布尔标记与内部枚举不进契约：由 application 映射成 `client/enum` 的显式枚举后再对外。

---

## 6. Gateway 与依赖倒置

Gateway 是领域层对外的出站端口：持久化、远程调用、消息发送都通过它暴露给领域模型。

本项目采用：

**端口由 domain 定义，实现由 infrastructure 提供。**

repository 与 DO 严格一对一：**一个 DO 只对应一个 repository**。写方法（经 Gateway 返回聚合）与读方法（直查列返回 VO）都挂在同一个 repository 上，不允许再建第二个 repository 指向同一个 DO —— 无论 DDD 还是 CQRS 都适用。

### 6.1 domain 定义 Gateway

```go
package order

import "context"

type Gateway interface {
    GetByID(
        ctx context.Context,
        id string,
    ) (*Order, error)

    Save(
        ctx context.Context,
        o *Order,
    ) error
}
```

接口放在 `domain/gateway.go`，方法签名只出现领域类型，不出现 `*gorm.DB`、HTTP 客户端等技术类型。

### 6.2 infrastructure 实现 Gateway

```go
package infrastructure

import (
    "context"

    "gorm.io/gorm"

    "project/internal/purchase/domain/order"
)

type OrderRepository struct {
    db *gorm.DB
}

func NewOrderRepository(
    db *gorm.DB,
) *OrderRepository {
    return &OrderRepository{
        db: db,
    }
}

func (r *OrderRepository) GetByID(
    ctx context.Context,
    id string,
) (*order.Order, error) {
    // 读取持久化 Model 并恢复领域对象。
    return nil, nil
}

func (r *OrderRepository) Save(
    ctx context.Context,
    o *order.Order,
) error {
    // 将领域对象映射为 Model 后写库。
    return nil
}

var _ order.Gateway = (*OrderRepository)(nil)
```

上面的代码为结构示意，实际实现需要补充持久化模型、映射及错误处理。

### 6.3 端口放置原则

1. 聚合自身的持久化端口（`XxxGateway`）放 `domain`；不建全局 `repository` Package。
2. 技术能力端口（密码散列、令牌签发、文件、短信等）由**使用方 `application` 定义**，不放 `domain`：
   这些能力不是领域概念，domain 不需要认识它们，放进来只会让领域层混入技术细节。
3. 端口方法使用领域语言（`GetByID`、`Save`、`FindByPhone`），不暴露 SQL、分页游标等存储细节。
4. 按能力建端口（`OrderGateway`、`PaymentGateway`），不做一个万能 Repository。
5. 查询型端口放 `application`（`UserQuery`），直接返回 `client/vo`，不为此加载完整聚合（见第 8 章）。

---

## 7. 适配器层：adapter 与 infrastructure

适配器负责连接应用核心与外部世界，由两个 Package 分别承担。

| 方向 | Package | 职责 |
|---|---|---|
| 入口 | `adapter/` | HTTP、MCP、MQ Consumer、定时任务，把外部请求转成用例调用 |
| 出口 | `infrastructure/` | 实现 domain 的 Gateway：数据库、缓存、MQ、外部 API |

适配器总则（所有入口适配器都必须遵守）：

1. 复用同一套契约：入参用 `client/dto`，出参用 `client/vo`，适配器不定义自己的请求/响应结构。
2. 复用同一套用例：所有适配器调用同一个 application 服务（如 `application.UserService`），不复制业务流程。
3. 适配器不写业务逻辑：校验规则、状态判断、字段拼装、展示文案一律下沉到 application 或 domain。
4. 适配器只做协议转换：参数绑定与序列化、状态码与错误映射、鉴权、路由注册。

同一上下文的 HTTP 与 MCP 入口就是这套原则的体现：两者共用 `dto.UserAuthRequest` 与 `vo.UserVO`，共用同一个 `application.UserService`。

### 7.1 入口适配器：adapter/

外部请求进入系统的入口。

例如：

- HTTP Controller
- MCP 工具入口
- MQ Consumer
- 定时任务入口
- CLI 命令

目录示例：

```text
adapter/
├── http/
│   └── user.go     # HTTP Controller
└── mcp/
    └── user.go     # MCP 工具入口
```

Controller 负责：

1. 接收请求。
2. 解析和校验输入。
3. 将请求转换为 `client/dto` 的 Request。
4. 调用 application Service。
5. 将结果转换为 Response。
6. 处理协议层错误和状态码。

Controller 不直接操作数据库，也不承载核心业务规则。

`adapter/` 按协议分子目录（`http/`、`mcp/`、`mq/`、`job/`），每个协议目录下按业务对象建文件，如 `http/user.go`。

### 7.2 出口实现：infrastructure/

domain 定义端口，infrastructure 提供实现。

例如：

- MySQL Repository（Gateway 实现）
- Redis 缓存
- MQ Producer
- 外部 API Client
- 文件存储

目录示例：

```text
infrastructure/
├── repository/
│   └── user.go     # 该 DO 唯一的 repository：写走 Gateway，读直查列
├── convertor/
│   └── user.go     # DO ↔ 领域对象转换
├── do/
│   └── db/
│       └── user.go # 表结构（DataObject）
└── util/
    ├── bool.go     # 类型转换
    └── password.go # 散列适配
```

infrastructure 可以依赖技术框架，但不能反向要求 domain 依赖这些技术。

### 7.3 MCP 入口：adapter/mcp/

MCP 入口把同一批用例暴露成工具供模型调用，只做参数映射与错误透出：复用 `client/dto`、`client/vo` 与同一套 application 逻辑，不定义自己的 DTO，也不实现协议。

协议实现用官方 SDK：`github.com/modelcontextprotocol/go-sdk`（官方 Go SDK，由 MCP 官方与 Google Go 团队共同维护）。不自己写 JSON-RPC 层，也不手写 tool schema。

```text
adapter/mcp/
└── user.go     # 工具注册 + handler
```

```go
// 注册：入参直接是 client/dto 的 UserAuthRequest，schema 由 SDK 依类型自动推导
sdk.AddTool(m.server, &sdk.Tool{
    Name:        ToolLogin,
    Description: "用邮箱和密码登录，返回访问令牌。",
}, m.login)

// handler：入参复用 dto，出参复用 vo，业务错误直接返回 error（SDK 转成 isError）
func (m *UserMcp) login(ctx context.Context, _ *sdk.CallToolRequest, in dto.UserAuthRequest) (*sdk.CallToolResult, *vo.UserTokenVO, error) {
    res := m.service.Login(ctx, &in)
    if err := res.Error(); err != nil {
        return nil, nil, err
    }

    return nil, res.Data, nil
}
```

传输与挂载：

1. 传输用 Streamable HTTP：`sdk.NewStreamableHTTPHandler(...)` 返回标准 `http.Handler`。
2. 挂载复用现有 hertz 服务，不新开端口：`h.Any("/v1/mcp", adaptor.HertzHandler(handler))`，`adaptor` 是 hertz 官方 `pkg/common/adaptor`。
3. 工具名用 `上下文_动作`（`user_login`、`user_register`、`user_me`），避免多上下文冲突。
4. 入参复用 `client/dto.UserAuthRequest`，出参复用 `client/vo.UserTokenVO`、`vo.UserVO`，MCP 适配器不定义任何 DTO/VO。
5. 工具最终落到 `application.UserService`，与 HTTP 入口共用同一份用例实现。
6. 工具 schema 由 DTO 的 `json` / `jsonschema` 标签推导：字段默认可选，必填语义写进字段描述，由业务层校验兜底。
7. 测试用 SDK 自带的内存传输（`sdk.NewInMemoryTransports` + `Client.Connect`）或 `httptest` + Streamable HTTP 客户端，不依赖数据库。

---

## 8. 读写分离：轻量 CQRS

本项目采用轻量 CQRS。

不要求所有查询都经过 Domain Aggregate，也不要求引入独立的 CQRS 基础设施。

核心规则：

- Command：通过领域模型执行写操作。
- Query：根据查询需求直接读取数据，返回查询 DTO。

### 8.1 Command 写操作

典型流程：

```text
HTTP Request
    |
    v
Handler
    |
    v
Application Service
    |
    v
Gateway.GetByID()
    |
    v
Domain Aggregate
    |
    v
执行领域行为
    |
    v
Gateway.Save()
    |
    v
Database
```

例如审批采购订单：

1. 根据订单 ID 加载 Aggregate。
2. 调用 `Approve()`。
3. Domain 校验是否允许审批。
4. 保存修改后的 Aggregate。
5. 提交事务。

Command 必须保证业务不变量。

### 8.2 Query 查询操作

查询优先走轻量 CQRS：`adapter → application → repository 的读方法`，不加载领域聚合。
读方法挂在**该 DO 唯一的 repository** 上，不另建 query repository。
例外：`domain` 的 Gateway 里已经有现成且可复用的方法时，直接用它，不再重复造查询。

```text
HTTP Request
    |
    v
Adapter
    |
    v
Application（调用读模型接口）
    |
    v
repository.ByID（同一个 repository，只查展示列）
    |
    v
Database
    |
    v
client/vo
    |
    v
HTTP Response
```

约定：

1. 读模型接口定义在 `application`（消费方定义接口），由该 DO 的 repository 实现。
2. 读方法只取展示列并组装 `client/vo`，不加载聚合；同一 repository 里严禁出现第二个 DO。
3. 查询直接取展示需要的列并组装 VO，不为列表、详情、报表加载完整聚合。
4. 只有需要业务规则校验或状态流转时才回到聚合（走 Command 路径）。

典型查询：用户列表、订单列表、订单详情、财务统计、库存报表、业务看板。

### 8.3 读模型示例

端口定义在 `application`：

```go
// application/user.go
type UserQuery interface {
    ByID(ctx context.Context, id uint) (*vo.UserVO, error)
}
```

实现在该 DO 唯一的 repository 里，直接查列并返回 VO：

```go
// infrastructure/repository/user.go
func (q *User) ByID(ctx context.Context, id uint) (*vo.UserVO, error) {
    do := q.Find(int(id))
    if err := q.Err; err != nil {
        return nil, err
    }
    if do == nil {
        return nil, mago.NewError(mago.ErrUriNotFound).WithMsg("用户不存在")
    }

    return vo.NewUserVO(do.Email, enum.FromActive(do.IsActive != 0)), nil
}
```

DO/实体到 VO 的组装写在 VO 自己的构造函数或方法上，不在每个查询里散写字段赋值。

不需要为了查询用户列表，先构造完整的 Domain Entity 再转 VO。

### 8.4 为什么不统一使用一个 Model？

系统中可能存在以下结构：

| 类型 | 职责 |
|---|---|
| Domain Entity | 业务状态、行为和业务不变量 |
| Application DTO | 用例输入和输出 |
| Persistence Model | 数据库表映射 |
| HTTP Request | HTTP 请求协议 |
| HTTP Response | HTTP 响应协议 |
| Query DTO | 查询结果 |

这些结构可能有相同字段，但职责并不相同。

例如：

```go
type User struct {
    // Domain Entity
}

type UserModel struct {
    // MySQL Persistence Model
}

type UserListItem struct {
    // Application Query DTO
}

type UserResponse struct {
    // HTTP Response
}
```

不要为了减少重复字段，把所有层强行绑定到一个结构体。

也不要为了形式上的 DDD，给每个简单查询都创建不必要的模型转换。

**只有当模型的职责、生命周期或变化原因不同时，才需要拆分。**

### 8.5 读写分离的约束

1. Command 必须遵守领域业务规则，走聚合与 Gateway。
2. Query 默认走 `adapter → application → repository 的读方法`，不加载聚合。
3. 只有 `domain` 的 Gateway 已有现成且可复用的方法时才复用，不再另建查询。
4. Query 不得绕过必要的权限和数据范围校验。
5. 查询返回 `client/vo`，不得直接暴露数据库 DO 或 domain 聚合。
6. 复杂统计、报表不需要为了 DDD 而加载 Aggregate。
7. 不要在没有实际需求时引入独立读写数据库。
8. 不要把轻量 CQRS 误解为必须引入消息总线、事件溯源或独立查询服务。

---

## 9. DTO 与 Model 命名规范

建议采用清晰的命名方式。

| 类型 | 命名示例 |
|---|---|
| Domain Entity | `Order` |
| Persistence DO | `db.Order`（`infrastructure/do/db`） |
| 请求 DTO | `CreateOrderRequest`（`client/dto`） |
| 响应 VO | `OrderVO`（`client/vo`） |
| 查询入参 | `OrderListQuery`（`client/dto`） |
| 对外枚举 | `OrderStatus`（`client/enum`） |
| Domain Event | `OrderApproved`（`client/api`） |

不需要机械地给所有类型添加 `DTO` 后缀。

VO 的组装逻辑写在 VO 自己身上，用构造函数或 `FromXxx` 方法，不散写在 application、query 或 adapter 里：

```go
// client/vo/user.go
func NewUserVO(email string, status enum.UserStatus) *UserVO {
    return &UserVO{Email: email, Status: status}
}
```

1. 一个 VO 的字段拼装只允许出现在这个 VO 的构造函数/方法里。
2. VO 不 import `domain`、`application`、`infrastructure`，转换所需的枚举与标量由调用方先算好。
3. 需要多来源字段时给 VO 加 `FromXxx` 方法，而不是在调用侧写一长串字段赋值。
4. 不采用 `FromUser(*domain.User)` 这种直接吃领域聚合的写法：`client` 是跨模块契约包，一旦依赖 `domain`，
   其他上下文 import 契约时会连带本上下文的领域模型，架构测试也会拦。COLA 单模块骨架里的 `fromEntity` 属于
   「DTO 与 domain 同模块」的形态，不适用于本项目的 client 契约包。

例如：

```go
type OrderListItem struct {
    ID     string
    Number string
    Status string
}
```

比：

```go
type OrderListItemDTO struct {
}
```

更加简洁。

命名应优先体现业务用途，而不是强调它属于哪一层。

---

## 10. 跨 Bounded Context 通信

不同业务上下文应保持相对独立。

例如：

```text
purchase
inventory
finance
logistics
```

这些模块不能随意直接操作对方内部的 Domain Entity 或 Aggregate。

### 10.1 不推荐的方式

采购模块直接导入库存模块的内部领域对象，并修改其内部状态。

```go
// 不推荐：直接操纵其他上下文的内部模型。
inventoryOrder.Status = "shipped"
```

这样会导致上下文之间形成强耦合。

### 10.2 推荐的通信方式

按调用性质选，不按方便程度选：

| 场景 | 方式 |
|---|---|
| 同进程同步调用 | 通过被调上下文的契约包 `internal/<ctx>/api` |
| 状态变更后通知其他上下文 | Domain Event / Integration Event |
| 对接外部系统或异构服务 | 调用方 infrastructure 内的 client + convertor（ACL） |

#### 契约包（同步调用）

每个上下文有一个 `client` 包，等价于 COLA 各业务模块的 `xxx-client`：只放接口、DTO/VO、枚举、错误码和事件结构，不暴露实现。

```text
internal/user/
├── client/
│   ├── api/user.go    # Service 接口
│   ├── dto/user.go    # 请求响应
│   ├── vo/user.go     # 对外视图（按需）
│   ├── enum/user.go   # 对外枚举（按需）
│   └── error.go       # 错误码扩展
├── application/       # 实现 client/api 的接口
├── domain/
└── infrastructure/
```

```go
// internal/user/client/api/user.go
package api

// Service 是 user 上下文对其他上下文暴露的用例。
type UserService interface {
    FindByID(id uint) (*UserDTO, error)
}

// UserDTO 只携带跨上下文需要的数据。
type UserDTO struct {
    ID    uint
    Email string
    Name  string
}
```

被调上下文在 `application` 里实现该接口，调用方只持有接口：

```go
type financeService struct {
    users userapi.UserService // 只依赖契约包，不依赖对方的 application / domain
}
```

装配仍然只在 `cmd/` 完成：

```go
userAPI := application.NewUserAPI(userRepo)   // 实现 api.UserService
financeSvc := finance.NewFinanceService(userAPI)
```

契约包规则：

1. `client` 只放接口、DTO/VO、枚举、错误码和事件结构；不得 import `domain`、`application`、`infrastructure`。
2. 契约只传数据（ID、必要字段），不传聚合；密码散列等敏感字段不得出现在契约里。
3. 调用方不得 import 对方的 `application`、`domain`、`infrastructure`，只能 import 对方的 `client`。
4. A 与 B 互相调用会形成 import 循环，Go 编译期直接失败；此时把共用契约下沉到 `internal/contract/`，或改成事件。

#### 事件（异步解耦）

采购订单审批完成后：

```text
Purchase Context
      |
      v
OrderApproved Event
      |
      v
Inventory Context
      |
      v
创建待入库业务记录
```

事件应该表达已经发生的业务事实，事件结构与 topic 常量同样放在发布方的 `api` 包，订阅方只依赖契约：

```go
// internal/purchase/client/api/event.go
type OrderApproved struct {
    OrderID    string
    ApprovedAt time.Time
}
```

### 10.3 跨上下文数据规则

1. 不同上下文不共享 Domain Entity。
2. 不同上下文不直接修改对方数据库记录。
3. 对外暴露稳定的业务契约。
4. 对外传递必要的标识和业务数据。
5. 通过事件实现最终一致性时，应考虑幂等、重试和消息可靠性。
6. 需要跨模块事务时，应明确采用本地事务、协调流程或 Saga 等策略，不能假设多个模块的数据库操作天然处于同一事务中。

---

## 11. 启动装配与依赖注入

`cmd/` 是应用的组合根（Composition Root）。

负责：

- 读取并校验配置。
- 初始化数据库、缓存、Logger、消息队列。
- 构造 infrastructure 实现。
- 构造 application Service 并注入 Gateway。
- 构造 adapter Controller 并注册路由。
- 启动 HTTP Server、MQTT、队列等 server。

### 11.1 手动依赖注入

默认采用构造函数注入。

```go
func NewOrderService(orders order.Gateway) OrderService {
    return &orderService{orders: orders}
}
```

装配顺序固定为自内向外：

```text
config → DB / Redis / MQ
       → infrastructure（实现 Gateway）
       → application（注入 Gateway）
       → adapter（注入 Service）
       → 注册路由并启动
```

示例：

```go
func NewApp(c *config.Conf) *mago.App {
    repo := infrastructure.NewOrderRepository(db)
    svc := application.NewOrderService(repo)
    ctrl := adapter.NewUserController(svc)

    hs := NewHttpServer(c, ctrl)
    return mago.New(c.AppName, []mago.Server{hs})
}
```

### 11.2 DI 规则

1. 默认使用手动依赖注入。
2. 不使用全局 Service Locator。
3. 不在业务代码中随意创建数据库连接。
4. 不在 Controller 中初始化 Repository。
5. 不把依赖装配逻辑放入 domain。
6. 不因为模块增多就立即引入反射式 DI 框架，`wire` 之类按需评估。
7. 只有当手动装配的复杂度已经明显影响维护时，才评估 DI 工具。

---

## 12. 跨上下文技术能力

技术能力不单独占一层：跨上下文复用的技术代码用独立 Package 承载，或直接使用框架能力（`config`、`logx`、`httpx`、`queuex` 等）。

它与 DDD 的 Shared Kernel 不是同一个概念。

### 12.1 这里可以放什么？

- 数据库连接和初始化。
- 日志基础设施。
- OpenTelemetry 初始化。
- HTTP Server 配置。
- 消息队列客户端。
- 统一 ID 生成器。
- 系统时钟抽象。
- 事务基础设施。

### 12.2 这里不应该放什么？

不要把所有公共代码都当作技术能力收集起来。

以下内容通常不应放入：

- 采购订单领域模型。
- 库存领域模型。
- 财务业务规则。
- 各业务模块的 DTO。
- 各业务模块专用 Gateway 实现。
- 业务状态枚举。

---

## 13. 错误处理

错误应该在合适的层级处理。

### 13.1 Domain 错误

领域层不自定义错误：不建 `error.go`，不声明 `ErrXxx` 变量，不定义错误类型。只能使用标准库与框架 `mago` 的全局错误码。

```go
func (u *User) CanLogin() error {
    if !u.IsActive {
        return mago.NewError(mago.ErrAuthFailed).WithMsg("账号已停用")
    }

    return nil
}
```

约束：

1. 领域错误统一由 `mago.NewError(code).WithMsg(msg)` 表达，code 取自框架内置错误码。
2. 错误信息要带业务语义，adapter 可以直接回显给客户端。
3. 需要新的错误码时必须在 `client/error.go` 扩展（见 13.2），domain 不得引用 `client`，只能用框架内置码。
4. 不鼓励在 domain 里字符串拼装错误，也不允许把 `error` 拆成自定义类型供上层做类型断言。

### 13.2 Application 错误

Application 可以处理用例级错误，例如：

- 资源不存在。
- 权限不足。
- 依赖服务不可用。
- 事务执行失败。

需要扩展框架的错误码时，统一在 `client/error.go` 中声明并注册文案：

```go
const (
    ErrUserDisabled mago.ErrorCode = 401501 // 认证类，HTTP 401
    ErrEmailTaken   mago.ErrorCode = 422101 // 参数类，HTTP 422
)

func init() {
    mago.InitErrorCodeMsg(ErrUserDisabled, "账号已停用")
    mago.InitErrorCodeMsg(ErrEmailTaken, "邮箱已被注册")
}
```

错误码规则：

1. 前三位是 HTTP 状态码，后三位是上下文内业务编号，同一上下文内不得重复。
2. 扩展只放在 `client/error.go`，adapter 与 application 可直接使用，其他上下文也能复用同一段号。
3. domain 只能使用框架内置错误码，不得引用这些扩展（否则 domain 会反向依赖契约层）。

### 13.3 Adapter 错误

HTTP Handler 将应用错误转换为 HTTP 响应。

例如：

| 错误类型 | HTTP 状态码 |
|---|---:|
| 参数错误 | 400 |
| 未认证 | 401 |
| 无权限 | 403 |
| 资源不存在 | 404 |
| 业务状态冲突 | 409 |
| 内部错误 | 500 |

实际错误码和业务错误响应格式应统一定义。

### 13.4 错误处理原则

1. 不允许静默忽略错误。
2. 不要在每一层重复包装相同错误信息。
3. 使用 `%w` 保留错误链。
4. 通过 `errors.Is` 和 `errors.As` 判断错误。
5. 不要将数据库原始错误直接返回给客户端。
6. 日志中应保留足够的排查上下文，但避免记录敏感数据。
7. 基础设施错误不得伪装成业务错误：查询失败、网关超时不能返回「密码错误」这类业务文案，应原样上抛。

---

## 14. 事务管理

事务边界由 Application 用例决定。

例如审批采购订单：

```text
Service.ApproveOrder
       |
       v
开始事务
       |
       v
读取订单
       |
       v
执行领域行为
       |
       v
保存订单
       |
       v
写入操作记录
       |
       v
提交事务
```

如果这些操作必须保持原子性，应处于同一个事务边界内。

### 14.1 事务规则

1. Domain 不依赖数据库事务实现。
2. Application 决定用例的事务边界。
3. infrastructure 负责具体数据库操作。
4. 事务基础设施放在跨上下文技术包（见第 12 章），不放进 domain。
5. 不要让 HTTP Handler 自己控制业务事务。
6. 外部 HTTP 调用和消息发送不能简单地认为可以与数据库事务原子提交。
7. 对数据库写入与消息发布之间的一致性要求较高时，应评估 Outbox Pattern。

事务接口可以根据实际项目需要设计，不应为了抽象而提前建立复杂的事务框架。

---

## 15. 测试策略

测试应尽量靠近对应的 Package。

### 15.1 Domain 测试

重点验证：

- Entity 行为。
- Aggregate 业务不变量。
- 状态流转。
- Value Object 约束。
- Domain Service 规则。

Domain 测试不应该依赖真实数据库。

### 15.2 Application 测试

重点验证：

- 用例执行流程。
- Repository 调用。
- 业务流程编排。
- 事务边界。
- 异常场景。

可以通过 Fake 或 Mock 实现依赖替换。

### 15.3 Adapter 测试

重点验证：

- HTTP 请求解析。
- 参数校验。
- Response 映射。
- 数据库持久化。
- 消息发布和消费。

### 15.4 集成测试

重点验证：

- 数据库真实读写。
- 事务行为。
- 多模块协作。
- 消息处理。
- 关键业务流程。

不要为了追求测试数量而编写大量没有实际价值的 Mock 测试。

---

## 16. Go 工程规范

### 16.1 基本规范

1. 使用 `gofmt` 格式化代码。
2. 使用 Go Modules 管理依赖。
3. 优先使用 Go 标准库。
4. 避免没有必要的第三方依赖。
5. 使用构造函数完成依赖注入。
6. 跨层方法一律 ctx 风格：application Service、domain Gateway、repository、端口实现的首参必须是 `context.Context`，
   并一路透传到 IO 调用（gorm 用 `WithContext(ctx)`、HTTP/MQ 客户端同理由 ctx 驱动），禁止在业务代码里用 `context.Background()`。
7. 遵循 Go 的错误处理习惯。
8. 优先使用小接口。
9. 不要为了设计模式而设计模式。
10. 避免过度抽象。

### 16.2 Package 命名

Package 名称应：

- 简短。
- 小写。
- 清晰。
- 不使用下划线。
- 不使用无意义的缩写。

推荐：

```go
package purchase
package inventory
package order
package adapter
package mysql
```

### 16.3 接口设计

Go 倾向于小接口。

推荐：

```go
type OrderReader interface {
    GetByID(ctx context.Context, id string) (*Order, error)
}
```

避免创建包含大量无关方法的万能接口。

接口应由使用它的一方定义，而不是为了每个结构体都创建一个接口。

接口实现关系不靠满屏的 `var _ Xxx = (*T)(nil)` 声明，按是否有调用方分两种写法：

1. 有调用方的接口：编译期在传入处自动校验（如 `NewUserService(users domain.UserGateway)` 传入 `*repository.User`），生产代码里不写断言。
2. 没有调用方的对外契约：在测试文件里写一次 `var _ api.UserService = (*userService)(nil)`，接口一漂移 `go test` 就失败，生产代码保持干净。
3. 不用 `var _ Xxx = &T{}`：会实例化零值，对带锁、带必填字段的类型不安全。

### 16.4 不要滥用 pkg/

本项目默认不需要创建全局 `pkg/` 目录。

`internal/` 已经可以表达应用内部代码的访问边界。

只有当某些代码确实需要作为可复用的公共 Go Package 对外发布时，才考虑 `pkg/`。

### 16.5 不创建全局技术分类目录

不推荐：

```text
internal/
├── handlers/
├── services/
├── repositories/
├── models/
└── dto/
```

这种结构容易让不同业务上下文的代码混杂在一起。

应优先按照业务上下文组织：

```text
internal/
├── user/
├── catalog/
└── finance/
```

### 16.6 依赖选型优先级

遇到**协议类**（MQTT、WebSocket、gRPC、MCP、SSE…）、**功能类**（加解密、压缩、时区、鉴权、序列化…）、**开放平台接入类**（钉钉、微信、支付、OSS、短信…）的代码，按下表从上到下取第一个可用方案，不得跳级：

| 优先级 | 来源 | 判断标准 | 示例 |
| --- | --- | --- | --- |
| 1 | Go 标准库 | 标准库已有等价能力，且性能可接受 | `crypto/*`、`encoding/*`、`net/http`、`context`、`time` |
| 2 | mago 仓库组件 | 本仓库已封装且语义匹配，避免重复造轮子 | 错误码、`AbstractRepository`、DTO/VO、配置、日志 |
| 3 | GitHub 社区官方库 | 由协议方或官方组织维护 | `modelcontextprotocol/go-sdk`、`cloudwego/hertz`、`gorm.io/gorm` |
| 4 | GitHub 高热度且仍在维护 | 官方库缺位时的可用替代，需多人维护、近期有提交 | 详见下方判定标准 |
| 5 | 自己实现 | 以上都不满足或都不合规时的最后手段 | 必须写清取舍理由并补测试 |

选型规则：

1. 优先级只能逐级下降，跳级必须在代码注释与提交说明里写明原因（例如官方库只提供客户端、与 hertz/gorm 生态冲突）。
2. 「高热度且仍在维护」不能只看 star：至少满足近 6 个月有提交、有两个以上活跃维护者、License 可商用。
3. 自己实现前先确认没有重复 mago 已有能力；自实现部分必须有单元测试，并对协议边界做异常用例。
4. 新增依赖时在提交说明中回答两个问题：为什么不用上一级？该依赖由谁维护、最近一次提交是什么时候？
5. 本原则与第 18.2 节「不要随意引入第三方框架」一致：选型不只看能不能用，还要看是否最靠前。

---

## 17. 架构依赖约束

这是所有开发人员和 AI Agent 都必须遵守的约束。

| 依赖方向 | 是否允许 | 说明 |
|---|---|---|
| `adapter` → `application` | 允许 | 入口适配器调用用例服务 |
| `adapter` → `client` | 允许 | 复用请求 / 响应结构与错误码 |
| `application` → `client` | 允许 | 实现 client/api 接口、使用 dto 与错误码 |
| `domain` → `client` | 禁止 | 领域层不认识对外契约 |
| `application` → `domain` | 允许 | 调用聚合、领域服务与 Gateway 接口 |
| `infrastructure/repository` → `domain` | 允许 | 实现 domain 定义的 Gateway 接口（写路径） |
| `infrastructure/repository` 的读方法 → `client/vo` | 允许 | 直查列并组装视图，不加载聚合 |
| 一个 `do/` 被两个 repository 引用 | 禁止 | 一个 DO 只对应一个 repository，读写方法都放它身上 |
| `application` 定义读模型接口 → 同一 repository 实现 | 允许 | 轻量 CQRS 查询路径 |
| `cmd/` → 任意层 | 允许 | 组合根负责装配，唯一例外 |
| `application` → `adapter` | 禁止 | 用例不能反向依赖适配器 |
| `application` → `infrastructure` | 禁止 | 用例只能通过 Gateway 接口访问外部 |
| `domain` → `application` | 禁止 | 领域层不能依赖用例编排 |
| `domain` → `adapter` | 禁止 | 领域层不能依赖协议 |
| `domain` → `infrastructure` | 禁止 | 领域层不能依赖具体技术实现 |
| `domain` → GORM/Hertz 等技术框架 | 禁止 | 领域层保持技术无关 |
| `adapter` → `domain` | 原则上禁止 | 需要领域能力时由 application 暴露 |
| `adapter` → `infrastructure` | 禁止 | 绕过用例直接读写数据会破坏分层 |
| 上下文 → 其他上下文的内部实现 | 原则上禁止 | 通过公开契约通信（见第 10 章） |

### 17.1 依赖规则补充

Go 编译器不会自动识别分层，也不会自动强制所有架构依赖规则。

必须通过以下方式约束：

1. 合理拆分 Package：`domain`、`application`、`adapter`、`infrastructure` 是四个真实 Package，不是同一 Package 下的文件名前缀。
2. 控制导出：`domain` 只导出聚合、值对象、领域服务、Gateway 接口与领域错误；`infrastructure` 的持久化 Model 不导出到模块外。
3. 依赖方向用自动化测试守护：在上下文内写 `arch_test.go`，用 `go/packages` 解析 import，断言 `domain` 的导入中不出现 `application`、`adapter`、`infrastructure`，`application` 的导入中不出现 `adapter`、`infrastructure`。
4. 也可使用 `go-arch-lint`、`depguard` 之类的静态检查工具，把规则固化到 CI。
5. 在代码评审中检查依赖方向。

特别注意：如果 Handler、Domain、Repository 全部放在同一个 Go Package，即使它们被分成不同文件，Go 编译器也不会为它们建立独立的依赖隔离。

因此，**需要严格隔离的架构职责，必须通过真正的 Package 边界实现，而不只是通过文件名或注释实现。**

---

## 18. AI Agent 开发约束

AI Agent 在新增、修改 Go 代码之前，必须先读完本文件 —— `README.md` 是架构规范的**唯一文档入口**（不另设 docs/、AGENTS.md）。快速索引见 `llms.txt`。

必须遵守本文件定义的：

- 项目目录结构（第 2 章）
- Go Package 设计规则（第 3 章）
- DDD 分层与依赖方向（第 7、17 章）
- Gateway 端口与「一个 DO 只有一个 repository」（第 6 章）
- 轻量 CQRS 读写分离（第 8 章）
- DTO / VO / DO 分离与 client 契约（第 9、10 章）
- 跨 Bounded Context 通信（第 10 章）
- ctx 风格透传（第 16.1 节）
- 数据迁移与修数脚本位置：`internal/<ctx>/adapter/script/`（不建 migrations 目录）

不得为了完成单个需求而破坏既有架构约束。

### 18.1 开发前

1. 阅读本架构规范（配合 `llms.txt` 的速查表）。
2. 确认需求所属的 Bounded Context。
3. 确认涉及的业务 Entity、Aggregate 和 Value Object。
4. 判断需求属于 Command 还是 Query。
5. 确认是否存在已有的 Application Service 方法。
6. 确认是否存在可复用的 Repository 和 Adapter。
7. 检查跨上下文依赖是否符合约束。

### 18.2 开发时

1. 优先在现有业务 Package 中扩展。
2. 不要为了一个简单需求创建大量新 Package。
3. 不要将业务规则写入 HTTP Handler。
4. 不要让 Domain 依赖数据库和 Web 框架。
5. 不要将数据库 Model 直接作为 HTTP Response。
6. 不要让 Query 强制经过 Domain Aggregate。
7. 不要为了减少字段重复而合并不同职责的模型。
8. 不要直接修改其他 Bounded Context 的内部领域对象。
9. 不要随意引入新的第三方框架；协议、功能、开放平台接入一律按第 16.6 节的优先级选型（标准库 → mago 组件 → 社区官方库 → 高热度且在维护的库 → 自己实现）。
10. 不要为了追求架构完整而引入当前没有实际需求的抽象。

### 18.3 开发完成后

必须检查：

- [ ] 是否符合 Package 边界？
- [ ] 是否符合依赖方向？
- [ ] 是否存在不合理的跨上下文依赖？
- [ ] Domain 是否保持技术无关？
- [ ] Command 是否正确执行领域规则？
- [ ] Query 是否采用合理的查询方式？
- [ ] DTO、Domain Entity 和 Persistence Model 是否职责清晰？
- [ ] 事务边界是否合理？
- [ ] 是否处理错误？
- [ ] 是否补充必要测试？
- [ ] 是否通过 `go test ./...`？
- [ ] 是否通过项目配置的代码检查？

如果现有架构无法满足新需求，Agent 应先说明原因和建议的架构调整，不得擅自破坏既有架构约束；冲突未裁决前不动手。

---

## 19. 最终架构原则

本项目遵循以下原则：

1. **业务优先：** 一级目录按照 Bounded Context 组织，而不是按照技术类型组织。
2. **Package 优先：** Go Package 是实际的代码封装和依赖边界。
3. **适度 DDD：** 复杂业务使用领域模型，简单 CRUD 不强行复杂化。
4. **依赖倒置：** 业务核心不依赖具体基础设施。
5. **轻量 CQRS：** 写操作通过领域模型保证业务规则，读操作可以直接查询数据库。
6. **模型职责分离：** Domain Entity、DTO、Persistence Model 和 HTTP Model 不强行合并。
7. **模块独立：** 不同 Bounded Context 不直接操纵对方内部领域模型。
8. **手动装配：** 默认使用构造函数注入，由 `cmd/` 统一装配。
9. **技术能力独立：** 跨上下文技术包不承载业务领域模型。
10. **避免过度设计：** 不为了形式上的架构完整而创建无实际价值的抽象、目录和接口。

最终目标不是追求最复杂的 DDD 架构，而是建立一套：

**符合 Go 语言习惯、业务边界清晰、便于长期维护、适合多人协作，并且 AI Agent 能够稳定遵循的后端工程架构。**

---

## 20. 参考资料

以下资料用于理解 Go 语言和相关架构思想。项目规范以本文件明确约定为准。

- Go 官方文档：https://go.dev/doc/
- Go 官方 Effective Go：https://go.dev/doc/effective_go
- Go 官方组织结构建议：https://go.dev/doc/modules/layout
- Go 官方 `internal` Package 机制：https://go.dev/doc/go1.4#internalpackages
- Go 官方代码审查建议：https://go.dev/wiki/CodeReviewComments
- Martin Fowler - CQRS：https://martinfowler.com/bliki/CQRS.html
- Hexagonal Architecture：https://alistair.cockburn.us/hexagonal-architecture/

注意：Go 社区并不存在一套强制适用于所有项目的官方目录结构。本文档定义的是本项目自己的工程约定，而不是 Go 语言官方强制标准。
