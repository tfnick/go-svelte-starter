# Add Notification Center Clear Action

## Goal

给首页左下角的 `Notifications` 通知中心增加清除功能，操作入口和 `Tasks` 面板保持一致。用户点击清除后，当前左下角通知列表立即清空，并且该用户名下已经产生的 notification 不再从用户侧可见。

## What I Already Know

* 用户希望在首页左下角 `Notifications` 通知中心增加清除按钮，按钮风格与 `Tasks` 面板一致。
* `frontend/src/components/TaskCenter.svelte` 已有 `ArchiveX` 清空按钮、loading 状态、错误提示，以及 `clearMyTasks()` API 调用。
* `frontend/src/components/NotificationCenter.svelte` 当前只有关闭按钮，没有清空按钮，也没有错误/loading 状态。
* `frontend/src/App.svelte` 目前把 WebSocket toast 写入内存数组 `notifications`，再传给 `NotificationCenter`。
* 当前左下角 `Notifications` 面板没有主动从后端拉取 notification 列表；它显示的是当前浏览器会话收到的实时通知。
* 后端已有 admin notification 列表：`GET /api/admin/notifications`。
* 后端已有用户任务清空接口：`POST /api/user/tasks/clear`，语义是对当前用户的终态任务做软清除，不物理删除任务。
* `notifications` 表当前没有 `cleared_at` / `hidden_at` 字段。
* 最近架构决策已经把业务 WebSocket 通知统一收口到 `SendNotification(StorePolicy...)`，因此通知台账会持续增长，用户侧需要一个明确的可见性清理入口。

## Confirmed Decisions

* “清除”表示用户侧不可见，而不是物理删除 notification 审计记录。
* 清除范围是当前登录用户自己的 notification，不影响其他用户。
* 管理端 `/api/admin/notifications` 仍可查看所有 notification 台账，用于审计和排障。
* 当前浏览器内存中的 notification 列表需要立即清空；后端也需要记录清除状态，避免未来用户侧通知列表恢复显示旧通知。

## Requirements

* 在 `NotificationCenter` 头部增加清除 icon button，位置和交互风格与 `TaskCenter` 的清除按钮一致。
* 点击清除时，调用新的当前用户接口，例如 `POST /api/user/notifications/clear`。
* 清除成功后，前端左下角当前通知列表立即变为空。
* 清除成功后，当前用户已存在的 notification 在用户侧视角不再可见。
* 清除操作只能作用于当前登录用户，未登录时返回 unauthorized。
* 清除是软清除：保留 `notifications` 表原始记录和管理端审计能力。
* 新通知到达时，仍然可以继续进入左下角通知中心；清除只影响清除动作发生前已存在或当前面板中的 notification。
* API 返回清除数量，和 Tasks 清空接口风格保持接近。

## Acceptance Criteria

* [x] `NotificationCenter` 有清除按钮，视觉和按钮状态与 `TaskCenter` 保持一致。
* [x] 点击清除按钮时，按钮显示 loading，重复点击被禁用。
* [x] 清除成功后，左下角 notification badge 归零，面板显示 `No notifications`。
* [x] 清除失败时，面板内显示错误提示，当前通知列表不被误清空。
* [x] 后端新增当前用户清除接口，只允许清除 `ctx.Actor.UserID` 下的 notification。
* [x] 后端清除不影响其他用户 notification。
* [x] 后端清除不物理删除 notification 台账，管理端列表仍能查看记录。
* [x] 新增/更新模型、usecase、route、前端 API 测试，并通过前端 build 验证组件编译。
* [x] `go test ./...` 通过。
* [x] `cd frontend && npm test` 和 `cd frontend && npm run build` 通过。

## Definition Of Done

* Tests added/updated for backend clear behavior and frontend API/helper behavior.
* Quality gate green.
* `.trellis/spec/` updated if this task establishes a reusable notification visibility contract.
* Task implementation committed before finish-work/archive.

## Technical Approach

Recommended approach: soft-clear user-visible notifications.

Backend:

* Add a `cleared_at TEXT NOT NULL DEFAULT ''` column to `notifications` via app migration.
* Add model function such as `ClearNotificationsByUser(ctx, userID) (int, error)` that sets `cleared_at` for rows where `user_id = ?` and `cleared_at = ''`.
* Add usecase `ClearMyNotifications(ctx)` that requires authenticated current user and returns `{ cleared_count }`.
* Add route `POST /api/user/notifications/clear`.
* Do not change admin list semantics unless implementation discovers the current list is reused for user-side visibility.

Frontend:

* Add `clearMyNotifications()` in `frontend/src/api.js`.
* Let `App.svelte` own a `clearNotifications()` callback that clears the parent `notifications` state after child clear success.
* Pass callback to `NotificationCenter`.
* Add clear/loading/error UI to `NotificationCenter.svelte` modeled after `TaskCenter.svelte`.
* Update frontend API tests for the new endpoint.

## Decision

**Context**: 当前左下角 notification 面板是实时消息内存列表，但后端已经有 durable notification 台账。仅清空前端内存会让“清除”变成一次性的 UI 操作，无法表达用户侧历史 notification 不再可见。

**Decision**: 使用软清除语义。用户点击清除时，前端清空当前面板；后端为当前用户 notification 设置 `cleared_at`，保留审计记录。

**Consequences**: 用户体验和 `Tasks` 清空一致；数据库保留排障能力；后续如果新增用户侧 notification 历史列表，可以自然过滤 `cleared_at IS NULL`。

## Open Questions

* None.

## Out Of Scope

* 不新增 notification 已读/未读状态。
* 不新增 notification 历史分页面板，除非实现中发现当前左下角必须从后端拉取。
* 不物理删除 notification 记录。
* 不改变 notification realtime 投递机制。

## Technical Notes

* Relevant frontend files:
  * `frontend/src/components/NotificationCenter.svelte`
  * `frontend/src/components/TaskCenter.svelte`
  * `frontend/src/App.svelte`
  * `frontend/src/api.js`
  * `frontend/src/api.test.js`
* Relevant backend files:
  * `api/models/notification.go`
  * `api/usecase/notification.go`
  * `api/routes/notification.go`
  * `index.go`
  * `api/db/migrations/app/`
* Existing analogous task clear path:
  * `POST /api/user/tasks/clear`
  * `api/usecase.ClearMyTasks`
  * `models.ClearTerminalAsyncTasksByUser`
