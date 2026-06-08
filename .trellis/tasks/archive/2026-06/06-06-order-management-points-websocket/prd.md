# Order Management UI with Points WebSocket

## Goal

在现有 Svelte 后台界面中实现订单管理能力：用户可以查看订单列表、创建订单、将订单标记为支付成功。支付成功时，后端必须利用 `api/framework/events` 的 `sync_tx` EventBus 能力，在同一个 usecase 事务中给用户积分账户发放积分；页面通过 WebSocket 显示用户最新积分。

## What I Already Know

* 当前已有订单后端能力：
  * `POST /api/orders` 创建订单。
  * `GET /api/orders/user/:user_id` 查询用户订单。
  * `GET /api/orders/:id` 查询订单详情。
  * `PATCH /api/orders/:id/status` 更新订单状态。
* 当前订单状态包括 `pending`, `paid`, `shipped`, `completed`, `cancelled`。
* 当前订单创建会扣减 `shared` DB 中的 product stock，并在 `app` DB 中创建 `orders` / `order_items`。
* 当前 EventBus 已支持：
  * `PublishAsync`：异步 best-effort。
  * `PublishInTx`：事务内同步 handler，失败则主事务回滚。
* 当前代码里还没有真实积分账户表、积分流水表或 WebSocket 基础设施。
* 当前前端已有 `Dashboard.svelte`，能查询用户订单，但不是完整订单管理页面。
* 当前前端 API client `frontend/src/api.js` 已支持内部 API envelope unwrap。

## Requirements

* 后端新增用户积分账户能力：
  * 能查询用户当前积分余额。
  * 能在支付成功时增加积分。
  * MVP 积分规则固定为每个订单支付成功发放 `10` 积分。
  * 积分发放必须可幂等，避免同一个订单重复支付导致重复发积分。
* 后端新增支付成功 usecase：
  * 从 `pending` 更新为 `paid`。
  * 在同一个 `fwusecase.WithAppTx` 事务中发布 `order.paid` 事件。
  * `order.paid` 的 `sync_tx` 订阅者同步发放积分。
  * 任一同步积分逻辑失败时，订单支付状态和积分变更一起回滚。
* 后端新增或调整 API：
  * 支付成功接口建议为 `POST /api/orders/:id/pay`，而不是让前端直接 PATCH 任意 status。
  * 积分余额查询接口。
  * WebSocket 接口，用于推送用户最新积分。
* 前端实现订单管理界面：
  * 显示当前用户订单列表。
  * 支持下订单。
  * 支持对 pending 订单触发“支付成功”。
  * 显示用户最新积分。
  * WebSocket 收到积分更新后，界面立即刷新积分显示。
* 前端订单 API helper 应集中在 `frontend/src/api.js` 或合理拆分后的 API module 中。
* 新增功能必须遵守 DTO 边界：返回给前端的 API 不能直接暴露 `models.*`。
* 新增事件必须使用 `api/framework/events`，不能直接依赖 raw EventBus，不能引入 outbox 或事件持久化表。

## Acceptance Criteria

* [ ] 登录后可以在界面看到订单列表。
* [ ] 登录后可以在界面创建订单。
* [ ] pending 订单可以在界面触发支付成功。
* [ ] 支付成功后订单状态变为 `paid`。
* [ ] 支付成功后用户积分账户同步增加积分。
* [ ] 如果积分发放失败，订单状态不会变成 `paid`。
* [ ] 同一个订单重复支付不会重复发放积分。
* [ ] 页面通过 WebSocket 展示用户最新积分；支付成功后无需手动刷新即可看到积分变化。
* [ ] 新增 API 响应符合内部 `{success,data}` / `{success,error}` envelope。
* [ ] 新增前端/后端测试覆盖核心流程。

## Definition of Done

* PRD 经确认并启动任务。
* 后端 migration、model、usecase、route、EventBus subscriber、WebSocket handler 完成。
* 前端订单管理 UI 和积分 WebSocket 展示完成。
* 运行并通过：
  * `go test ./...`
  * `cd frontend && npm test`
  * `cd frontend && npm run build`
* 必要时更新 `.trellis/spec/`。
* 提交本任务代码。

## Technical Approach

### Backend

* 新增 app migration：
  * `point_accounts`：`user_id`, `balance`, timestamps。
  * `point_transactions`：`id`, `user_id`, `order_id`, `points`, `type`, `created_at`，并对 `order_id/type` 建唯一约束用于幂等。
* 新增 `api/models/points.go`：
  * 查询积分余额。
  * 事务内创建账户或更新余额。
  * 插入积分流水，重复流水时保持幂等。
* 新增或扩展 `api/usecase/points.go`：
  * `GetUserPoints`。
  * `AwardPointsForPaidOrder`，供 EventBus sync handler 调用。
* 扩展订单 usecase：
  * 新增 `PayOrder(ctx, PayOrderCmd)`。
  * 只允许 pending 订单支付。
  * 在 `WithAppTx` 内更新订单为 paid，并 `events.PublishInTx(txCtx, OrderPaidEvent(...))`。
* EventBus：
  * 定义 `order.paid` topic 和 payload。
  * 注册 `points.award_on_order_paid` sync subscriber。
  * 注册位置需要在应用启动阶段完成。
* WebSocket：
  * 使用 Go 标准或 Echo 兼容方式新增 `/api/points/ws`。
  * 连接应基于当前登录用户，只推送当前用户积分。
  * 支付成功后向对应用户连接推送最新积分。

### Frontend

* 整理 Dashboard 或新增订单管理页面。
* API helper 新增：
  * `createOrder(payload)`
  * `payOrder(orderId)`
  * `getUserPoints(userId)` 或当前用户积分接口。
  * `connectPointsSocket(...)`。
* UI 需要包含：
  * 积分显示区。
  * 创建订单表单。
  * 订单列表。
  * pending 订单支付按钮。
  * 成功/错误提示。

## Decision (ADR-lite)

**Context**: 支付成功送积分属于不允许丢失的业务动作，应与订单支付状态保持一致。

**Decision**: 使用 EventBus `sync_tx`，在 `PayOrder` 的 app transaction 内发布 `order.paid`，由积分模块同步订阅并写入积分账户。WebSocket 只负责把最新积分推送给页面，不作为持久化依据。

**Consequences**: 订单支付和积分发放具备事务一致性；WebSocket 断开时不影响业务提交，页面可通过重新查询积分恢复状态。

## Out of Scope

* 不接入真实支付网关。
* 不实现支付失败、退款、取消支付。
* 不实现复杂积分规则配置，MVP 固定为每个订单支付成功发放 `10` 积分。
* 不实现 WebSocket 集群广播或跨进程连接管理。
* 不引入 outbox、消息队列或事件持久化表。

## Technical Notes

* 当前工作区仍有运行时 DB 文件未提交：`data/app.db-shm`、`data/app.db-wal`，本任务不得纳入提交。
* 当前 `Dashboard.svelte` 和部分前端文案存在历史乱码，订单管理 UI 实现时应一并整理相关页面文案。
* 当前 `frontend/src/api.js` 仍是单一 API client 文件；如果订单/积分 helper 变多，可以考虑拆分，但 MVP 可先保留。

## Open Questions

* None.

## Completion Record

### Implementation Summary

* 新增 `point_accounts` 与 `point_transactions` 业务表；`point_transactions` 通过 `UNIQUE(order_id, type)` 保证同一订单支付送积分幂等。
* 新增 `PayOrder` usecase；支付入口在 `WithAppTx` 内把订单置为 `paid`，并通过 EventBus `sync_tx` 发布 `order.paid`。
* 新增 `points.award_on_order_paid` 同步订阅器；订阅器在同一事务内写积分账户与流水，失败则订单支付状态一起回滚。
* 积分 WebSocket 推送通过 `RegisterAfterCommit` 执行，避免事务未提交时前端提前看到积分变化。
* 新增 `/api/orders/:id/pay`、`/api/points/me`、`/api/points/ws`、`/api/products`。
* 重构 Dashboard 为订单管理页面，支持商品下单、订单列表、支付成功和积分实时显示。
* 更新 `frontend/src/api.js` helper、`frontend/vite.config.js` WebSocket proxy、生产 `frontend/dist`。
* 更新 `.trellis/spec/backend/api-contracts.md`、`.trellis/spec/backend/eventing-guidelines.md`、`.trellis/spec/frontend/svelte-vite-embed.md`。

### Verification

* `go test ./...` 通过。
* `cd frontend && npm test` 通过。
* `cd frontend && npm run build` 通过。
* 使用临时数据库启动生产构建服务后，浏览器 smoke check 通过：首页可见 `订单列表`、`下订单`、`积分余额`，未登录态控件正确禁用。
