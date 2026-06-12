# Consolidate Experiment Realtime Notifications

## Goal

收口 `Experiment -> Realtime` 以及其他业务路径里的 WebSocket 发送能力：业务操作不再直接调用 `realtime.Publish`，而是统一调用 notification 相关封装接口。notification 封装负责决定是否写入 `notifications` 表，并负责通过 WebSocket 发送实时消息，让左下角 notification 面板与实时 Toast 行为保持一致。

## What I Already Know

* 用户希望收口 Experiment 的 realtime websocket 发送功能。
* 用户希望所有业务操作通过 notification 封装接口发送，而不是散落调用 websocket/realtime。
* notification 封装需要通过 `StorePolicy` 枚举暴露是否 store 的参数。
* `StorePolicyDefault` / `StorePolicyStore` 时需要写入 `notifications`，用户能在左下角 notification 列表看到。
* `StorePolicyTransient` 时不写入 `notifications`，但仍由 notification 封装统一发送 WebSocket 实时消息。
* 用户应能收到实时 Toast。
* 现有 `api/usecase/notification.go` 已有 `CreateNotification`，会先写 `notifications`，再发布 realtime `notification` 消息。
* 现有 `api/usecase/notifications.go` 的 `TriggerExportToast` 是 Experiment 触发路径，目前直接 `realtime.Publish`，不会落库。
* 现有 `api/usecase/order_export.go` 同时直接发布 `async_export_task` 并调用 `CreateNotification`，职责重复且入口不统一。
* 现有 `api/usecase/heavy_task.go` 的非订单导出任务完成通知直接 `realtime.Publish`，不会落库。
* 现有 `api/usecase/events/order_paid_points.go` 的积分余额刷新直接 `realtime.Publish`，更像状态刷新而不是 durable notification，但也应通过 notification 封装进行发送。
* WebSocket route 和 notification usecase 内部仍需要保留 `api/framework/realtime` 作为底层传输能力。

## Requirements

* 所有业务 WebSocket 发送都必须统一通过 notification 封装接口创建，不能由业务 usecase 直接调用 `realtime.Publish`。
* notification 封装需要通过 `StorePolicy` 枚举暴露“是否持久化”的能力，用于区分 durable notification 与 transient realtime message。
* 默认推荐持久化到 `notifications`，只有积分刷新、局部 UI refresh、实验性 transient toast 等场景显式声明不持久化。
* `StorePolicyDefault` / `StorePolicyStore` 的 notification 必须写入 `notifications` 表，并在左下角 notification 面板可见。
* `StorePolicyTransient` 的 notification 不写入 `notifications` 表，但仍通过统一 notification 封装发送 WebSocket 消息。
* 需要 Toast 的业务消息必须通过同一 notification 封装触发实时 Toast。
* Experiment 的 realtime 测试按钮必须改为调用 notification 封装路径，不能直接发布 websocket 消息。
* 订单导出完成通知不再拆成“直接 websocket + notification ledger”两条路径。
* 非订单导出 heavy task 完成通知如属于用户可见通知，也应通过 notification 封装落库。
* 积分余额刷新等纯状态同步消息可使用 `StorePolicyTransient`，但仍不能直接调用 `realtime.Publish`。
* 保留 WebSocket transport 层的底层 realtime 发布/订阅能力，用于 route、hub 和 notification usecase 内部实现。
* 测试需要覆盖 Experiment 触发路径会创建 notification ledger 并发布 notification realtime message。
* 测试需要覆盖 `StorePolicyTransient` 路径不会创建 notification ledger，但仍会发布统一 realtime message。
* 测试需要覆盖至少一个真实业务路径，避免再次绕过 notification 封装。

## Open Questions

* 已确认使用 `StorePolicy` 枚举表达持久化策略。

## Acceptance Criteria

* [x] `TriggerExportToast` 不再直接调用 `realtime.Publish`，而是调用 notification 封装。
* [x] 点击 Experiment realtime 测试按钮后，数据库中产生一条 `notifications` 记录。
* [x] 点击 Experiment realtime 测试按钮后，前端通过 WebSocket 收到 notification 类型消息并显示 Toast。
* [x] 左下角 notification 面板能看到该通知。
* [x] `StorePolicyTransient` 的业务消息通过 notification 封装发送，且不会产生 `notifications` 记录。
* [x] 订单导出完成通知不再拆成“直接 websocket + notification ledger”两条路径。
* [x] 非订单导出 heavy task 完成通知如属于用户可见通知，也应通过 notification 封装落库。
* [x] 业务层新增或保留的 `realtime.Publish` 调用只出现在 transport/notification 内部等明确边界内，并有说明和测试保护。
* [x] `go test ./...` 通过。
* [x] 涉及前端 API/消息处理时，`cd frontend && npm test` 和 `cd frontend && npm run build` 通过。

## Definition Of Done

* Tests added/updated for backend notification behavior.
* Frontend tests updated if message type/API contract changes.
* Quality gate green.
* `.trellis/spec/` updated if this task establishes a new architecture rule.
* Task implementation committed before finish-work/archive.

## Technical Approach

推荐方案：建立一个业务 notification 封装边界，让业务 usecase 只表达“发送一条业务通知/实时消息”，由 notification 封装负责：

* normalize notification payload。
* 根据 `StorePolicy` 决定是否 insert `notifications` ledger。
* publish realtime notification/message。
* expose enough payload for Toast and notification center。

`api/framework/realtime` 继续作为底层传输机制存在，但只允许在以下边界直接使用：

* websocket route/subscription infrastructure。
* notification usecase 内部发布 realtime notification/message。

实现建议：

* 使用显式 `StorePolicy`，例如 `StorePolicyDefault`、`StorePolicyStore`、`StorePolicyTransient`。
* `StorePolicyDefault` 与 `StorePolicyStore` 都按持久化处理，防止调用方漏传导致重要通知丢失 ledger。
* `StorePolicyTransient` 只用于积分刷新、局部 UI refresh、实验性 transient toast 等不需要通知中心留痕的消息。
* 前端尽量继续消费统一 realtime message；如果要保留 `points`、`async_export_task` 等类型，也应由 notification 封装构造并发送，而不是业务层直发。

## Decision Draft

**Context**: 当前业务代码中直接 websocket 发布与 notification ledger 创建并存，导致 Experiment 测试按钮、异步任务完成等用户可见消息可能绕过 `notifications` 表；积分刷新等状态同步消息也散落依赖底层 realtime。

**Decision**: 业务 WebSocket 消息统一通过 notification usecase/helper 发送，禁止业务路径直接 `realtime.Publish`。notification 封装通过 `StorePolicy` 暴露 store 能力，支持 durable 与 transient 两种发送形态。

**Consequences**: 通知链路更一致，业务入口统一；`StorePolicyDefault` / `StorePolicyStore` 消息可在左下角 notification 和后台通知列表中追溯，`StorePolicyTransient` 消息保持轻量实时投递；需要梳理现有 `async_export_task`、`heavy_task`、`points` 等 realtime message 类型的迁移方式。

## Out Of Scope

* 不重构 WebSocket transport/hub 本身。
* 不改变 `/api/user/realtime/ws` 的连接协议，除非实现中发现必须兼容消息类型。
* 不新增 SMS/email 实际发送能力。
* 不做 notification 已读/未读状态设计。

## Research References

* [`research/current-realtime-notification-state.md`](research/current-realtime-notification-state.md) - 记录当前直接 `realtime.Publish` 调用点和推荐边界。

## Technical Notes

* 相关后端文件：
  * `api/usecase/notification.go`
  * `api/usecase/notifications.go`
  * `api/usecase/order_export.go`
  * `api/usecase/heavy_task.go`
  * `api/usecase/events/order_paid_points.go`
  * `api/routes/points.go`
* 相关前端文件：
  * `frontend/src/pages/Experiments.svelte`
  * `frontend/src/App.svelte`
  * `frontend/src/helpers/realtimeMessages.js`
  * `frontend/src/components/NotificationCenter.svelte`
