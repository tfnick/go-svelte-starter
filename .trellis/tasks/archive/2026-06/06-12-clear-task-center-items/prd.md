# Clear Task Center Items

## Goal

在左下角 `Tasks` 面板增加一个清空图标。用户点击后，当前任务中心里已有的任务以后不再出现在该用户的 `Tasks` 面板中，行为类似“归档/隐藏”，而不是删除任务执行记录或取消正在执行的任务。

## What I Already Know

* 当前任务中心组件位于 `frontend/src/components/TaskCenter.svelte`，面板顶部已有刷新和关闭图标。
* 前端通过 `listMyTasks()` 调用 `GET /api/user/tasks` 加载任务，通过 `GET /api/user/tasks/:id/download` 下载已完成的订单导出文件。
* 后端任务列表在 `api/usecase/heavy_task.go` 的 `ListMyTasks`，数据来自 `api/models/async_task.go` 的 `async_tasks` 表。
* `async_tasks` 当前只有执行状态：`queued`、`processing`、`completed`、`failed`，没有面向用户可见性的 `cleared/archived` 字段。
* 任务记录承载执行历史、下载结果、失败原因和队列处理状态；本需求不应物理删除任务记录，否则会影响审计和后续排查。

## Requirements

* 在 `Tasks` 面板标题栏增加一个清空图标按钮，使用图标按钮而不是文字按钮。
* 点击清空后，只将当前用户已经进入终态的任务标记为“已清空/已归档/已隐藏”。
* 终态任务定义为 `completed` 和 `failed`。
* 清空后的任务不再出现在该用户后续打开的 `Tasks` 面板中。
* 清空操作只影响当前登录用户自己的任务，不能影响其他用户。
* 清空不处理正在排队或执行中的任务；`queued` 和 `processing` 任务仍保留在任务中心。
* 新创建的任务不受历史清空操作影响，应继续出现在 `Tasks` 面板。
* `GET /api/user/tasks` 默认只返回未清空的任务。
* 清空操作应通过第一方用户 API 暴露，推荐路径：`POST /api/user/tasks/clear`。
* 前端调用清空成功后，应立即刷新/置空任务列表，并保留失败错误提示。

## Acceptance Criteria

* [ ] `Tasks` 面板顶部出现清空图标，并有清晰的 `aria-label`。
* [ ] 用户点击清空后，当前面板中的 `completed` / `failed` 任务从列表消失。
* [ ] 用户点击清空后，当前面板中的 `queued` / `processing` 任务仍然保留。
* [ ] 重新打开 `Tasks` 面板或刷新页面后，被清空的任务仍不再出现。
* [ ] 清空只作用于当前登录用户，后端必须使用 `ctx.Actor.UserID` 约束。
* [ ] 清空不改变任务执行状态语义；只标记已经处于 `completed/failed` 的任务。
* [ ] 清空后创建的新任务仍能出现在任务中心。
* [ ] 后端测试覆盖当前用户只清空终态任务、跨用户隔离、列表过滤。
* [ ] 前端 API helper 测试覆盖 `POST /api/user/tasks/clear`。
* [ ] `go test ./...`、`cd frontend && npm test`、`cd frontend && npm run build`、`git diff --check` 通过。

## Technical Approach

推荐实现为“可见性维度”，避免复用执行状态字段：

* 数据库：给 `async_tasks` 增加 `cleared_at TEXT NOT NULL DEFAULT ''` 或等价字段，并为 `user_id, cleared_at, created_at` 增加查询索引。
* Model：新增批量清空当前用户终态任务的方法，例如 `ClearCompletedAsyncTasksByUser(ctx, userID)`；`ListAsyncTasksByUser` 和 `CountAsyncTasksByUser` 默认过滤 `cleared_at = ''`。
* Usecase：新增 `ClearMyTasks(ctx)`，从 `ctx.Actor.UserID` 获取 subject，禁止从请求体传 user id。
* Route：新增 `POST /api/user/tasks/clear`，返回清空数量或简单成功响应。
* Frontend API：新增 `clearMyTasks()` helper。
* TaskCenter：标题栏增加 `Trash2` 或 `Archive` 图标；点击后调用 `clearMyTasks()`，成功后清空本地列表或刷新列表。

## Decision (ADR-lite)

**Context**: `async_tasks.status` 表示执行生命周期；用户点击“清空”是任务中心展示层行为，不等价于执行状态变化。同时用户希望只清空已经成功或失败的终态任务，避免隐藏仍在执行的任务。

**Decision**: 不复用 `status = archived/cleared`。新增独立的可见性字段，让任务执行状态继续只表达队列/执行结果；清空操作只标记 `completed/failed`，列表默认隐藏已清空任务。

**Consequences**: 查询和迁移需要小幅扩展，但执行状态语义更干净；正在执行的任务不会被误藏；未来如果需要任务历史页或管理员排查，仍可看到完整记录。

## Open Questions

* None.

## Out of Scope

* 不实现单条任务归档。
* 不实现撤销清空。
* 不物理删除 `async_tasks` 记录。
* 不新增管理员任务历史管理页。
* 不改变订单导出任务的下载鉴权规则。

## Technical Notes

* Likely backend files:
  * `api/db/migrations/app/*.sql`
  * `api/models/async_task.go`
  * `api/usecase/heavy_task.go`
  * `api/routes/heavy_task.go`
  * route registration in `index.go`
* Likely frontend files:
  * `frontend/src/api.js`
  * `frontend/src/api.test.js`
  * `frontend/src/components/TaskCenter.svelte`
* Existing related contracts:
  * `GET /api/user/tasks`
  * `GET /api/user/tasks/:id/download`
  * `POST /api/user/orders/export`
  * `POST /api/admin/orders/export`
