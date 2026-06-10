# 通知中心与任务中心

## Goal

在左下角实现通知中心和任务中心。用户提交重任务时，后端通过消息队列排队异步处理，处理完成后通过 SSE 推送通知到前端通知中心。worker 处理线程数可在 Settings 页面配置。

## What I Already Know

### 用户需求
* 左下角通知中心 + 任务中心 UI
* 重任务走消息队列排队异步处理
* 处理完成后 SSE 推送通知
* worker 线程数在 Settings 可配置

### 代码基础
* **队列系统**：`api/framework/queue/queue.go` — 基于 goqite（SQLite 消息队列），已有 3 个队列：`scheduled-tasks`、`domain-events`、`integration-webhooks`。`JSONRunner` 支持配置并发数（`limit`）、轮询间隔、消息超时扩展。
* **SSE 基础设施**：`api/framework/realtime/realtime.go` — 内存 pub/sub hub，按 userID 分组订阅。已有 `NewNotificationMessage()` 消息类型。目前只有 `/api/user/points/sse` 一个 SSE endpoint。
* **设置存储**：`api/models/setting.go` — `app_settings` 表（key-value JSON），当前仅存 site logo。`UpsertAppSetting(ctx, key, valueJSON)` / `GetAppSetting(ctx, key)`。
* **前端布局**：`App.svelte` → `Header` + `AppSidebar`（daisyUI drawer），左侧 sidebar w-72，主内容区 flex column。
* **通知模型**：`api/models/notification.go` — `notifications` 表已有 `notification_type`、`user_id`、`title`、`summary`、`status` 等字段。
* **realtime 消息分发**：`frontend/src/helpers/realtimeMessages.js` — `dispatchRealtimeMessage()` 已支持 `presentation: toast` 类型，但前端尚无全局 toast 组件显示。
* **Settings 页面**：`frontend/src/pages/Settings.svelte` — daisyUI tabs，目前只有 General（logo 上传）和 Retain（空占位）两个 tab。

## Decision (ADR-lite)

**Context**: 确定任务中心接入范围
**Decision**: 先做通用框架 — 提供可复用的任务入队/状态追踪/SSE 通知基础设施。MVP 接入 1 个端到端验证（如通知导出 toast），后续重任务按相同模式接入。不一次性改造全部已有异步操作。
**Consequences**: 重点是框架的通用性和扩展性，而非覆盖所有场景。

**Context**: 通知中心与任务中心的 UI 关系
**Decision**: 两个独立面板 — 两个图标，各自弹出独立面板，各自独立展开/关闭。
**Consequences**: 每个面板各自维护状态，互不干扰，适合各自信息独立增长的场景。

**Context**: 任务记录是否需要持久化
**Decision**: DB 持久化 — 新增 `async_tasks` 表，记录 task_id、user_id、task_type、status（queued/processing/completed/failed）、payload、result、error_message、created_at、updated_at。MVP 保留全部记录，后续可扩展清理策略（如保留 N 天）。
**Consequences**: 支持刷新后仍可查看任务历史、故障排查；比纯内存多一张表和 model 层代码。

**Context**: worker 线程数是全局统一还是按队列配置
**Decision**: 全局统一 — Settings 中一个 `heavy_task_worker_limit` 配置项，控制 heavy-tasks 队列的 `JSONRunner` limit 参数。默认值 1，可在 Settings UI 修改。
**Consequences**: 简单直观；如果后续需要按队列差异化配置，可扩展为 per-queue 的 JSON 结构。

**Context**: 队列消息处理失败后是否需要最大重试限制
**Decision**: 加入最大重试次数 — `async_tasks` 表记录 `retry_count`，超过阈值（默认 3）后标记 `failed` 不再重试，并推送失败通知。
**Consequences**: 防止死循环重试占满 worker；goqite 超时自动重试 + 业务层 retry_count 双重控制。

## Assumptions (temporary)

* 通知中心的"通知"分为两类：(1) 任务完成/失败的状态通知；(2) 系统其他 SSE 通知
* MVP 接入 1 个具体任务做端到端验证
* 通知中心为全局组件，无论用户在哪个页面都能看到和交互
* 任务中心展示当前用户的异步任务列表及状态

## Open Questions

（暂无）

## Requirements (evolving)

* 左下角通知中心：显示 SSE 推送的系统通知，支持未读标记
* 左下角任务中心：显示当前用户提交的异步任务及处理状态
* 后端：通用任务入队框架 → worker 异步处理 → 完成/失败时 SSE 通知
* Settings：可配置 worker 线程数
* MVP 接入 1 个具体任务做端到端验证
* 任务失败自动重试，超过最大次数（默认 3）标记失败并通过 SSE 通知用户

## Acceptance Criteria (evolving)

* [ ] 通知中心面板在左下角可打开/关闭
* [ ] 任务中心面板在左下角可打开/关闭
* [ ] 用户提交异步任务后，任务中心显示排队/处理中/完成/失败状态
* [ ] 任务完成后前端收到 SSE 通知并在通知中心展示
* [ ] Settings 中可修改 worker 线程数并生效
* [ ] MVP 任务端到端验证通过

## Out of Scope (explicit)

* 批量改造已有异步操作为任务中心模式（后续按需接入）
* 合并 points SSE 到通用 SSE（points SSE 保持现状，后续统一）
* 管理员查看全部用户任务列表（MVP 只看自己的任务）
* 邮件/短信/Webhook 多渠道通知

## Technical Notes

* 队列：复用 `api/framework/queue` 的 `JSONRunner`，新增一个 `heavy-tasks` 队列
* SSE：新增通用 SSE endpoint（如 `GET /api/user/events`），替代仅 points 专用的 endpoint
* 设置：key 用 `heavy_task.worker_limit` 或其他命名，存 JSON number
* 前端：全局组件放在 `App.svelte` 层级，位置固定左下角
