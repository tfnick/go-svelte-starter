# Current Realtime Notification State

## Findings

* `api/usecase/notification.go` already provides the durable notification path:
  `CreateNotification` writes a row into `notifications`, then publishes a
  `notification` realtime message through `publishRealtimeNotification`.
* `api/usecase/notifications.go` is the Experiment realtime trigger path. It
  currently calls `realtime.Publish` directly with an `async_export_task`
  message, so it does not create a notification ledger row.
* `api/usecase/order_export.go` currently publishes an `async_export_task`
  realtime message directly and then also calls `CreateNotification`. This can
  produce realtime UI behavior, but the business path is split across direct
  websocket publishing and notification ledger creation.
* `api/usecase/heavy_task.go` directly publishes a `heavy_task` realtime toast
  for non-order-export heavy tasks and does not create a notification ledger row.
* `api/usecase/events/order_paid_points.go` directly publishes a `points`
  realtime refresh after a successful transaction commit. This is a state-sync
  message rather than a user-facing notification.
* `api/routes/points.go` owns the websocket route and initial points snapshot.
  This route is infrastructure/transport boundary and should continue to use
  `api/framework/realtime` directly.
* Frontend `frontend/src/pages/Experiments.svelte` listens to websocket messages
  and displays local stream logs. Global app websocket handling in
  `frontend/src/App.svelte` adds toast/notification center items for recognized
  realtime messages.

## Decision

Business operations that send WebSocket messages should call a notification
usecase/helper, not `realtime.Publish` directly. The notification boundary owns
both durable and transient delivery through `StorePolicy`:

* `StorePolicyDefault` / `StorePolicyStore` writes `notifications` and then
  publishes realtime.
* `StorePolicyTransient` publishes realtime through the same boundary without
  writing `notifications`.

The low-level realtime package remains available to websocket
route/infrastructure code and to the notification usecase implementation itself.
Refresh-only state sync messages, such as points balance refresh, use
`StorePolicyTransient`.
