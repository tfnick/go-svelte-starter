# Architecture Change: Replace SSE With WebSocket

## Goal

系统中所有使用 SSE 技术的地方全部替换为 WebSocket。迁移完成后，运行时代码、前端交互、API 路径、测试、种子数据和用户可见文案都不再使用 SSE；实时消息仍沿用现有 `realtime` envelope 和用户隔离语义。

## What I Already Know

* 当前后端实时分发核心在 `api/framework/realtime`，它是 user/client subscription hub，消息已经是 JSON envelope。
* 当前 SSE route 有两个：
  * `api/routes/points.go`: `PointsSSE`，挂载在 legacy `/api/points/sse` 与新路径 `/api/user/points/sse`。
  * `api/routes/heavy_task.go`: `UserEventsSSE`，挂载在 `/api/user/events`。
* 当前前端使用 `EventSource` 的地方：
  * `frontend/src/App.svelte`: 全局事件流，处理通知 toast 和任务刷新。
  * `frontend/src/pages/Dashboard.svelte`: 积分实时刷新。
  * `frontend/src/pages/Experiments.svelte`: SSE 实验页。
* 当前 API helper：
  * `eventsSSEURL()`
  * `pointsSSEURL()`
* 当前业务命名/数据也带有 SSE：
  * `api/usecase/notification.go`: `NotificationTypeSSE`、`publishSSENotification`。
  * `api/usecase/heavy_task.go`: `publishTaskSSE`。
  * `api/db/migrations/app/002_seed.sql`: notification type seed `sse` / `SSE`。
  * 多个测试、文档、营销页文案包含 `SSE`。
* 浏览器原生 `WebSocket` 构造器不能设置自定义 `Authorization` header，因此现有 `access_token` query token 认证方式仍有价值，但命名不应再绑定 SSE。

## Requirements

* 新增一个第一方登录用户 WebSocket 实时通道，推荐路径：`GET /api/user/realtime/ws?access_token=<jwt>&client_id=<id>`。
* WebSocket 连接必须复用当前内部登录态解析能力，确保只能订阅当前登录用户自己的实时消息。
* 保留现有 realtime envelope：
  * `type`
  * `presentation`
  * `payload`
* 后端 `realtime.Hub` 继续按 `user_id` 和可选 `client_id` 隔离消息。
* 所有 SSE route、handler、helper、测试和文案都迁移为 WebSocket/realtime 命名。
* 前端移除所有 `EventSource` 使用，统一通过 `WebSocket` 客户端连接实时通道。
* 前端应能自动重连，且登录退出时关闭连接。
* 积分刷新、通知 toast、异步任务完成刷新必须保持现有行为。
* notification type 从 `sse` 迁移到 WebSocket/realtime 命名；新 seed 不再出现 `sse`。
* 如存在旧数据，需要通过 migration 将 notification type `sse` 平滑迁移到新值。
* 文档、spec、PRD、营销文案、测试名中不再把当前系统能力描述为 SSE。

## Acceptance Criteria

* [x] `rg -n "EventSource|text/event-stream|points/sse|SSE|sse"` 不再命中运行时代码、前端源码、测试和当前文档中的 SSE 实现/产品文案；历史归档、本任务 PRD/research、以及迁移旧数据必须保留的 `'sse'` 字面值可作为例外。
* [x] 后端存在 WebSocket route，并通过登录 token 建立当前用户实时订阅。
* [x] WebSocket 连接收到的消息仍符合现有 realtime envelope，前端 `dispatchRealtimeMessage` 可继续处理。
* [x] Dashboard 的 points balance 可通过 WebSocket 消息刷新。
* [x] App 全局通知 toast 和 task center refresh 可通过 WebSocket 消息触发。
* [x] Experiments 页面从 SSE 实验改为 WebSocket/realtime 实验。
* [x] 旧 SSE endpoint 不再注册或不再作为 SSE 返回 `text/event-stream`。
* [x] notification type seed 和 usecase 常量不再使用 `sse`。
* [x] 后端测试覆盖 WebSocket 连接鉴权、用户隔离、消息发送。
* [x] 前端测试覆盖 WebSocket URL/helper、连接生命周期、消息分发。
* [x] `go test ./...` 通过。
* [x] `cd frontend && npm test` 通过。
* [x] `cd frontend && npm run build` 通过。

## Technical Approach

推荐方案：单一用户实时 WebSocket 通道。

```text
Browser/App frontend
  -> ws(s)://<host>/api/user/realtime/ws?access_token=<jwt>&client_id=<optional>
  -> route authenticates user
  -> realtime.SubscribeClient(user.ID, clientID)
  -> write existing realtime envelope JSON frames to WebSocket
```

后端：

* 引入 `github.com/coder/websocket`。
* 在 `api/routes` 中新增 WebSocket handler，例如 `UserRealtimeWebSocket`。
* 连接建立后订阅 `realtime.Hub`，将 `sub.Messages` 逐条写为 WebSocket text/JSON message。
* 用 ping/heartbeat 或写超时检测断线，并在断开时 `sub.Close()`。
* 删除或停止注册 `PointsSSE`、`UserEventsSSE` 及 SSE writer helper。

前端：

* 新增 `realtimeWebSocketURL()` helper。
* 新增轻量 WebSocket client lifecycle，替换 `eventsSSEURL()`、`pointsSSEURL()` 和页面内 `EventSource`。
* Dashboard、App、Experiments 共享同一连接优先；如先做 MVP，也可以每个页面独立连接，但命名必须是 WebSocket/realtime。

数据与命名：

* `NotificationTypeSSE` 改为 `NotificationTypeWebSocket` 或更通用的 `NotificationTypeRealtime`。
* `publishSSENotification` 改为 `publishRealtimeNotification`。
* `publishTaskSSE` 改为 `publishTaskRealtime`。
* seed 中的 `sse` 改为 `websocket` 或 `realtime`。

## Decision (ADR-lite)

**Context**: SSE 只能服务单向 server push，当前需求明确要求系统不再使用 SSE。现有系统已经具备 transport-agnostic 的 realtime hub，迁移重点是替换传输协议和命名，而不是重做业务事件模型。

**Decision**: 使用单一 WebSocket realtime 通道 `/api/user/realtime/ws` 承载当前所有用户级实时消息，保留现有 JSON envelope 和 user/client 隔离。

**Consequences**:

* 前端可以复用一个双向连接承载通知、积分、任务等实时消息。
* 后续如果需要客户端发起订阅/ack/ping，也可以在同一 WebSocket 协议上扩展。
* 需要处理 WebSocket 鉴权、断线重连、关闭握手和测试方式。

## Confirmed Decisions

* WebSocket endpoint 使用 `/api/user/realtime/ws`。
* 前端页面统一复用 `createRealtimeWebSocketClient()` 管理 WebSocket 生命周期，页面只处理 parsed realtime envelope 的业务分发。
* notification channel 的持久化类型命名使用更通用的 `realtime`，避免再次把业务渠道绑定到具体传输技术。

## Implementation Status

* Backend:
  * 新增 `GET /api/user/realtime/ws?access_token=<jwt>&client_id=<id>`，使用 `github.com/coder/websocket`，通过当前登录用户订阅 `realtime.Hub`。
  * 移除 `/api/points/sse`、`/api/user/points/sse`、`/api/user/events` 的 SSE route 注册和 SSE writer helper。
  * notification/task 相关命名从 `SSE` 改为 `Realtime`，保留现有 realtime envelope。
  * 新增 migration `api/db/migrations/app/016_rename_realtime_notification_type.sql`，把历史 `notification_type='sse'` 平滑迁移到 `realtime`。
* Frontend:
  * `realtimeWebSocketURL()` 生成 `ws://` / `wss://` URL，并附加 `access_token` 和可选 `client_id`。
  * `createRealtimeWebSocketClient()` 统一封装 open/message/error/close/disconnect/reconnect 生命周期。
  * App、Dashboard、Experiments 均改用 WebSocket；Experiments tab 和用户可见文案改为 Realtime/WebSocket。
  * Vite `/api` proxy 开启 `ws: true`。
* Specs/docs:
  * `.trellis/spec/backend/*` 与 `.trellis/spec/frontend/svelte-vite-embed.md` 已同步 WebSocket realtime 契约。
  * 营销页文案从 SSE 改为 WebSocket realtime。

## Verification

* `go test ./...`：通过。
* `cd frontend && npm test`：通过，43 个前端测试。
* `cd frontend && npm run build`：通过。
* `git diff --check`：通过，仅有 Windows CRLF 提示。
* 当前运行代码与规格扫描：

```sh
rg -n "EventSource|text/event-stream|pointsSSEURL|eventsSSEURL|points/sse|/api/user/events|/api/points/sse|NotificationTypeSSE|\\bSSE\\b" . --glob "!.git/**" --glob "!frontend/node_modules/**" --glob "!frontend/dist/**"
```

扫描结果无当前运行契约命中；历史任务文档和 migration 里保留旧 `sse` 字面值仅用于描述/迁移旧状态。

## Out of Scope

* 不引入 `/open-api` 第三方 WebSocket。
* 不实现跨实例 WebSocket fanout；当前仍沿用进程内 `realtime.Hub`。
* 不重做 notification center、task center 的业务模型。
* 不引入复杂订阅协议；MVP 连接成功后默认订阅当前用户所有实时消息。

## Definition of Done

* 代码实现完成并通过后端/前端测试。
* 文档与 spec 同步，不再描述当前系统使用 SSE。
* PRD 中记录实际落地路径、剩余风险和验证命令。
* 工作提交完成后再归档任务。

## Research References

* [`research/websocket-library-selection.md`](research/websocket-library-selection.md): 推荐使用 `github.com/coder/websocket`，保留现有 realtime hub，仅替换传输层。

## Technical Notes

* Inspected:
  * `index.go`
  * `api/routes/points.go`
  * `api/routes/heavy_task.go`
  * `api/framework/realtime`
  * `api/usecase/notification.go`
  * `api/usecase/heavy_task.go`
  * `api/framework/http/middleware/auth.go`
  * `frontend/src/api.js`
  * `frontend/src/App.svelte`
  * `frontend/src/pages/Dashboard.svelte`
  * `frontend/src/pages/Experiments.svelte`
  * `go.mod`
* Relevant specs:
  * `.trellis/spec/backend/index.md`
  * `.trellis/spec/backend/route-handler-guidelines.md`
  * `.trellis/spec/backend/api-contracts.md`
  * `.trellis/spec/backend/directory-structure.md`
  * `.trellis/spec/backend/quality-guidelines.md`
  * `.trellis/spec/frontend/index.md`
  * `.trellis/spec/frontend/svelte-vite-embed.md`
