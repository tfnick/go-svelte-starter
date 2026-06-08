# Replace WebSocket with SSE

## Goal

Replace the current browser-to-server WebSocket realtime transport with Server-Sent Events (SSE), while preserving the existing realtime features: points balance refresh after paid orders and async export toast notifications. The replacement should keep the existing realtime message envelope and hub semantics where possible, so business behavior remains stable and only the transport changes.

## What I already know

* User asked for a technical component replacement task: use new SSE technology to replace the current WebSocket technology.
* Previous WebSocket endpoint was registered as `GET /api/points/ws`.
* Previous backend WebSocket implementation was in `api/routes/points.go`.
* Previous frontend WebSocket URL helper was `pointsWebSocketURL()` in `frontend/src/api.js`.
* Existing frontend connection logic is in `frontend/src/pages/Dashboard.svelte`.
* Existing realtime message parsing/dispatch is transport-neutral in `frontend/src/helpers/realtimeMessages.js`.
* Existing realtime backend hub is `api/framework/realtime`, with user and client subscriptions.
* Existing realtime message envelope includes `type`, `presentation`, and `payload`.
* Existing realtime features:
  * `points` + `refresh` updates points balance.
  * `async_export_task` + `toast` displays export notification toasts.
* Existing auth middleware supports `Authorization: Bearer <token>` and `access_token` query param.
* WebSocket was using query token because browsers cannot set custom headers on WebSocket connections.
* SSE `EventSource` also cannot set custom headers in the native browser API, so query token or another browser-compatible auth strategy is still needed.
* Previous Vite proxy set `ws: true` only because WebSocket was used.

## Assumptions

* This task replaces WebSocket for browser realtime only; the internal `api/framework/realtime` hub can stay as the in-process publish/subscribe abstraction.
* Native `EventSource` is preferred over adding a new client dependency.
* The SSE endpoint should be protected by the existing auth middleware using the existing `access_token` query parameter.
* The existing `realtime` JSON envelope should be sent as SSE `data:` payload.
* This task does not introduce bidirectional client-to-server realtime commands; existing WebSocket receive loop only keeps the connection alive and ignores client messages.
* Resolved: remove the old `/api/points/ws` WebSocket endpoint immediately; do not keep a compatibility alias.

## Open Questions

* Resolved: remove `/api/points/ws` immediately and expose only the SSE stream endpoint.

## Requirements

* Add an SSE endpoint for the points/realtime stream.
* Remove the old WebSocket endpoint and frontend WebSocket URL helper.
* Replace browser `WebSocket` usage in `Dashboard.svelte` with `EventSource`.
* Replace `pointsWebSocketURL()` with an SSE URL helper.
* Preserve initial points message behavior: when the stream opens, the client should receive the current points balance.
* Preserve `points` + `refresh` updates after order payment.
* Preserve `async_export_task` + `toast` notifications.
* Preserve safe behavior for malformed realtime messages.
* Preserve disconnected/error UI behavior in Dashboard.
* Ensure disabled/missing/invalid users are rejected by existing auth behavior.
* Update tests that currently assert WebSocket helper URLs.
* Update specs to describe SSE instead of WebSocket.

## Acceptance Criteria

* [x] Frontend no longer constructs `new WebSocket(...)`.
* [x] Frontend uses `EventSource` to connect to the realtime stream.
* [x] Backend exposes an SSE stream endpoint with `Content-Type: text/event-stream`.
* [x] SSE stream sends the same realtime JSON envelope in `data: ...`.
* [x] Initial points balance is delivered over SSE.
* [x] Paying an order still updates points through realtime refresh.
* [x] Triggering export toast still displays a toast through realtime.
* [x] `frontend/src/helpers/realtimeMessages.js` behavior remains compatible.
* [x] Existing HTTP points refresh still works as fallback.
* [x] Relevant backend and frontend tests pass.

## Definition of Done

* Tests added/updated for SSE endpoint and frontend URL/helper behavior.
* `go test ./...` passes.
* `cd frontend && npm test` passes.
* `cd frontend && npm run build` passes.
* Specs updated to replace WebSocket contract with SSE contract.
* Work is committed before Trellis finish-work archival.

## Out of Scope

* Introducing bidirectional realtime commands.
* Persisting realtime messages across reconnects.
* Cross-process fanout or distributed realtime hub.
* Replacing the internal `api/framework/realtime` hub abstraction.
* Adding browser notification permissions or service workers.

## Technical Notes

* Route registration is now `index.go` -> `protected.GET("/points/sse", user.PointsSSE)`.
* Backend SSE route lives in `api/routes/points.go`; it sends initial points, forwards `framework/realtime` messages, and emits keepalive comments.
* Existing realtime hub remains `api/framework/realtime/realtime.go`.
* Frontend helper is now `pointsSSEURL()` in `frontend/src/api.js`.
* Frontend connection logic in `frontend/src/pages/Dashboard.svelte` now uses native `EventSource`.
* Frontend envelope dispatch remains `frontend/src/helpers/realtimeMessages.js`.
* Vite proxy no longer sets `ws: true`.
* Added backend route tests in `api/routes/points_test.go`.
* Updated query-token auth test in `api/framework/http/middleware/auth_test.go`.
* Updated frontend helper tests in `frontend/src/api.test.js`.
* Verified: `go test ./api/routes ./api/framework/http/middleware ./api/framework/realtime`, `cd frontend && npm test`, `cd frontend && npm run build`.
