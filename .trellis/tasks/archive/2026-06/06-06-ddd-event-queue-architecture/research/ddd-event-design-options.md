# DDD Event Design Options

## Context

The repository now has a transaction-aware `goqite` queue wrapper and project-owned durable event tables:

* `api/framework/queue` owns raw `maragu.dev/goqite` usage.
* `domain_events` stores the event fact.
* `domain_event_deliveries` stores per-subscriber delivery state.
* `QueueDomainEvents` carries JSON envelopes that point to one event and one subscriber.

The current `api/framework/events` package still mixes three delivery styles:

* `async_best_effort` via raw `github.com/asaskevich/EventBus`.
* `sync_tx` via raw EventBus, used by `order.paid -> points.award_on_order_paid`.
* `durable_async` via goqite and domain event tables.

The user's goal is a new DDD event design that completely replaces the existing EventBus approach.

## Common Pattern Fit

For this codebase, the closest fit is a transactional-outbox-like durable domain event design:

1. The publishing usecase mutates aggregate state inside an app transaction.
2. The same transaction persists a domain event fact.
3. The same transaction creates one delivery row and one queue message per subscriber.
4. Queue workers process one subscriber delivery at a time.
5. Handlers are idempotent because queue retry can invoke them more than once.

This keeps module decoupling at the event boundary while preserving rollback behavior for event publication itself.

## Option A: Queue-First Durable Events (Recommended)

All cross-domain event subscribers are durable async subscribers. Raw EventBus and sync event handlers are removed.

How it works:

* `events.Publish(ctx, event)` validates and persists the event.
* The publisher writes event/delivery/message rows in the active app transaction when one exists.
* Each subscriber is invoked later by the `domain-events` queue runner.
* Business invariants that must be committed with the publisher are not modeled as event subscribers; they stay as explicit usecase/domain service calls.

Pros:

* Best module decoupling.
* One delivery path to understand and test.
* Failures are retryable and inspectable.
* Removes raw EventBus dependency completely.

Cons:

* Synchronous cross-module behavior changes to eventual consistency.
* `PayOrder` currently guarantees points are awarded in the same transaction; this would need either eventual points or an explicit in-transaction call outside the event mechanism.

Best when:

* The product accepts that cross-domain side effects may finish after the main command returns.
* The architecture values decoupling and durability over immediate synchronous side effects.

## Option B: Queue-Backed Events Plus Explicit Sync Policy

Remove raw EventBus, but keep a framework-level synchronous transaction handler path implemented without third-party EventBus.

How it works:

* Durable async handlers still use goqite.
* A narrow `RegisterSync` / `PublishInTx` equivalent remains for same-transaction side effects.
* Sync handlers are invoked directly from an internal registry, not through raw EventBus.

Pros:

* Preserves current `PayOrder` atomic points behavior.
* Removes the raw EventBus dependency.
* Lower migration risk for existing tests.

Cons:

* Still couples publisher runtime to subscriber runtime.
* Leaves two mental models: sync tx handlers and durable async handlers.
* Sync event handlers can hide business dependencies behind an event-shaped API.

Best when:

* Some existing side effects are true business invariants and must remain atomic with the publisher.

## Option C: Phased Compatibility Facade

Keep old function names temporarily, but reimplement them over the new event facade and migrate call sites over time.

How it works:

* `PublishAsync`, `PublishInTx`, `RegisterSync`, etc. remain temporarily.
* Internals route durable paths to the new queue-backed implementation.
* Tests and specs mark old APIs deprecated.

Pros:

* Lowest implementation shock.
* Allows one smaller PR to land before complete API cleanup.

Cons:

* Does not satisfy "completely replace existing event bus scheme" as cleanly.
* Deprecated APIs tend to linger.
* More code paths to test during transition.

Best when:

* The team wants a risk-managed migration, accepting temporary compatibility debt.

## Recommendation

Choose Option A unless `order.paid -> points` or similar flows are hard business invariants that must commit atomically with the publisher.

If Option A is chosen, update the order payment design explicitly:

* `PayOrder` transaction marks order paid and publishes `order.paid` durably.
* A queue subscriber awards points idempotently.
* The API returns payment success once the event is queued, not once points are awarded.
* Realtime points notification is sent after the points subscriber commits.

If atomic points are required, use Option B and make the sync policy explicit in the new design.
