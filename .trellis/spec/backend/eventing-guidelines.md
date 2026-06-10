# Eventing Guidelines

> 本文是 DDD domain event 的权威实现说明。当前设计是 queue-first durable only：`api/framework/events` 只负责 durable event facade，底层通过 `api/framework/queue` 使用 goqite，不再使用 raw EventBus、best-effort async 或 sync transaction subscriber。

---

## Overview

业务代码只能使用 `api/framework/events` 发布和订阅领域事件。该 package 不 import `github.com/asaskevich/EventBus`；raw goqite 仍只允许出现在 `api/framework/queue`。

领域事件的唯一交付路径：

```text
usecase transaction
  -> events.Publish(txCtx, event)
  -> domain_events
  -> domain_event_deliveries
  -> goqite queue: domain-events
  -> events.HandleMessage
  -> subscriber handler
```

设计目标：

* 发布方只表达领域事实，不直接依赖订阅模块。
* 跨领域副作用最终一致，必须可重试、可观察、幂等。
* 事件事实、delivery rows、goqite message 在 active app transaction 中一起 commit/rollback。
* 每个 subscriber 独立 delivery row 和独立 queue message。

---

## Public API

```go
type Event struct {
    ID            string
    Topic         string
    AggregateType string
    AggregateID   string
    PayloadJSON   []byte
    MetadataJSON  []byte
    OccurredAt    time.Time
}

type Subscription struct {
    Topic      string
    Subscriber string
}

type Handler interface {
    Handle(context.Context, Event) error
}

func Configure(store Store, sender QueueSender)
func Register(sub Subscription, handler Handler) error
func RegisterTransactional[T any](sub Subscription, fn TransactionalHandlerFunc[T]) error
func Subscribers(topic string) []Subscription
func NewPayloadEvent[T any](topic string, aggregateType string, aggregateID string, payload T) (Event, error)
func NewPayloadEventWithOptions[T any](opts PayloadEventOptions, payload T) (Event, error)
func DecodePayload[T any](event Event) (T, error)
func TransactionalHandler[T any](fn TransactionalHandlerFunc[T]) Handler
func Publish(ctx fwusecase.Context, event Event) error
func HandleMessage(ctx context.Context, message []byte) error
```

Queue message envelope:

```json
{"event_id":"...","subscriber":"points.award_on_order_paid","topic":"order.paid"}
```

Retired API:

* `RegisterAsync`
* `RegisterDurable`
* `RegisterSync`
* `PublishAsync`
* `PublishInTx`
* `DeliveryMode`
* `async_best_effort`
* `sync_tx`

---

## Boundary Rules

* raw `github.com/asaskevich/EventBus` must not appear anywhere in production code.
* raw `maragu.dev/goqite` can only be imported by `api/framework/queue`.
* usecase may import `api/framework/events` and publish events.
* models must not import `api/framework/events`.
* routes must not publish domain events.
* subscriber registration happens during application startup, never in request path.
* `(topic, subscriber)` must be unique.
* event facade sends queue messages through `QueueSender`; it must not use raw goqite directly.

---

## Event Payload Rules

事件 payload 是稳定 DTO，不是 raw model。每个业务事件应在 usecase-owned event package 中提供 payload struct 和 typed constructor。

```go
const OrderPaidTopic = "order.paid"

type OrderPaidPayload struct {
    OrderID string `json:"order_id"`
    UserID  string `json:"user_id"`
    Amount  int64  `json:"amount"`
    Points  int64  `json:"points"`
}

func NewOrderPaidEvent(order *models.Order) (events.Event, error) {
    return events.NewPayloadEvent(OrderPaidTopic, "order", order.ID, OrderPaidPayload{
        OrderID: order.ID,
        UserID:  order.UserID,
        Amount:  order.Amount,
        Points:  OrderPaidPoints,
    })
}
```

Prefer typed transactional subscribers when the handler writes app DB state:

```go
err := events.RegisterTransactional[OrderPaidPayload](events.Subscription{
    Topic:      OrderPaidTopic,
    Subscriber: OrderPaidSubscriber,
}, func(txCtx fwusecase.Context, event events.Event, payload OrderPaidPayload) error {
    return AwardOrderPaidPoints(txCtx, AwardOrderPaidPointsCmd{
        UserID:  payload.UserID,
        OrderID: payload.OrderID,
        Points:  payload.Points,
    })
})
```

`RegisterTransactional` decodes `PayloadJSON`, creates a system usecase context, and runs the handler inside `fwusecase.WithAppTx`. Business code still owns side-effect idempotency.

规则：

* `Topic` 必填，例如 `order.paid`。
* `ID` 为空时 framework 自动生成。
* `OccurredAt` 为空时 framework 自动设置。
* `MetadataJSON` 为空时 framework 写入 `{}`。
* `PayloadJSON` 只放稳定业务字段。
* 不放 password、API key、session ID、reset token、完整 model。

---

## Scenario: Durable DDD Events

### 1. Scope / Trigger

新增领域事件、修改 `api/framework/events`、修改 `api/framework/queue` event runner、修改 `api/usecase/events/durable_store.go` 或修改 `domain_events` / `domain_event_deliveries` 时，必须遵守本节。

### 2. Signatures

注册：

```go
err := events.Register(events.Subscription{
    Topic:      "order.paid",
    Subscriber: "points.award_on_order_paid",
}, handler)
```

发布：

```go
err := fwusecase.WithAppTx(ctx, func(txCtx fwusecase.Context) error {
    event, err := usecaseevents.NewOrderPaidEvent(order)
    if err != nil {
        return err
    }
    return events.Publish(txCtx, event)
})
```

启动：

```go
queueManager, _ := queue.NewManager()
events.Configure(usecaseevents.DurableStore{}, queueManager)
runner, _ := queueManager.NewJSONRunner(queue.QueueDomainEvents, 1, 500*time.Millisecond)
runner.Register(events.HandleMessage)
go runner.Start(appCtx)
```

### 3. Contracts

* `Publish` with no subscribers returns `nil` and does not persist an event.
* If subscribers exist but store/sender is not configured, `Publish` returns `ErrDurableEventsNotReady`.
* `Publish` normalizes event `ID`, `OccurredAt`, and `MetadataJSON`.
* In active app transaction, `domain_events`, `domain_event_deliveries`, and `goqite` rows must roll back together.
* Fan-out creates one queue message and one delivery row per subscriber.
* `HandleMessage` loads persisted event by `event_id` and invokes exactly one subscriber handler.
* Handler success marks that delivery `succeeded`; queue runner deletes the message after handler returns nil.
* Handler failure marks that delivery `failed` and returns error so goqite timeout/retry keeps the message retryable.
* Subscriber handlers that write app DB should prefer `RegisterTransactional` or `TransactionalHandler`; custom handlers must open their own `fwusecase.WithAppTx` unless already intentionally operating inside another usecase transaction.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| empty event topic | `ErrInvalidEvent` |
| empty subscription topic | `ErrInvalidSubscription` |
| empty subscriber | `ErrInvalidSubscription` |
| nil handler | `ErrInvalidSubscription` |
| duplicate `(topic, subscriber)` | `ErrDuplicateSubscription` |
| subscribers exist but store/sender missing | `ErrDurableEventsNotReady` |
| durable message missing `event_id` or `subscriber` | `ErrInvalidEvent` wrapped error |
| subscriber handler returns error | delivery row becomes `failed`; queue message remains retryable |
| one subscriber fails and another succeeds | success delivery remains `succeeded`; failed delivery remains retryable |

### 5. Good/Base/Bad Cases

Good: one `order.paid` event creates independent messages for `points.award_on_order_paid` and `audit.record_order_paid`; retrying the points subscriber never re-runs audit.

Base: no subscriber for a topic returns `nil` and writes no event rows.

Bad: one queue message loops over all subscribers; a single failure would retry already-successful subscribers and blur delivery state.

### 6. Tests Required

* `api/framework/events/events_test.go` covers duplicate registration, no-subscriber publish, missing durable config, fan-out, per-subscriber failure isolation, and transaction rollback.
* Usecase event tests should verify publisher transaction commits event/delivery/message rows atomically.
* Subscriber tests should verify handler idempotency when the same event is retried.
* `api/framework/archguard/layer_boundary_test.go` must reject raw EventBus imports and raw goqite outside `api/framework/queue`.
* Run `go test ./...`.

### 7. Wrong vs Correct

#### Wrong

```go
_ = queue.SendJSON(ctx.Std(), queue.SendOptions{Queue: queue.QueueDomainEvents}, event)
```

#### Correct

```go
event, err := usecaseevents.NewOrderPaidEvent(order)
if err != nil {
    return err
}
return events.Publish(txCtx, event)
```

---

## Scenario: Order Paid Points

### 1. Scope / Trigger

`order.paid -> points.award_on_order_paid` 是 queue-first durable event 的示例。`PayOrder` 只保证订单支付事务提交且 event 已可靠入队；积分奖励由 durable subscriber 异步、幂等完成。

### 2. Signatures

```go
const (
    OrderPaidTopic      = "order.paid"
    OrderPaidSubscriber = "points.award_on_order_paid"
    OrderPaidPoints     = int64(10)
)

func NewOrderPaidEvent(order *models.Order) (events.Event, error)
func RegisterEventHandlers(award AwardOrderPaidPointsFunc) error
```

### 3. Contracts

* `PayOrder` 发布前确认 `events.Subscribers(OrderPaidTopic)` 非空，避免订单已支付但积分 subscriber 未注册。
* `PayOrder` 在同一个 `WithAppTx` 中更新订单状态并调用 `events.Publish`。
* 支付成功后 points balance 不要求立即变化。
* `points.award_on_order_paid` handler uses `RegisterTransactional` so `AwardOrderPaidPoints` runs inside the subscriber app transaction.
* `point_transactions` 使用 `UNIQUE(order_id, type)` 保证同一订单的 `order_paid` 幂等。
* realtime points message 只在 subscriber 的积分事务 commit 后通过 `RegisterAfterCommit` 推送；推送失败不影响已提交业务数据。

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| event queue write fails inside `PayOrder` tx | order status rolls back to `pending`; no event rows remain |
| point account insert fails in subscriber | order remains `paid`; delivery becomes `failed`; queue retries |
| duplicate `order_id/type` point transaction | no extra points are awarded |
| realtime publish fails after subscriber commit | ignore realtime error; committed points remain valid |

### 5. Good/Base/Bad Cases

Good: `PayOrder` returns paid order after durable event is queued; queue runner later handles `points.award_on_order_paid`, writes point account/ledger, and after commit pushes realtime points.

Base: already-paid order returns idempotently and does not publish another `order.paid`.

Bad: assuming points are available immediately after `PayOrder` returns.

### 6. Tests Required

* `api/usecase/order_payment_points_test.go` verifies event/message enqueue, async points award, duplicate payment idempotency, duplicate event idempotency, queue failure rollback, and subscriber failure isolation.

### 7. Wrong vs Correct

#### Wrong

```go
points, _, err := AwardOrderPaidPoints(txCtx, cmd)
if err != nil {
    return err
}
return events.Publish(txCtx, event)
```

#### Correct

```go
event, err := usecaseevents.NewOrderPaidEvent(order)
if err != nil {
    return err
}
return events.Publish(txCtx, event)
```

---

## Common Mistakes

* Importing raw EventBus anywhere.
* Publishing a domain event from `routes` or `models`.
* Registering a subscriber in request path.
* Calling `queue.SendJSON` directly for a domain event.
* Wrapping durable event messages with `goqite/jobs.Create`; the queue body must stay a stable JSON envelope.
* Adding project delivery state to `goqite` instead of `domain_event_deliveries`.
* Treating durable subscribers as exactly-once. Handlers must be idempotent.
* Expecting cross-domain side effects to be synchronous with the publisher.
