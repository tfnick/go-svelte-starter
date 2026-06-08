# Framework EventBus Dual-Mode Design

## Goal

Design an EventBus-based domain event mechanism under `api/framework/events`.
The project will keep EventBus as the eventing foundation and will not introduce
an outbox pattern.

The framework must support two scenarios:

* **Async best-effort**: the main usecase publishes an event, listeners run
  asynchronously, and loss is acceptable. Example: sending a message.
* **Sync transaction**: the main usecase publishes an event, listeners complete
  synchronously inside the same app transaction, and all related writes commit
  or roll back together. Example: order-created points reward.

## What I Already Know

* The user wants a design task, not immediate implementation.
* The user explicitly wants to keep only the EventBus-based scheme and no
  longer wants an outbox pattern.
* A published domain event may be listened to and consumed by multiple modules.
* `github.com/asaskevich/EventBus` is the desired event dispatch foundation.
* EventBus supports in-process publish/subscribe; transaction persistence and
  after-commit semantics must be added by the project framework around it.
* Current backend layering is `routes -> usecase -> models -> db`.
* `api/framework/` owns business-agnostic infrastructure.
* `api/usecase` owns business transaction boundaries through
  `fwusecase.WithAppTx(ctx, fn)`.
* `api/models` access storage via transaction-aware `db.ExecutorFor` /
  `db.DynamicExecutorFor`.
* `db.WithTx` intentionally supports only the `app` database.
* Order creation currently writes to `shared` first for stock reservation, then
  writes `orders` / `order_items` in an `app` transaction, with explicit shared
  compensation on failure.
* App migrations are embedded under `api/db/migrations/app`.
* SQLite runs in single-connection mode with WAL enabled.

## Requirements

* Design only the EventBus-based eventing scheme.
* Do not design or implement an outbox package, outbox tables, background
  worker, retry queue, dead-letter queue, or replay mechanism.
* Add framework package placement and API boundaries for `api/framework/events`.
* Keep `github.com/asaskevich/EventBus` behind the framework boundary; usecase,
  models, and routes should not import it directly.
* Support two subscriber modes:
  * `async_best_effort`: async listener, no durability guarantee, failure logs
    only, no rollback coupling;
  * `sync_tx`: synchronous listener, participates in the active app transaction,
    failure rolls back the usecase transaction.
* If an async event is published inside `fwusecase.WithAppTx`, dispatch it only
  after the app transaction commits successfully. If the transaction rolls back,
  discard the async event.
* If a sync event is published inside `fwusecase.WithAppTx`, execute all
  `sync_tx` listeners before the app transaction commits.
* Persist sync transaction event execution:
  * insert one `domain_events` row in the active app transaction;
  * insert one `domain_event_executions` row per `sync_tx` subscriber;
  * update execution rows before the transaction completes.
* Support one event being consumed by multiple modules with stable subscriber
  names.
* Identify migration, transaction hook, logging, and test implications for a
  later implementation task.
* Keep this task research/design-only; do not add dependencies or implement
  production code in this task.

## Acceptance Criteria

* [x] PRD states that EventBus is the only eventing direction.
* [x] PRD explicitly removes outbox from the design.
* [x] PRD captures both async best-effort and sync transaction scenarios.
* [x] Research document exists under `research/`.
* [x] Design supports multiple subscribers per domain event.
* [x] Design explains after-commit async dispatch.
* [x] Design explains sync transaction dispatch and rollback.
* [x] Design identifies framework package placement.
* [x] Design identifies migration, transaction hook, logging, and test
  requirements for a later implementation task.

## Definition of Done

* Research artifacts are written to files.
* No production code or dependency changes are made.
* User can decide whether to proceed with EventBus implementation based on the
  design.

## Recommended Direction

Build `api/framework/events` as the only eventing framework package.

Use `github.com/asaskevich/EventBus` for in-process topic dispatch, but wrap it
with project-owned typed contracts so the framework can separate async
best-effort listeners from sync transaction listeners.

Recommended core shape:

```text
api/framework/events
  Event             typed event envelope
  Subscription      topic + stable subscriber name + delivery mode
  AsyncHandler      best-effort listener using context.Context
  TxHandler         transaction listener using fwusecase.Context
  Registry          stores subscriptions and registers EventBus callbacks
  Publisher         PublishAsync / PublishInTx entrypoints
  Store             app-transaction-aware sync event persistence
```

There should be no `api/framework/outbox` package in this plan.

## Event Modes

### Async Best-Effort

Use this mode for side effects that may be lost:

* sending messages;
* fire-and-forget notifications;
* non-critical telemetry;
* non-critical external calls.

Semantics:

* listener runs asynchronously;
* listener does not share the main app transaction;
* listener failure does not roll back the usecase;
* framework logs failure;
* event and execution rows are not required because loss is acceptable.

When published inside `fwusecase.WithAppTx`, async dispatch should be queued and
flushed only after successful commit. This prevents async handlers from acting
on data that later rolls back.

### Sync Transaction

Use this mode for side effects that must not be lost and can participate in the
same app transaction:

* order-created points reward;
* internal audit rows that must commit with the aggregate;
* module-local bookkeeping in the app DB;
* projections that must be atomic with the usecase.

Semantics:

* listener runs synchronously before the usecase transaction commits;
* listener receives `fwusecase.Context` and writes through the active app
  transaction;
* listener failure returns from `PublishInTx`;
* `fwusecase.WithAppTx` rolls back business writes, event row, and execution
  rows together.

Sync transaction mode is limited to work that can join the app transaction.
It cannot make external APIs or the `shared` database atomic with app DB writes.

## Candidate Framework Contracts

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

type Publisher interface {
    PublishAsync(ctx fwusecase.Context, event Event) error
    PublishInTx(ctx fwusecase.Context, event Event) error
}
```

Subscriber names must be explicit and stable. Do not derive subscriber IDs from
function names or Go type names because renames would change persistence keys.

## Publish Semantics

`PublishAsync`:

1. resolves `async_best_effort` subscribers for the topic;
2. if the current context has an active app transaction hook, queues dispatch
   for after commit;
3. otherwise dispatches immediately through EventBus async handling;
4. logs handler failures;
5. does not persist event or execution rows.

`PublishInTx`:

1. resolves both subscriber groups for the topic;
2. persists one `domain_events` row if there are `sync_tx` subscribers;
3. persists one `domain_event_executions` row per `sync_tx` subscriber;
4. dispatches `sync_tx` subscribers synchronously through EventBus;
5. updates sync execution rows to `processed` or `failed`;
6. returns an aggregate error if any `sync_tx` subscriber fails;
7. queues `async_best_effort` subscribers for after commit if the transaction
   succeeds.

This lets one event topic support both examples:

* `order.points_reward` as `sync_tx`;
* `order.message_notification` as `async_best_effort`.

## Transaction Hook Requirement

The framework needs an after-commit hook around `fwusecase.WithAppTx`.

Conceptually:

```go
err := fwusecase.WithAppTx(ctx, func(txCtx fwusecase.Context) error {
    // business writes
    // events.PublishAsync(txCtx, event) queues async callbacks
    // events.PublishInTx(txCtx, event) runs sync callbacks now
    return nil
})
// if commit succeeded, queued async callbacks are published
```

The hook is not an outbox. It is only an in-memory list of callbacks/events for
the current usecase call. If the process crashes after commit but before
dispatch, async events can be lost by design.

## Decision (ADR-lite)

**Context**: The project wants EventBus as the only eventing direction and has
two distinct consistency needs: best-effort async side effects and
transactional synchronous module reactions.

**Decision**: Build `api/framework/events` around `github.com/asaskevich/EventBus`
with two modes: `async_best_effort` and `sync_tx`. Add transaction hook support
so async listeners published inside a transaction run after commit, while sync
listeners run inside the transaction and roll back with it on failure.

**Consequences**: The design stays simpler than outbox and matches the current
project stage. Async work may be lost and has no retry/replay. Sync work is
safe only for app-DB transaction participants and runs before the usecase
returns.

## Out of Scope

* `api/framework/outbox`.
* `outbox_events` / `outbox_deliveries` tables.
* Background event workers.
* Retry queues, dead-letter queues, replay, backfill, or eventual delivery.
* Using async EventBus handlers for transaction-participating logic.
* Letting routes, usecases, or models import `github.com/asaskevich/EventBus`
  directly.
* Publishing real order events in this research task.
* Reworking shared DB compensation.
* Changing Open API or frontend behavior.
* Touching unrelated runtime database files.

## Research References

* `research/eventbus-transaction-design.md` - EventBus-only dual-mode event
  design.

## Technical Notes

* Relevant current files:
  * `api/db/db.go`
  * `api/db/tx.go`
  * `api/framework/usecase/context.go`
  * `api/framework/usecase/transaction.go`
  * `api/usecase/order.go`
  * `api/models/order.go`
  * `api/db/migrations/app/*.sql`
  * `.trellis/spec/backend/database-guidelines.md`
  * `.trellis/spec/backend/directory-structure.md`
* Existing dirty files `data/app.db-shm` and `data/app.db-wal` are unrelated
  runtime files and should remain excluded.
