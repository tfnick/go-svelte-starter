# brainstorm: ddd event queue architecture

## Goal

基于已经集成的 `goqite` 消息队列能力，设计并实现新的 DDD domain event 封装，用于替换当前 `api/framework/events` 中基于 `github.com/asaskevich/EventBus` 的 event bus 方案。新方案应让不同业务模块通过领域事件解耦，并把跨模块异步副作用建立在 durable queue + project-owned event tables 上。

## What I already know

* 用户目标：设计新的 DDD event 方案，底层复用现有 goqite 队列能力，完成领域事件封装，实现模块/领域解耦，并完全替换现有 EventBus 方案。
* 当前仓库已经有 `api/framework/queue`，其中 `QueueDomainEvents = "domain-events"`，支持 `SendJSON`、事务内 `SendAndGetIDTx`、`NewJSONRunner`。
* 当前 `api/framework/events` 是混合实现：raw EventBus 负责 `async_best_effort` 和 `sync_tx`，durable path 已经通过 `domain_events` / `domain_event_deliveries` + goqite fan-out 实现一部分可靠异步交付。
* 当前 `api/usecase/order.go` 的 `PayOrder` 依赖 `events.PublishInTx`，并在支付前检查 `SyncSubscribers(order.paid)`，以保证订单支付和积分奖励在同一个 app transaction 中提交或回滚。
* 当前 `api/usecase/events/order_paid_points.go` 注册 `DeliveryModeSyncTx` handler，积分写入同步发生；realtime points message 通过 after-commit 发布。
* 当前 `api/db/migrations/app/009_add_domain_event_delivery.sql` 已创建 `domain_events` 和 `domain_event_deliveries`，每个 durable subscriber 有独立 delivery row。
* 当前 archguard 仍有旧约束：raw `github.com/asaskevich/EventBus` 只能在 `api/framework/events` 中导入，并禁止创建 `api/framework/outbox`。替换方案需要同步更新这些约束。
* `go.mod` 仍直接依赖 `github.com/asaskevich/EventBus`，新方案完成后应移除该依赖。

## Assumptions (temporary)

* 新设计应优先复用现有 `domain_events` / `domain_event_deliveries` 表，不新增 `domain_event_executions`，也不把项目状态字段塞进 `goqite` 表。
* `goqite` 仍是 framework component-owned table，业务只通过 `api/framework/queue` 间接使用。
* 新 DDD event facade 仍放在 `api/framework/events` 或重命名后的 framework package 中；业务模块不直接 import raw goqite。
* 当前任务会覆盖后端事件框架、业务事件注册、订单支付积分示例、spec/archguard 更新；除非实现需要，不新增前端功能。

## Research References

* [`research/ddd-event-design-options.md`](research/ddd-event-design-options.md) - 记录仓库约束下的 queue-first DDD event 可选方案和推荐方向。

## Open Questions

* None currently. Requirements are ready for final confirmation.

## Requirements (evolving)

* 设计新的 DDD event API，使发布方只表达领域事实，不直接依赖订阅模块。
* 用 goqite-backed durable delivery 替代 raw EventBus delivery。
* 删除或退役 raw EventBus 依赖、API、测试和 spec 约束。
* 保留事务一致性：在 app transaction 中发布事件时，事件事实、delivery rows 和 goqite messages 必须随主事务一起 commit/rollback。
* 每个 subscriber 必须独立 delivery row 和独立 queue message，避免一个 subscriber 的失败影响其他 subscriber 的处理状态。
* handler failure 必须返回 error 给 queue runner，让 goqite timeout/retry 机制接管重试；delivery state 应记录 running/succeeded/failed/attempts/last_error。
* 业务事件 payload 应有稳定 schema；推荐通过每个领域的 typed constructor / payload struct 生成 event，而不是在调用处拼散装 JSON。
* 需要迁移现有 `order.paid -> points.award_on_order_paid` 示例到新方案，并同步调整测试。
* 选择 Queue-first durable only：所有跨领域事件 subscriber 都走 goqite durable queue；不保留 raw EventBus，也不保留事务内同步 subscriber。
* `PayOrder` 的成功语义调整为：订单支付事务提交且 `order.paid` event 已可靠入队即返回成功；积分奖励由 durable subscriber 异步完成，达到最终一致性。
* 选择轻量 typed constructor API：framework 继续提供通用 `events.Event`，每个领域在 usecase-owned event package 中提供 payload struct 和 `New<EventName>Event(...)` 构造器。
* 不引入泛型 typed event/handler，不要求每个 event type 实现领域接口。

## Acceptance Criteria (evolving)

* [ ] `go test ./...` 通过。
* [ ] `go.mod` / `go.sum` 不再需要 `github.com/asaskevich/EventBus`。
* [ ] `api/framework/events` 不再 import raw EventBus，或旧 package 被新的 queue-backed DDD event facade 完全替代。
* [ ] archguard 能阻止业务层直接 import raw goqite，并能阻止 raw EventBus 重新出现。
* [ ] 事务 rollback 后不留下 `domain_events`、`domain_event_deliveries` 或 `goqite` messages。
* [ ] 一个 event 对多个 subscriber fan-out 时，每个 subscriber 有独立 queue message 和 delivery state。
* [ ] 某个 subscriber 失败时只标记自身 failed，不影响其他 subscriber succeeded。
* [ ] 现有订单支付/积分测试按 queue-first 语义更新并通过：支付成功不要求积分已同步到账，但要求 `order.paid` durable delivery 已入队。
* [ ] `order.paid` durable subscriber 幂等处理积分奖励，queue retry 不会重复发放积分。
* [ ] `order.paid` 事件由 typed constructor 创建，调用点不直接拼散装 JSON。
* [ ] `.trellis/spec/backend/eventing-guidelines.md`、database/quality/archguard 相关说明更新到新 DDD event 方案。

## Definition of Done

* Tests added/updated for framework event facade, durable delivery, transaction rollback, and migrated order-paid points flow.
* `go test ./...` green.
* Specs updated if behavior or architecture contracts change.
* Rollback and retry behavior explicitly documented.
* Old EventBus design removed from production code and project guidelines.

## Technical Approach (draft)

Selected direction is **queue-first durable DDD events**:

* Publisher calls a small domain event facade, for example `events.Publish(ctx, event)` or `events.PublishInTx(ctx, event)`.
* Publishing persists `domain_events`, creates one `domain_event_deliveries` row per subscriber, and enqueues one JSON message per delivery to `QueueDomainEvents`.
* The `domain-events` JSON runner loads the persisted event by `event_id`, dispatches exactly one subscriber handler per message, and updates that subscriber delivery state.
* Subscriber registration stays in usecase/domain module code, but registration declares durable handlers only by topic + subscriber name.
* Domain packages expose typed constructors such as `NewOrderPaidEvent(order)` or `NewOrderPaidEvent(payload)` that return `events.Event`.
* The old `async_best_effort` and raw EventBus `sync_tx` paths are removed.
* Synchronous cross-domain side effects are not modeled as event subscribers. If a side effect must be part of the same transaction, it should be an explicit usecase/domain service call instead of a DDD event subscriber.

## Decision (ADR-lite, draft)

**Context**: Existing EventBus gives fast in-process sync/async dispatch, but it couples modules at runtime, has best-effort async behavior, and now duplicates responsibilities with the newly introduced goqite durable path.

**Decision**: Use Queue-first durable only. Move cross-domain event handling to durable queue delivery and treat transaction-coupled invariants as explicit usecase/domain service calls rather than event subscribers.

**Consequences**:

* Pros: stronger module decoupling, recoverable async processing, inspectable delivery state, no raw EventBus dependency.
* Cons: existing synchronous side effects such as points-on-payment become eventual unless modeled as explicit command calls before publishing the event.
* Risk: changing order payment semantics may affect user-visible points timing and tests; tests must assert queued delivery and idempotent subscriber behavior instead of immediate balance changes.

## API Decision (ADR-lite)

**Context**: The event facade needs enough structure to keep payload schemas stable without overbuilding a generic framework.

**Decision**: Use lightweight typed constructors. Keep `events.Event` as the framework contract and move per-domain schema safety into payload structs plus constructors in `api/usecase/events`.

**Consequences**:

* Pros: small API, low migration cost, readable call sites, no generic reflection-heavy handler machinery.
* Cons: handlers still decode JSON explicitly; compile-time payload typing is at constructor/handler boundaries rather than end-to-end.

## Implementation Plan

* PR1: Replace raw EventBus internals with a queue-first durable event registry/facade; remove async best-effort and sync tx modes.
* PR2: Migrate `order.paid -> points.award_on_order_paid` to durable-only semantics with typed constructor and idempotent subscriber tests.
* PR3: Update archguard, `go.mod`, and backend specs so raw EventBus cannot return and durable DDD event contracts are documented.

## Out of Scope (explicit)

* New frontend UI beyond existing queue/message inspection, unless implementation discovers a required operational view gap.
* Distributed broker support outside SQLite/goqite.
* Exactly-once external side effects; handlers should be idempotent and retry-safe instead.
* General saga/orchestration framework.

## Technical Notes

* Inspected `api/framework/events/events.go`: raw EventBus is used for async and sync dispatch; durable queue path already exists inside the same Bus.
* Inspected `api/framework/queue/queue.go`: queue manager supports transaction-aware JSON send and `domain-events` runner.
* Inspected `api/models/domain_event.go` and migration `009_add_domain_event_delivery.sql`: durable event state exists but may need richer query/update semantics for attempts and idempotency.
* Inspected `api/usecase/order.go` and `api/usecase/events/order_paid_points.go`: current business-critical example uses sync tx subscriber.
* Inspected `api/framework/archguard/layer_boundary_test.go`: old EventBus boundary and outbox prohibition need to be revised.
* Relevant specs: `.trellis/spec/backend/eventing-guidelines.md`, `.trellis/spec/backend/database-guidelines.md`, `.trellis/spec/backend/quality-guidelines.md`, `.trellis/spec/backend/directory-structure.md`.
