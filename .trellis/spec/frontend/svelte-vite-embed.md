# Svelte Vite Embed

> 本文是 Svelte/Vite 前端、Go embed、前端 API client 的权威说明。

---

## Scope

修改以下内容时使用本文：

* `frontend/` Svelte/Vite 源码或配置。
* `frontend/src/api.js`。
* `dev.bat`、`build.bat`、`verify-build.bat`。
* `static.go` 的嵌入前端服务逻辑。
* 被 Svelte 消费的内部 `/api/*` 契约。

---

## Development Contract

开发模式：

* Go 后端服务 `/api/*`。
* Vite 服务 Svelte app，默认 `http://127.0.0.1:5173`。
* `frontend/vite.config.js` 把 `/api` 代理到 `BACKEND_PORT`，默认 `3000`。
* `dev.bat` 启动 Go 和 Vite。
* Go 只有显式 `--dev` 时才把页面 route 重定向到 Vite。
* 编辑 Svelte 源码应通过 Vite HMR 更新浏览器。

---

## Production Embed Contract

生产模式：

* `npm run build` 输出到 `frontend/dist/`。
* `static.go` 使用 `//go:embed frontend/dist`。
* 最终 executable 必须在没有 `frontend/`、`frontend/dist/` 或 `public/` 的磁盘文件时仍能服务 SPA。
* 非 API 路径 fallback 到 embedded `index.html`。
* `/api/*` 不能 fallback 到 `index.html`。
* 直接运行 executable 默认是生产行为，不重定向到 Vite。

---

## Frontend API Client

`frontend/src/api.js` 是当前内部 API 的单一前端边界。Svelte 组件应调用 API helper，不直接 `fetch('/api/...')`。

`request(path, options)` 当前规则：

* 使用相对 `/api/*` URL。
* 登录态使用 `localStorage` 中的 `app_access_token`，不使用 cookie credentials。
* 如果存在 access token，且调用方没有显式提供 `Authorization`，自动设置 `Authorization: Bearer <token>`。
* plain object 和 array body 自动 JSON stringify。
* 未提供 `Content-Type` 时才设置 `application/json`。
* `FormData`、`Blob`、raw string 不强制 JSON header。
* 成功 envelope `{success:true,data:...}` 自动返回 `data`。
* 失败 envelope `{success:false,error:{message}}` 抛出 safe message。
* `204`、空响应、没有 `data` 的成功 envelope 返回 `null`。
* 迁移期仍兼容 legacy flat `error` 或 `message`。

当前 helper：

```js
getAccessToken()
setAccessToken(token)
getAuthStatus()
login(payload)
register(payload)
logout()
forgotPassword(payload)
resetPassword(payload)
getUser(id)
listUsers({ page, pageSize })
setUserActive(id, active)
getDictionaries(types)
listDictionaryTypes()
createDictionaryType(payload)
updateDictionaryType(id, payload)
setDictionaryTypeEnabled(id, enabled)
listDictionaryValues(typeId)
createDictionaryValue(typeId, payload)
updateDictionaryValue(typeId, id, payload)
setDictionaryValueEnabled(id, enabled)
getUserOrders(userId, { page, pageSize })
createOrder(payload)
payOrder(orderId)
getMyPoints()
getProducts()
triggerExportToast()
summarizeTextWithLLM({ text, prompt, dimensions })
listNotifications({ page, pageSize, type, email, phone })
listScheduledTasks()
createScheduledTask(payload)
updateScheduledTask(id, payload)
setScheduledTaskEnabled(id, enabled)
listScheduledTaskHistory(id)
listEvents({ page, pageSize })
listEventDeliveries(eventId)
listMessages(queue)
listParameterIntegrationSchemas(scenario)
listParameterIntegrationChannels(scenario)
createParameterIntegrationChannel(payload)
updateParameterIntegrationChannel(id, payload)
setParameterIntegrationChannelEnabled(id, enabled)
listVariables()
createVariable(payload)
updateVariable(id, payload)
setVariableEnabled(id, enabled)
pointsSSEURL(locationObject = globalThis.location)
```

`login(payload)` 和 `register(payload)` 会从后端响应保存 `access_token`；`logout()` 会清除本地 token。`pointsSSEURL()` 会把本地 token 放入 `access_token` query parameter。

---

## Date Time Display Contract

后端 API 的时间语义统一为 UTC；页面展示时根据浏览器 local timezone 渲染。

规则：

* Svelte 页面不要直接写 `new Date(value).toLocaleString()`。
* 展示后端时间字段时，使用 `frontend/src/helpers/dateTime.js` 的 `formatLocalDateTime(value)`。
* helper 会把既有 SQLite UTC 字符串 `YYYY-MM-DD HH:mm:ss` 明确解析为 UTC，再交给 `toLocaleString()` 按 local 展示。
* 带 `Z` 或显式 offset 的 RFC3339 字符串保持按标准解析。
* 只有业务需要编辑调度时间等用户输入时，才在页面层处理 local input；提交给后端的跨层时间仍应带 timezone 或由后端规范化为 UTC。

示例：

```svelte
import { formatLocalDateTime } from '../helpers/dateTime.js';

<td>{formatLocalDateTime(row.updated_at)}</td>
```

---

## Scenario: Order Management Realtime UI

### 1. Scope / Trigger

订单管理页面涉及创建 Creem checkout 台账订单、发起支付、积分查询和 SSE 实时积分。任何修改 `Dashboard.svelte`、`frontend/src/api.js` 或 `frontend/vite.config.js` 的相关行为时，都要遵守本节。`GET /api/products` helper 仍存在，但当前 Creem checkout 页面不依赖本地商品列表。

### 2. Signatures

前端 API helper：

```js
createOrder({ user_id })
createOrderPaymentCheckout(orderId)
getUserOrders(userId, { page, pageSize })
payOrder(orderId)
getMyPoints()
getProducts()
triggerExportToast()
listNotifications({ page, pageSize, type, email, phone })
pointsSSEURL(locationObject = globalThis.location)
```

Vite proxy：

```js
proxy: {
  '/api': {
    target: `http://127.0.0.1:${backendPort}`,
    changeOrigin: true
  }
}
```

### 3. Contracts

* Svelte 组件必须通过 `frontend/src/api.js` helper 调用内部 `/api/*`，不要直接 `fetch('/api/...')`。
* Creem checkout 页面通过 `createOrder({ user_id })` 创建本地 `pending` 订单台账，然后调用 `createOrderPaymentCheckout(order.id)` 获取 `checkout_url` 并跳转。
* 当前 Creem checkout 页面不加载 `getProducts()`，不展示本地商品选择器，不提交 `product_id` 或 `quantity`。本地 `products` 只保留给 legacy/demo/admin 视图。
* 如果订单列表中的 `order.amount` 为 `0`，页面不要格式化成实际货币金额；应显示 provider/Creem 价格来源，避免把本地占位金额误认为实收金额。
* `pointsSSEURL()` 根据当前页面协议和 host 生成相对部署可用的 `http://host/api/points/sse` 或 `https://host/api/points/sse`，并在本地存在 token 时附加 `access_token` query。
* 开发模式下 `/api/points/sse` 通过普通 Vite HTTP proxy 转发，不需要 `ws: true`。
* SSE message 使用 realtime envelope；后端把 envelope 写入 SSE `data:`，Svelte 页面通过 `frontend/src/helpers/realtimeMessages.js` 解析和分发，不要在页面里 hard-code 单个消息 shape：

```json
{"type":"points","presentation":"refresh","payload":{"user_id":"u001","client_id":"...","balance":10}}
```

```json
{"type":"async_export_task","presentation":"toast","payload":{"task_id":"export-1","status":"completed","message":"Export completed"}}
```

```json
{"type":"notification","presentation":"toast","payload":{"id":"notification-id","title":"Order paid","summary":"Your points have been awarded","source_type":"order","source_id":"order-id"}}
```

* `points` 默认展示方式是 `refresh`，用于刷新积分余额；`async_export_task` 和 `notification` 默认展示方式是 `toast`，用于页面可见通知。
* Header 登录态下登出按钮后可以放一个验证按钮，调用 `triggerExportToast()`，由后端发布 `async_export_task` toast；按钮不要在前端本地伪造 toast。
* SSE 断开不应阻塞支付；页面支付成功后不应主动调用 `getMyPoints()` 刷新积分。积分展示应等待 SSE `points` + `refresh` 通知，初始加载和用户显式刷新仍可调用 `getMyPoints()`。

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| user not logged in | order form and refresh/pay controls disabled |
| user clicks create/pay | call `createOrder({ user_id })`, then `createOrderPaymentCheckout(order.id)` |
| create order API fails | show API client safe message and stay on the page |
| checkout API returns `checkout_url` | redirect with `globalThis.location.assign(checkout_url)` |
| pending order row is clicked | call `createOrderPaymentCheckout(order.id)` again |
| `order.amount` is `0` | display provider/Creem amount label, not `$0.00` as a charge |
| SSE message malformed | ignore message; keep page usable |
| SSE disconnected | show disconnected/error status; HTTP refresh remains available |
| local access token missing or invalid | API throws safe unauthorized message; EventSource fails to connect |

### 5. Good/Base/Bad Cases

Good: `Dashboard.svelte` calls `createOrder({ user_id })`, then `createOrderPaymentCheckout(order.id)`, redirects to Creem, and later receives `points` + `refresh` over SSE after webhook-confirmed payment.

Base: If checkout creation fails after local order creation, the pending order remains visible and can be retried from the order list.

Bad: Hard-code `http://localhost:3000/api/points/sse`; this breaks Vite dev proxy, non-local hosts, HTTPS deployments, and JWT token propagation.

Bad: Load `getProducts()` and require a selected local product before creating Creem checkout; Creem product/price is configured in the payment channel for this MVP.

Bad: Call `loadPoints()` inside payment completion handling; this hides whether the points update came from SSE and makes payment-time UI state depend on an extra HTTP query.

### 6. Tests Required

* `cd frontend && npm test`：assert helper paths and `pointsSSEURL` scheme/host behavior。
* `cd frontend && npm run build`：assert Svelte page compiles and `frontend/dist` is regenerated for Go embed。
* Browser smoke check：打开生产服务首页，确认 `Orders`、`Creem Checkout`、`Points balance` 可见。

### 7. Wrong vs Correct

#### Wrong

```js
const events = new EventSource('http://localhost:3000/api/points/sse');
```

#### Correct

```js
const events = new EventSource(pointsSSEURL());
```

---

## Scenario: Experiment LLM and SSE UI

### 1. Scope / Trigger

Modify `frontend/src/pages/Experiments.svelte`, `/experiments` app routing, the LLM summary API helper, or SSE demo controls according to this section. The Experiment page is for functional research and demonstrations; it should reuse existing app capabilities instead of introducing parallel transport paths.

### 2. Signatures

Frontend helpers:

```js
summarizeTextWithLLM({ text, prompt, dimensions })
triggerExportToast()
pointsSSEURL(locationObject = globalThis.location)
```

Route entry:

```js
{ path: '/experiments', label: 'Experiment', description: 'LLM and SSE' }
```

Backend API paths:

```text
POST /api/llm/summaries
POST /api/notifications/test-export-toast
GET  /api/points/sse?access_token=<jwt>
```

### 3. Contracts

* `Experiments.svelte` must call helpers from `frontend/src/api.js`; no direct `fetch('/api/...')` in the component.
* The right side uses the same daisyUI `tabs tabs-lift` + radio input pattern as `Parameters.svelte`.
* The `LLM` tab sends `text`, `prompt`, and `dimensions` to `summarizeTextWithLLM()`. For a single free-form summary demo, use `dimensions:['summary']` and render `response.summary.summary`.
* The `SSE` tab owns the demo button that calls `triggerExportToast()`. Do not place this test action in the global `Header`.
* The `SSE` tab connects through `new EventSource(pointsSSEURL())`, parses realtime envelopes with `dispatchRealtimeMessage()`, and displays received messages as demo output.
* The page may keep chat/event history in component state only; do not persist experiment history unless a new storage contract is added.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| empty LLM text | show local validation message and do not call API |
| empty LLM prompt | show local validation message and do not call API |
| LLM API fails | show API client safe message and append a failed assistant message |
| SSE disconnected | show disconnected/error state; reconnect remains available |
| export trigger API fails | show API client safe message; do not fabricate success toast |
| realtime message malformed | ignore or log a safe malformed-message entry; keep page usable |

### 5. Good/Base/Bad Cases

Good: User opens `/experiments`, submits text + prompt in the `LLM` tab, and sees the DeepSeek-backed summary appended to the chat surface with model/channel metadata.

Base: User opens the `SSE` tab, stream connects through `pointsSSEURL()`, clicks `Trigger export completed`, and sees the backend `async_export_task` toast message arrive through SSE.

Bad: Component directly calls `fetch('/api/llm/summaries')` or creates a fake toast immediately after clicking the SSE button; this bypasses helper tests and hides backend delivery behavior.

### 6. Tests Required

* `frontend/src/api.test.js` covers `summarizeTextWithLLM()` path, method, and JSON body.
* `frontend/src/router.test.js` covers `/experiments.html` alias, menu label, app-route detection, and title.
* `cd frontend && npm test`
* `cd frontend && npm run build`

### 7. Wrong vs Correct

#### Wrong

```svelte
await fetch('/api/llm/summaries', { method: 'POST' });
```

#### Correct

```svelte
const result = await summarizeTextWithLLM({
  text,
  prompt,
  dimensions: ['summary']
});
```

---

## Scenario: Logged-In App Shell

### 1. Scope / Trigger

修改登录后默认首页、应用菜单、页面路由或移动端布局时，遵守本节。认证页仍是独立页面壳，登录后的业务页统一进入 app shell。

### 2. Signatures

路由定义集中在 `frontend/src/router.js`：

```js
appRoutes = [
  { path: '/', label: 'Dashboard', description: 'Welcome' },
  { path: '/orders', label: 'Order', description: 'Orders and points' },
  { path: '/users', label: 'User', description: 'Accounts' },
  { path: '/scheduler', label: 'Scheduler', description: 'Reserved' },
  { path: '/events', label: 'Event', description: 'Domain deliveries' },
  { path: '/dictionary', label: 'Dictionary', description: 'Selectable values' },
  { path: '/variables', label: 'Variable', description: 'Global controls' }
]

normalizePath(pathOrLocation)
navigate(path)
isAuthRoute(path)
isAppRoute(path)
routeTitle(path)
```

应用壳组件：

```svelte
<AppSidebar {path} {auth}>
  {#snippet children()}...{/snippet}
</AppSidebar>
```

### 3. Contracts

* `/` 是登录后的 Dashboard 欢迎页，不承载订单管理表单。
* `/orders` 承载现有订单管理能力。
* `/users` 承载 User 管理列表和启禁用操作。
* `/scheduler` 承载 scheduled task 管理、execution history 和 read-only queue message inspection。
* `/dictionary` 承载 dictionary type 和 dictionary value 管理。
* `/variables` 承载 typed global parameter 和 logic-control variable 管理。
* `/login`、`/register`、`/forgot-password`、`/reset-password` 不包裹 `AppSidebar`。
* 未登录用户访问 app route 时不展示登录后菜单，应进入登录入口或认证提示。
* 新增登录后菜单项时，先更新 `appRoutes`，再在 `App.svelte` 中挂载对应页面组件。
* 桌面端使用左侧菜单；移动端使用 daisyUI drawer，不能产生页面级水平滚动。

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| logged in and request `/` | show Dashboard welcome page with app sidebar |
| logged in and request `/orders` | show order management page with app sidebar |
| logged in and request `/users` | show User management page with app sidebar |
| logged in and request `/scheduler` | show Scheduler page with task form, task list, history panel, and queue message table |
| logged in and request `/dictionary` | show Dictionary page with dictionary type list, value form, and value table |
| logged in and request `/variables` | show Variable page with typed create/edit form and variable table |
| unauthenticated and request app route | show login entry, no app sidebar |
| auth route requested | show auth page without app sidebar |
| mobile viewport | sidebar is hidden behind drawer toggle, content has no horizontal overflow |

### 5. Good/Base/Bad Cases

Good: Add a new menu page by extending `appRoutes`, rendering the page in `App.svelte`, and covering route helpers in `router.test.js`.

Base: Scheduler remains inside the same app shell and uses `frontend/src/api.js` helpers for every backend call.

Bad: Hard-code a separate menu array inside `AppSidebar.svelte`; it will drift from route tests and document titles.

### 6. Tests Required

```sh
cd frontend && npm test
cd frontend && npm run build
```

Assert route aliases, auth route detection, app route labels, and Svelte compilation. For layout-sensitive changes, also smoke check desktop and mobile widths in the browser.

### 7. Wrong vs Correct

#### Wrong

```svelte
const menu = [{ path: '/orders', label: 'Order' }];
```

#### Correct

```svelte
import { appRoutes } from '../router.js';
```

---

## Scenario: User Management UI

### 1. Scope / Trigger

修改 `frontend/src/pages/Users.svelte`、user management API helper、`/users` app route，或后端 `/api/users*` DTO 时遵守本节。

### 2. Signatures

Frontend helpers:

```js
listUsers({ page, pageSize })
setUserActive(userId, active)
```

Backend API paths:

```text
GET /api/users?page=1&page_size=10
PATCH /api/users/:id/active
```

Route entry:

```js
{ path: '/users', label: 'User', description: 'Accounts' }
```

### 3. Contracts

* `Users.svelte` must call helpers from `frontend/src/api.js`; no direct `fetch('/api/...')` in the component.
* User list uses standard `{items, pagination}` payload from the internal API envelope after `request()` unwraps `data`.
* Page includes a paginated table with user identity, email, verification status, active/disabled status, created time, and row actions.
* Disable/Enable action uses `setUserActive(user.id, !user.is_active)` and refreshes the current page after success.
* Current logged-in user row should be visually identifiable and its Disable action disabled in the UI; backend still enforces the same rule.
* Pagination controls use daisyUI `join` / `btn` classes and remain horizontally scrollable on narrow widths.
* Page displays safe API client error messages through `Notice`; successful enable/disable can show a short success notice.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| user not logged in | app shell auth flow shows login, not User controls |
| user list empty | page shows empty state |
| API request fails | page displays safe API client message through `Notice` |
| disabling current logged-in user | button is disabled when the row can be identified; backend validation message is displayed if request still happens |
| target user missing during toggle | page displays `user not found` safe message |
| mobile viewport | table can scroll horizontally and page layout avoids page-level horizontal overflow |

### 5. Good/Base/Bad Cases

Good: Load page 1 with `listUsers({ page: 1, pageSize: 10 })`, click Disable on another user, then refresh the current page.

Base: Enable on a disabled row calls the same helper with `active:true` and shows the updated badge after refresh.

Bad: Hard-code `/api/users/${id}/active` directly inside `Users.svelte`; helper tests will not cover it.

### 6. Tests Required

* `frontend/src/api.test.js` covers user helper paths, pagination query params, encoded user IDs, verbs, and JSON body.
* `frontend/src/router.test.js` covers `/users.html` alias, app route label, app-route detection, and title.
* `cd frontend && npm test`
* `cd frontend && npm run build`
* Browser smoke check at `/users` on desktop and mobile widths when layout changes.

### 7. Wrong vs Correct

#### Wrong

```js
await fetch(`/api/users/${user.id}/active`, { method: 'PATCH' });
```

#### Correct

```js
await setUserActive(user.id, !user.is_active);
```

---

## Scenario: Scheduler Operations UI

### 1. Scope / Trigger

修改 `frontend/src/pages/Scheduler.svelte`、scheduler/message API helper、或后端 `/api/scheduler/*` / `/api/messages` DTO 时遵守本节。

### 2. Signatures

Frontend helpers:

```js
listScheduledTasks()
createScheduledTask({ name, job_name, schedule_type, schedule_value, payload_json, enabled })
updateScheduledTask(id, { name, job_name, schedule_type, schedule_value, payload_json, enabled })
setScheduledTaskEnabled(id, enabled)
listScheduledTaskHistory(id)
listMessages(queue = '')
```

Backend API paths:

```text
GET    /api/scheduler/tasks
POST   /api/scheduler/tasks
PUT    /api/scheduler/tasks/:id
PATCH  /api/scheduler/tasks/:id/enabled
GET    /api/scheduler/tasks/:id/history
GET    /api/messages?queue=<queue>
```

### 3. Contracts

* `Scheduler.svelte` must call helpers from `frontend/src/api.js`; no direct `fetch('/api/...')` in the component.
* Page includes task create/edit form, task table, enable/disable action, history table, and read-only queue message table.
* MVP `job_name` is `scheduler.noop`; backend rejects unregistered job names.
* `schedule_type` accepts `cron` and `once_at`; `schedule_value` is cron expression or RFC3339 timestamp.
* Queue message table is read-only and filters by queue name through `GET /api/messages?queue=...`.
* UI displays safe API client error messages through `Notice`.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| user not logged in | app shell auth flow shows login, not scheduler controls |
| invalid cron or invalid JSON payload | API throws safe validation message; page keeps form values |
| create/update succeeds | task table refreshes and form enters edit state for saved task |
| selected task has no history | history panel shows empty state |
| message queue is empty | queue panel shows empty state |

### 5. Good/Base/Bad Cases

Good: Submit through `createScheduledTask`, then refresh tasks and show the computed `next_run_at`.

Base: Queue filter can be empty to list all queues or set to `scheduled-tasks` / `domain-events`.

Bad: Component manually encodes `/api/scheduler/tasks/${id}` with string concatenation and direct `fetch`; helper tests will not cover it.

### 6. Tests Required

* `frontend/src/api.test.js` covers scheduler/message helper paths, verbs, encoded IDs, and JSON body handling.
* `cd frontend && npm test`
* `cd frontend && npm run build`
* Browser smoke check: login, open `/scheduler`, create a `scheduler.noop` task, open history, filter queue messages.

### 7. Wrong vs Correct

#### Wrong

```js
await fetch(`/api/messages?queue=${queue}`);
```

#### Correct

```js
messages = await listMessages(queueFilter.trim());
```

---

## Scenario: Event Management UI

### 1. Scope / Trigger

修改 `frontend/src/pages/Events.svelte`、event API helper、`/events` app route，或后端 `/api/events*` DTO 时遵守本节。

### 2. Signatures

Frontend helpers:

```js
listEvents({ page, pageSize })
listEventDeliveries(eventId)
```

Backend API paths:

```text
GET /api/events?page=1&page_size=10
GET /api/events/:id/deliveries
```

Route entry:

```js
{ path: '/events', label: 'Event', description: 'Domain deliveries' }
```

### 3. Contracts

* `Events.svelte` must call helpers from `frontend/src/api.js`; no direct `fetch('/api/...')` in the component.
* Event list uses standard `{items, pagination}` payload from the internal API envelope after `request()` unwraps `data`.
* Event page is read-only: no replay, no retry, no delivery state mutation.
* Delivery records are lazy-loaded by selected event id through `listEventDeliveries(event.id)`.
* Pagination controls use daisyUI `join` / `btn` classes and remain horizontally scrollable on narrow widths.
* Event rows should show enough context to identify `topic`, `aggregate_type`, `aggregate_id`, event id, occurrence time, and payload preview.
* Delivery rows should show `subscriber`, `status`, `attempts`, `message_id`, `last_error`, and timestamps.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| user not logged in | app shell auth flow shows login, not Event controls |
| event list empty | page shows empty state |
| selected event has no delivery records | delivery panel shows empty state |
| API request fails | page displays safe API client message through `Notice` |
| payload or metadata is invalid JSON | page displays original text instead of crashing |
| mobile viewport | event table can scroll horizontally and page layout stacks vertically |

### 5. Good/Base/Bad Cases

Good: Load page 1 with `listEvents({ page: 1, pageSize: 10 })`, auto-select the first visible event, then call `listEventDeliveries(event.id)`.

Base: If page changes and the selected event is not on the new page, clear old delivery records before selecting the first event on the new page.

Bad: Hard-code `/api/events/${id}/deliveries` directly inside `Events.svelte`; helper tests will not cover it.

### 6. Tests Required

* `frontend/src/api.test.js` covers event helper paths, pagination query params, and encoded event IDs.
* `frontend/src/router.test.js` covers `/events.html` alias, app route label, app-route detection, and title.
* `cd frontend && npm test`
* `cd frontend && npm run build`
* Browser smoke check at `/events` on desktop and mobile widths when layout changes.

### 7. Wrong vs Correct

#### Wrong

```js
await fetch(`/api/events/${event.id}/deliveries`);
```

#### Correct

```js
deliveries = await listEventDeliveries(event.id);
```

---

## Scenario: Notification Management UI

### 1. Scope / Trigger

修改 `frontend/src/pages/Notifications.svelte`、`/notifications` admin 菜单路由、notification API helper、或 realtime `notification` toast 行为时，遵守本节。

### 2. Signatures

Frontend helpers:

```js
getDictionaries(['notification_type'])
listNotifications({ page, pageSize, type, email, phone })
```

Backend API path:

```text
GET /api/notifications?page=1&page_size=10&type=sse&email=ada@example.com&phone=138
```

Route entry:

```js
{ path: '/notifications', label: 'Notification', description: 'Delivery ledger', adminOnly: true }
```

Realtime message:

```json
{"type":"notification","presentation":"toast","payload":{"id":"notification-id","title":"Order paid","summary":"Your points have been awarded","source_type":"order","source_id":"order-id"}}
```

### 3. Contracts

* `Notifications.svelte` must call helpers from `frontend/src/api.js`; no direct `fetch('/api/...')` in the component.
* `/notifications` is an admin-only app route. Regular users do not see the menu entry; backend still enforces `RequireAdmin()`.
* Page is read-only for MVP. Do not add a create form or POST helper until backend exposes an explicit admin creation contract.
* Type filter options come from dynamic dictionary lookup `notification_type`.
* List data uses standard `{items, pagination}` payload from the internal API envelope after `request()` unwraps `data`.
* Filters are `type`, `email`, and `phone`; applying filters resets to page 1.
* Table should show notification identity, type label, status, recipient, source, created time, sent time, and safe `last_error`.
* Use `formatLocalDateTime(value)` for notification timestamps.
* `frontend/src/helpers/realtimeMessages.js` treats `notification` as a toast by default and builds toast text from `summary`, then `title`, before falling back.
* Realtime `notification` payload must not assume or display raw `payload_json`.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| user not logged in | app shell auth flow shows login, not Notification controls |
| non-admin user | menu entry hidden; direct path falls back to accessible dashboard |
| dictionary lookup fails | page displays safe API client message through `Notice` |
| list request fails | page displays safe API client message through `Notice` |
| notification list empty | page shows empty state |
| invalid filter type reaches backend | page displays backend validation message |
| mobile/narrow width | table scrolls horizontally and page avoids page-level horizontal overflow |

### 5. Good/Base/Bad Cases

Good: Admin opens `/notifications`, filters by `type=sms` and `phone=138`, then pages through the ledger.

Base: A notification without `sent_at` renders the fallback display from `formatLocalDateTime`.

Bad: Add a local hard-coded notification type enum in the page; types are dictionary-managed and must be loaded dynamically.

### 6. Tests Required

* `frontend/src/api.test.js` covers `listNotifications({page,pageSize,type,email,phone})` path and query encoding.
* `frontend/src/router.test.js` covers `/notifications.html` alias, admin-only menu visibility, app-route detection, and title.
* `frontend/src/realtimeMessages.test.js` covers `notification` default toast presentation and toast text.
* `cd frontend && npm test`
* `cd frontend && npm run build`
* Browser smoke check at `/notifications` on desktop and mobile widths when layout changes.

### 7. Wrong vs Correct

#### Wrong

```svelte
await fetch(`/api/notifications?type=${filters.type}`);
```

#### Correct

```svelte
notifications = await listNotifications({
  page,
  pageSize: notificationPageSize,
  type: filters.type,
  email: filters.email.trim(),
  phone: filters.phone.trim()
});
```

---

## Scenario: Dictionary Management UI

### 1. Scope / Trigger

修改 `frontend/src/pages/Dictionary.svelte`、`/dictionary` 菜单路由、dictionary management API helper、或动态 dictionary lookup helper 时，遵守本节。该页面管理 dictionary type 和 dictionary value；普通表单仍通过 `getDictionaries(types)` 获取 enabled lookup options。

### 2. Signatures

Frontend helpers:

```js
getDictionaries(types)
listDictionaryTypes()
createDictionaryType(payload)
updateDictionaryType(id, payload)
setDictionaryTypeEnabled(id, enabled)
listDictionaryValues(typeId)
createDictionaryValue(typeId, payload)
updateDictionaryValue(typeId, id, payload)
setDictionaryValueEnabled(id, enabled)
```

Route entry:

```js
{ path: '/dictionary', label: 'Dictionary', description: 'Selectable values' }
```

Backend API paths:

```text
GET   /api/dictionaries?types=product_category,region
GET   /api/dictionary/types
POST  /api/dictionary/types
PUT   /api/dictionary/types/:id
PATCH /api/dictionary/types/:id/enabled
GET   /api/dictionary/types/:type_id/values
POST  /api/dictionary/types/:type_id/values
PUT   /api/dictionary/types/:type_id/values/:id
PATCH /api/dictionary/values/:id/enabled
```

### 3. Contracts

* `Dictionary.svelte` 必须通过 `frontend/src/api.js` helper 调用内部 API，不直接 `fetch('/api/...')`。
* 页面左侧管理 dictionary types，右侧管理当前 selected type 的 values。
* Type 表单字段包括 `type_key`、`name`、`enabled`、`description`。
* Value 表单字段包括 `value_code`、`label`、`sort_order`、`enabled`、`description`。
* create/update 成功后刷新对应列表，并让表单进入 saved row 的编辑状态。
* enable/disable type 后刷新 type 列表和当前 value 列表。
* enable/disable value 后刷新 value 列表；如果当前表单正在编辑该 value，同步表单状态。
* 普通 lookup consumer 使用 `getDictionaries(types)`，只接收后端返回的 enabled options。
* 页面错误提示显示 API client 抛出的 safe message。

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| user not logged in | app shell auth flow shows login, not Dictionary controls |
| dictionary list empty | page shows empty state and disables value save until a type exists |
| selected dictionary has no values | value table shows empty state |
| API request fails | page displays safe API client message through `Notice` |
| create/update succeeds | relevant list refreshes and form enters edit state for saved row |
| backend rejects duplicate type/value | page shows backend safe conflict message |
| mobile/narrow width | value table scrolls horizontally and page avoids page-level horizontal overflow |

### 5. Good/Base/Bad Cases

Good: 创建 `order_status`，并添加 `pending`、`paid`、`cancelled` values，业务表单通过 `getDictionaries(['order_status'])` 读取。

Base: 禁用某个 value 后，Dictionary 页面仍能看到它，普通 lookup 不再返回它。

Bad: 在 Svelte 组件中直接调用 `/api/dictionary/types`，或在普通页面里手写 value label map；这会绕过 helper 测试和动态 dictionary contract。

### 6. Tests Required

* `frontend/src/api.test.js` 覆盖 dictionary management helper 路径、method、encoded ID 和 body。
* `frontend/src/router.test.js` 覆盖 `/dictionary.html` alias、menu label、app-route detection 和 title。
* `cd frontend && npm test`
* `cd frontend && npm run build`
* 布局敏感变更建议 browser smoke check `/dictionary` 桌面和移动宽度。

### 7. Wrong vs Correct

#### Wrong

```svelte
await fetch('/api/dictionary/types');
```

#### Correct

```svelte
dictionaryTypes = await listDictionaryTypes();
```

---

## Scenario: Parameter Integration Management UI

### 1. Scope / Trigger

修改 `frontend/src/pages/Parameters.svelte`、`/parameters` 菜单路由、parameter integration API helper、或 adapter schema 动态表单时，遵守本节。该页面管理外部集成 channel/credential 配置，但不展示 provider raw payload 或 legacy credential ciphertext/masked fields。`credential_value` 只在 `auth.user.is_admin=true` 的后台管理员页面内编辑。

### 2. Signatures

Frontend helpers:

```js
listParameterIntegrationSchemas('payment')
listParameterIntegrationChannels('payment')
createParameterIntegrationChannel(payload)
updateParameterIntegrationChannel(id, payload)
setParameterIntegrationChannelEnabled(id, enabled)
```

Route entry:

```js
{ path: '/parameters', label: 'Parameter', description: 'Integration settings' }
```

Backend API paths:

```text
GET   /api/parameters/integration-schemas?scenario=payment|llm|sms|email|oss
GET   /api/parameters/integration-channels?scenario=payment|llm|sms|email|oss
POST  /api/parameters/integration-channels
PUT   /api/parameters/integration-channels/:id
PATCH /api/parameters/integration-channels/:id/enabled
```

OSS providers such as Cloudflare R2 and Aliyun OSS are Parameter configuration scenarios only: the UI loads `scenario=oss`, captures provider `endpoint_url` / `bucket` / `region` config and `s3_access_key` credentials, and does not execute upload/download or SDK connection tests.

OSS primary-provider UI is scoped to the OSS tab only. The form renders a `Primary provider` toggle for OSS channels, submits `is_primary` only as a meaningful OSS field, and treats the backend response as the source of truth for the final primary state. Zero primary OSS channels is a valid state; the UI must not auto-promote another channel when the current primary is unchecked or disabled.

### 3. Contracts

* `Parameters.svelte` 必须通过 `frontend/src/api.js` helper 调用内部 API，不直接 `fetch('/api/...')`。
* 页面左侧是 create/edit 表单，右侧是 daisyUI `tabs tabs-lift` + radio input pattern 的 tab content。
* Tab 固定为 `Payment`、`LLM`、`SMS`、`Email`、`OSS`，分别查询 `scenario=payment|llm|sms|email|oss` 的不分页列表。
* `Parameter` 菜单项是 admin-only route；普通登录用户不显示该菜单，后端仍通过 `RequireAdmin()` 做最终权限保护。
* 进入场景时先调用 `listParameterIntegrationSchemas(scenario)`，再按 `adapter_key` 匹配当前 schema。
* 如果当前 `adapter_key` 有 schema，页面根据 `config_fields` 渲染结构化 config 表单，并同步写回 `form.config_json`；Advanced JSON 仍保留，用于额外非敏感字段。
* 如果当前 `adapter_key` 没有 schema，或用户选择 `Custom adapter`，页面显示原始 `adapter_key` 输入和 credential value/Advanced JSON 输入，不能阻塞暂未建 schema 的渠道。
* `credential_format=plain` 时，结构化 credential 输入保存为原始 `credential_value` 字符串；`credential_format=json_object` 时，保存为 JSON object 字符串。
* schema field 的 `dictionary_type` 可以通过 `getDictionaries(types)` 动态加载 options；如果字典没有值，使用 schema 内置 `options` 兜底。
* config 和 credential 的 schema field 如果返回 `help_text`，页面应在字段名旁渲染 `?` 类型 tooltip，鼠标移入或键盘聚焦时展示说明；例如 Aliyun SMTP password 必须提示使用邮箱客户端授权密码，而不是账号登录密码。
* 固定的 `Webhook` 开关虽然不是 adapter schema field，也必须在 label 旁渲染与 schema `help_text` 一致的 `?` tooltip；提示内容展示通用 provider callback URL 格式 `https://<public-domain>/api/integrations/<scenario>/<channel_code>/webhooks/<provider_code>`，并用当前 `form.scenario`、`form.channel_code`、`form.provider_code` 代入，空值保留占位符。当前已实现的 Payment/Creem 后端 route 是 `/api/integrations/payment/<channel_code>/webhooks/creem`；SMS/Email/LLM 页面只展示统一格式提示，不代表已有真实 webhook route。
* `environment` 使用 `integration_environment` 字典，缺失时 fallback 为 `test` 和 `production`；provider API URL 不走 dictionary，也不使用 schema `options`，应渲染为普通 URL 输入框，因为 dictionary/option value 是规范化 code。
* `credential_type` 使用 `integration_credential_type` 字典下拉，当前默认值为 `payment_bundle`、`api_key`、`smtp_password` 和 `s3_access_key`；缺失时前端 fallback 到同一组值，后端保存仍按启用字典值校验。
* 创建和编辑时使用 `credential_value`；编辑表单从 admin DTO 回填当前值。schema 中 `kind=secret` 的字段必须默认使用 password 输入遮罩，并提供显示/隐藏按钮。
* 不再使用 `credential_masked_value`。列表只展示 `credential_type` 和是否 configured，不展示具体 credential value。
* 前端不得提交或展示 legacy `credential_plaintext`、`ciphertext`、`key_version`、`masked_value`。
* 前端只做辅助格式化和渲染，required、URL、number、boolean、敏感字段位置等最终以 backend schema validation 为准。

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| user not logged in | app shell auth flow shows login, not Parameter controls |
| selected tab has no channels | tab content shows empty state |
| schema API returns no schema for scenario | form still allows Custom adapter and Advanced JSON |
| API request fails | page displays safe API client message through `Notice` |
| create/update succeeds | current scenario list refreshes and form enters edit state for saved channel |
| enable/disable succeeds | current scenario list refreshes and selected form state updates if it was editing that row |
| credential value empty on edit | frontend sends empty string; backend preserves existing credential |
| OSS primary toggled on and save succeeds | refreshed list shows only the backend-returned primary row |
| primary OSS channel disabled | refreshed row shows `enabled=false` and not primary; no local auto-promote occurs |
| non-OSS tab active | primary provider toggle is hidden |
| Webhook help icon hover/focus | tooltip shows the provider callback URL format using current scenario/channel/provider values and does not change the save payload |
| backend rejects sensitive config key or invalid schema field | page shows backend safe validation message |
| mobile/narrow width | tables scroll horizontally and page avoids page-level horizontal overflow |

### 5. Good/Base/Bad Cases

Good: In the `OSS` tab, mark Cloudflare R2 as `Primary provider` and save; after refresh, only the backend-returned primary row shows the primary badge.

Base: In the `OSS` tab, uncheck the current primary provider and save; zero primary OSS channels remains valid, and every OSS row shows not primary.

Bad: Enforce primary-provider uniqueness only by hiding or disabling other OSS row controls. The backend response must remain the page source of truth.

Good: 在 `Payment` tab 新增 `creem-prod`，通过 schema 字段填写 `base_url/product_id/success_url`，通过 credential 字段填写 secret bundle。

Base: 在 `LLM` tab 编辑 `deepseek-prod` 的 `priority` 和 `metadata_json`，credential 字段保持当前回填值，后端保存同一个 `credential_value`。

Bad: 在 Svelte 组件中直接调用 `/api/parameters/integration-channels`，或在列表中展示具体 `credential_value`；这会绕过 helper 测试和后台配置边界。

Bad: 将 Webhook tooltip 写死为 Creem 专用文案，或改成表单内常驻说明段落；这会隐藏 payment/sms/email/llm 的统一 callback 格式，也会挤占配置表单空间。

### 6. Tests Required

* `frontend/src/api.test.js` must cover Parameter integration helper bodies including `is_primary`.
* `frontend/src/api.test.js` 覆盖 parameter helper 路径、method、encoded ID 和 body。
* `frontend/src/router.test.js` 覆盖 `/parameters.html` alias、menu label、app-route detection 和 title。
* `cd frontend && npm test`
* `cd frontend && npm run build`
* 布局敏感变更建议 browser smoke check `/parameters` 桌面和移动宽度。

### 7. Wrong vs Correct

#### Wrong

```svelte
await fetch('/api/parameters/integration-channels?scenario=payment');
```

#### Correct

```svelte
channels = await listParameterIntegrationChannels('payment');
```

#### Wrong

```svelte
<td>{channel.credential_value}</td>
```

#### Correct

```svelte
<td>{channel.credential_value ? 'configured' : '--'}</td>
```

---

## Scenario: Variable Management UI

### 1. Scope / Trigger

修改 `frontend/src/pages/Variables.svelte`、`/variables` 菜单路由、或 variable API helper 时，遵守本节。该页面管理 typed global parameter 和 logic-control values，不负责具体业务逻辑消费变量，也不存储 secrets。

### 2. Signatures

Frontend helpers:

```js
listVariables()
createVariable(payload)
updateVariable(id, payload)
setVariableEnabled(id, enabled)
```

Route entry:

```js
{ path: '/variables', label: 'Variable', description: 'Global controls' }
```

Backend API paths:

```text
GET   /api/variables
POST  /api/variables
PUT   /api/variables/:id
PATCH /api/variables/:id/enabled
```

### 3. Contracts

* `Variables.svelte` 必须通过 `frontend/src/api.js` helper 调用内部 API，不直接 `fetch('/api/...')`。
* 页面左侧是 create/edit 表单，右侧是 variables table。
* 表单字段包括 `key`、`name`、`value_type`、`value_json`、`enabled`、`description`。
* `value_type` options 固定为 `string`、`number`、`boolean`、`json`。
* `string` 类型可用普通 input 输入，提交时由后端规范化为 JSON string。
* `number` 类型可用普通 input 输入，但最终必须由后端校验为 JSON number。
* `boolean` 类型使用 select 或等价二选控件，提交 `true` 或 `false`。
* `json` 类型使用 textarea，前端可尝试格式化，但最终校验由后端执行。
* 列表展示 `key`、`name`、`value_type`、`value_json`、enabled 状态、description 和更新时间。
* create/update 成功后刷新列表，并让表单进入 saved row 的编辑状态。
* enable/disable 成功后刷新列表；如果当前表单正在编辑该变量，同步表单状态。
* 页面错误提示显示 API client 抛出的 safe message。

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| user not logged in | app shell auth flow shows login, not Variable controls |
| variable list empty | page shows empty state |
| API request fails | page displays safe API client message through `Notice` |
| create/update succeeds | variable table refreshes and form enters edit state for saved variable |
| enable/disable succeeds | table refreshes and selected form state updates if it was editing that row |
| backend rejects duplicate key | page shows `variable key already exists` safe message |
| backend rejects invalid typed value | page shows backend validation message and keeps form values |
| mobile/narrow width | table scrolls horizontally and page avoids page-level horizontal overflow |

### 5. Good/Base/Bad Cases

Good: 创建 `checkout.max_retry`，`value_type=number`，`value_json=3`，用于未来业务逻辑按 key 读取。

Base: 创建 `feature.new_checkout`，`value_type=boolean`，`value_json=true`，通过 enabled 控制是否参与逻辑判断。

Bad: 在 Svelte 组件中直接调用 `/api/variables`，或把 API key/token 放进 variable value；这会绕过 helper 测试或破坏 secret 边界。

### 6. Tests Required

* `frontend/src/api.test.js` 覆盖 variable helper 路径、method、encoded ID 和 body。
* `frontend/src/router.test.js` 覆盖 `/variables.html` alias、menu label、app-route detection 和 title。
* `cd frontend && npm test`
* `cd frontend && npm run build`
* 布局敏感变更建议 browser smoke check `/variables` 桌面和移动宽度。

### 7. Wrong vs Correct

#### Wrong

```svelte
await fetch('/api/variables');
```

#### Correct

```svelte
variables = await listVariables();
```

#### Wrong

```js
await createVariable({ key: 'api.secret', value_type: 'string', value_json: 'sk-live-secret' });
```

#### Correct

```js
await createVariable({ key: 'feature.new_checkout', value_type: 'boolean', value_json: 'true' });
```

---

## UI Contract

* Tailwind CSS 和 daisyUI 是现有样式系统。
* 常用 UI 使用 daisyUI class，例如 `btn`、`input`、`card`、`alert`、`navbar`、`loading`。
* 不要引入第二套 Svelte UI component library，除非任务明确要求。
* 组件中的错误提示应显示 API client 抛出的 safe message。

---

## Scenario: Settings Logo UI

### 1. Scope / Trigger

Modify `frontend/src/pages/Settings.svelte`, `/settings` app routing, header logo rendering, or settings API helpers according to this section.

### 2. Signatures

Frontend helpers:

```js
getSiteSettings()
uploadSiteLogo(file)
```

Route entry:

```js
{ path: '/settings', label: 'Setting', description: 'Site preferences', adminOnly: true }
```

Backend API paths:

```text
GET  /api/settings/site
POST /api/settings/site/logo
GET  /api/settings/public/logo
```

Header logo:

```svelte
<img src={siteSettings.logo_url || '/logo.png'} width="110" height="25" />
```

### 3. Contracts

* `Header.svelte` renders an image in `navbar-start`, not the text `Svelte Go Starter`.
* The logo renders at `110x25` with object containment and falls back to `/logo.png` on load error.
* `App.svelte` loads site settings with `getSiteSettings()` and passes them into `Header`.
* `/settings` is admin-only in `appRoutes`; regular users do not see the menu item and backend upload remains protected by `RequireAdmin()`.
* `Settings.svelte` uses daisyUI `tabs tabs-lift` and includes `General` and `Retain` tabs.
* `General` uploads the logo through `uploadSiteLogo(file)` using `FormData`; it must not call `fetch('/api/settings/site/logo')` directly.
* `frontend/public/logo.png` is the default public asset and must be present so Vite copies it to `frontend/dist/logo.png` during build.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| settings API fails | header keeps `/logo.png` |
| no configured logo | header displays `/logo.png` |
| configured logo URL fails to load | image `onerror` resets to `/logo.png` |
| regular user opens menu | `/settings` is absent from `visibleAppRoutes()` |
| user selects no file | page shows local `Logo file is required` and does not call upload |
| upload API fails | page displays API client safe message through `Notice` |
| upload succeeds | page clears file input, refreshes site settings, and header updates |

### 5. Good/Base/Bad Cases

Good: Admin opens `/settings`, selects a PNG/WebP/JPEG logo, uploads it, and the header image switches to `/api/settings/public/logo?v=...`.

Base: Fresh install has only `frontend/public/logo.png`; header still shows a 110x25 image before any settings API response arrives.

Bad: Component directly calls `fetch('/api/settings/site/logo')`, or forces `Content-Type: application/json` for `FormData`; this breaks the API helper contract.

Bad: Header stores the configured logo URL in localStorage; the source of truth is the backend settings API.

### 6. Tests Required

* `frontend/src/api.test.js` covers settings helper paths, `FormData`, auth header, and no forced JSON content type.
* `frontend/src/router.test.js` covers `/settings.html`, app-route detection, route title, and admin-only visibility.
* `cd frontend && npm test`
* `cd frontend && npm run build` and verify `frontend/dist/logo.png` exists.

### 7. Wrong vs Correct

#### Wrong

```svelte
await fetch('/api/settings/site/logo', { method: 'POST', body: formData });
```

#### Correct

```svelte
await uploadSiteLogo(selectedLogo);
await onSettingsChanged?.();
```

---

## Dictionary and Enum Contract

* 静态 enum 放在 `frontend/src/enums/*.ts`。
* label 和 options 必须从同一个定义生成。
* 动态 dictionary 通过后端 `/api/dictionaries` 获取。
* dictionary store 位于 `frontend/src/stores/`。
* 纯工具/cache helper 位于 `frontend/src/helpers/`。
* dictionary store 应批量加载缺失类型，并复用缓存和 pending request。
* Svelte 组件不把资源 ID 翻译成名称；如果 DTO 有展示 ID，应显示后端提供的 `xxx_name`。

---

## Validation Matrix

| Condition | Expected behavior |
| --- | --- |
| production request `/login` | serve embedded `index.html` |
| production request `/api/unknown` | JSON 404, not SPA index |
| direct executable run | no Vite redirect |
| Svelte source changes in dev | Vite HMR updates browser |
| API success envelope | `request()` returns `data` |
| API error envelope | `request()` throws `error.message` |
| plain object body | JSON encoded |
| `FormData` body | body/header untouched |
| API DTO has `xxx_id` for display | component displays `xxx_name` |

---

## Tests Required

Frontend API client change：

```sh
cd frontend && npm test
cd frontend && npm run build
```

Frontend serving/embed change：

```sh
cd frontend && npm run build
go test ./...
```

Packaging change：

```bat
build.bat
verify-build.bat
```

---

## Wrong vs Correct

### Wrong

```svelte
const response = await fetch('http://localhost:3000/api/auth/status');
```

### Correct

```svelte
import { getAuthStatus } from './api.js';

const status = await getAuthStatus();
```

### Wrong

```go
router.Static("/", "public/")
```

### Correct

```go
//go:embed frontend/dist
var frontendDistFS embed.FS
```

---

## Common Mistakes

* Svelte 组件直接 fetch 内部 API。
* 在组件中使用绝对 localhost API URL。
* 前端把 `xxx_id` 翻译成名称。
* `FormData` 请求被强制设置 `Content-Type: application/json`。
* `/api/*` fallback 到 SPA `index.html`。
* 以为 Go embed 会自动反映 Svelte 源码修改。
