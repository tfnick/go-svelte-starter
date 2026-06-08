# Order List Pagination

## Goal

为订单列表增加分页能力，并在后端形成可复用的标准分页查询规范。前端订单管理页需要使用 daisyUI 分页组件风格，并在移动端与桌面端都保持可用、可读、可操作。

## What I Already Know

* 用户希望为订单列表实现分页。
* 前端需要考虑使用 daisyUI 的分页组件，并支持自适应布局。
* 后端需要形成标准分页查询规范，而不是只为订单接口临时拼一个参数。
* 当前订单列表接口是 `GET /api/orders/user/:user_id`，由 `api/routes/order.go` 的 `GetUserOrders` 处理。
* 当前 usecase 查询对象是 `UserOrdersQry{UserID string}`，返回 `[]OrderCo`。
* 当前 model 查询是 `SELECT * FROM orders WHERE user_id = ? ORDER BY created_at DESC`，一次返回用户所有订单。
* 当前前端 API helper 是 `frontend/src/api.js` 的 `getUserOrders(userId)`，直接返回数组。
* 当前订单管理 UI 在 `frontend/src/pages/Dashboard.svelte`，`loadOrders()` 直接加载全部订单并渲染表格。
* 项目已使用 Tailwind/daisyUI，`frontend/package.json` 中 daisyUI 版本为 `^4.12.24`。
* daisyUI 分页官方文档使用 `join` 分组 `btn/join-item`，活动项使用 `btn-active`。

## Assumptions

* 使用页码分页，页码从 1 开始，便于和 daisyUI 按钮式分页 UI 对齐。
* 后端查询参数使用 `page` 与 `page_size`。
* 默认 `page_size` 为 10，最大 `page_size` 为 50。
* 后端响应使用统一分页结构：`items` 承载当前页数据，`pagination` 承载分页元信息。
* 订单排序使用稳定排序：`created_at DESC, id DESC`。
* 本任务只分页订单列表，不引入搜索、筛选、排序切换或无限滚动。

## Requirements

* 后端订单列表接口支持 `page` 和 `page_size` 查询参数。
* 后端对分页参数进行标准化与校验：缺省值、正整数要求、最大页大小限制。
* 后端响应返回当前页数据和分页元信息，包括当前页、页大小、总条数、总页数、是否有上一页、是否有下一页。
* 后端分页规范应沉淀为可复用代码和项目规范，后续列表接口可以复用。
* 订单查询需要使用 `LIMIT/OFFSET`，并单独查询总数。
* 前端 `getUserOrders` 支持传入分页参数。
* Dashboard 订单管理页维护当前页状态，翻页时重新请求后端。
* 创建订单后回到第一页并刷新订单列表。
* 支付订单后刷新当前页，避免状态与后端不一致。
* 分页控件使用 daisyUI `join` + `btn`/`join-item` 组合，并体现当前页、上一页、下一页和禁用状态。
* 分页区域在小屏上可换行/横向容纳，在桌面端和表格信息并排展示。

## Acceptance Criteria

* [x] `GET /api/orders/user/:user_id?page=1&page_size=10` 返回分页对象，而不是无元信息数组。
* [x] 非法分页参数返回现有错误响应风格的校验错误。
* [x] 后端查询不会一次性加载用户所有订单。
* [x] 前端订单列表能在多页数据间切换。
* [x] 前端分页按钮具有当前页高亮、边界禁用、加载禁用状态。
* [x] 移动端宽度下分页区域不遮挡、不溢出主要内容。
* [x] 相关后端/前端测试更新或新增。
* [x] 质量检查通过，包括 Go 测试与前端测试/构建。

## Definition of Done

* Tests added/updated where behavior changes.
* Lint/typecheck/build relevant to changed layers passes.
* Backend pagination contract is documented in `.trellis/spec`.
* Existing unrelated database working-tree changes are not modified or reverted.

## Out of Scope

* 订单搜索、筛选、排序切换。
* Cursor pagination / keyset pagination。
* 全站所有列表接口迁移到分页。
* 移动端卡片式订单列表重构。

## Technical Notes

* Likely backend files:
  * `api/routes/order.go`
  * `api/usecase/order.go`
  * `api/models/order.go`
  * `api/framework/usecase/` or nearby framework package for shared pagination types.
  * `.trellis/spec/backend/api-contracts.md` for the standard contract.
* Likely frontend files:
  * `frontend/src/api.js`
  * `frontend/src/pages/Dashboard.svelte`
* Existing response envelope comes from `api/framework/http/response/httpresponse.go`.
* DaisyUI reference: https://daisyui.com/components/pagination/

## Open Questions

* 是否确认采用推荐分页契约：`page` 从 1 开始，`page_size` 默认 10、最大 50，响应为 `{ items, pagination }`？
