# Architecture Diagnosis: Audit Actor Values for Multi-Channel APIs

## Goal

诊断当系统同时支持内部 `/api` 和第三方 `/open-api`，且展示层扩展为 Web Browser、Desktop App、小程序、第三方应用时，业务表上的 `creator` / `updator` 应该存储什么值。目标是给出一条稳定的审计字段语义，避免把展示层、调用方应用、API key、真实业务主体混在同一字段里。

## What I Already Know

* 当前后端已有两类 API surface：
  * `/api/*`：内部管理/前端 API。
  * `/open-api/v1/*`：第三方开放 API。
* `index.go` 中 `/api` 使用普通登录认证，`/open-api/v1` 使用开放 API key。
* `api/framework/usecase/context.go` 已定义：
  * `SurfaceInternalAPI = "api"`
  * `SurfaceOpenAPI = "open-api"`
  * `SurfaceSystem = "system"`
  * `ActorContext`：内部登录用户身份。
  * `ConsumerContext`：开放 API 消费者身份，包括 `KeyID`、`PartnerID`、`AccountID`、`Environment`、`Scopes`。
* `api/framework/http/context/usecase_context.go` 已把内部用户映射到 `ctx.Actor`，把开放 API key 映射到 `ctx.Consumer`。
* 当前 schema 中没有发现统一的 `creator` / `updator` 审计列；这是一个预备架构决策。

## Diagnosis

`creator` / `updator` 不应该存“展示层”，例如 `web`、`desktop`、`mini_program`、`third_party_app`。这些值描述的是入口、客户端或渠道，不是业务上的操作者。

推荐语义是：`creator` / `updator` 存储本次写操作的最终责任业务主体，也就是可以被追责、授权、归属、查询的 principal。

展示层、API surface、第三方 partner、API key 等上下文应该进入独立字段或审计日志，而不是挤进 `creator` / `updator`。

如果采用 typed actor 表达，例如 `user:019...`、`openapi_account:acc_...`、`service:scheduler`、`integration:creem`，则必须明确：这些字段仍然是审计 actor，不是数据权限归属字段。数据权限应基于资源自己的 owner / subject / tenant 字段判断，而不是基于 `creator` / `updator` 判断。

## Recommended Rule

`creator` / `updator` 存业务主体 ID，优先使用系统内可归属的 account/user ID：

| 调用来源 | API surface | `creator` / `updator` 推荐值 | 额外审计上下文 |
| --- | --- | --- | --- |
| Web Browser 后台 | `/api` | 登录用户 `Actor.UserID` | `surface=api`，可选 `client=web` |
| Desktop App | `/api` | 登录用户 `Actor.UserID` | `surface=api`，可选 `client=desktop` |
| 小程序 | `/api` | 登录用户 `Actor.UserID` | `surface=api`，可选 `client=mini_program` |
| 第三方应用 | `/open-api/v1` | `Consumer.AccountID` | `surface=open-api`，`partner_id`，`key_id`，可选 `client=third_party_app` |
| 系统任务/事件订阅 | internal/system | 固定系统主体，例如 `system` 或专用 service account ID | `surface=system`，`job/subscriber/event_id` |

如果业务表的 `creator` / `updator` 需要外键指向 `users.id`，那么开放 API 的写操作应存 `Consumer.AccountID`，因为当前项目中 `open_api_partners.account_id` 已绑定到 `users.id`。

如果未来需要支持非用户型主体，例如企业租户、纯机器账号、服务账号，那么不应继续让 `creator` / `updator` 裸存字符串 ID，而应升级为 typed actor：

* `creator_type` / `creator_id`
* `updator_type` / `updator_id`

示例类型：`user`、`open_api_account`、`service_account`、`system`。

## Rationale

* 同一个人可能从 Web、Desktop、小程序执行同一操作。审计字段应该显示“谁改了”，而不是“从哪个端改了”。
* 第三方应用通过 `/open-api` 写入时，API key 只是凭证，partner 是调用组织或应用，最终业务归属通常是绑定的 account/user。
* 如果把 `creator` 写成 `web`、`desktop`、`open-api-key-xxx`，后续权限、数据归属、报表、问题追踪都会变得困难。
* 如果只写 `Consumer.AccountID` 而完全丢弃 `key_id` / `partner_id`，又会损失第三方调用排查能力，所以这些信息应该在额外审计上下文中保留。

## Proposed Architecture

在 usecase 层统一从 `fwusecase.Context` 解析审计 actor：

* `SurfaceInternalAPI`：要求 `ctx.Actor.Authenticated == true`，返回 actor type `user` 和 `ctx.Actor.UserID`。
* `SurfaceOpenAPI`：要求 `ctx.Consumer.Authenticated == true`，返回 actor type `user` 或 `open_api_account`，当前项目推荐优先返回 `ctx.Consumer.AccountID`。
* `SurfaceSystem`：返回 `system` 或后续扩展的 service account。

业务模型写入时不要直接关心 HTTP route、header、客户端类型或 API key 解析细节，只接收已经规范化的审计 actor。

### Permission Model Impact

数据权限会受影响，但影响点不是“typed actor 能不能用”，而是必须拆清楚三类概念：

* `actor`：本次操作是谁/什么发起的，用于审计，例如 `user:019...`、`service:scheduler`、`integration:creem`。
* `subject` / `principal`：请求上下文中被授权的业务主体，表示这次请求“代表谁”访问系统，例如当前登录用户、开放 API key 绑定的 account/user、service account。
* `owner`：资源表上的归属或隔离字段，用于和 subject/principal 的 claims 比对，例如 `orders.user_id`、`documents.owner_user_id`、`account_id`、`tenant_id`。

权限查询必须使用资源 owner 字段，而不是 `creator` / `updator`：

* 用户查看本人订单：subject 是 `user:019...`，资源 owner 过滤为 `WHERE orders.user_id = subject.user_id`。
* 第三方应用查看绑定账号的数据：subject 是开放 API key 绑定的 account/user，资源 owner 过滤为 `WHERE resource.owner_user_id = subject.account_id`，同时校验 `partner_id`、`key_id`、`scope`。
* 租户内数据访问：subject 带有允许访问的 tenant claims，资源 owner 过滤为 `WHERE resource.tenant_id IN subject.tenant_ids`。
* 定时任务更新用户数据：`updated_by = service:scheduler`，但 `owner_user_id` 不变；用户仍然能看到自己的数据。
* 支付回调更新订单：`updated_by = integration:creem`，但订单归属仍由 `orders.user_id` 决定。

因此，不建议用 `creator` / `updator` 实现“我的数据”过滤。它只能回答“谁创建/最后修改了这条记录”，不能可靠回答“这条记录属于谁”。

### Scenario: Multi-Channel Order List

当同一个 order list 同时服务 Web Browser、Desktop App、小程序以及第三方 `/open-api` 时，复用边界应该放在 usecase 查询能力和 model 数据访问能力上，而不是复用 HTTP route DTO。

推荐分层：

```text
presentation/client
  -> transport route adapter
  -> usecase application query + permission policy
  -> model repository/query
  -> db
```

对应当前项目目录，短期可以保持 flat package，但按文件边界拆清：

```text
api/
  routes/
    order.go              # 内部 /api 订单 DTO 和 handler
    open_api_order.go     # 第三方 /open-api/v1 订单 DTO 和 handler
  usecase/
    order.go              # 订单 command/核心业务，已有文件可逐步拆分
    order_query.go        # ListOrders / GetOrderDetail 等查询用例
    order_access.go       # subject -> owner filter / permission policy
  models/
    order.go              # Order struct 与基础写入
    order_query.go        # ListOrdersByFilter / CountOrdersByFilter
frontend/
  src/
    api.js                # Web Browser 内部 /api client
```

未来如果桌面 App 和小程序不复用 Svelte 前端代码，也建议各自维护客户端 SDK/UI 代码，但后端仍然复用 `/api` transport 和 usecase：

```text
clients/
  desktop/                # 可选：桌面 app 代码或 SDK
  mini-program/           # 可选：小程序代码或 SDK
  openapi-sdk/            # 可选：第三方开放 API SDK
```

内部端与开放 API 的 route 必须分开：

* `api/routes/order.go`：面向内部产品端，使用内部 envelope，字段可以服务后台/前台需要，但仍不能泄露敏感字段。
* `api/routes/open_api_order.go`：面向第三方，使用 Open API envelope，字段契约稳定，不复用内部 DTO。

usecase 应提供统一的查询入口，例如：

```go
type ListOrdersQry struct {
    Scope    OrderAccessScope
    Page     int
    PageSize int
}

type OrderAccessScope struct {
    Mode      string // "mine", "admin", "account", "tenant"
    UserID    string
    AccountID string
    TenantID  string
}

func ListOrders(ctx fwusecase.Context, qry ListOrdersQry) (OrdersCo, error)
```

更稳妥的实现形态是“多个语义明确的 usecase 入口 + 一个内部复用核心”：

```go
func ListMyOrders(ctx fwusecase.Context, qry ListMyOrdersQry) (OrdersCo, error)
func ListAdminOrders(ctx fwusecase.Context, qry ListAdminOrdersQry) (OrdersCo, error)
func ListOpenAPIOrders(ctx fwusecase.Context, qry ListOpenAPIOrdersQry) (OrdersCo, error)

func listOrders(ctx fwusecase.Context, filter OrderListFilter) (OrdersCo, error)
```

其中公开 usecase 入口负责解析 subject、校验权限、决定 owner filter；内部 `listOrders` 只复用分页、排序、查询和 `Co` 组装。这样可以避免 route 层构造一个过强的 `Scope` 参数，从而绕过“我的订单”和“订单管理”的权限差异。

route 层不应该允许普通用户传入任意 `user_id` 来决定列表范围。route 层只负责把 HTTP 输入转换为“意图”，真正的 owner 过滤条件由 usecase 根据 `fwusecase.Context` 推导：

* Web/Desktop/小程序的“我的订单”：subject 来自 `ctx.Actor.UserID`，usecase 推导 `owner.user_id = ctx.Actor.UserID`。
* 内部管理员订单列表：需要 `ctx.Actor.IsAdmin == true`，允许指定 `user_id/status/page` 等筛选。
* 第三方开放 API 订单列表：subject 来自 `ctx.Consumer.AccountID`，usecase 推导 `owner.account_id/user_id = ctx.Consumer.AccountID`，同时校验 scope，例如 `orders:read`。

model 层只接收已经由 usecase 决定好的查询条件，不读取 `fwusecase.Context`、不判断 JWT/API key、不知道 Web/Desktop/小程序/Open API：

```go
type OrderListFilter struct {
    UserID    string
    AccountID string
    TenantID  string
    Status    string
    Limit     int
    Offset    int
}
```

当前代码中的 `GET /api/orders/user/:user_id` 和 `usecase.GetUserOrders(ctx, UserOrdersQry{UserID: ...})` 适合作为早期 demo，但在多端和数据权限场景下存在风险：普通登录用户如果能控制 path 中的 `user_id`，就可能请求别人的订单。推荐改为：

```text
GET /api/user/orders
GET /api/admin/orders?user_id=...
GET /open-api/v1/orders
```

其中：

* `/api/user/orders`：只看当前登录用户自己的订单。
* `/api/admin/orders`：后台管理列表，必须 admin，允许跨用户筛选。
* `/open-api/v1/orders`：第三方只看 API key 绑定 account 的订单，字段契约独立。

数据流建议：

```text
Web/Desktop/Mini Program
  -> GET /api/user/orders
  -> routes.ListMyOrders
  -> usecase.ListOrders(ctx, scope from Actor.UserID)
  -> models.ListOrders(filter.UserID = Actor.UserID)
  -> routes.OrderResponse

Internal Admin
  -> GET /api/admin/orders?user_id=u1
  -> routes.ListAdminOrders
  -> usecase.ListOrders(ctx, admin scope + optional filters)
  -> models.ListOrders(filter)
  -> routes.AdminOrderResponse

Third-party App
  -> GET /open-api/v1/orders
  -> routes.ListOpenAPIOrders
  -> usecase.ListOrders(ctx, scope from Consumer.AccountID + scope orders:read)
  -> models.ListOrders(filter.AccountID/UserID = Consumer.AccountID)
  -> routes.OpenAPIOrderResponse
```

复用原则：

* 复用 `usecase.ListOrders`、分页规范、排序规范、owner 过滤策略、model 查询。
* 不复用内部 `/api` DTO 给 `/open-api`。
* 不复用前端 Web 的 `frontend/src/api.js` 给第三方应用；第三方应使用公开 API 文档或独立 SDK。
* 不让客户端决定 owner，只允许客户端传普通筛选条件，例如 status、time range、page。
* 数据权限检查靠 usecase policy，不靠 route 命名约定，不靠 model 自行猜测。

#### Permission Boundary Placement

权限边界应分两层放置：

* route / middleware 层负责入口级权限：是否登录、是否 admin、是否携带 Open API key、是否具备 API scope、这个 HTTP endpoint 是否允许当前 surface 访问。
* usecase 层负责业务级和数据级权限：当前 subject 能访问哪些 owner 范围、是否允许跨用户查询、系统任务或第三方回调能否修改目标资源。

因此，`/api/admin/orders` 应在 route/middleware 层先要求 `RequireAdmin()`；`/open-api/v1/orders` 应在 route/middleware 层先要求 API key 和 `orders:read` scope。通过入口校验后，usecase 仍然要根据 `ctx.Actor` / `ctx.Consumer` 推导 owner filter，不能相信 route 传入的 owner 条件。

推荐规则：

```text
routes/middleware = gatekeeper for endpoint access
usecase           = authority for business/data access
models            = storage query only
```

如果只把数据权限放在 routes，后续同一个 usecase 被 Open API、系统任务、事件订阅、测试 helper 或其他非 HTTP 入口复用时，容易绕过权限约束。如果只把权限放在 usecase，route 的入口语义又不够清晰，错误 endpoint 可能先进入业务层才被拒绝。

#### Native SQL Filter Convention

订单列表的 owner filter 和普通筛选条件应使用项目既有的原生 SQL 动态条件语法：`db.DynamicExecutorFor(ctx, "app")` + `#[ ... ]`。不要在 usecase 或 model 中手写字符串拼接 WHERE 条件。

职责分配：

* route/middleware：校验入口权限，并把 HTTP query 转成普通筛选意图，例如 `status`、`page`、`page_size`。
* usecase：根据 `ctx.Actor` / `ctx.Consumer` 决定强制 owner 条件和是否允许跨用户查询。
* model：接收 usecase 已经决定好的 `OrderQuery`，使用 `#[ ... ]` 执行动态 SQL。

示例：

```go
type OrderQuery struct {
    UserID    string
    AccountID string
    TenantID  string
    Status    string
    Limit     int
    Offset    int
}

func ListOrders(ctx context.Context, query OrderQuery) ([]Order, error) {
    eng, err := db.DynamicExecutorFor(ctx, "app")
    if err != nil {
        return nil, fmt.Errorf("database unavailable: %w", err)
    }

    sql := `
        SELECT * FROM orders
        WHERE 1=1
            #[ AND user_id = :user_id ]
            #[ AND account_id = :account_id ]
            #[ AND tenant_id = :tenant_id ]
            #[ AND status = :status ]
        ORDER BY created_at DESC, id DESC
        LIMIT :limit OFFSET :offset
    `

    var orders []Order
    if err := eng.Select(&orders, sql, query); err != nil {
        return nil, fmt.Errorf("list orders failed: %w", err)
    }
    return orders, nil
}
```

关键约束：`UserID` / `AccountID` / `TenantID` 是否出现，不应由客户端直接决定；这些 owner 条件必须由 usecase 权限策略填入。`#[ ... ]` 只是 SQL 执行层的条件开关，不是权限判断本身。

### Scenario: Non-HTTP Entrypoints

定时任务入口和消息监听处理入口不应该都放到 `api/routes`。`routes` 的职责是 HTTP transport adapter：读取 HTTP path/query/body、构造 `fwusecase.Context`、调用 usecase、映射 response DTO，并返回 HTTP envelope。

推荐边界：

```text
api/routes/*                  # HTTP API only
index.go / bootstrap          # process startup, runner registration
api/framework/queue           # queue primitive and runner wrapper
api/framework/events          # domain event bus and durable message dispatch
api/usecase/scheduler.go      # scheduled task management and job execution usecase
api/usecase/events/*          # domain event subscriber adapters to usecase
api/usecase/payment_webhook.go # webhook receipt/job business handling
```

可以放在 `routes` 的内容：

* 定时任务管理 API，例如 `GET /api/scheduler/tasks`、`POST /api/scheduler/tasks`、`PATCH /api/scheduler/tasks/:id/enabled`。
* 消息/事件的后台查询 API，例如 `GET /api/messages`、`GET /api/events`。

不应该放在 `routes` 的内容：

* 定时器 tick loop，例如周期性调用 `EnqueueDueScheduledTasks(...)`。
* queue runner 注册和启动，例如 `scheduledRunner.Register(...)`、`durableRunner.Register(...)`。
* domain event subscriber 的消息处理入口，例如 `fwevents.HandleMessage` 或 `order.paid` subscriber。
* webhook queue job 处理入口，例如支付回调已经落库入队后的异步处理。

决策理由：

* 这些入口没有 HTTP request/response 生命周期，放进 `routes` 会错误绑定 Echo、HTTP envelope、middleware、status code 等概念。
* 定时任务和消息监听通常使用 `SurfaceSystem` 或 integration actor，不是用户请求；认证/授权模型不同于 HTTP route。
* queue message 的 ACK、retry、visibility timeout、idempotency、dead-letter 风险都属于 worker/queue 语义，不属于 route 语义。
* 同一个 usecase 应能被 HTTP、scheduler、queue subscriber、CLI/test helper 复用；如果入口逻辑混入 routes，会降低复用并绕开非 HTTP 场景。
* 当前项目已经采用该边界：`api/routes/scheduler.go` 只做 HTTP 管理接口；`index.go` 中的 `startQueueRunners` / `runSchedulerLoop` 负责 worker 启动；`api/framework/events` 负责 durable event dispatch；`api/usecase/events/*` 负责 subscriber 到业务 usecase 的适配。

如果后续非 HTTP 入口增多，可以从 `index.go` 抽出一个 bootstrap/worker registration 文件或包，但仍不应放到 `api/routes`。

### Scenario: First-Party Apps vs Personas

“第一方应用”只表示客户端由本系统/本组织控制，使用内部认证体系和内部 API 契约；它不表示所有第一方客户端都应该共享同一个 route。

推荐拆成三层判断：

```text
ownership: first-party / third-party
persona: parent / teacher / admin / operator / user
surface: web / desktop app / mini program / mobile app
```

其中：

* `ownership` 决定走 `/api` 还是 `/open-api`。
* `persona` 决定 route namespace、权限策略、owner/scope 过滤。
* `surface` 通常不决定 route；只有 DTO、同步协议、性能或设备能力明显不同时才单独拆 endpoint。

示例：

| 客户端 | ownership | persona | 推荐 API |
| --- | --- | --- | --- |
| 家长小程序 | first-party | parent | `/api/parent/...` |
| 老师小程序 | first-party | teacher | `/api/teacher/...` |
| 管理 Web | first-party | admin/staff | `/api/admin/...` |
| 管理 App | first-party | admin/staff | `/api/admin/...` |
| 第三方应用 | third-party | partner/app consumer | `/open-api/v1/...` |

因此，家长小程序和老师小程序虽然都是 first-party，也都可能是 mini program surface，但它们不是同一个权限场景：家长通常按 `parent_id` / `student_id` / guardian relationship 过滤；老师通常按 `teacher_id` / `class_id` / school scope 过滤。它们应优先拆成不同 persona route 和不同 usecase 入口，再复用底层查询核心。

管理 Web 和管理 App 如果只是展示层不同、角色和权限一致，可以共用 `/api/admin/...`。如果管理 App 需要离线同步、设备授权、轻量字段或不同操作能力，可以新增专用 endpoint，但仍属于 `/api` 内部契约，而不是 `/open-api`。

决策规则：

* 同 ownership、同 persona、同权限语义、同 DTO 契约：可以共享 route。
* 同 ownership、不同 persona 或 owner/scope 过滤不同：拆 route/usecase 入口，复用内部查询核心。
* 同 persona、不同 surface：默认共享 route；只有契约或协议不同才拆 endpoint。
* third-party 永远不要复用内部 `/api` DTO 和 route，走 `/open-api/v1`。

### 4.1 Route Design

本任务确认采用 persona-first 的 route 设计：先按 persona 和权限语义拆 route，再判断 surface 是否需要特殊 endpoint。Web、App、小程序、Desktop 这类 surface 默认不直接决定 route；它们只有在 DTO、性能、同步协议、设备能力或发布节奏存在明确差异时，才新增专用 endpoint。

统一命名规范：

```text
/api/<persona>/<resource>
/api/<persona>/<feature-module>/<resource>
/open-api/v1/<resource>
/open-api/v1/<feature-module>/<resource>
```

其中：

* `ownership` 决定 `/api` vs `/open-api`：第一方应用使用 `/api`，第三方开放调用使用 `/open-api/v1`。
* `persona` 决定 route namespace 和权限边界，例如 `/api/parent/...`、`/api/teacher/...`、`/api/admin/...`。
* `surface` 决定 DTO、性能、设备能力和同步协议差异；默认共享同 persona route，不按 Web/App/小程序机械拆分。

路由和 DTO 应按契约稳定性做版本化，避免某个 surface 升级影响其他客户端。内部 `/api` 可以通过新 endpoint 或新 DTO 字段渐进演进；公开 `/open-api/v1` 必须保持外部契约稳定，必要时引入 `/open-api/v2`。

确认原则：

* Route 按 persona/权限拆，不按 Web/App/小程序拆。
* Ownership 决定 `/api` vs `/open-api`。
* Surface 决定 DTO/性能/设备差异，默认共享 route。
* `creator` / `updator` 是业务责任主体审计字段，不做权限过滤。
* 统一解析 actor/subject，并与资源 owner/scope 强制比对。
* 测试覆盖内部 persona、管理端、Open API 三类入口，保证审计与权限一致。

### Scenario: Third-Party Open API vs Provider Webhook

第三方应用调用 `/open-api/v1` 和项目中已有的 webhook 都属于“外部系统边界”，但它们不是同一种入口。

核心区别：

| 维度 | Third-party Open API | Provider Webhook |
| --- | --- | --- |
| 调用方向 | 第三方应用主动调用我们 | 外部 provider 回调我们 |
| 典型路径 | `/open-api/v1/orders` | `/api/integrations/payment/:channel_code/webhooks/creem` |
| 调用方身份 | partner / app consumer | provider，例如 `creem` |
| 认证方式 | API key / OAuth client / scope | provider signature / webhook secret |
| 权限模型 | consumer 能代表哪个 account/tenant 访问哪些资源 | provider 只能通知它负责的事件，不能任意查询业务数据 |
| 数据范围 | request subject 与 resource owner/scope 比对 | 从 provider event 映射到本地资源，例如 order_id / subscription_id |
| 响应语义 | 返回业务数据或操作结果 | 快速 ACK，通常不返回业务数据 |
| 幂等重点 | API idempotency key / repeated client request | provider_event_id / payload hash 去重 |
| 异步处理 | 可同步，也可异步 | 推荐落库入队后异步处理 |
| DTO 契约 | 我们定义公开 API 契约 | provider 定义入站 payload，我们做 normalize |

联系：

* 二者都不是普通第一方展示层，不应复用内部 `/api` 的页面 DTO。
* 二者都需要独立认证、日志、幂等、审计和安全字段控制。
* 二者最终都应进入 usecase 层，由 usecase 处理业务一致性、事务、owner/scope 校验和副作用。
* 二者的 actor 都不是普通登录用户：Open API 通常是 `openapi_account:<id>` 或 partner consumer；webhook 通常是 `integration:creem`。

本项目当前已有 webhook 边界：

```text
POST /api/integrations/payment/:channel_code/webhooks/creem
  -> routes.ReceivePaymentWebhook
  -> usecase.ReceivePaymentWebhook
  -> verify provider signature
  -> persist integration_webhook_receipts
  -> enqueue integration-webhooks
  -> usecase.HandlePaymentWebhookJob
```

这个 webhook endpoint 虽然挂在 `/api/integrations/...` 下，但它不是第一方前端 API，也不走登录态或 Open API key。它是一类 provider ingress：按照 provider 的标准接收 payload，验证签名，快速 ACK，然后异步处理业务。

决策规则：

* 第三方需要主动查询/创建/更新我们的业务资源：设计 `/open-api/v1/...`。
* 外部 provider 要通知我们某个外部事件发生：设计 `/api/integrations/<scenario>/<channel>/webhooks/<provider>` 或未来独立 `/webhooks/...` namespace。
* Open API 的 subject 来自 consumer credential；webhook 的 actor 来自 provider/integration channel。
* Open API 通过 owner/scope 控制“能访问哪些资源”；webhook 通过签名、channel 配置、event idempotency 和本地资源映射控制“能影响哪些资源”。

## Requirements

* 明确 `creator` / `updator` 的语义是业务责任主体，不是展示层或入口类型。
* 明确 `creator` / `updator` 是审计字段，不作为“只能查看本人数据”的权限过滤字段。
* `/api` 写操作使用登录用户 ID。
* Web Browser、Desktop App、小程序如果都走 `/api` 且共享用户登录体系，审计主体保持一致，客户端来源另存。
* `/open-api` 写操作使用开放 API 消费者绑定的业务 account/user ID。
* 第三方调用的 `partner_id`、`key_id`、`surface` 应保留在审计上下文、操作日志或扩展审计字段中。
* 系统任务必须有明确的系统主体，不允许空值或随意写 `admin`。
* 涉及“本人数据”时，业务表必须有明确的 owner 字段，例如 `user_id`、`owner_user_id`、`account_id` 或 `tenant_id`，并与请求 subject/principal 的 claims 做比对。

## Acceptance Criteria

* [ ] 团队确认 `creator` / `updator` 字段语义为业务主体 ID。
* [ ] 团队确认开放 API 写入时使用 `Consumer.AccountID`，而不是 `KeyID` 或 `PartnerID`，除非后续采用 typed actor 字段。
* [ ] 如果进入实现阶段，新增统一的 audit actor 解析函数或规范，避免每个 usecase 自行判断。
* [ ] 如果进入实现阶段，新增测试覆盖 `/api`、`/open-api`、`system` 三类 surface 的 actor 解析。

## Open Questions

* 是否要保持 `creator` / `updator` 为单列用户 ID，还是直接升级为 `creator_type` / `creator_id` 和 `updator_type` / `updator_id`？
* 是否需要在资源表中统一引入 owner 字段命名规范，例如 `owner_user_id`、`account_id`、`tenant_id`？

## Out of Scope

* 本任务当前只做架构诊断，不直接修改数据库 schema。
* 本任务当前不实现具体审计日志表。
* 本任务当前不设计第三方应用的完整授权模型。

## Technical Notes

* Relevant files inspected:
  * `index.go`
  * `api/framework/usecase/context.go`
  * `api/framework/http/context/usecase_context.go`
  * `api/framework/http/context/usecase_context_test.go`
  * `api/models/open_api_key.go`
  * `api/models/open_api_account.go`
  * `api/db/migrations/app/001_schema.sql`
* Current project already has a strong separation between `Surface`, `ActorContext`, and `ConsumerContext`; audit-field design should preserve that separation.
* Naming note: `updator` appears to mean `updater` / `updated_by`. If this is a new schema design, prefer `created_by` / `updated_by` or `creator_id` / `updater_id` for clarity.

## Implementation Plan For Confirmation

### MVP: Internal Order List Permission Boundary

当前最适合立即落地的内容是订单列表的 persona-first route 和 owner 过滤边界。它能直接修正现有 `GET /api/orders/user/:user_id` 让客户端传任意 `user_id` 的风险，同时不会引入新的数据库字段。

本轮统一采用 persona-first 路由，避免混用 `/api/orders/me` 这种 resource-first 路径：

```text
/api/<persona>/<resource>
```

#### Step 1: Add persona-specific internal routes

新增或调整内部 route：

```text
GET /api/user/orders
GET /api/admin/orders?user_id=&status=&page=&page_size=
```

推荐处理现有 route：

```text
GET /api/orders/user/:user_id
```

短期保留但标记为 legacy，并加权限保护：只允许 `:user_id == ctx.Actor.UserID` 或 `ctx.Actor.IsAdmin == true`。同时前端改用 `/api/user/orders`，后续再移除旧 route。

命名理由：

* `/api/user/orders`：用户自服务视角，只能查看当前登录用户自己的订单，owner 来自 `ctx.Actor.UserID`。
* `/api/admin/orders`：管理视角，可以跨用户查询，但必须 admin。
* 未来如果出现家长/老师 persona，可自然扩展为 `/api/parent/orders`、`/api/teacher/orders`。
* 不使用 `/api/orders/me`，避免和 persona-first 规范混用。

#### Step 2: Split usecase entrypoints, reuse internal query core

新增语义明确的 usecase 入口：

```go
func ListMyOrders(ctx fwusecase.Context, qry ListMyOrdersQry) (OrdersCo, error)
func ListAdminOrders(ctx fwusecase.Context, qry ListAdminOrdersQry) (OrdersCo, error)
```

内部复用：

```go
func listOrders(ctx fwusecase.Context, query models.OrderQuery, page fwusecase.PageQuery) (OrdersCo, error)
```

权限规则：

* `ListMyOrders` 从 `ctx.Actor.UserID` 强制生成 `OrderQuery.UserID`。
* `ListAdminOrders` 要求 `ctx.Actor.IsAdmin == true`，才允许使用 query 中的 `user_id`。
* 普通客户端不能通过 request 参数决定 owner。

#### Step 3: Move order list model query to dynamic SQL

新增 `models.OrderQuery`，使用项目既有 `db.DynamicExecutorFor(ctx, "app")` + `#[ ... ]`：

```go
type OrderQuery struct {
    UserID string
    Status string
    Limit  int
    Offset int
}
```

对应：

```sql
SELECT * FROM orders
WHERE 1=1
    #[ AND user_id = :user_id ]
    #[ AND status = :status ]
ORDER BY created_at DESC, id DESC
LIMIT :limit OFFSET :offset
```

同时新增 count 查询，保证分页 metadata 一致。

#### Step 4: Update frontend internal API helper

新增或替换：

```js
getMyOrders(pagination)
listAdminOrders(filters)
```

页面侧不再传当前用户 ID 来查询“我的订单”。

#### Step 5: Tests

后端测试：

* `ListMyOrders` 只能返回 `ctx.Actor.UserID` 的订单。
* `ListAdminOrders` 非 admin 返回 forbidden。
* admin 可以按 `user_id` 过滤。
* 旧 `/api/orders/user/:user_id` 如果保留，则普通用户只能查自己，不能查别人。
* model 动态 SQL 覆盖 `user_id`、`status`、分页。

前端测试：

* `getMyOrders()` 调用 `/api/user/orders`。
* `listAdminOrders()` 调用 `/api/admin/orders` 并正确编码筛选条件。

验证命令：

```sh
go test ./...
cd frontend && npm test
```

### Deferred: Open API Order List

`GET /open-api/v1/orders` 建议作为第二步，不和 MVP 混在一起。原因是它需要补充 Open API scope，例如 `orders:read`，以及公开 DTO、文档、测试矩阵。

建议后续独立实现：

```text
GET /open-api/v1/orders
```

范围：

* 新增 Open API order route 和 DTO，不复用内部 DTO。
* 新增 Open API scope 校验，例如 `orders:read`。
* usecase 新增 `ListOpenAPIOrders`，从 `ctx.Consumer.AccountID` 推导 owner。
* 更新 `OPENAPI_CLIENT_GUIDE.md`。

### Deferred: Audit Actor Persistence

当前 schema 尚无统一 `creator` / `updator` 字段。本任务先不改表。等具体业务表需要审计字段时，再引入：

```text
creator_type / creator_id
updator_type / updator_id
```

或先在单表内采用明确字段，并同步新增 audit actor 解析 helper。

## Unified API Path Migration Plan

### Target Namespaces

统一按入口语义划分 API namespace：

```text
/api/public/...       # 第一方公开/无需登录或可选登录能力
/api/auth/...         # 登录、注册、OAuth、token/session 相关
/api/user/...         # 当前登录用户自服务
/api/admin/...        # 管理端/运营端，必须 admin
/api/integrations/... # 外部 provider webhook ingress
/open-api/v1/...      # 第三方主动调用的开放 API
```

迁移原则：

* persona-first：`/api/user/...`、`/api/admin/...`、未来 `/api/parent/...`、`/api/teacher/...`。
* 不按 Web/App/小程序拆 route；surface 只有在 DTO/性能/同步协议不同才新增 endpoint。
* 同一个资源的不同 HTTP method 可以共享 path，前提是 DTO 契约一致或兼容，且 method-level middleware/handler 能清楚表达权限。
* 读写不必机械拆 path；当读是公开资源、写是管理操作时，可以使用同一 resource path，通过 method 权限区分。
* 当 user/admin 的数据范围、DTO 字段、操作语义明显不同，才拆 persona path。
* legacy path 可短期保留，但必须加权限保护，并在前端迁到新 path 后移除。
* 内部 `/api` 与公开 `/open-api/v1` 的 DTO 不复用。

### Current To Target Mapping

| 当前路径 | 目标路径 | 分类 | 处理建议 |
| --- | --- | --- | --- |
| `POST /api/auth/register` | 保持 | auth | 暂不迁移 |
| `POST /api/auth/login` | 保持 | auth | 暂不迁移 |
| `POST /api/auth/logout` | 保持 | auth | 暂不迁移 |
| `GET /api/auth/status` | 保持 | auth | 暂不迁移 |
| `GET /api/auth/me` | `GET /api/user/me` | user | 新增新路径，旧路径短期保留 |
| `POST /api/auth/forgot-password` | 保持 | auth | 暂不迁移 |
| `POST /api/auth/reset-password` | 保持 | auth | 暂不迁移 |
| OAuth routes | 保持 `/api/auth/oauth/...` | auth | 暂不迁移 |
| `GET /api/dictionaries` | `GET /api/public/dictionaries` | public | 新增新路径，旧路径短期保留 |
| `GET /api/settings/site` | `GET /api/public/settings/site` | public | 新增新路径，旧路径短期保留 |
| `GET /api/settings/public/logo` | `GET /api/public/settings/logo` | public | 新增新路径，旧路径短期保留 |
| `GET /api/products` | 保持 `GET /api/products` | public read | 当前用于 checkout catalog，读语义清晰，可保留共享 resource path；如未来 DTO/权限明显变化，再考虑 `/api/public/products` |
| `POST /api/orders` | `POST /api/user/orders` | user | 新增新路径；usecase 强制使用 current user，不让客户端传任意 `user_id` |
| `GET /api/orders/user/:user_id` | `GET /api/user/orders` | user | 前端迁移；旧路径加 owner/admin 保护后废弃 |
| `GET /api/orders/:id` | 倾向保持 `GET /api/orders/:id`，按 actor 权限返回 | user/admin | 如果 user/admin DTO 一致，可共享 path；handler/usecase 根据 actor 做 owner/admin 判断 |
| `POST /api/orders/:id/pay` | `POST /api/user/orders/:id/pay` | user | user owner 保护；是否保留手动 pay 待业务确认 |
| `POST /api/orders/:id/payment-checkout` | `POST /api/user/orders/:id/payment-checkout` | user | user owner 保护 |
| `PATCH /api/orders/:id/status` | `PATCH /api/admin/orders/:id/status` | admin | 管理操作，应收敛到 admin |
| `GET /api/points/me` | `GET /api/user/points` | user | 新增新路径，旧路径短期保留 |
| `GET /api/points/sse` | `GET /api/user/points/sse` | user | 新增新路径，旧路径短期保留 |
| `POST /api/notifications/test-export-toast` | `POST /api/user/notifications/test-export-toast` | user | 新增新路径，旧路径短期保留 |
| `POST /api/admin/reload-shared-db` | 保持 `POST /api/admin/reload-shared-db` | admin | 路径已是 admin 语义，需确认注册在 RequireAdmin 保护下 |
| `GET /api/users` | `GET /api/admin/users` | admin | 当前只 RequireAuth，建议迁到 admin 并加 RequireAdmin |
| `GET /api/users/:id` | `GET /api/admin/users/:id` 和 `GET /api/user/me` | admin/user | 管理详情走 admin；当前用户详情走 user/me |
| `POST /api/users` | `POST /api/admin/users` | admin | 用户管理 |
| `PUT /api/users/:id` | `PUT /api/admin/users/:id` | admin | 用户管理 |
| `PATCH /api/users/:id/active` | `PATCH /api/admin/users/:id/active` | admin | 用户管理 |
| `DELETE /api/users/:id` | `DELETE /api/admin/users/:id` | admin | 用户管理 |
| `POST/PUT/PATCH /api/products...` | 保持 `/api/products...` | admin by method | 商品写操作通过 method-level admin 权限控制，不必机械迁到 `/api/admin/products` |
| `GET/POST/PUT/PATCH /api/dictionary...` | `/api/admin/dictionary...` | admin | 字典管理；public dictionary read 另走 `/api/public/dictionaries` |
| `/api/scheduler/tasks...` | `/api/admin/scheduler/tasks...` | admin | 定时任务管理 |
| `/api/events...` | `/api/admin/events...` | admin | 事件投递后台 |
| `/api/messages` | `/api/admin/messages` | admin | 队列消息后台 |
| `/api/parameters/...` | `/api/admin/parameters/...` | admin | 已 RequireAdmin，补齐路径 |
| `GET /api/notifications` | `GET /api/admin/notifications` | admin | 已 RequireAdmin，补齐路径 |
| `POST /api/settings/site/logo` | `POST /api/admin/settings/site/logo` | admin | 已 RequireAdmin，补齐路径 |
| `/api/variables...` | `/api/admin/variables...` | admin | 当前只 RequireAuth，建议迁到 admin |
| `POST /api/llm/summaries` | `POST /api/user/llm/summaries` 或 `/api/admin/llm/summaries` | user/admin 待确认 | 当前实验页使用，按产品定位决定 |
| `/api/integrations/payment/:channel_code/webhooks/creem` | 保持或未来迁到 `/webhooks/payment/:channel_code/creem` | webhook ingress | 不纳入 persona；provider 标准优先 |
| `/open-api/v1/health` | 保持 | open-api | 公开健康检查 |
| `/open-api/v1/account/me` | 保持 | open-api | 第三方 consumer 自查 |

### Proposed Batches

### Implementation Status

本轮已落地订单列表 MVP：

* 新增 `GET /api/user/orders`：当前登录用户自服务订单列表，owner 来自 `ctx.Actor.UserID`。
* 新增 `GET /api/admin/orders?user_id=&status=&page=&page_size=`：管理端订单列表，必须 admin，可跨用户筛选。
* 保留 legacy `GET /api/orders/user/:user_id`，但增加 owner/admin 保护：普通用户只能查询自己，admin 可跨用户。
* 前端订单页面已迁移到 `getMyOrders()` / `/api/user/orders`。
* model 层订单列表改为 `OrderQuery` + `#[ ... ]` 动态 SQL 条件查询。

本轮未落地、仍属于后续批次：

* `/api/user/points`、`/api/user/points/sse`、`/api/user/me` 别名。
* `/api/admin/users`、`/api/admin/scheduler/tasks`、`/api/admin/events`、`/api/admin/messages` 等管理端路径迁移。
* `/api/public/...` namespace cleanup。
* `/open-api/v1/orders`。

#### Batch 1: Add aliases and migrate frontend user/admin reads

低风险新增新路径，旧路径保留：

```text
GET /api/user/orders
GET /api/user/points
GET /api/user/points/sse
GET /api/user/me
GET /api/admin/orders
GET /api/admin/users
GET /api/admin/scheduler/tasks
GET /api/admin/events
GET /api/admin/messages
```

同时更新 `frontend/src/api.js` 和测试，使页面优先使用新路径。

#### Batch 2: Tighten permissions

将明显管理端能力收紧到 admin 权限或 admin namespace：

```text
/api/admin/users...
/api/admin/dictionary...
/api/admin/parameters...
/api/admin/variables...
/api/admin/settings...
/api/products...                # writes keep resource path, require admin by method
```

旧路径短期保留但加 `RequireAdmin()` 或 owner 保护。

#### Batch 3: Public namespace cleanup

新增公开读取路径：

```text
GET /api/public/dictionaries
GET /api/public/settings/site
GET /api/public/settings/logo
```

旧 public 路径保留到前端迁移完成。`/api/products` 是否迁入 `/api/public/products` 暂不作为 MVP 决策；如果商品读写 DTO 一致，优先保留 `/api/products`，通过 HTTP method 区分公开读和 admin 写。

#### Batch 4: Open API and webhook follow-up

单独设计：

```text
GET /open-api/v1/orders
```

Webhook 路径暂不强制改动，因为它是 provider ingress，不属于 persona-first `/api/<persona>` 体系。若要统一，可后续迁到：

```text
POST /webhooks/payment/:channel_code/creem
```

但这会影响 provider 配置和外部回调地址，应独立变更。
