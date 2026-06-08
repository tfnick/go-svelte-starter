# WebSocket Message Presentation Refactor

## Goal

Refactor the frontend/backend WebSocket communication model from a points-only payload into a reusable realtime message protocol. The backend should be able to publish multiple message types and optionally instruct the browser how to present them. The browser should dispatch messages by server instruction, with sensible defaults per message type.

## What I already know

* Current backend realtime transport is `api/framework/realtime`.
* Current WebSocket endpoint is `GET /api/points/ws` in `api/routes/points.go`.
* Current frontend WebSocket consumer lives in `frontend/src/pages/Dashboard.svelte`.
* Current message shape is points-specific: `{"type":"points.balance","user_id":"...","client_id":"...","balance":10}`.
* Current frontend handles only `payload.type === "points.balance"` and updates local `points`.
* Current event subscriber file is `api/usecase/points_events.go`.
* `points_events.go` subscribes to `order.paid`, calls points usecase logic, and publishes realtime points messages after app transaction commit.
* `routes` is HTTP adapter code and should not own domain/application event subscriptions.
* `usecase` owns orchestration, events, transaction boundaries, and cross-module coordination.

## Requirements

* Backend realtime messages must use a shared envelope that supports:
  * message type
  * presentation instruction
  * optional payload
  * optional client/user metadata where useful
* Supported message types for this task:
  * `points`
  * `async_export_task`
* Supported presentation modes for this task:
  * `refresh`
  * `toast`
* Default presentation behavior:
  * `points` defaults to refreshing/updating the points display.
  * `async_export_task` defaults to showing a toast.
  * If the backend explicitly provides a presentation mode, the browser follows that instruction.
* The browser client must dispatch incoming WebSocket messages by message type and presentation mode instead of hard-coding a single points-only payload shape.
* Existing points updates after order payment must continue to work.
* The frontend should ignore malformed or unsupported realtime messages without breaking the page.
* `api/usecase/points_events.go` should be moved into a child event directory, but not under `routes`.
* Architecture boundaries must remain:
  * `routes`: WebSocket HTTP upgrade, auth context, transport concerns.
  * `usecase`: application event subscription, points awarding orchestration, post-commit realtime publish.
  * `framework/realtime`: business-agnostic transport/envelope primitives.

## Acceptance Criteria

* [ ] Backend defines a reusable realtime message envelope in `api/framework/realtime`.
* [ ] Backend points publish path sends the new envelope format.
* [ ] Backend includes an async export task realtime message type/payload model, even if a full export job workflow is out of scope.
* [ ] Frontend parses the new envelope and dispatches `points` refresh messages to update points.
* [ ] Frontend dispatches `async_export_task` toast messages to a visible toast/notification UI.
* [ ] A logged-in user can click a header button after the logout button to trigger a backend `async_export_task` toast notification for verification.
* [ ] Existing WebSocket JWT behavior using `access_token` continues to work.
* [ ] Existing points update after payment continues to use after-commit publish.
* [ ] `points_events.go` event subscriber code is no longer at `api/usecase/points_events.go`; it is organized under a usecase event subdirectory or an equivalent usecase-owned event package.
* [ ] No domain/application event subscription code is moved into `api/routes`.
* [ ] Unit tests cover backend envelope defaults and frontend dispatch behavior.
* [ ] `go test ./...` passes.
* [ ] `cd frontend && npm test` passes.
* [ ] `cd frontend && npm run build` passes.

## Technical Approach

### Realtime Envelope

Recommended backend envelope shape:

```json
{
  "type": "points",
  "presentation": "refresh",
  "payload": {
    "user_id": "u001",
    "client_id": "...",
    "balance": 10
  }
}
```

Async export example:

```json
{
  "type": "async_export_task",
  "presentation": "toast",
  "payload": {
    "task_id": "task-001",
    "status": "completed",
    "message": "Export completed"
  }
}
```

The framework should provide default presentation resolution, so callers can omit `presentation` and still get type-specific defaults.

### Frontend Dispatch

Recommended frontend shape:

* Keep `frontend/src/api.js` responsible only for WebSocket URL and API helpers.
* Add a small realtime message dispatcher/helper, for example under `frontend/src/helpers/` or `frontend/src/stores/`.
* `Dashboard.svelte` should call the dispatcher from `onmessage`.
* Points refresh updates `points`.
* Async export task toast appends a transient toast notification.
* Add a minimal logged-in verification action in the header that calls a protected API and lets the backend publish an `async_export_task` message over WebSocket.

### Event File Placement Analysis

`points_events.go` should not move to `routes`.

Reason:

* It registers application/domain event subscribers.
* It calls usecase logic (`AwardOrderPaidPoints`).
* It coordinates transaction after-commit behavior.
* None of that is HTTP adapter responsibility.

Recommended placement is under usecase, for example:

```text
api/usecase/events/
  order_paid_points.go
```

Important Go dependency note:

* If `api/usecase/events` imports parent package `api/usecase`, then parent package files cannot import `api/usecase/events` without an import cycle.
* To avoid cycles, event constants and payload builders used by core usecases may need to move into the child event package, while handler registration receives dependencies by function/interface injection from startup, or the parent package must stop importing the child package.
* The implementation should keep event subscription ownership in the usecase layer and avoid moving it to routes merely to dodge import cycles.

## Decision (ADR-lite)

**Context**: Realtime messages are currently points-specific. New message types such as async export task notifications would otherwise lead to ad hoc payload checks spread across backend and frontend.

**Decision**: Introduce a reusable realtime envelope and frontend dispatcher. Keep business event subscriptions in the usecase layer under a child event package. Do not move event subscriber code into routes.

**Consequences**:

* Adding new realtime message types becomes mostly registration/configuration work.
* Frontend message handling becomes easier to test.
* The usecase event subpackage needs careful dependency direction to avoid Go import cycles.

## Out of Scope

* Building a full async export job engine.
* Building a full async export task workflow.
* Persisting realtime messages.
* Multi-node WebSocket fanout.
* Replacing the current WebSocket library.
* Adding outbox/event persistence.

## Technical Notes

* Current WebSocket route: `api/routes/points.go`.
* Current realtime hub: `api/framework/realtime/realtime.go`.
* Current points realtime payload: `api/usecase/points.go`.
* Current order paid points subscriber: `api/usecase/points_events.go`.
* Current frontend consumer: `frontend/src/pages/Dashboard.svelte`.
* Current frontend WebSocket helper: `frontend/src/api.js`.
* Relevant specs:
  * `.trellis/spec/backend/eventing-guidelines.md`
  * `.trellis/spec/backend/route-handler-guidelines.md`
  * `.trellis/spec/backend/api-contracts.md`
  * `.trellis/spec/frontend/svelte-vite-embed.md`

## Open Questions

* None. Updated decision: include a minimal protected verification trigger for `async_export_task` toast messages, but do not build a full export workflow.
