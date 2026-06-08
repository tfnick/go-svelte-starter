# Framework EventBus Dual-Mode Design Research

## Summary

The user clarified the final direction: keep only the EventBus-based scheme and
do not use the outbox pattern.

The framework should support two EventBus delivery modes:

* `async_best_effort`: for listeners that run asynchronously and may be lost,
  such as sending messages.
* `sync_tx`: for listeners that must complete in the same app transaction, such
  as awarding points after order creation.

The recommended package is `api/framework/events`, backed by
`github.com/asaskevich/EventBus` for in-process topic dispatch and wrapped by
project-owned contracts for transaction boundaries, subscriber mode, and
logging.

## Current Project Constraints

### Layering

Current dependency direction is:

```text
routes -> usecase -> models -> db
```

`api/framework` owns reusable, business-agnostic infrastructure. That makes
`api/framework/events` the right package for EventBus integration.

Usecase code should call framework event APIs, not import
`github.com/asaskevich/EventBus` directly. Models should not own event
publishing or transaction orchestration.

### Database / Transaction Model

The app DB is the only transactional database for business flows:

* `db.WithTx(ctx, "app", fn)` opens/reuses an app transaction.
* `db.WithTx(ctx, "shared", fn)` is intentionally rejected.
* Models use `db.ExecutorFor(ctx, name)` so writes can join an active app
  transaction when present.

`sync_tx` handlers must use the same `fwusecase.Context` that is inside
`fwusecase.WithAppTx`. That makes their app DB writes commit or roll back with
the usecase.

`async_best_effort` handlers must not use the active SQL transaction. When an
async event is published inside an app transaction, dispatch should wait until
the app transaction commits.

### Order Flow

Order creation currently:

1. validates user/product input;
2. reserves product stock in the `shared` DB;
3. writes order and order items inside an app DB transaction;
4. compensates `shared` stock if the app transaction fails.

For an order-created event:

* `sync_tx` listener: award points in the app DB and roll back with the order if
  it fails.
* `async_best_effort` listener: send a message after the app transaction commits
  successfully; if the process dies before dispatch, the message can be lost.

## Non-Goals

* Do not add an outbox package.
* Do not create `outbox_events` or `outbox_deliveries`.
* Do not add a background worker.
* Do not design retry queues, dead-letter queues, replay, or backfill.
* Do not expose `github.com/asaskevich/EventBus` outside `api/framework/events`.
* Do not let async handlers share the active SQL transaction.

## Recommended Architecture

### Package Placement

Create:

```text
api/framework/events/
```

This package should own:

* EventBus dependency integration;
* typed event envelope;
* subscriber metadata and stable subscriber names;
* delivery mode routing;
* async handler contracts;
* transaction-aware handler contracts;
* sync event and execution persistence;
* after-commit dispatch hooks;
* structured logging for async and sync handler failures.

Domain modules can register handlers during application startup, but the
reusable eventing machinery stays in framework.

### Components

```text
api/framework/events
  Event             framework event envelope
  DeliveryMode      async_best_effort or sync_tx
  Subscription      topic + subscriber + mode
  Registry          typed registration metadata
  Bus               wraps github.com/asaskevich/EventBus
  Publisher         PublishAsync / PublishInTx entrypoints
  Store             app transaction-aware sync persistence
  TxHooks           after-commit queue integration
```

### Why a Wrapper Is Required

Raw EventBus dispatch is not enough for the framework semantics:

* usecases need stable APIs instead of raw `Publish`;
* framework must route subscribers by delivery mode;
* sync handlers need `fwusecase.Context`;
* async handlers need post-commit dispatch when published in a transaction;
* framework must collect sync handler errors;
* framework must log async handler failures.

Therefore EventBus should be retained as the dispatch foundation, but it should
sit behind `api/framework/events`.

## Core Contracts

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

type DeliveryMode string

const (
    DeliveryModeAsyncBestEffort DeliveryMode = "async_best_effort"
    DeliveryModeSyncTx          DeliveryMode = "sync_tx"
)

type Subscription struct {
    Topic      string
    Subscriber string
    Mode       DeliveryMode
}

type AsyncHandler interface {
    Handle(ctx context.Context, event Event) error
}

type TxHandler interface {
    Handle(ctx fwusecase.Context, event Event) error
}

type Registry interface {
    RegisterAsync(subscription Subscription, handler AsyncHandler) error
    RegisterSync(subscription Subscription, handler TxHandler) error
}

type Publisher interface {
    PublishAsync(ctx fwusecase.Context, event Event) error
    PublishInTx(ctx fwusecase.Context, event Event) error
}
```

Subscriber names must be stable because sync executions store them in
`domain_event_executions`. Do not use function names or Go type names as
implicit subscriber IDs.

## Async Best-Effort Mode

### Semantics

Use for tasks where loss is acceptable:

* sending messages;
* non-critical notifications;
* telemetry;
* optional external calls.

Behavior:

1. Usecase calls `events.PublishAsync(ctx, event)`.
2. Framework resolves `async_best_effort` subscribers for the topic.
3. If the usecase is inside `fwusecase.WithAppTx`, framework queues dispatch
   for after commit.
4. If the usecase is not inside an app transaction, framework dispatches
   immediately.
5. EventBus runs handlers asynchronously.
6. Handler errors are logged and do not affect the usecase result.

Async events are not persisted. This is intentional because the scenario allows
loss.

### After-Commit Queue

The framework should add a small in-memory transaction hook around
`fwusecase.WithAppTx`.

Conceptual flow:

```go
err := fwusecase.WithAppTx(ctx, func(txCtx fwusecase.Context) error {
    // business writes
    events.PublishAsync(txCtx, OrderCreated(order))
    return nil
})
// if commit succeeded, queued async events are published
```

If the app transaction rolls back, queued async events are discarded.

If the process crashes after commit but before dispatch, the async event may be
lost. That is accepted by this mode.

## Sync Transaction Mode

### Semantics

Use for tasks that must commit or roll back with the main usecase:

* order-created points reward;
* internal audit rows;
* module-local app DB bookkeeping;
* app DB projections that must be atomic with the usecase.

Behavior:

1. Usecase calls `events.PublishInTx(txCtx, event)` inside
   `fwusecase.WithAppTx`.
2. Framework resolves both subscriber groups for the topic.
3. Framework persists one `domain_events` row if sync subscribers exist.
4. Framework persists one `domain_event_executions` row per sync subscriber.
5. Framework dispatches sync subscribers through EventBus synchronously.
6. Sync handlers receive `fwusecase.Context` and write through the active app
   transaction.
7. Framework marks sync executions as `processed` or `failed`.
8. If any sync handler fails, `PublishInTx` returns an error and the app
   transaction rolls back.
9. Async subscribers for the same event are queued for after-commit dispatch
   only if the app transaction succeeds.

This preserves module decoupling while keeping required work atomic.

### EventBus Dispatch Shape

The framework can register each typed sync handler as a normal synchronous
EventBus callback:

```go
func (r *Registry) RegisterSync(sub Subscription, handler TxHandler) error {
    r.syncSubs[sub.Topic] = append(r.syncSubs[sub.Topic], sub)

    return r.bus.Subscribe(syncTopic(sub.Topic), func(dispatch *SyncDispatch) {
        err := handler.Handle(dispatch.TxContext, dispatch.Event)
        dispatch.Record(sub, err)
    })
}
```

`PublishInTx` calls the sync topic with a framework-owned dispatch object:

```go
bus.Publish(syncTopic(event.Topic), dispatch)
```

Do not use async EventBus callbacks for `sync_tx` handlers because they cannot
safely share the active SQL transaction.

## Database Design

Only sync transaction events require persistence.

### Events Table

One immutable row per sync-published domain event:

```sql
CREATE TABLE IF NOT EXISTS domain_events (
    id             TEXT PRIMARY KEY,
    topic          TEXT NOT NULL,
    aggregate_type TEXT NOT NULL,
    aggregate_id   TEXT NOT NULL,
    payload_json   TEXT NOT NULL,
    metadata_json  TEXT NOT NULL DEFAULT '{}',
    occurred_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Recommended indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_domain_events_topic_created
    ON domain_events(topic, created_at);

CREATE INDEX IF NOT EXISTS idx_domain_events_aggregate
    ON domain_events(aggregate_type, aggregate_id, created_at);
```

### Executions Table

One row per sync subscriber:

```sql
CREATE TABLE IF NOT EXISTS domain_event_executions (
    id             TEXT PRIMARY KEY,
    event_id       TEXT NOT NULL,
    subscriber     TEXT NOT NULL,
    mode           TEXT NOT NULL DEFAULT 'sync_tx',
    status         TEXT NOT NULL DEFAULT 'pending',
    last_error     TEXT,
    started_at     DATETIME,
    finished_at    DATETIME,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (event_id) REFERENCES domain_events(id) ON DELETE CASCADE,
    UNIQUE (event_id, subscriber)
);
```

Recommended indexes:

```sql
CREATE INDEX IF NOT EXISTS idx_domain_event_executions_event
    ON domain_event_executions(event_id);

CREATE INDEX IF NOT EXISTS idx_domain_event_executions_subscriber_status
    ON domain_event_executions(subscriber, status, created_at);
```

Possible statuses:

```text
pending -> processing -> processed
pending -> processing -> failed
```

No retry status is needed because this design intentionally has no background
delivery or retry queue.

## Fit for Order Flow

For order creation:

1. reserve stock in shared DB;
2. open app transaction;
3. insert order and order items;
4. call `events.PublishInTx(txCtx, OrderCreated(order))`;
5. sync points-reward handler runs inside the app transaction;
6. async message handler is queued for after commit;
7. if the points handler fails, app transaction rolls back and the async message
   is not dispatched;
8. shared stock compensation still runs on app transaction failure.

This matches the user's two examples:

* sending messages: `async_best_effort`;
* order-created points reward: `sync_tx`.

## Logging

Async failures are only visible through logs.

Required log fields:

* mode;
* event ID;
* topic;
* aggregate type and ID;
* subscriber;
* handler status;
* error message;
* usecase/request correlation fields if available.

Sync handler failures should also be logged because their `failed` execution
rows roll back when the usecase transaction rolls back.

## Testing Requirements

Async best-effort:

* `PublishAsync` outside a transaction dispatches async subscribers.
* `PublishAsync` inside a transaction dispatches only after successful commit.
* `PublishAsync` inside a rolled-back transaction does not dispatch.
* Async handler failure is logged and does not return a usecase error.
* Async events do not create `domain_events` or `domain_event_executions` rows.

Sync transaction:

* `PublishInTx` persists one `domain_events` row when sync subscribers exist and
  the transaction commits.
* `PublishInTx` persists one `domain_event_executions` row per sync subscriber
  when the transaction commits.
* One event with two sync subscribers invokes both handlers through EventBus.
* Sync handler writes join the active app transaction.
* Sync handler failure rolls back business writes, event row, and execution
  rows.
* Sync handler failure is logged with topic, event ID, and subscriber.
* Async subscribers on the same event are dispatched after commit only when sync
  processing succeeds.
* Usecase, routes, and models do not import `github.com/asaskevich/EventBus`.
* No `api/framework/outbox` package or outbox migration is created.

## Implementation Phases

### Phase 1: Dependency + Framework Contracts

* Add `github.com/asaskevich/EventBus` to `go.mod`.
* Add `api/framework/events` package.
* Define event, delivery mode, subscription, async handler, sync handler,
  registry, and publisher contracts.
* Keep the external EventBus dependency private to the framework package.

### Phase 2: Transaction Hooks

* Extend framework transaction handling so `fwusecase.WithAppTx` can collect
  after-commit callbacks.
* Ensure callbacks run only after app transaction commit succeeds.
* Ensure callbacks are discarded on rollback.
* Test hook ordering and rollback behavior.

### Phase 3: Sync Persistence

* Add app migration for `domain_events`.
* Add app migration for `domain_event_executions`.
* Implement transaction-aware store methods using the app executor from
  `fwusecase.Context`.
* Test commit and rollback behavior.

### Phase 4: EventBus Dispatch

* Implement registry registration with stable subscriber names.
* Register async callbacks for `async_best_effort`.
* Register sync callbacks for `sync_tx`.
* Implement `PublishAsync`.
* Implement `PublishInTx`.
* Implement aggregate error handling and structured logging.
* Test multi-subscriber fan-out across both modes.

### Phase 5: First Domain Event

* Add a small `order.created` event publisher inside order app transaction.
* Register a sync points-reward demo/test handler.
* Register an async message demo/test handler.
* Keep shared stock compensation unchanged.

## Recommendation

Proceed with EventBus-only dual-mode eventing:

> Build `api/framework/events` around `github.com/asaskevich/EventBus`, with
> `async_best_effort` for loss-tolerant async listeners and `sync_tx` for
> transaction-participating listeners.

Do not build outbox infrastructure for this project stage.
