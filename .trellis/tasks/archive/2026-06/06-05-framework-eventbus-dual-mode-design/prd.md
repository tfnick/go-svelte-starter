# Framework EventBus Dual-Mode Module Decoupling

## Goal

Implement a framework-level module decoupling mechanism based on
`github.com/asaskevich/EventBus`.

The design must support two event handling scenarios:

* **Async best-effort**: the main transaction publishes an event, listeners
  handle it asynchronously, and loss is acceptable. Example: sending messages.
* **Sync transaction**: the main transaction publishes an event, listeners run
  synchronously in the same app transaction, and all related writes persist or
  roll back together. Example: awarding points after order creation.

The detailed design is complete. This task now proceeds with a small framework
implementation that keeps EventBus behind `api/framework/events` and avoids any
event persistence tables.

## What I Already Know

* The user wants a new clean task because the previous research task became
  muddled.
* The user wants `github.com/asaskevich/EventBus` as the foundation.
* The user wants module decoupling, not an outbox implementation.
* The system currently uses `routes -> usecase -> models -> db`.
* `api/framework` owns reusable infrastructure.
* `api/usecase` owns business transaction boundaries through
  `fwusecase.WithAppTx(ctx, fn)`.
* `api/models` use `db.ExecutorFor` / `db.DynamicExecutorFor`, so app writes can
  join the active transaction through `txCtx.Std()`.
* `db.WithTx` only supports the `app` database. Shared DB writes are not part of
  app transactions.
* Current `fwusecase.WithAppTx` does not expose after-commit hooks yet.
* EventBus supports in-process pub/sub, sync callback dispatch, and async
  callback dispatch, but framework code must provide typed contracts, error
  collection, transaction hooks, and logging.
* The user explicitly rejected framework event/execution tables; the design
  must not include them even as an optional audit path.

## Requirements

* Create and implement the `api/framework/events` package.
* Add only EventBus-based eventing to the design. Do not introduce outbox.
* Keep `github.com/asaskevich/EventBus` behind the framework boundary.
* Provide two delivery modes:
  * `async_best_effort`;
  * `sync_tx`.
* Allow one topic to have multiple subscribers across both modes.
* Use stable subscriber names; do not infer subscriber identity from function or
  Go type names.
* For `async_best_effort`:
  * dispatch asynchronously;
  * do not persist event or execution rows;
  * log handler failures;
  * if published inside an app transaction, dispatch only after commit;
  * if the app transaction rolls back, discard the async event.
* For `sync_tx`:
  * require publishing inside `fwusecase.WithAppTx`;
  * run handlers synchronously before transaction commit;
  * pass `fwusecase.Context` to handlers so model writes join the active app
    transaction;
  * do not add framework event/execution tables;
  * persist the listeners' business writes in the same transaction;
  * return an error when any sync handler fails, so the usecase transaction
    rolls back.
* Define and implement API contracts, package placement, transaction hook
  changes, logging expectations, and test requirements.
* Document what this design does not guarantee.

## Acceptance Criteria

* [x] PRD exists and captures the two scenarios.
* [x] Detailed design exists in `info.md`.
* [x] EventBus library notes exist under `research/`.
* [x] Design avoids outbox, background workers, retry queues, replay, and
  dead-letter queues.
* [x] Design identifies how async events are delayed until after commit.
* [x] Design identifies how sync handlers join the app transaction and roll
  back on failure.
* [x] Design includes transaction sequence diagrams for order-created points
  reward success and rollback.
* [x] Design removes framework event/execution tables entirely.
* [x] Design includes business-side example code for async SMS and sync points
  reward scenarios.
* [x] Design identifies framework package boundaries and API contracts.
* [x] Design identifies transaction hook, logging, and test requirements.
* [x] `github.com/asaskevich/EventBus` is added as a framework-only dependency.
* [x] `api/framework/usecase` supports after-commit hooks for successful app
  transactions.
* [x] `api/framework/events` provides async best-effort and sync transaction
  publishing APIs.
* [x] Tests cover transaction hooks, async dispatch, sync rollback, and
  architecture import boundaries.

## Definition of Done

* `prd.md`, `info.md`, and research notes remain aligned with the implemented
  design.
* EventBus remains hidden behind `api/framework/events`.
* No outbox package, framework event tables, event execution tables, retry
  queues, replay, or background worker is introduced.
* Go tests for the framework eventing and transaction behavior pass.

## Decision (ADR-lite)

**Context**: The project needs module decoupling. Some event consumers are
loss-tolerant and should run asynchronously. Other consumers must be atomic with
the main app transaction.

**Decision**: Build `api/framework/events` around
`github.com/asaskevich/EventBus`, with two delivery modes:
`async_best_effort` and `sync_tx`. Add framework-level after-commit hooks so
async events published during a transaction are dispatched only after a
successful commit.

**Consequences**: The design is intentionally simpler than outbox. Async events
can be lost and are not retried. Sync events are reliable only for work that can
participate in the same app DB transaction.

## Out of Scope

* Outbox pattern.
* Durable async events.
* Background workers.
* Retry queues.
* Dead-letter queues.
* Replay/backfill.
* Cross-database atomicity with `shared`.
* External API atomicity.
* Framework event/execution tables.
* Framework-level event audit tables.

## Research References

* `research/eventbus-library-notes.md` - EventBus capabilities and constraints.
* `info.md` - Detailed project design.

## Technical Notes

Relevant current files:

* `api/framework/usecase/context.go`
* `api/framework/usecase/transaction.go`
* `api/db/tx.go`
* `api/framework/logging/logging.go`
* `api/usecase/order.go`
* `.trellis/spec/backend/database-guidelines.md`
* `.trellis/spec/backend/directory-structure.md`

Existing dirty files `data/app.db-shm` and `data/app.db-wal` are unrelated
runtime files and should remain excluded.
