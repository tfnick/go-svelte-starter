# Current DDD Event Architecture Review

## Executive Summary

当前 lite DDD event 设计已经具备核心可用性：domain event 通过 `api/framework/events` 发布，底层复用 `api/framework/queue` 的 goqite 持久化队列能力；一个 event 可以 fan-out 到多个 subscriber，每个 subscriber 拥有独立 delivery row 和 queue message；publisher transaction 内的 event、delivery、queue message 能一起 commit/rollback；subscriber 失败会保留 queue message 以便 retry。

主要可优化点不在“能不能工作”，而在“边界表达和业务接入负担”：

* framework 与业务职责大体清晰，但 `DurableStore` 放在 `api/usecase/events` 会让基础设施持久化适配看起来像业务代码；可考虑移到 framework/model adapter 层或显式命名为 app adapter。
* 业务开发者仍要手写 JSON marshal/unmarshal、handler transaction、registration boilerplate、subscriber missing check 和幂等表设计；可以通过 typed event helper、typed handler adapter、subscriber bootstrap registry、idempotency helper 降低负担。
* 最终一致性已成立，但 exactly-once 没有也不应该承诺；framework 只能保障 per-subscriber delivery/retry 轨迹，业务侧必须保障幂等写。
* 非 DDD 基本消息发送仍支持，能力在 `api/framework/queue`，不应塞回 `api/framework/events`。建议把文档和 API 命名明确成 “domain events facade” vs “generic durable queue” 两条路。

## Current Responsibility Boundary

### `api/framework/events`

Framework event facade 当前负责：

* 定义通用 `Event`、`Subscription`、`Handler`。
* 注册 topic/subscriber 到 handler。
* publish 时查询 subscribers。
* 持久化 event。
* 为每个 subscriber 创建独立 queue message 和 delivery row。
* 消费 queue message 时加载 event，调用对应 handler。
* 维护 delivery 状态：`queued -> running -> succeeded/failed`。
* 在 handler 失败时返回 error，让 queue runner 保留消息并后续 retry。

它不应该负责：

* 业务 payload struct 的字段选择。
* 业务幂等语义。
* 业务事务内写哪些表。
* 通知短信、积分、审计等具体副作用。

### `api/framework/queue`

Generic queue 当前负责：

* 封装 raw goqite。
* 提供 `Send` / `SendJSON`。
* 提供 raw queue runner / JSON runner。
* 提供 `CreateJob` 支持 goqite jobs 风格的 named job。
* 在存在 app SQL transaction 时使用 goqite tx send。

这说明原来的非 DDD 基本消息发送并没有丢失，只是与 DDD event facade 分离了。

### Business / Usecase Side

业务侧当前负责：

* 定义 event topic / subscriber name。
* 定义 payload struct。
* 编写 typed constructor，例如 `NewOrderPaidEvent(order)`.
* 在 publisher usecase transaction 内调用 `events.Publish(txCtx, event)`.
* 注册 subscriber handler。
* 在 subscriber handler 中 unmarshal payload。
* 在 subscriber handler 中开启自己的 `fwusecase.WithAppTx`。
* 实现业务幂等，例如积分使用 `UNIQUE(order_id, type)`。
* 处理 after-commit realtime side effects。

### Boundary Assessment

边界方向是对的：DDD event 和 generic queue 已拆开，业务无法 import raw goqite，raw EventBus 已被 archguard 禁止。

但仍有两个表达层面的模糊点：

1. `api/usecase/events/durable_store.go` 是 framework events 的 persistence adapter，但它在 `usecase/events` 包中，容易让人误以为业务事件包同时负责基础设施存储。
2. `events.Register` 的 handler map 使用 `subscriber` 作为 key，而不是 `(topic, subscriber)`；规范说唯一性是 `(topic, subscriber)`，实现隐含要求 subscriber 全局唯一。

## Business Developer Burden

新增一个 DDD event/subscriber 目前至少需要：

1. 新增 topic/subscriber 常量。
2. 新增 payload struct。
3. 新增 event constructor，并手写 `json.Marshal`。
4. 在 publisher usecase 里调用 constructor + `events.Publish`。
5. 如果 publisher 依赖某个 subscriber 必须存在，还要手写 `events.Subscribers(topic)` guard。
6. 新增 handler struct，实现 `Handle(context.Context, events.Event) error`。
7. handler 内手写 `json.Unmarshal`。
8. handler 内手写 `fwusecase.NewContext` 和 `WithAppTx`。
9. handler 内调用业务 usecase。
10. 设计业务幂等键/表/唯一约束。
11. 在 startup 注册 handler。
12. 写 publish、fan-out、retry、idempotency 测试。

这对框架作者是清晰的，但对普通业务开发者偏重。

## Final Consistency And Idempotency

### What Is Already Supported

最终一致性链路已经成立：

* publisher 在 app transaction 内更新业务状态并 `events.Publish(txCtx, event)`。
* `domain_events`、`domain_event_deliveries`、`goqite` queue rows 都使用同一个 app transaction。
* transaction rollback 时 event/delivery/message 都回滚。
* transaction commit 后 queue runner 最终消费 message。
* subscriber 成功后 delivery 标记 `succeeded`，queue message 删除。
* subscriber 失败后 delivery 标记 `failed`，queue runner 不删除 message，等待 retry。
* 多 subscriber 是独立 queue message，单个 subscriber 失败不会影响已成功的 subscriber。

### What Is Not Guaranteed

当前不保证：

* exactly-once handler execution。
* 多 subscriber 物理并行执行。
* handler side effects 自动幂等。
* 外部系统调用自动去重。
* failed delivery 的人工补偿/后台治理 UI。

这些不一定是缺陷，但必须明确写进 contract。

### Business Idempotency

积分示例的幂等是业务层实现的：

* `point_transactions` 使用 `UNIQUE(order_id, type)`。
* insert 使用 ignore semantics。
* retry 同一个 event 时不会重复加积分。

framework 当前提供的是 delivery 维度的尝试轨迹，不提供业务幂等写 helper。

## Generic Non-DDD Message Support

当前仍支持非 DDD 语义消息：

* `queue.Manager.Send(ctx, queue.SendOptions{...})`
* `queue.Manager.SendJSON(ctx, opts, payload)`
* `queue.Manager.NewRunner(queueName, limit, pollInterval)`
* `queue.Manager.NewJSONRunner(queueName, limit, pollInterval)`
* `queue.Manager.CreateJob(...)`

示例：scheduler 仍使用 `DefaultQueueManager.CreateJob(..., queue.QueueScheduledTasks, ...)`。

因此普通消息不应该通过 `api/framework/events` 发送。建议文档中明确：

* Domain facts / cross-domain business side effects -> `api/framework/events`.
* Generic background jobs / command messages / scheduled jobs -> `api/framework/queue`.

## Recommended Optimizations

### P0: Align Subscriber Key Contract

当前规范说 `(topic, subscriber)` 唯一，但 handler lookup 用 `subscriber`。二选一：

* 推荐：让 subscriber 全局唯一，并把规范改成 “subscriber name is globally unique”。实现也在 register 时检查所有 topic 中是否已有相同 subscriber。
* 或者：handler map 改成 `topic + "\x00" + subscriber`，`HandleMessage` 同时校验 message topic 与 loaded event topic。

从业务理解看，subscriber 名称用 `points.award_on_order_paid` 这种全局语义名更自然，建议采用全局唯一策略，简单且可观测。

### P1: Add Typed Payload Helpers

可以在 framework 提供泛型 helper，减少业务 JSON boilerplate：

```go
func NewEventPayload[T any](topic, aggregateType, aggregateID string, payload T) (Event, error)
func DecodePayload[T any](event Event) (T, error)
```

业务代码保留 payload struct 和 constructor，但不再手写 marshal/unmarshal。

### P1: Add Transactional Handler Adapter

提供 adapter 帮业务 handler 自动完成 decode + system context + app transaction：

```go
func TransactionalHandler[T any](fn func(fwusecase.Context, T) error) Handler
```

这样业务 subscriber 可以从：

```go
func (h handler) Handle(ctx context.Context, event events.Event) error {
    var payload Payload
    json.Unmarshal(event.PayloadJSON, &payload)
    ucCtx := fwusecase.NewContext(ctx, fwusecase.SurfaceSystem)
    return fwusecase.WithAppTx(ucCtx, func(txCtx fwusecase.Context) error {
        return h.do(txCtx, payload)
    })
}
```

降到只写业务函数。

### P1: Make Startup Registration More Explicit

当前 `RegisterEventHandlers` 使用 package-level `sync.Once`，测试和多 app instance 场景容易不直观。可考虑：

* 将 registration 汇总到一个 `RegisterSubscribers(reg events.Registry, deps SubscriberDeps) error`。
* production 传 default registry，测试可传 fresh bus。
* 避免 package-level once，改由 application bootstrap 保证只注册一次。

### P1: Add Idempotency Scaffold

framework 不应该替业务决定幂等键，但可以提供 scaffold：

```go
type IdempotencyStore interface {
    TryBegin(ctx context.Context, scope, key string) (acquired bool, err error)
    MarkSucceeded(ctx context.Context, scope, key string) error
    MarkFailed(ctx context.Context, scope, key string, message string) error
}
```

或者更轻量：规范要求每个 subscriber 在 PRD/test 中声明 idempotency key，并提供测试 helper 验证同一 message 重放不会重复 side effect。

### P2: Improve Operational Observability

当前 delivery row 有 status/attempts/last_error，但没有 query API 或 admin view。后续可以加：

* List failed deliveries.
* Requeue / retry now.
* Mark ignored.
* Correlate event id to queue message id.

这对最终一致性体系的运营很关键，但不是 MVP 阻塞项。

### P2: Clarify Generic Queue Docs

更新 eventing/queue spec：

* `events` 只服务 DDD domain facts。
* `queue` 服务 generic background messages/jobs。
* 业务普通任务可以使用 `queue.Manager`，但 raw goqite 仍只允许 framework queue。
* domain event 不允许直接用 `queue.SendJSON`。

## Suggested Implementation Plan

如果用户希望从评审进入实现，建议分三步：

1. Contract hardening:
   * subscriber 全局唯一或 topic/subscriber 复合 key。
   * `HandleMessage` 校验 message topic 与 persisted event topic。
   * 补测试。
2. Business ergonomics:
   * typed payload constructor/decode helper。
   * transactional typed handler adapter。
   * 改造 `order.paid` 示例。
3. Documentation and idempotency:
   * 明确 DDD events vs generic queue。
   * 增加 subscriber idempotency checklist/test helper。
   * 更新 `.trellis/spec/backend/eventing-guidelines.md`。

## Answer To The Four Questions

### 1. Framework vs Business Boundary

基本清晰，但可进一步硬化。framework 负责 durable fan-out、delivery tracking、queue integration、retry contract；业务负责 event meaning、payload、side effects、idempotency。需要优化的是 persistence adapter 位置和 subscriber key contract。

### 2. Business Developer Burden

当前偏高。业务开发者要理解太多 framework plumbing。建议用 typed helper 和 transactional handler adapter 把开发体验收敛到：定义 payload、写业务 handler、声明幂等键、注册 subscriber。

### 3. Final Consistency And Idempotent Consumption

支持最终一致性，且提供 delivery/retry 技术基础。但业务幂等只被文档和示例保障，没有 framework scaffold。这个边界正确，但建议提供 helper/test pattern 降低误用概率。

### 4. Generic Non-DDD Message Sending

支持。`api/framework/queue` 仍是通用消息/任务通道；`api/framework/events` 是 DDD event facade。建议明确文档边界，不要把 generic messages 重新塞进 events。
