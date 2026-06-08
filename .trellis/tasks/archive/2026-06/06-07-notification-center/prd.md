# Notification Center

## Goal

先澄清并实现“通知中心”的业务定位：通知中心不是渠道 provider adapter，也不是新的外部渠道防腐层；项目已经通过 `api/usecase/integrations/<scenario>` ports、`api/integrations/<scenario>/<provider>` adapters、`integration_channels`、`integration_credentials` 提供了渠道配置和防腐边界。

通知中心应作为业务侧的通知编排与通知台账：接收业务通知意图，形成统一的内部通知记录/任务，保存接收人快照、通知类型、业务来源、内容摘要、状态与投递结果，并在需要真实发送时调用既有渠道 port，而不是直接理解 provider DTO 或 SDK。

## What I Already Know

* 用户明确要求创建通知中心任务。
* 通知中心需要统一管理用户通知，并支持分页查询。
* 查询至少支持按通知类型、用户邮箱、手机号筛选。
* 通知类型包括 `sse`、`sms`、`email`、微信公众号消息等。
* 通知类型必须纳入 Dictionary 管理。
* 用户补充了关键架构约束：系统已经提供渠道 port 和渠道防腐，通知中心需要先澄清定位，不能重复建设渠道层。
* 现有外部集成规范要求：
  * `api/usecase/integrations/<scenario>` 定义业务稳定 port 和 DTO。
  * `api/integrations/<scenario>/<provider>` 实现 provider adapter，只做 provider DTO mapping、HTTP/SDK 调用和 provider error normalization。
  * `api/usecase` 通过 registry/bootstrap 使用 adapter，不直接导入 `api/integrations`。
  * 渠道配置和凭证通过 DB 管理，例如 `integration_channels`、`integration_credentials`、`integration_operation_configs`。
* 现有项目已有 `api/framework/realtime` 和 `/api/points/sse`，SSE 消息使用统一 realtime envelope。
* 现有 `api/usecase/notifications.go` 只有测试导出 toast 的实时发送能力，没有持久化通知台账。
* 现有 Dictionary 模块可复用来维护 `notification_type`。

## Positioning

### Notification Center Owns

* 通知业务意图的统一入口，例如“给某用户发送一条订单通知/系统通知/营销通知”。
* 通知记录与查询，包括通知类型、业务来源、用户、邮箱/手机号快照、标题、摘要、payload 安全快照、状态、失败原因、创建/发送时间。
* 跨渠道的业务层状态，例如 `pending`、`sent`、`failed`、`skipped`。
* 管理端分页查询和筛选。
* 通知类型字典 `notification_type`，用于展示、筛选和校验。
* 未来可以拥有业务策略，例如是否生成站内/SSE 通知、是否同时走 SMS/Email、是否允许人工重发。

### Notification Center Does Not Own

* 不实现 SMS、Email、微信公众号 provider SDK/HTTP 调用。
* 不保存 provider secret，不绕过 `integration_credentials`。
* 不直接接收或返回 provider-specific DTO。
* 不决定 provider adapter 内部的签名、鉴权、回调解析、防腐映射。
* 不替代 `integration_invocations` / callback receipt 这类外部调用技术台账。

### Relationship To Channel Ports

Notification Center 处在业务 usecase 层。它可以调用未来的 `sms`、`email`、`wechat` 等场景 port，也可以发布 SSE/realtime 消息，但它只传递业务稳定 DTO。具体 provider 选择、渠道配置、凭证、provider error normalization 仍由现有 integration boundary 负责。

推荐调用方向：

```text
business feature or domain event subscriber
  -> notification center usecase
  -> notification record / delivery task in app DB
  -> scenario port, e.g. sms/email/wechat/realtime
  -> provider adapter under api/integrations/<scenario>/<provider>
```

### Calling Rule

业务场景如果表达的是“通知某个用户/一批用户”，默认调用 Notification Center，而不是直接调用 SMS/SSE/Email/WeChat port。Notification Center 负责记录通知、应用通知策略、决定或承载 channel 选择，并通过稳定 port 发起投递。

业务场景可以通过两种方式进入 Notification Center：

* 同步调用：业务 usecase 在当前流程中直接调用通知中心 usecase，例如完成某个业务动作后立即创建 SSE 通知。
* 异步调用：领域事件 subscriber 监听业务事件后调用通知中心 usecase，例如 `order.paid` 事件被处理时创建通知。

业务场景只有在以下情况才直接调用 port：

* 它不是用户通知，而是纯技术实时同步，例如页面状态刷新、短生命周期进度流、内部调试消息。
* 它是渠道能力本身的管理/测试/探活，例如参数页测试某个 SMS channel 是否可用。
* 它位于 Notification Center 内部实现中，正在执行通知投递。

判断口径：如果这条消息未来需要在通知中心里查询、审计、重发、按类型统计、受通知策略控制，就必须先进入 Notification Center；如果只是当前连接的一次性 UI 状态推送，可以直接走 SSE/realtime primitive 或对应 scenario port。

## Assumptions

* 第一版优先建设“通知台账 + 管理端查询 + 类型字典”，为后续真实渠道发送和重发策略留接口。
* 第一版不新增真实 SMS/Email/微信公众号 provider adapter；这些属于各自 integration scenario 的后续任务。
* 手机号不是当前 `users` 表字段；通知记录保存 `recipient_phone` 快照，便于查询 SMS/公众号类通知。
* 邮箱筛选使用通知记录中的 `recipient_email` 快照；同时保留 `user_id` 关联当前用户。
* 邮箱和手机号筛选默认使用包含匹配，便于管理端搜索。

## Open Questions

* None. Requirements are ready for final confirmation.

## Requirements

* 新增 app DB 通知记录表，保存业务侧通知记录。
* 通知记录 ID 使用 UUID v7。
* 通知记录至少包含：
  * `id`
  * `notification_type`
  * `source_type` / `source_id` 或等价业务来源字段
  * `user_id`
  * `recipient_email`
  * `recipient_phone`
  * `title`
  * `summary`
  * `payload_json`
  * `status`
  * `last_error`
  * `sent_at`
  * `created_at`
  * `updated_at`
* 通知状态第一版只支持：
  * `pending`
  * `sent`
  * `failed`
  * `skipped`
* 通知状态不纳入 Dictionary 管理；状态是 Notification Center 的代码拥有状态机，应使用 Go constants + DB `CHECK` constraint 校验。
* 前端展示状态可以使用 route DTO 返回的 `status`，必要时由前端或后端固定映射 label；管理端不能新增、禁用或删除通知状态。
* 第一版不实现 `read` / `unread` / `retrying` 等用户收件箱或重试编排状态。
* 新增 `notification_type` dictionary type，并 seed 至少以下 enabled values：
  * `sse`
  * `sms`
  * `email`
  * `wechat_official_account`
* 类型筛选必须校验 against enabled `notification_type` dictionary values。
* 新增内部 API：`GET /api/notifications?page=&page_size=&type=&email=&phone=`。
* 第一版不提供管理端 `POST /api/notifications` 创建接口；创建通知只暴露为后端 usecase，供业务场景调用。
* 创建通知 usecase 可以被业务 usecase 同步调用，也可以被领域事件 subscriber 异步调用。
* 列表 API 返回统一 envelope，包含 `items` 和 `pagination`。
* Notification DTO 不直接暴露 model；应返回 route-local DTO。
* DTO 应包含通知类型 code 和 label，避免前端自行拼接字典显示。
* 前端新增 `/notifications` 页面和菜单入口，提供类型下拉、邮箱输入、手机号输入、分页表格和空状态。
* `/notifications` 页面和 `GET /api/notifications` 只允许 admin 访问。
* 类型下拉通过 dictionary lookup 加载 `notification_type`，不在前端写死。
* 页面错误提示使用现有 API client safe message。
* 第一版采用 Ledger + SSE Slice：在通知台账基础上，增加最小 SSE 投递链路，复用现有 realtime hub。
* 新增业务入口 usecase：创建通知时先写入通知记录，再在 `notification_type=sse` 时发送 realtime/SSE 消息。
* `CreateNotification` command 第一版字段保持轻量：
  * `user_id`
  * `notification_type`
  * `source_type`
  * `source_id`
  * `recipient_email`
  * `recipient_phone`
  * `title`
  * `summary`
  * `payload_json`
* SSE slice 新增统一 realtime message type `notification`，默认 `presentation=toast`。
* SSE `notification` payload 只包含安全展示字段：
  * `id`
  * `title`
  * `summary`
  * `source_type`
  * `source_id`
* SSE payload 不推送完整 `payload_json`；完整扩展数据只保存在通知台账中。
* SSE 投递成功时通知记录状态更新为 `sent`；投递失败时更新为 `failed` 并保存 safe `last_error`。
* 非 `sse` 类型第一版只落台账，状态记为 `skipped`，不真实调用 SMS/Email/WeChat provider。

## Acceptance Criteria

* [ ] Fresh app DB migrate 后包含通知记录表、必要索引和 `notification_type` 字典 seed。
* [ ] 通知状态通过 code-owned constants 和 DB `CHECK` constraint 约束，不创建 `notification_status` 字典。
* [ ] `GET /api/notifications` 支持分页，并按 `type`、`email`、`phone` 单独或组合筛选。
* [ ] 不存在管理端创建通知 HTTP API；业务代码通过后端 usecase 创建通知。
* [ ] 通知创建 usecase 可被普通业务 usecase 和领域事件 subscriber 复用。
* [ ] 非 admin 访问 `GET /api/notifications` 会返回 forbidden safe message。
* [ ] 非 admin 用户在前端菜单中不可见 `/notifications`。
* [ ] 禁用或不存在的通知类型作为筛选条件时返回 validation safe message。
* [ ] 列表 DTO 包含通知类型 code 和 label。
* [ ] 前端 `/notifications` 页面可通过字典下拉选择类型，并展示分页结果。
* [ ] 实现中不新增 provider SDK 调用，不绕过现有 integration port/adapter 边界。
* [ ] 创建 `sse` 通知时先写通知记录，再通过现有 realtime hub 投递给目标用户。
* [ ] SSE 投递使用 realtime message type `notification`，默认 `presentation=toast`。
* [ ] SSE `notification` payload 只包含 `id/title/summary/source_type/source_id`，不包含完整 `payload_json`。
* [ ] `sse` 通知投递结果会反映到通知记录状态。
* [ ] 第一版 `sms`、`email`、`wechat_official_account` 通知只进入台账，不真实发送 provider 请求，状态为 `skipped`。
* [ ] Go tests 覆盖 model/usecase/route 的分页、筛选和类型校验。
* [ ] Go tests 覆盖 SSE 通知创建、记录状态和 realtime message 投递。
* [ ] Frontend tests 覆盖 API helper path、query 参数和 router/menu。

## Definition of Done

* Run `go test ./...`.
* Run `cd frontend && npm test`.
* Run `cd frontend && npm run build` if a Svelte page is implemented.
* Update `.trellis/spec/` if notification center boundary becomes a durable project rule.
* Commit implementation changes before archiving the task.

## Feasible MVP Approaches

### Approach A: Notification Ledger First

只实现通知记录表、字典、查询 API 和管理端页面。业务发送入口和真实投递留到后续任务。

Pros:
* 最小化和现有渠道 port 的耦合风险。
* 快速建立统一查询能力。
* 不会在通知中心里误放 provider 逻辑。

Cons:
* 第一版不能证明“业务通知 -> 投递”的闭环。

### Approach B: Ledger + SSE Delivery Slice

在 Approach A 基础上，增加一个最小 `CreateNotification` usecase：写通知记录，并对 `notification_type=sse` 调用现有 realtime hub 发送。SMS/Email/公众号仍只记录，不真实发送。

Pros:
* 能验证通知中心作为业务编排入口的形状。
* 复用现有 realtime/SSE，无需新增外部 provider adapter。
* 给后续 SMS/Email/公众号 port 接入留下清晰模式。

Cons:
* 范围比纯查询略大，需要定义 create command、状态流转和错误处理。

### Approach C: Full Notification Orchestration

第一版同时定义 SMS/Email/微信公众号 ports，并打通发送、失败记录、重试或人工重发。

Pros:
* 一次性覆盖完整业务闭环。

Cons:
* 容易和已有 integration channel/credential/callback 架构交叉，任务过大；真实 provider adapter 也会拉高复杂度。

Selected: Approach B。它既避免重建渠道防腐，又能验证通知中心“业务编排 + 台账”的核心定位。

## Technical Approach

* DB:
  * Add `notifications` table to app baseline.
  * Add `CHECK (status IN ('pending', 'sent', 'failed', 'skipped'))`; do not seed a `notification_status` dictionary.
  * Add indexes for `(notification_type, created_at)`, `(recipient_email, created_at)`, `(recipient_phone, created_at)`, and `(user_id, created_at)`.
  * Seed `notification_type` dictionary rows in app seed SQL.
* Backend:
  * Add `api/models/notification.go` with dynamic SQL list/count helpers.
  * Add `api/usecase/notification.go` with pagination normalization and dictionary type validation.
  * Add a create/send usecase that writes notification record first and sends only the SSE/realtime slice through existing `api/framework/realtime`.
  * Define `CreateNotificationCmd` with `user_id`、`notification_type`、`source_type`、`source_id`、`recipient_email`、`recipient_phone`、`title`、`summary`、`payload_json`.
  * Add `notification` realtime message type/payload in `api/framework/realtime`.
  * Add `api/routes/notification.go` with route-local query parsing and DTO mapping for list only.
  * Register list route behind admin auth.
  * Keep notification creation as usecase-only API; no create HTTP route in this task.
* Frontend:
  * Add API helper `listNotifications(query)`.
  * Add `/notifications` route and menu entry.
  * Mark `/notifications` as admin-only.
  * Build a dense management page with filters and paginated table, matching existing admin/tool pages.

## Decision (ADR-lite)

**Context**: The project already has channel ports and provider anti-corruption boundaries. A notification center that directly sends SMS/Email/WeChat through provider SDKs would duplicate and weaken that architecture.

**Decision**: Position Notification Center as a business notification orchestration and ledger boundary. It owns notification intent, record, query, and user-facing/management state; channel ports own provider delivery.

**Consequences**: Notification records and queries are unified across channels, while provider-specific behavior remains behind existing integration boundaries. Some future work is required to add real SMS/Email/WeChat ports/adapters, but the notification center does not need to change its query contract when those arrive.

### MVP Delivery Slice

**Context**: A pure ledger would clarify query/storage but would not prove the business flow from notification intent to delivery. Full SMS/Email/WeChat delivery would be too broad and would blur the channel anti-corruption boundary.

**Decision**: First implementation uses Ledger + SSE Slice. `sse` notifications are persisted and then delivered through the existing realtime hub. `sms`、`email`、`wechat_official_account` records are persisted for ledger/query only and do not call providers in this task.

**Consequences**: The MVP validates the notification center orchestration shape without introducing new provider adapters. Later channel ports can plug into the same create/send usecase and record status model.

## Out of Scope

* Real SMS provider sending.
* Real email provider sending.
* Real WeChat official account API integration.
* Provider credential management beyond existing Parameter/integration management.
* Management HTTP API for manually creating notifications.
* Raw provider request/response storage in notification records.
* Automatic cross-channel failover.
* Retry queue or manual resend workflow unless explicitly included later.
* User-side read/unread inbox UX unless explicitly included.

## Technical Notes

* Existing realtime code: `api/framework/realtime/realtime.go`.
* Existing SSE route: `api/routes/points.go`.
* Existing temporary notification usecase: `api/usecase/notifications.go`.
* Existing integration boundary spec: `.trellis/spec/backend/directory-structure.md`, Scenario: External Integration Boundary.
* Existing external integration design task: `.trellis/tasks/06-06-external-integration-anticorruption-layer/prd.md`.
* Existing dictionary model/usecase/routes/frontend page can be reused for `notification_type`.
* Existing pagination examples: user management, domain events, orders.
* Existing frontend router/menu: `frontend/src/router.js`.
