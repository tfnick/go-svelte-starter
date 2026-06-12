# API Contracts

> 本文是后端对外 JSON 契约和 DTO 边界的权威说明。Route handler 的执行流程见 [Route Handler Guidelines](./route-handler-guidelines.md)。

---

## Overview

所有给前端的内部 `/api/*` 响应必须返回 DTO，不直接暴露 `models.*`。
Open API 是独立公开契约，详见 [Open API Guidelines](./open-api-guidelines.md)。
时间字段遵守 [Database Guidelines / Timezone Contract](./database-guidelines.md#timezone-contract)：后端时间语义为 UTC，前端展示时再转换为用户 local timezone。

当前内部 API 的统一响应格式是：

```json
{"success":true,"data":{}}
```

失败格式是：

```json
{"success":false,"error":{"code":"snake_case","message":"safe message"}}
```

前端 `frontend/src/api.js` 会自动 unwrap 成 `data`，所以 Svelte 组件拿到的是 payload 本身。

---

## Internal API Envelope

内部 `/api/*` route 必须使用 `api/framework/http/response`：

```go
httpresponse.OK(c, dto)
httpresponse.Created(c, dto)
httpresponse.OKMessage(c, "message")
httpresponse.OKEmpty(c)
httpresponse.BadRequest(c, "invalid request data")
httpresponse.InternalUsecaseError(c, err)
```

不要在内部 route 中直接调用 `c.JSON(...)`。`api/framework/archguard/layer_boundary_test.go` 会检查这一点。

---

## Pagination Contract

内部 `/api/*` 列表接口需要分页时，统一使用 page-number pagination：

* Query parameters use `page` and `page_size`.
* `page` is 1-based.
* Missing `page` defaults to `1`.
* Missing `page_size` defaults to `10`.
* `page_size` max is `50`.
* Explicit `page <= 0`, `page_size <= 0`, non-numeric values, or too-large `page_size` return `CodeValidation`.
* route layer parses pagination query parameters with `api/framework/http/request.PageQuery(c)`.
* usecase layer uses `fwusecase.PageQuery` / `fwusecase.PageResult` for standard normalization and metadata.
* model layer receives only concrete SQL parameters such as `limit` and `offset`.
* route layer maps usecase result to route-local DTO and returns it through `httpresponse.OK`.

Standard response payload inside the internal success envelope:

```json
{
  "items": [],
  "pagination": {
    "page": 1,
    "page_size": 10,
    "total_items": 0,
    "total_pages": 0,
    "has_previous": false,
    "has_next": false
  }
}
```

排序字段必须稳定，避免翻页时重复或遗漏。例如订单列表使用 `ORDER BY created_at DESC, id DESC`。

---

## DTO Boundary

DTO 边界固定为：

```text
models -> usecase.Co -> routes.DTO -> httpresponse envelope
```

规则：

* `models.*` 是存储结构，不是前端契约。
* `usecase.XxxCo` 是 route-independent client object，可承载业务层组装后的数据。
* `routes` 定义 request DTO、response DTO，以及 `ToXxxResponse(...)` mapper。
* mapper 应显式字段映射，不使用 reflection 或“自动模型转 DTO”。
* request DTO 只接受客户端允许输入的字段，不直接 bind 到 model。

示例：

```go
type UserResponse struct {
    ID            string `json:"id"`
    Name          string `json:"name"`
    Email         string `json:"email"`
    EmailVerified bool   `json:"email_verified"`
    IsActive      bool   `json:"is_active"`
}

func ToUserResponse(user usecase.UserCo) UserResponse {
    return UserResponse{
        ID:            user.ID,
        Name:          user.Name,
        Email:         user.Email,
        EmailVerified: user.EmailVerified,
        IsActive:      user.IsActive,
    }
}
```

---

## Response DTO Location

DTO 与 mapper 放在对应的 `api/routes/*.go` 文件中：

| Domain | Current file | Current DTO examples |
| --- | --- | --- |
| Auth | `api/routes/auth.go` | `CurrentUserResponse`, `AuthStatusResponse` |
| User | `api/routes/user.go` | `UserResponse`, `ToUserResponse` |
| Dictionary | `api/routes/dictionaries.go` | `DictionaryOptionResponse`, `DictionaryBatchResponse` |
| Order | `api/routes/order.go` | `OrderResponse`, `OrderItemResponse`, `OrderDetailResponse` |
| Domain Event | `api/routes/domain_event.go` | `DomainEventResponse`, `DomainEventDeliveryResponse`, `DomainEventsResponse` |
| Notification | `api/routes/notification.go` | `NotificationResponse`, `NotificationsResponse` |
| Admin | `api/routes/admin.go` | 当前只返回 message，无资源 DTO |
| Open API | `api/routes/open_api_*.go` | 独立公开 DTO，不与内部 API 混用 |

不要新增共享 `api/types`，该目录已退役。

---

## ID Name Pairing

如果 DTO 暴露用于页面展示的关联 ID，应同时暴露后端组装好的名称字段：

```json
{
  "user_id": "u1",
  "user_name": "Ada"
}
```

规则：

* 前端显示 `xxx_name`，不要在 Svelte 里把 `xxx_id` 翻译成名称。
* 后端用 `api/framework/data/namelookup` 做批量去重查询，避免 N+1。
* lookup key 和 model batch loader 绑定在 `api/usecase/translate`。
* usecase 收集列表对象上的 ID 时优先使用 `namelookup.Collect(batch, key, items, idFunc)`，避免每个场景手写重复循环。
* 缺失名称时保留原始 `xxx_id`，`xxx_name` 可为空字符串。

当前示例：

* `OrderResponse` 包含 `user_id` 和 `user_name`。
* `OrderItemResponse` 包含 `product_id` 和 `product_name`。

示例：

```go
names, err := translate.Resolve(ctx.Std(), func(batch *namelookup.Batch) {
    namelookup.Collect(batch, translate.UserDisplayName, orders, func(order models.Order) string {
        return order.UserID
    })
})
```

---

## Auth and User Contract

认证/用户相关内部响应当前规则：

* `POST /api/auth/login`、`POST /api/auth/register` 返回 `AuthTokenResponse`：

```json
{"access_token":"...","token_type":"Bearer","expires_in":604800,"expires_at":"2026-06-13T08:00:00+08:00","user":{"id":"u001","name":"Ada","email":"ada@example.com","email_verified":true}}
```

* `POST /api/auth/logout` 返回 `data.message`，服务端不保存 JWT session，前端负责丢弃本地 token。
* `GET /api/auth/status` 返回 `{logged_in:boolean, user?:{id,name}}`。
* `GET /api/user/me` 返回 `{user:{id,name,email,email_verified}}`；legacy `GET /api/auth/me` 迁移期保留。
* User detail/create/update/delete 返回 `UserResponse` 或空响应；User 管理列表使用分页 envelope。
* `GET /api/admin/users?page=1&page_size=10` 返回 `UsersResponse`，其中 `items` 为 `[]UserResponse`，`pagination` 遵守 [Pagination Contract](#pagination-contract)。legacy `/api/users` 迁移期保留，但也必须走 admin gate。
* `PATCH /api/admin/users/:id/active` 用于启用/禁用用户，request body 为 `{active:boolean}`，返回更新后的 `UserResponse`。legacy `/api/users/:id/active` 迁移期保留，但也必须走 admin gate。
* 禁用用户沿用 `users.is_active=0` 语义；登录和受保护 API 已通过 auth/usecase/middleware 拒绝 disabled user。

敏感字段永远不能出现在 DTO 中：

* `password`
* `password_hash`
* `session_id`
* raw JWT signing secret
* reset token
* raw API key

---

## Scenario: OAuth Login API

### 1. Scope / Trigger

Modify Google OAuth login, GitHub OAuth login, auth callback handling, OAuth identity storage, OAuth env wiring, or frontend token exchange according to this section.

### 2. Signatures

Backend API:

```text
GET  /api/auth/oauth/:provider/start?redirect_path=/orders
GET  /api/auth/oauth/:provider/callback?code=<provider-code>&state=<state>
POST /api/auth/oauth/exchange
```

Exchange request:

```json
{"token":"one-time-result-token"}
```

Usecase:

```go
type OAuthStartCmd struct {
    Provider       string
    RedirectPath   string
    RequestBaseURL string
}

type OAuthCallbackCmd struct {
    Provider       string
    Code           string
    State          string
    RequestBaseURL string
}

type OAuthExchangeCmd struct {
    Token string
}
```

DB:

```sql
oauth_identities(id, provider, provider_user_id, user_id, email, email_verified, display_name, created_at, updated_at, UNIQUE(provider, provider_user_id))
oauth_states(id, state_hash, provider, redirect_path, expires_at, used_at, created_at)
oauth_login_results(id, token_hash, user_id, redirect_path, expires_at, used_at, created_at)
```

### 3. Contracts

* Supported providers are `google` and `github`.
* MVP provider credentials come from runtime env: `GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_CLIENT_SECRET`, `GITHUB_OAUTH_CLIENT_ID`, `GITHUB_OAUTH_CLIENT_SECRET`, and `APP_PUBLIC_BASE_URL`.
* Runtime env may be provided by OS/container variables or dotenv-style files loaded by the executable at startup. System env wins; env files only fill missing keys.
* Callback URLs are `<APP_PUBLIC_BASE_URL>/api/auth/oauth/google/callback` and `<APP_PUBLIC_BASE_URL>/api/auth/oauth/github/callback`.
* `APP_PUBLIC_BASE_URL` is the browser-facing origin. In Vite development it may be `http://127.0.0.1:5173` because Vite proxies `/api` to the Go backend.
* OAuth `state` values and exchange tokens are stored only as SHA-256 hashes.
* The callback must not put an app JWT in the URL. It creates an `oauth_login_results` row and redirects to `/login/oauth/callback?token=<one-time-token>&redirect_path=<path>`.
* `POST /api/auth/oauth/exchange` returns the same `AuthTokenResponse` DTO shape as password login/register.
* Existing linked `(provider, provider_user_id)` wins. Otherwise a verified provider email may auto-link an existing local user or create a new active local user.
* Provider access tokens are not persisted for login-only OAuth.
* Route layer issues the app JWT after exchange; usecase returns `AuthCo`.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| unsupported provider | `CodeValidation`, safe message `OAuth provider is not supported` |
| missing provider env config | `CodeValidation`, safe message `OAuth provider is not configured` |
| missing callback `code` or `state` | `CodeValidation`, safe message `OAuth callback is missing code or state` |
| invalid, expired, or reused state | `CodeValidation`, safe message `OAuth state is invalid or expired` |
| provider returns missing or unverified email | `CodeValidation`, safe message `OAuth verified email is required` |
| disabled linked or matched local user | `CodeForbidden`, safe message `account is disabled` |
| invalid, expired, or reused exchange token | `CodeValidation`, safe message `OAuth login result is invalid or expired` |
| provider request fails | safe usecase error; do not leak provider secrets or raw tokens |

### 5. Good/Base/Bad Cases

Good: User clicks Google on the login page, backend creates hashed state, provider redirects back, backend resolves a verified email, creates a one-time exchange token, and frontend exchanges it for the normal JWT response.

Base: GitHub profile email is empty, so the adapter fetches `/user/emails` and selects a verified email before account mapping.

Bad: Redirect to the frontend with `?access_token=<jwt>`; URLs are logged and copied, so JWTs must never appear there.

### 6. Tests Required

* `api/models/oauth_test.go` covers migration-backed state/result token one-time use and identity uniqueness.
* `api/usecase/auth_oauth_test.go` covers user creation, verified-email auto-linking, unverified email rejection, disabled user rejection, and exchange token one-time use.
* `frontend/src/api.test.js` covers OAuth helper URL generation and exchange token storage.
* `frontend/src/router.test.js` covers `/login/oauth/callback` auth-route classification.
* Run `go test ./api/...`, `cd frontend && npm test`, and `cd frontend && npm run build`.

### 7. Wrong vs Correct

#### Wrong

```go
return c.Redirect(http.StatusFound, "/login?access_token="+jwt)
```

#### Correct

```go
values := url.Values{}
values.Set("token", result.ResultToken)
values.Set("redirect_path", result.RedirectPath)
return c.Redirect(http.StatusFound, "/login/oauth/callback?"+values.Encode())
```

---

## Order Product Admin Contract

订单相关内部响应当前规则：

* `CreateOrderResponse` 是 create route 的 wrapper，包含 `message` 和 `order`。
* `PayOrderResponse` 是 pay route 的 wrapper，包含 `message` 和 `order`。支付成功必须通过 `POST /api/orders/:id/pay`，不能通过 `PATCH /api/orders/:id/status` 直接设置 `paid`。
* `OrderDetailResponse` 是 detail route 的 wrapper，包含 `order` 和 `items`。
* 当前用户下单使用 `POST /api/user/orders`，request 不接受 owner 语义的 `user_id`；route 必须从 current user 推导 `CreateOrderCmd.UserID`。legacy `POST /api/orders` 迁移期保留，但必须限制为 `body.user_id == current user` 或 admin。
* 当前用户订单 list 使用 `GET /api/user/orders?page=1&page_size=10&status=pending`，返回 `UserOrdersResponse`，其中 `items` 为 `[]OrderResponse`，`pagination` 遵守 [Pagination Contract](#pagination-contract)。owner 过滤必须来自当前 `ctx.Actor.UserID`，不能由客户端传入。
* 管理端订单 list 使用 `GET /api/admin/orders?user_id=<id>&status=pending&page=1&page_size=10`，必须 admin，可选按 `user_id` 和 `status` 筛选。
* Legacy 订单 list 路径 `GET /api/orders/user/:user_id` 仅迁移期保留，必须限制为 `:user_id == ctx.Actor.UserID` 或 `ctx.Actor.IsAdmin == true`。
* `UpdateOrderStatus` 返回 `data.message`。
* Creem checkout MVP 中，`POST /api/user/orders` 只要求 `product_id`，创建 `pending` 本地订单台账；`items` 可作为 legacy payload 被接受，但当前支付流不使用本地 `quantity`、`products.stock` 或本地价格。
* 新建 Creem checkout 台账订单的 `amount` 可以为 `0`；真实收费金额由 Creem checkout 的配置产品决定，前端不应把本地 `0` 渲染成实际收费金额。

产品列表通过 `GET /api/products` 返回 `[]ProductResponse`，用于 legacy/demo/admin 商品展示。产品 DTO 只暴露 `id`、`name`、`description`、`price`、`stock`，不返回 `models.Product`。

积分相关内部响应当前规则：

* `GET /api/user/points` 返回 `PointsResponse`，字段为 `user_id` 和 `balance`；legacy `GET /api/points/me` 迁移期保留。
* `GET /api/user/realtime/ws?access_token=<jwt>` 是当前用户自服务 WebSocket 实时通道，不使用 HTTP JSON envelope。连接成功后以 text frame 推送 realtime envelope，例如 `{"type":"points","presentation":"refresh","payload":{"user_id":"...","client_id":"...","balance":10}}`。
* `POST /api/user/notifications/test-export-toast` 是登录态验证入口，返回 `data.message`，并向当前用户发布 `async_export_task` + `toast` realtime envelope；legacy `/api/notifications/test-export-toast` 迁移期保留。

Admin routes 统一放在 `/api/admin/...`；legacy 管理路径迁移期可以保留，但必须挂在 `RequireAdmin()` 后。当前包括 users、orders status、dictionary management、products write、scheduler、events、messages、parameters、notifications、settings upload、variables 和 `POST /api/admin/reload-shared-db`。

---

## Scenario: Async Order Excel Export API

### 1. Scope / Trigger

Modify order export, async task result download, OSS-backed export files, `POST /api/user/orders/export`, `POST /api/admin/orders/export`, `GET /api/user/tasks/:id/download`, or the frontend order export/task download helpers according to this section.

### 2. Signatures

Backend API:

```text
POST /api/user/orders/export?status=paid
POST /api/admin/orders/export?user_id=<id>&status=paid
GET  /api/user/tasks/:id/download
GET  /api/user/tasks?page=1&page_size=20
```

Usecase:

```go
type EnqueueMyOrdersExcelExportQry struct {
    Status string
}

type EnqueueAdminOrdersExcelExportQry struct {
    UserID string
    Status string
}

type OrderExportTaskCo struct {
    TaskID string
}

type TaskDownloadQry struct {
    TaskID string
}

type TaskDownloadCo struct {
    URL       string
    ExpiresAt string
    Filename  string
}
```

Task payload/result JSON:

```json
{"scope":"user","requester_user_id":"u1","user_id":"u1","status":"paid","created_at":"2026-06-12T00:00:00Z"}
```

```json
{"object_key":"exports/orders/2026/06/task.xlsx","filename":"orders-20260612-120000.xlsx","content_type":"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet","size":12345,"row_count":10,"channel_code":"primary-r2","provider_code":"cloudflare_r2","adapter_key":"oss.cloudflare_r2.s3_compatible"}
```

### 3. Contracts

* Current-user export owner filter must come from `ctx.Actor.UserID`; clients cannot pass `user_id` to `/api/user/orders/export`.
* Admin export requires `ctx.Actor.IsAdmin == true`; optional `user_id` and `status` filters must match `GET /api/admin/orders` semantics.
* Count matching orders before enqueue. If count is greater than `100000`, return validation error and do not insert `async_tasks`.
* Export worker must re-derive the scoped `OrderQuery` from task payload. User-scope payload must force `UserID = requester_user_id` even if payload contains another `user_id`.
* Worker must use bounded batches from DB and Excel streaming writer. Do not build a full in-memory slice of all matching orders.
* XLSX bytes are written to a temporary file, then uploaded through the configured primary OSS provider. DB task result stores object metadata, not file bytes.
* Download route must verify `async_tasks.user_id == ctx.Actor.UserID`, task status is `completed`, and task type is `orders_excel_export` before returning a presigned GET URL.
* Presigned download URL TTL is one hour for MVP.
* Terminal success/failure must refresh the TaskCenter and create a realtime notification with `source_type="async_task"` for the requesting user.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| unauthenticated `/api/user/orders/export` | `CodeUnauthorized`, safe message `not logged in` |
| non-admin `/api/admin/orders/export` | `CodeForbidden`, safe message `admin access is required` |
| invalid `status` | `CodeValidation`, safe message `invalid order status` |
| primary OSS provider missing before enqueue | validation/internal safe error; no queued export task |
| matching count > `100000` | `CodeValidation`, safe message `order export cannot exceed 100000 rows`; no queued export task |
| queue manager missing | `CodeInternal`, safe message `queue manager is not configured` |
| worker storage/upload failure | task retries; after max retries task becomes `failed` and notification is emitted |
| download task not found | `CodeNotFound`, safe message `task not found` |
| download by non-owner | `CodeForbidden`, safe message `cannot download another user's task result` |
| download before completion | `CodeConflict`, safe message `task is not completed` |
| completed task has no export result/object key | `CodeInternal`, safe message `task result is missing export object` |

### 5. Good/Base/Bad Cases

Good: User clicks Export on `/app/orders`; frontend calls `exportMyOrders()`; backend enqueues `orders_excel_export`; worker streams batches into XLSX, uploads to OSS, marks the task completed with result JSON, creates a notification, and TaskCenter shows a download button that calls `GET /api/user/tasks/:id/download`.

Base: Admin calls `/api/admin/orders/export?user_id=u1&status=paid`; task owner is the admin actor, but exported data is filtered to user `u1` because the admin scope explicitly allows cross-user query.

Bad: Worker trusts `payload.user_id` for user-scope tasks. User-scope exports must always force `payload.requester_user_id` as the owner filter.

Bad: Route returns `object_key` directly as a public URL or lets the client presign arbitrary keys. Download must go through an owner-scoped task endpoint.

### 6. Tests Required

* Usecase tests cover current-user scope, admin scope, non-admin rejection, invalid status, row-count limit before enqueue, worker completion, result JSON, notification row creation, and owner-only download.
* Model tests or usecase integration tests cover bounded iteration with keyset order `created_at DESC, id DESC`.
* Frontend API tests cover `exportMyOrders()`, `exportAdminOrders()`, and `getTaskDownload()` paths/methods.
* Run `go test ./...`, `cd frontend && npm test`, `cd frontend && npm run build`, and `git diff --check`.

### 7. Wrong vs Correct

#### Wrong

```go
orders, _ := models.ListOrders(ctx, models.OrderQuery{Limit: total})
_ = xlsx.WriteAll(orders)
```

#### Correct

```go
err := models.IterateOrders(ctx, query, 1000, func(batch []models.Order) error {
    return streamRows(batch)
})
```

#### Wrong

```go
return httpresponse.OK(c, map[string]string{"url": publicBaseURL + objectKey})
```

#### Correct

```go
download, err := usecase.GetMyTaskDownload(ctx, usecase.TaskDownloadQry{TaskID: c.Param("id")})
return httpresponse.OK(c, TaskDownloadResponse{URL: download.URL, ExpiresAt: download.ExpiresAt, Filename: download.Filename})
```

---

## Scenario: LLM Summary API

### 1. Scope / Trigger

Modify text summarization, DeepSeek-backed LLM operation wiring, `POST /api/llm/summaries`, or the frontend `summarizeTextWithLLM()` helper according to this section.

### 2. Signatures

Backend API:

```text
POST /api/llm/summaries
```

Request:

```json
{
  "text": "source text",
  "prompt": "summarize for an executive audience",
  "dimensions": ["summary"]
}
```

Usecase:

```go
type SummarizeTextWithLLMCmd struct {
    Text       string
    Prompt     string
    Dimensions []string
}
```

Response payload inside the internal success envelope:

```json
{
  "summary": {"summary": "concise result"},
  "model_code": "summary-fast",
  "channel_code": "deepseek-prod",
  "invocation_id": "invocation-id"
}
```

### 3. Contracts

* `text` is required and is the original source text.
* `prompt` is optional for backward compatibility, but experiment/chat-style callers should send it to describe summary style, language, and focus.
* `dimensions` is required and controls the exact JSON keys expected from the provider response. A single summary demo should pass `["summary"]`.
* The usecase loads the enabled LLM channel/model from integration configuration for `scenario=llm` and operation `text_summary`.
* For MVP compatibility with Parameter-created DeepSeek channels, if `integration_operation_configs` or `integration_model_options` are absent, the resolver may select the enabled `llm.deepseek.openai_compatible` channel by priority and synthesize a `deepseek-chat` model option. Explicit model options still take precedence.
* Provider output must be parsed as a strict JSON object containing every requested dimension key.
* Route layer maps `usecase.LLMSummaryCo` to route-local `LLMSummaryResponse` and returns `httpresponse.OK`.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| empty `text` | `CodeValidation`, safe message `text is required` |
| `text` over 20000 runes | `CodeValidation`, safe message `text is too long` |
| `prompt` over 4000 runes | `CodeValidation`, safe message `prompt is too long` |
| empty `dimensions` | `CodeValidation`, safe message `dimensions are required` |
| more than 8 dimensions | `CodeValidation`, safe message `too many dimensions` |
| missing LLM channel/config/credential, or unsupported missing model option | `CodeInternal`, safe message `LLM channel is not configured` |
| provider request fails | `CodeInternal`, safe message `failed to generate summary`; invocation is marked failed |
| provider JSON missing a requested key | `CodeInternal`, safe message `failed to parse LLM summary`; invocation is marked failed |

### 5. Good/Base/Bad Cases

Good: Experiment UI calls `summarizeTextWithLLM({text, prompt, dimensions:['summary']})`; usecase sends both source text and requirement prompt to DeepSeek and returns `summary.summary`.

Base: Legacy caller omits `prompt` and still receives dimension-based summary output.

Base: A DeepSeek channel created from Parameter without operation/model rows still resolves through the enabled channel and the `deepseek-chat` fallback model.

Bad: Route returns raw provider text directly or accepts free-form output without checking requested dimension keys; this breaks frontend rendering assumptions and invocation auditability.

### 6. Tests Required

* `api/usecase/llm_summary_test.go` covers config loading, prompt inclusion in the provider request, JSON parsing, metadata, invocation success, usage recording, and validation.
* `api/models/integration_test.go` covers explicit LLM model lookup and Parameter-created DeepSeek channel-only fallback to `deepseek-chat`.
* `api/routes/llm_test.go` covers internal envelope, request DTO binding including `prompt`, and response DTO shape.
* `frontend/src/api.test.js` covers helper path, method, and JSON body.
* Run `go test ./api/usecase -run TestSummarizeTextWithLLM`, `go test ./api/routes -run TestSummarizeTextWithLLM`, and `cd frontend && npm test`.

### 7. Wrong vs Correct

#### Wrong

```go
return httpresponse.OK(c, result.Content)
```

#### Correct

```go
summary, err := usecase.SummarizeTextWithLLM(ctx, usecase.SummarizeTextWithLLMCmd{
    Text:       req.Text,
    Prompt:     req.Prompt,
    Dimensions: req.Dimensions,
})
return httpresponse.OK(c, ToLLMSummaryResponse(summary))
```

---

## Scenario: User Management API

### 1. Scope / Trigger

修改 User 管理页面、`GET /api/admin/users` 分页契约、`users.is_active` 启禁用语义、或前端 `listUsers` / `setUserActive` helper 时，遵守本节。

### 2. Signatures

Backend API:

```text
GET /api/admin/users?page=1&page_size=10
PATCH /api/admin/users/:id/active
```

Patch request:

```json
{"active":false}
```

Usecase:

```go
type ListUsersQry struct {
    Page     int
    PageSize int
}

type SetUserActiveCmd struct {
    ID     string
    Active bool
}

func ListUsers(ctx fwusecase.Context, qry ListUsersQry) (UsersCo, error)
func SetUserActive(ctx fwusecase.Context, cmd SetUserActiveCmd) (UserCo, error)
```

DB:

```sql
users(id, name, email, password_hash, email_verified, is_active, created_at, updated_at)
```

### 3. Contracts

* `GET /api/admin/users` uses [Pagination Contract](#pagination-contract) and returns `UsersResponse`.
* `UsersResponse.items` is `[]UserResponse` with `id`, `name`, `email`, `email_verified`, `is_active`, `created_at`, and `updated_at`.
* `UserResponse` must never include `password_hash`, reset tokens, sessions, or JWT details.
* User list ordering must be stable: `ORDER BY created_at DESC, id DESC`.
* `PATCH /api/admin/users/:id/active` only mutates `users.is_active` and `updated_at`; it does not delete rows or change profile fields.
* Disabling the current logged-in user is forbidden. Usecase checks `ctx.Actor.UserID` and returns validation error before touching storage.
* A disabled user cannot log in and cannot continue using protected API requests because auth middleware reloads the user and checks `is_active`.
* route layer uses route-local DTOs and `httpresponse.OK`; it must not return `models.User` directly.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| invalid `page` / `page_size` | `CodeValidation` -> internal `400` envelope |
| empty `id` in `SetUserActiveCmd` | `CodeValidation`, safe message `missing user ID` |
| disabling current logged-in user | `CodeValidation`, safe message `cannot disable current user` |
| target user does not exist | `CodeNotFound`, safe message `user not found` |
| database query/update fails | `CodeInternal`, safe message `failed to load users`, `failed to count users`, or `failed to update user active state` |

### 5. Good/Base/Bad Cases

Good: User page calls `GET /api/admin/users?page=1&page_size=10`, then calls `PATCH /api/admin/users/:id/active` from a row action and refreshes the current page.

Base: Re-enabling a disabled user returns the updated `UserResponse` with `is_active:true`; the page refreshes the row/list.

Bad: Use `DELETE /api/admin/users/:id` to disable a user; this removes the row and breaks account history and foreign-key relationships.

### 6. Tests Required

* `api/usecase/user_management_test.go` covers pagination metadata, stable ordering, active-state toggling, missing user, and current-user disable validation.
* `api/routes/user_management_test.go` covers internal envelope, pagination query validation, toggle response shape, current-user disable validation, and not found mapping.
* `frontend/src/api.test.js` covers `listUsers({page,pageSize})` and `setUserActive(id, active)` paths, verbs, encoded IDs, and JSON body.
* `frontend/src/router.test.js` covers `/users.html` alias, menu label, app-route detection, and title.
* Run `go test ./...`, `cd frontend && npm test`, and `cd frontend && npm run build`.

### 7. Wrong vs Correct

#### Wrong

```go
return httpresponse.OK(c, models.User{})
```

#### Correct

```go
users, err := usecase.ListUsers(ctx, qry)
return httpresponse.OK(c, ToUsersResponse(users))
```

#### Wrong

```go
_ = models.DeleteUser(ctx.Std(), userID)
```

#### Correct

```go
user, err := usecase.SetUserActive(ctx, usecase.SetUserActiveCmd{
    ID:     userID,
    Active: false,
})
```

---

## Scenario: Event Management API

### 1. Scope / Trigger

修改 Event 管理页面、`domain_events` / `domain_event_deliveries` 只读查询、或前端 `listEvents` / `listEventDeliveries` helper 时，遵守本节。

### 2. Signatures

Backend API:

```text
GET /api/admin/events?page=1&page_size=10
GET /api/admin/events/:id/deliveries
```

Usecase:

```go
type DomainEventsQry struct {
    Page     int
    PageSize int
}

type DomainEventDeliveriesQry struct {
    EventID string
}

func ListDomainEvents(ctx fwusecase.Context, qry DomainEventsQry) (DomainEventsCo, error)
func ListDomainEventDeliveries(ctx fwusecase.Context, qry DomainEventDeliveriesQry) ([]DomainEventDeliveryCo, error)
```

DB:

```sql
domain_events(id, topic, aggregate_type, aggregate_id, payload_json, metadata_json, occurred_at, created_at)
domain_event_deliveries(id, event_id, subscriber, message_id, status, attempts, last_error, created_at, updated_at)
```

### 3. Contracts

* `GET /api/admin/events` uses [Pagination Contract](#pagination-contract) and returns `DomainEventsResponse`.
* `DomainEventsResponse.items` is `[]DomainEventResponse` with `id`, `topic`, `aggregate_type`, `aggregate_id`, `payload_json`, `metadata_json`, `occurred_at`, and `created_at`.
* `GET /api/admin/events/:id/deliveries` returns `[]DomainEventDeliveryResponse` with `id`, `event_id`, `subscriber`, `message_id`, `status`, `attempts`, `last_error`, `created_at`, and `updated_at`.
* Event management API is read-only: no replay, no retry, and no delivery state mutation.
* Event list ordering must be stable: `ORDER BY created_at DESC, id DESC`.
* Delivery list is queried by one event id and ordered for fan-out scanning: `ORDER BY created_at ASC, subscriber ASC`.
* route layer uses route-local DTOs and `httpresponse.OK`; it must not return `models.DomainEvent` or `models.DomainEventDelivery` directly.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| invalid `page` / `page_size` | `CodeValidation` -> internal `400` envelope |
| empty `event_id` in usecase query | `CodeValidation`, safe message `event ID is required` |
| selected event has no delivery rows | success envelope with empty `[]` |
| database query fails | `CodeInternal`, safe message `failed to load domain events` or `failed to load domain event deliveries` |

### 5. Good/Base/Bad Cases

Good: Event page calls `GET /api/admin/events?page=1&page_size=10`, then lazy-loads `GET /api/admin/events/:id/deliveries` when the user selects an event.

Base: An event with no delivery rows returns an empty list and the page shows an empty state; do not treat this as 404.

Bad: Read or mutate `goqite` to infer domain event delivery state; business delivery state belongs to `domain_event_deliveries`.

### 6. Tests Required

* `api/usecase/domain_event_test.go` covers event pagination metadata, stable ordering, delivery lookup by event, and empty event ID validation.
* `api/routes/domain_event_test.go` covers internal envelope, pagination query validation, and delivery DTO shape.
* `frontend/src/api.test.js` covers `listEvents({page,pageSize})` and `listEventDeliveries(eventId)` paths and URL encoding.
* `frontend/src/router.test.js` covers `/events` route alias, menu label, app-route detection, and title.
* Run `go test ./...`, `cd frontend && npm test`, and `cd frontend && npm run build`.

### 7. Wrong vs Correct

#### Wrong

```go
return httpresponse.OK(c, models.DomainEvent{})
```

#### Correct

```go
events, err := usecase.ListDomainEvents(ctx, qry)
return httpresponse.OK(c, ToDomainEventsResponse(events))
```

---

## Scenario: Notification Center API

### 1. Scope / Trigger

修改 Notification Center、`notifications` 台账、业务通知创建 usecase、realtime notification envelope、或 admin `/api/notifications` 查询接口时，遵守本节。

Notification Center 的定位是业务通知入口与台账，不替代外部渠道 port/provider adapter。业务场景需要发送用户通知时默认调用 `usecase.CreateNotification(...)`；也可以由领域事件 subscriber 调用该 usecase。纯技术实时刷新、渠道连通性测试、provider 防腐层内部逻辑，才直接调用对应 port 或 realtime primitive。

### 2. Signatures

Admin query API:

```text
GET /api/notifications?page=1&page_size=10&type=realtime&email=ada@example.com&phone=138
```

Usecase:

```go
type CreateNotificationCmd struct {
    NotificationType string
    SourceType       string
    SourceID         string
    UserID           string
    RecipientEmail   string
    RecipientPhone   string
    Title            string
    Summary          string
    PayloadJSON      string
}

func CreateNotification(ctx fwusecase.Context, cmd CreateNotificationCmd) (NotificationCo, error)
func ListNotifications(ctx fwusecase.Context, qry NotificationsQry) (NotificationsCo, error)
```

DB:

```sql
notifications(
  id, notification_type, source_type, source_id, user_id,
  recipient_email, recipient_phone, title, summary, payload_json,
  status, last_error, sent_at, created_at, updated_at
)
dictionary_types(type_key='notification_type')
dictionary_values(value_code='realtime'|'sms'|'email'|'wechat_official_account')
```

### 3. Contracts

* Notification type is dictionary-managed through `notification_type`; usecases validate against enabled dictionary values.
* Notification status is code-owned, not dictionary-managed. Allowed values are `pending`, `sent`, `failed`, and `skipped`; DB must enforce them with `CHECK`.
* There is no admin HTTP create API for MVP. Creation is usecase-only so business usecases and event subscribers share the same entry.
* `GET /api/notifications` is admin-only and uses the standard pagination contract.
* Filter parameters are `type`, `email`, and `phone`; email/phone are substring filters over `recipient_email` and `recipient_phone`.
* List DTO includes both `notification_type` and `notification_type_label`.
* Non-realtime types are ledger-only in the MVP and must be stored as `skipped`; they do not call SMS, email, or WeChat provider ports yet.
* Realtime notifications must create the ledger record first, publish realtime message type `notification`, then update status to `sent` or `failed`.
* Realtime notification payload is a safe presentation snapshot only: `id`, `title`, `summary`, `source_type`, and `source_id`. Do not push raw `payload_json` or provider/channel secrets.
* `payload_json` must be a JSON object and remains an internal ledger payload.

Successful list response:

```json
{
  "items": [
    {
      "id": "notification-id",
      "notification_type": "realtime",
      "notification_type_label": "Realtime",
      "source_type": "order",
      "source_id": "order-id",
      "user_id": "user-id",
      "recipient_email": "",
      "recipient_phone": "",
      "title": "Order paid",
      "summary": "Your points have been awarded",
      "payload_json": "{}",
      "status": "sent",
      "last_error": "",
      "sent_at": "2026-06-07 10:00:00",
      "created_at": "2026-06-07 10:00:00",
      "updated_at": "2026-06-07 10:00:00"
    }
  ],
  "pagination": {}
}
```

Realtime message:

```json
{"type":"notification","presentation":"toast","payload":{"id":"notification-id","title":"Order paid","summary":"Your points have been awarded","source_type":"order","source_id":"order-id"}}
```

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| missing or unknown `notification_type` | `CodeValidation`, safe message `notification type is required` or `notification type is invalid` |
| disabled dictionary value | `CodeValidation`, same as unknown type |
| missing title | `CodeValidation`, safe message `notification title is required` |
| realtime without `user_id` | `CodeValidation`, safe message `user ID is required for realtime notification` |
| invalid or non-object `payload_json` | `CodeValidation` |
| realtime publish fails | ledger status becomes `failed`, `last_error` stores a safe message, caller receives `CodeInternal` |
| invalid pagination query | `CodeValidation` -> internal `400` envelope |
| non-admin list request | middleware returns `403` before handler |

### 5. Good/Base/Bad Cases

Good: Order payment usecase calls `CreateNotification` with `notification_type=realtime`, `source_type=order`, `source_id=...`, and a safe title/summary; user receives a toast and admin can inspect the ledger row.

Base: A future SMS notification creates a `skipped` ledger row until the SMS delivery chain is implemented behind Notification Center.

Bad: Business usecase calls SMS provider port directly and separately inserts a notification row; this splits status and audit behavior.

### 6. Tests Required

* `api/usecase/notification_test.go` covers realtime create/publish/status, non-realtime skipped ledger, dictionary type validation, and list filters.
* `api/routes/notification_test.go` covers admin list DTO envelope, pagination, filters, and invalid type mapping.
* `api/framework/realtime/realtime_test.go` covers `notification` default presentation.
* `frontend/src/api.test.js`, `frontend/src/router.test.js`, and `frontend/src/realtimeMessages.test.js` cover helper path, admin route, and realtime toast behavior.
* Run `go test ./...`, `cd frontend && npm test`, and `cd frontend && npm run build`.

### 7. Wrong vs Correct

#### Wrong

```go
_ = smsPort.Send(ctx, req)
_ = models.InsertNotification(ctx.Std(), row)
```

#### Correct

```go
notification, err := usecase.CreateNotification(ctx, usecase.CreateNotificationCmd{
    NotificationType: usecase.NotificationTypeRealtime,
    UserID: userID,
    Title: "Order paid",
    Summary: "Your points have been awarded",
})
```

---

## Scenario: Dictionary Management API

### 1. Scope / Trigger

Modify Dictionary management, `/api/dictionary*` routes, `/api/public/dictionaries` lookup behavior, `dictionary_types` / `dictionary_values` DB schema, or frontend dictionary helpers according to this section. Dictionaries provide selectable values for UI and business forms; they are not credential or secret storage.

### 2. Signatures

Lookup API:

```text
GET /api/public/dictionaries?types=product_category,region
```

Management API:

```text
GET   /api/dictionary/types
POST  /api/dictionary/types
PUT   /api/dictionary/types/:id
PATCH /api/dictionary/types/:id/enabled
GET   /api/dictionary/types/:type_id/values
POST  /api/dictionary/types/:type_id/values
PUT   /api/dictionary/types/:type_id/values/:id
PATCH /api/dictionary/values/:id/enabled
```

Request DTOs:

```json
{"type_key":"order_status","name":"Order status","enabled":true,"description":"Order lifecycle"}
```

```json
{"value_code":"pending","label":"Pending","sort_order":10,"enabled":true,"description":"Waiting for payment"}
```

Usecase:

```go
func GetDictionaries(ctx fwusecase.Context, qry DictionaryBatchQry) (DictionaryBatchCo, error)
func ListDictionaryTypes(ctx fwusecase.Context, qry ListDictionaryTypesQry) ([]DictionaryTypeCo, error)
func CreateDictionaryType(ctx fwusecase.Context, cmd SaveDictionaryTypeCmd) (DictionaryTypeCo, error)
func UpdateDictionaryType(ctx fwusecase.Context, cmd SaveDictionaryTypeCmd) (DictionaryTypeCo, error)
func SetDictionaryTypeEnabled(ctx fwusecase.Context, cmd SetDictionaryTypeEnabledCmd) (DictionaryTypeCo, error)
func ListDictionaryValues(ctx fwusecase.Context, qry ListDictionaryValuesQry) ([]DictionaryValueCo, error)
func CreateDictionaryValue(ctx fwusecase.Context, cmd SaveDictionaryValueCmd) (DictionaryValueCo, error)
func UpdateDictionaryValue(ctx fwusecase.Context, cmd SaveDictionaryValueCmd) (DictionaryValueCo, error)
func SetDictionaryValueEnabled(ctx fwusecase.Context, cmd SetDictionaryValueEnabledCmd) (DictionaryValueCo, error)
```

DB:

```sql
dictionary_types(id, type_key, name, enabled, description, created_at, updated_at)
dictionary_values(id, dictionary_type_id, value_code, label, sort_order, enabled, description, created_at, updated_at)
```

### 3. Contracts

* Existing `GET /api/public/dictionaries?types=...` response shape stays `{dictionaries:{type:[{value,label}]}}`.
* Lookup API returns only enabled dictionary types and enabled dictionary values.
* Lookup API returns an empty array for a requested missing or disabled type.
* `type_key` and `value_code` are normalized to lowercase and must match `^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`.
* `type_key` is globally unique.
* `value_code` is unique within one dictionary type.
* Value ordering is stable: `sort_order ASC, value_code ASC`.
* Management routes are authenticated and use route-local DTOs plus `httpresponse`.
* Route DTOs must not return `models.DictionaryType` or `models.DictionaryValue` directly.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| missing `types` on lookup | `400` validation envelope, safe message `types is required` |
| missing or invalid `type_key` | `CodeValidation` |
| missing dictionary type name | `CodeValidation` |
| duplicate `type_key` | `CodeConflict`, safe message `dictionary type already exists` |
| missing or invalid `value_code` | `CodeValidation` |
| missing value label | `CodeValidation` |
| duplicate `(dictionary_type_id, value_code)` | `CodeConflict`, safe message `dictionary value already exists` |
| missing type when listing/creating/updating values | `CodeNotFound`, safe message `dictionary type not found` |
| update/toggle missing value | `CodeNotFound`, safe message `dictionary value not found` |

### 5. Good/Base/Bad Cases

Good: Add `order_status` with `pending`, `paid`, and `cancelled` values; forms load it through `getDictionaries(['order_status'])`.

Base: Disable one value so management still shows it, but lookup consumers no longer receive it.

Bad: Store provider API keys or secrets as dictionary values; use credential-specific storage for secrets.

### 6. Tests Required

* `api/models/dictionary_test.go` covers CRUD, enabled lookup filtering, ordering, and duplicate conflicts.
* `api/usecase/dictionary_test.go` covers key normalization, validation, conflict mapping, and enable/disable.
* `api/routes/dictionaries_test.go` covers lookup envelope and management DTOs.
* `frontend/src/api.test.js` covers dictionary management helper paths, methods, encoded IDs, and bodies.
* `frontend/src/router.test.js` covers `/dictionary`, `/dictionary.html`, menu label, app-route detection, and title.
* Run `go test ./...`, `cd frontend && npm test`, and `cd frontend && npm run build`.

### 7. Wrong vs Correct

#### Wrong

```go
return httpresponse.OK(c, models.DictionaryValue{})
```

#### Correct

```go
value, err := usecase.CreateDictionaryValue(ctx, cmd)
return httpresponse.Created(c, toDictionaryValueResponse(value))
```

---

## Scenario: Parameter Integration Channel API

### 1. Scope / Trigger

Modify `Parameter` management, external integration channel configuration, credential write boundaries, adapter schema registry, `/api/admin/parameters/*` routes, or frontend parameter helpers according to this section. This scenario is the management-surface slice of the external integration anti-corruption layer.

### 2. Signatures

Backend API:

```text
GET   /api/admin/parameters/integration-schemas?scenario=payment|llm|sms|email|oss
GET   /api/admin/parameters/integration-channels?scenario=payment|llm|sms|email|oss
POST  /api/admin/parameters/integration-channels
PUT   /api/admin/parameters/integration-channels/:id
PATCH /api/admin/parameters/integration-channels/:id/enabled
```

Request DTO:

```json
{
  "scenario": "payment",
  "channel_code": "creem-prod",
  "provider_code": "creem",
  "adapter_key": "payment.creem.hosted_checkout",
  "environment": "production",
  "enabled": true,
  "priority": 100,
  "webhook_enabled": true,
  "is_primary": false,
  "config_json": "{\"base_url\":\"https://api.creem.io\",\"product_id\":\"prod_123\"}",
  "metadata_json": "{}",
  "credential_type": "payment_bundle",
  "credential_value": "{\"api_key\":\"...\",\"webhook_secret\":\"...\"}"
}
```

Usecase:

```go
func ListParameterIntegrationSchemas(ctx fwusecase.Context, qry ListParameterIntegrationSchemasQry) ([]ParameterIntegrationAdapterSchemaCo, error)
func ListParameterIntegrationChannels(ctx fwusecase.Context, qry ListParameterIntegrationChannelsQry) ([]ParameterIntegrationChannelCo, error)
func CreateParameterIntegrationChannel(ctx fwusecase.Context, cmd SaveParameterIntegrationChannelCmd) (ParameterIntegrationChannelCo, error)
func UpdateParameterIntegrationChannel(ctx fwusecase.Context, cmd SaveParameterIntegrationChannelCmd) (ParameterIntegrationChannelCo, error)
func SetParameterIntegrationChannelEnabled(ctx fwusecase.Context, cmd SetParameterIntegrationChannelEnabledCmd) (ParameterIntegrationChannelCo, error)
```

DB:

```sql
integration_channels(id, scenario, channel_code, provider_code, adapter_key, environment, enabled, priority, credential_id, policy_id, webhook_enabled, is_primary, config_json, metadata_json)
integration_webhook_receipts(id, scenario, channel_id, channel_code, provider_code, provider_event_id, idempotency_key, status, attempts, message_id)
integration_credentials(id, credential_type, value_text, enabled, rotated_at)
dictionary_types(type_key='integration_environment')
dictionary_values(value_code='test'|'production')
dictionary_types(type_key='integration_credential_type')
dictionary_values(value_code='payment_bundle'|'api_key'|'smtp_password'|'s3_access_key')
```

`smtp_password` is present in baseline `002_seed.sql` and is also backfilled for already-migrated databases by `008_add_email_integration_seed.sql`. `s3_access_key` is present in baseline `002_seed.sql` and is backfilled by `011_add_oss_integration_seed.sql`.

### 3. Contracts

#### Adapter Schema Contract

* `GET /api/admin/parameters/integration-schemas` 返回 code-owned adapter schema DTO；该 DTO 用于前端动态渲染，也用于后端保存时校验。
* schema 按 `adapter_key` 定义，DB 只存 `adapter_key` 字符串，不存可执行逻辑。
* schema DTO 字段包括 `scenario`、`adapter_key`、`label`、`description`、`provider_code`、`credential_type`、`credential_format`、`advanced_json`、`config_fields`、`credential_fields`。
* field DTO 字段包括 `key`、`label`、`kind`、`required`、`placeholder`、`help_text`、`default_value`、`dictionary_type`、`sensitive`、`options`。
* `credential_format` 当前只允许 `plain` 和 `json_object`：`plain` 表示 `credential_value` 是原始字符串；`json_object` 表示 `credential_value` 是 JSON object 字符串。
* `advanced_json=true` 表示前端可以保留折叠的 Advanced JSON 编辑区；schema 外的额外 `config_json` key 可以保留，但仍不得含敏感 key。
* provider API URL / `base_url` 这类自由 URL 字段应使用 `kind=url`，且不设置 `dictionary_type` 或 `options`；Payment、LLM、SMS、Email 保持一致由用户输入 URL。

当前 code-owned schemas：

| Adapter key | Scenario | Provider | Credential format | Config fields | Credential fields |
| --- | --- | --- | --- | --- | --- |
| `payment.creem.hosted_checkout` | `payment` | `creem` | `json_object` | `base_url` required URL, `product_id` required text, optional `success_url` URL, optional `units` number | `api_key`, `webhook_secret` |
| `llm.deepseek.openai_compatible` | `llm` | `deepseek` | `plain` | `base_url` required URL | `api_key` |
| `sms.aliyun.adapter` | `sms` | `aliyun` | `plain` | optional `base_url` URL, optional `sign_name` text, optional `template_code` text | `api_key` |
| `email.aliyun.smtp` | `email` | `aliyun` | `json_object` | `smtp_host` required text default `smtp.qiye.aliyun.com`, `smtp_port` required number default `465`, `security` required option default `ssl`, `from_email` required text, optional `from_name` text | `username`, `password` with help text reminding admins to use the mailbox client authorization password instead of the account login password |
| `email.resend.api` | `email` | `resend` | `plain` | `base_url` required URL default `https://api.resend.com`, `from_email` required text, optional `from_name` text | `api_key` |
| `oss.cloudflare_r2.s3_compatible` | `oss` | `cloudflare_r2` | `json_object` | `endpoint_url` required URL, `bucket` required text, optional `region` default `auto`, optional `use_path_style` boolean default `true`, optional `public_base_url` URL, optional `key_prefix` text | `access_key_id`, `secret_access_key` |
| `oss.aliyun_oss.s3_compatible` | `oss` | `aliyun` | `json_object` | `endpoint_url` required URL, `bucket` required text, optional `region`, optional `use_path_style` boolean override, optional `public_base_url` URL, optional `key_prefix` text | `access_key_id`, `secret_access_key` |

* API only manages `integration_channels + integration_credentials`; it does not manage `integration_operation_configs`, `integration_model_options`, policy, webhook receipts, invocation raw data, provider request/response, prompt, stream chunks, OSS upload/download runtime, presigned URLs, or artifact lifecycle.
* Parameter APIs are protected internal admin configuration APIs. Routes must run behind `RequireAuth()` and `RequireAdmin()`; admin access is represented by `users.is_admin=1`.
* `scenario` only allows `payment`, `llm`, `sms`, `email`, and `oss`.
* `credential_type` uses the `integration_credential_type` dictionary. Current seeded values are `payment_bundle`, `api_key`, `smtp_password`, and `s3_access_key`; save usecases reject values that are not enabled dictionary values.
* `config_json` and `metadata_json` must be JSON objects and may only contain non-sensitive config. Obvious sensitive keys such as `api_key`, `secret`, `password`, `token`, and `private_key` are rejected recursively, including inside arrays.
* OSS `use_path_style` is an optional boolean addressing-style override. Cloudflare R2 schema defaults it to `true`; Aliyun OSS should usually omit it so the provider adapter uses AWS SDK virtual-hosted addressing by default.
* create requires `credential_value`; update with empty `credential_value` preserves the existing credential value.
* `credential_value` is stored as administrator-managed configuration in `integration_credentials.value_text`; it is not encrypted by this module. Legacy `ciphertext/key_version/masked_value` columns may exist for migration compatibility, but they are not part of the current API contract.
* Responses return `credential_type` and `credential_value` for the protected admin page. They must never return `credential_plaintext`, `ciphertext`, `key_version`, or `masked_value`.
* `is_primary` is meaningful only for `scenario=oss`; it marks the primary OSS provider/channel. Zero primary OSS channels is valid, and there must be at most one primary OSS channel.
* Saving an enabled OSS channel with `is_primary=true` must atomically clear `is_primary` on every other OSS channel in the same app DB transaction before returning the saved row. Saving `is_primary=false` must not auto-promote another channel.
* Disabling any integration channel clears that channel's `is_primary` flag. Non-OSS create/update requests with `is_primary=true` are normalized to false.
* route layer uses route-local DTOs and `httpresponse`; it must not return `models.IntegrationChannel` or `models.IntegrationCredential` directly.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| invalid or missing `scenario` | `CodeValidation`, safe message `invalid integration scenario` |
| missing `channel_code` / `provider_code` / `adapter_key` / `credential_type` | `CodeValidation` |
| `credential_type` not in enabled `integration_credential_type` dictionary | `CodeValidation`, safe message `invalid credential type` |
| create without `credential_value` | `CodeValidation`, safe message `credential value is required` |
| `config_json` / `metadata_json` invalid or not an object | `CodeValidation` |
| `config_json` / `metadata_json` contains sensitive key | `CodeValidation` |
| schema `scenario` / `provider_code` / `credential_type` mismatch | `CodeValidation` |
| schema required config or credential field missing | `CodeValidation` with safe field message |
| schema URL field invalid or non-http(s) | `CodeValidation` |
| schema number/boolean field has wrong type | `CodeValidation` |
| schema field with `options` has a value outside the allowed option values | `CodeValidation` |
| `credential_format=json_object` value is not a JSON object | `CodeValidation` |
| duplicate `(scenario, channel_code, environment)` | `CodeConflict`, safe message `integration channel already exists` |
| create/update enabled OSS channel with `is_primary=true` | saved row becomes primary and all other OSS rows become non-primary |
| create/update non-OSS channel with `is_primary=true` | response returns `is_primary=false` and no OSS primary state changes |
| disable current primary OSS channel | channel returns `enabled=false` and `is_primary=false`; zero primary OSS channels remains valid |
| update/toggle missing channel | `CodeNotFound`, safe message `integration channel not found` |
| credential storage or database failure | `CodeInternal` with safe message and internal cause |

### 5. Good/Base/Bad Cases

Good: Create a `payment.creem.hosted_checkout` channel with non-sensitive `base_url/product_id/success_url` in `config_json`, and `api_key/webhook_secret` in admin-managed `credential_value`.

Good: Create an `email.aliyun.smtp` channel with SMTP host/port/security/from fields in `config_json`, and username/password in admin-managed `credential_value`.

Good: Create an `oss.cloudflare_r2.s3_compatible` channel with R2 endpoint/bucket/region/use_path_style in `config_json`, and access key id plus secret access key in admin-managed `credential_value`; no OSS SDK call is made by the Parameter API.

Good: Create an `oss.aliyun_oss.s3_compatible` channel with Aliyun OSS endpoint/bucket/region in `config_json`, and access key id plus secret access key in admin-managed `credential_value`; no OSS SDK call is made by the Parameter API and virtual-hosted addressing remains the adapter default.

Good: Mark `oss.cloudflare_r2.s3_compatible` as `is_primary=true`; backend clears any previous OSS primary channel and leaves non-OSS scenarios untouched.

Base: Edit an LLM channel's `priority` and `metadata_json` with empty `credential_value`; backend preserves the previous credential value.

Base: Disable the current primary OSS channel; backend clears only that channel's primary flag, and zero primary OSS channels is allowed until an admin selects another.

Bad: Store `api_key` inside `config_json`, or expose legacy `ciphertext/masked_value` through route DTO. This breaks the external integration anti-corruption credential boundary.

Bad: Enforce primary-provider uniqueness only in the frontend. The backend transaction and database unique constraint must be the authority.

### 6. Tests Required

* `api/models/integration_test.go` covers channel/credential CRUD, admin credential value view, duplicate conflict, OSS primary uniqueness, and clearing primary on disable.
* `api/usecase/dictionary_test.go` covers seeded `integration_credential_type` values, including `smtp_password` and `s3_access_key`.
* `api/usecase/parameter_test.go` covers schema listing including OSS `use_path_style`, credential value persistence, empty value preservation, non-empty value update, sensitive JSON key rejection including arrays, schema required field validation, URL validation, plain vs JSON object credential formats, enable/disable, OSS primary at-most-one behavior, zero-primary behavior, and non-OSS primary normalization.
* `api/routes/parameter_test.go` covers schema route DTO, internal envelope, DTO without legacy plaintext/ciphertext/masked fields, create response, `is_primary` mapping, and enable/disable response.
* `frontend/src/api.test.js` covers parameter helper paths, HTTP methods, encoded IDs, `is_primary`, and bodies.
* `frontend/src/router.test.js` covers `/parameters`, `/parameters.html`, menu label, and title.
* Run `go test ./...`, `cd frontend && npm test`, and `cd frontend && npm run build`.

### 7. Wrong vs Correct

#### Wrong

```go
return httpresponse.OK(c, models.IntegrationCredential{})
```

#### Correct

```go
channel, err := usecase.CreateParameterIntegrationChannel(ctx, cmd)
return httpresponse.Created(c, toParameterIntegrationChannelResponse(channel))
```

#### Wrong

```json
{"config_json":"{\"api_key\":\"secret\"}"}
```

#### Correct

```json
{"config_json":"{\"base_url\":\"https://api.creem.io\",\"product_id\":\"prod_123\"}","credential_value":"{\"api_key\":\"secret\",\"webhook_secret\":\"secret\"}"}
```

---

## Scenario: Payment Provider Webhook Ingress

### 1. Scope / Trigger

Modify payment provider webhook ingress, Creem webhook signature verification, webhook receipt persistence, or payment webhook queueing according to this section. This is an external provider ingress contract, not a frontend API contract.

### 2. Signatures

Provider callback URL configured in Creem:

```text
https://<public-domain>/api/integrations/payment/<channel_code>/webhooks/creem
```

Backend route:

```text
POST /api/integrations/payment/:channel_code/webhooks/creem
Header: creem-signature: <provider signature>
Body: raw provider JSON payload
```

Usecase:

```go
func ReceivePaymentWebhook(ctx fwusecase.Context, cmd ReceivePaymentWebhookCmd) (PaymentWebhookReceiptCo, error)
```

### 3. Contracts

* The route is public provider ingress under `/api/integrations/.../webhooks/...`; it must not be placed behind `RequireAuth()`, `RequireAdmin()`, or Open API key middleware.
* `channel_code` is only a routing/config lookup key. It is not treated as a secret.
* Creem webhook authentication is endpoint-level and provider-specific: read the raw request body and `creem-signature`, then verify the HMAC signature with the channel credential `webhook_secret` before trusting parsed JSON.
* `webhook_secret` belongs in the admin-managed payment channel `credential_value` JSON together with `api_key`; it must not be stored in `config_json`.
* Successful accepted Creem webhooks return `200 OK` with an empty body, not the internal `{success:true,data}` envelope.
* Verified webhooks are persisted to `integration_webhook_receipts` and enqueued on `integration-webhooks`; duplicate provider events must not enqueue duplicate work.
* Invalid or missing signatures persist a failed receipt for audit when the channel can be resolved, return a safe unauthorized error, and must not enqueue webhook work.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| missing `channel_code` | `CodeValidation`, safe message `payment channel code is required` |
| unknown channel | `CodeNotFound`, safe message `payment channel not found` |
| webhook disabled for channel | `CodeForbidden`, safe message `payment webhook is not enabled` |
| empty body | `CodeValidation`, safe message `payment webhook payload is required` |
| body over route limit | `413`, safe message `payment webhook payload is too large` |
| missing or invalid `creem-signature` | `CodeUnauthorized`, safe message `payment webhook signature is invalid`; no queue message |
| valid signature and new provider event | persist queued receipt, enqueue one `integration-webhooks` job, return `200 OK` |
| valid duplicate provider event | return `200 OK`, keep existing receipt, do not enqueue duplicate job |

### 5. Good/Base/Bad Cases

Good: Creem dashboard stores `https://example.com/api/integrations/payment/creem-prod/webhooks/creem`, and the `creem-prod` Parameter channel stores `{"api_key":"...","webhook_secret":"..."}` in `credential_value`.

Base: Route reads raw body with a maximum size guard, passes `channel_code`, raw body, and `creem-signature` to usecase, and returns provider ACK after usecase acceptance.

Bad: Put Creem signature verification into global middleware, rely on JWT/Open API credentials, trust parsed JSON before signature verification, or ACK a valid Creem webhook with `204 No Content` when Creem requires `200 OK`.

### 6. Tests Required

* `api/integrations/payment/creem/creem_test.go` covers valid signature mapping, invalid signature rejection, and missing signature rejection.
* `api/usecase/payment_test.go` covers receipt persistence, duplicate no-op queueing, invalid/missing signature failed receipts, and no queue message for auth failures.
* `api/routes/payment_test.go` covers route-level provider ACK status and empty ACK body.
* Run `go test ./...`.

### 7. Wrong vs Correct

#### Wrong

```go
protected.POST("/integrations/payment/:channel_code/webhooks/creem", user.ReceivePaymentWebhook)
return c.NoContent(http.StatusNoContent)
```

#### Correct

```go
api.POST("/integrations/payment/:channel_code/webhooks/creem", user.ReceivePaymentWebhook)
return c.NoContent(http.StatusOK)
```

---

## Scenario: Payment Subscription Upgrade Cancellation

### 1. Scope / Trigger

修改会员订阅升级、旧订阅取消、Creem subscription cancel adapter、或 `orders.subscription_status` 语义时，遵守本节。这里是外部 provider 出站合同和本地会员订单状态合同，不是普通内部 API DTO。

### 2. Signatures

Usecase port:

```go
type CancelSubscriptionRequest struct {
    SubscriptionID string
    Mode           string // "scheduled" or "immediate"
    OnExecute      string // "cancel" or "pause" for scheduled mode
}

type CancelSubscriptionResult struct {
    Status string
}
```

Creem request:

```text
POST /v1/subscriptions/{subscription_id}/cancel
Header: x-api-key: <api key>
Header: Content-Type: application/json
```

Scheduled cancellation body:

```json
{"mode":"scheduled","onExecute":"cancel"}
```

### 3. Contracts

* 升级到更高会员等级时，`ApplyOrderMembership` 查询同一用户旧的 active 订阅订单，不包含当前新订单。
* 旧订阅筛选条件必须要求 `provider_subscription_id` 非空、`subscription_status='active'`、且商品会员等级属于 `premium` 或 `super`。
* 本地 DB 事务内更新用户会员、标记旧订单 `subscription_status='canceled'`、标记新订单 `membership_applied_at`。
* Creem 取消请求必须通过 `RegisterAfterCommit` 在 DB commit 后执行，不能放进 DB 事务。
* 默认取消策略是 scheduled cancel：发送 `mode:"scheduled"` 和 `onExecute:"cancel"`，不要发送空 body。
* 本地旧订单标记为 `canceled` 表示它不再参与 active subscription 查询；Creem 返回 `scheduled_cancel` 时仍可保持本地 `canceled`，避免重复扣费路径继续把旧订阅视为 active。
* provider adapter 返回的 raw provider error 必须先归一化为 `providererror`，不要把 provider response body 泄露到客户端或日志。

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| empty `SubscriptionID` | adapter returns validation provider error |
| missing payment channel/config/credential | usecase returns internal/safe error where user-facing; after-commit cancellation may fail without rolling back committed membership |
| Creem cancel endpoint returns non-2xx | adapter returns normalized provider error |
| old subscription has empty provider subscription id | not sent to provider; it is not selected by active old subscription query |
| duplicate membership application (`membership_applied_at` already set) | no-op; do not cancel again |

### 5. Good/Base/Bad Cases

Good: Premium active order `sub_old` exists, user pays a Super order, `ApplyOrderMembership` commits Super membership, old order becomes locally `canceled`, and after commit Creem receives scheduled cancel body for `sub_old`.

Base: Creem returns `{"status":"scheduled_cancel"}`; adapter returns that status while local old order remains excluded from future active subscription queries.

Bad: Call Creem cancel with `http.NoBody`; Creem requires the cancellation mode payload and may leave the subscription active.

Bad: Call Creem inside `WithAppTx`; slow or failed provider IO would hold the SQLite transaction and can roll back unrelated local state.

### 6. Tests Required

* `api/integrations/payment/creem/creem_test.go` asserts cancel request method/path, `Content-Type: application/json`, and body `mode:"scheduled"` / `onExecute:"cancel"`.
* `api/usecase/product_membership_checkout_test.go` asserts upgrading from old active premium subscription to super calls `CancelSubscription` with the old provider subscription id and marks only the old order canceled.
* `go test ./...` after changing the payment adapter port because every fake adapter must implement the full interface.

### 7. Wrong vs Correct

#### Wrong

```go
http.NewRequestWithContext(ctx, http.MethodPost, cancelURL, http.NoBody)
```

#### Correct

```go
body := strings.NewReader(`{"mode":"scheduled","onExecute":"cancel"}`)
http.NewRequestWithContext(ctx, http.MethodPost, cancelURL, body)
```

---

## Scenario: Variable Management API

### 1. Scope / Trigger

Modify Variable management, typed application configuration, `/api/admin/variables*` routes, `variables` DB schema, or frontend variable helpers according to this section. Variables are app-level configuration records used as global parameters or logic-control inputs; they are not encrypted credentials and should not store secrets.

### 2. Signatures

Backend API:

```text
GET   /api/admin/variables
POST  /api/admin/variables
PUT   /api/admin/variables/:id
PATCH /api/admin/variables/:id/enabled
```

Request DTO:

```json
{
  "key": "checkout.max_retry",
  "name": "Checkout max retry",
  "value_type": "number",
  "value_json": "3",
  "enabled": true,
  "description": "Maximum checkout retries"
}
```

Usecase:

```go
func ListVariables(ctx fwusecase.Context, qry ListVariablesQry) ([]VariableCo, error)
func CreateVariable(ctx fwusecase.Context, cmd SaveVariableCmd) (VariableCo, error)
func UpdateVariable(ctx fwusecase.Context, cmd SaveVariableCmd) (VariableCo, error)
func SetVariableEnabled(ctx fwusecase.Context, cmd SetVariableEnabledCmd) (VariableCo, error)
```

DB:

```sql
variables(id, variable_key, name, value_type, value_json, enabled, description, created_at, updated_at)
```

### 3. Contracts

* Route DTO uses JSON key `key`; DB uses column `variable_key` to avoid SQL keyword ambiguity.
* `value_type` only allows `string`, `number`, `boolean`, and `json`.
* `value_json` is always stored as valid JSON:
  * `string` stores a JSON string, for example `"hello"`.
  * `number` stores a JSON number, for example `3`.
  * `boolean` stores `true` or `false`.
  * `json` stores any valid JSON value, with empty input defaulting to `{}`.
* `key` is normalized to lowercase and must match `^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`.
* `key` is globally unique. Business logic should reference variables by key, not by ID.
* Secret-like values belong in credential-specific storage, not in variables.
* Route layer maps `usecase.VariableCo` to route-local `VariableResponse` and returns `httpresponse.OK` or `httpresponse.Created`.

Successful response DTO:

```json
{
  "id": "variable-id",
  "key": "checkout.max_retry",
  "name": "Checkout max retry",
  "value_type": "number",
  "value_json": "3",
  "enabled": true,
  "description": "Maximum checkout retries",
  "created_at": "2026-06-07 10:00:00",
  "updated_at": "2026-06-07 10:00:00"
}
```

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| missing or invalid `key` | `CodeValidation`, safe message `variable key is required` or `variable key is invalid` |
| missing `name` | `CodeValidation`, safe message `variable name is required` |
| invalid `value_type` | `CodeValidation`, safe message `invalid variable value type` |
| `number` value is not a JSON number | `CodeValidation`, safe message `variable value must be a number` |
| `boolean` value is not a JSON boolean | `CodeValidation`, safe message `variable value must be a boolean` |
| invalid JSON for non-string typed values | `CodeValidation`, safe message `variable value is invalid` |
| duplicate `key` | `CodeConflict`, safe message `variable key already exists` |
| update/toggle missing variable | `CodeNotFound`, safe message `variable not found` |
| database failure | `CodeInternal` with safe message and internal cause |

### 5. Good/Base/Bad Cases

Good: Store `checkout.max_retry` as `value_type:"number"` and `value_json:"3"`, then let future business logic read it by stable key.

Base: Store `feature.new_checkout` as `value_type:"boolean"` and disable the row when the flag should not be considered active.

Bad: Store API keys, tokens, or provider secrets in variables; use integration credentials or another secret-specific boundary instead.

### 6. Tests Required

* `api/models/variable_test.go` covers migration-backed CRUD, stable ordering, enable/disable, and duplicate key conflict.
* `api/usecase/variable_test.go` covers key normalization, typed value normalization, invalid key/type/value validation, conflict mapping, and enable/disable.
* `api/routes/variable_test.go` covers internal envelope, route-local DTO shape, create response, and enable/disable response.
* `frontend/src/api.test.js` covers variable helper paths, HTTP methods, encoded IDs, and JSON bodies.
* `frontend/src/router.test.js` covers `/variables`, `/variables.html`, menu label, app-route detection, and title.
* Run `go test ./...`, `cd frontend && npm test`, and `cd frontend && npm run build`.

### 7. Wrong vs Correct

#### Wrong

```go
return httpresponse.OK(c, models.Variable{})
```

#### Correct

```go
variable, err := usecase.CreateVariable(ctx, cmd)
return httpresponse.Created(c, toVariableResponse(variable))
```

#### Wrong

```json
{"key":"api.secret","value_type":"string","value_json":"sk-live-secret"}
```

#### Correct

```json
{"key":"feature.new_checkout","value_type":"boolean","value_json":"true"}
```

---

## Scenario: Order Payment Points API

### 1. Scope / Trigger

订单支付、积分余额、商品列表和积分 WebSocket realtime 属于跨层契约，涉及 route/usecase/model/db/frontend。任何扩展支付、积分或订单管理页面时，都必须先对齐本节。

### 2. Signatures

后端 API：

```text
GET  /api/user/orders?page=1&page_size=10&status=pending
GET  /api/admin/orders?user_id=<id>&status=pending&page=1&page_size=10
GET  /api/orders/user/:user_id?page=1&page_size=10        # legacy guarded
POST /api/user/orders
POST /api/orders                                           # legacy guarded
POST /api/orders/:id/payment-checkout
POST /api/orders/:id/pay
GET  /api/user/points
GET  /api/user/realtime/ws?access_token=<jwt>
POST /api/user/notifications/test-export-toast
GET  /api/products
POST /api/admin/products
PUT  /api/admin/products/:id
```

usecase：

```go
type CreateOrderCmd struct {
    UserID string
    Items  []CreateOrderItemCmd // legacy-compatible; ignored by current Creem checkout flow
}

type PayOrderCmd struct {
    OrderID string
}

func CreateOrder(ctx fwusecase.Context, cmd CreateOrderCmd) (OrderCo, error)
func ListMyOrders(ctx fwusecase.Context, qry ListMyOrdersQry) (UserOrdersCo, error)
func ListAdminOrders(ctx fwusecase.Context, qry ListAdminOrdersQry) (UserOrdersCo, error)
func GetUserOrders(ctx fwusecase.Context, qry UserOrdersQry) (UserOrdersCo, error) // legacy guarded
func CreateOrderPaymentCheckout(ctx fwusecase.Context, cmd CreateOrderPaymentCheckoutCmd) (PaymentCheckoutCo, error)
func PayOrder(ctx fwusecase.Context, cmd PayOrderCmd) (OrderCo, error)
func GetUserPoints(ctx fwusecase.Context, qry PointsBalanceQry) (PointsCo, error)
func ListProducts(ctx fwusecase.Context) ([]ProductCo, error)
```

DB：

```sql
point_accounts(user_id TEXT PRIMARY KEY, balance INTEGER, updated_at DATETIME)
point_transactions(id TEXT PRIMARY KEY, user_id TEXT, order_id TEXT, points INTEGER, type TEXT, created_at DATETIME, UNIQUE(order_id, type))
```

### 3. Contracts

`POST /api/user/orders` Creem checkout MVP request:

```json
{"product_id":"p001"}
```

`POST /api/user/orders` 成功：

```json
{"success":true,"data":{"message":"order created","order":{"id":"...","user_id":"u001","user_name":"Ada","amount":0,"status":"pending","created_at":"..."}}}
```

Legacy `items` payload may still be accepted but must not be used as the current Creem payment source of truth:

```json
{"user_id":"u001","items":[{"product_id":"p001","quantity":1}]}
```

`POST /api/orders/:id/payment-checkout` 成功：

```json
{"success":true,"data":{"order":{"id":"...","status":"pending"},"checkout_id":"chk_...","checkout_url":"https://...","provider":"creem","status":"open"}}
```

`POST /api/orders/:id/pay` 成功：

```json
{"success":true,"data":{"message":"order paid","order":{"id":"...","user_id":"...","user_name":"...","amount":1000,"status":"paid","created_at":"..."}}}
```

`GET /api/user/points` 成功：

```json
{"success":true,"data":{"user_id":"u001","balance":10}}
```

`GET /api/products` 成功：

```json
{"success":true,"data":[{"id":"p001","name":"iPhone 15","description":"...","price":699900,"stock":100}]}
```

`POST /api/user/notifications/test-export-toast` 成功：

```json
{"success":true,"data":{"message":"export notification sent"}}
```

`GET /api/user/realtime/ws` WebSocket text message：

```json
{"type":"points","presentation":"refresh","payload":{"user_id":"u001","client_id":"...","balance":10}}
```

异步导出任务通知也使用同一 envelope，默认展示方式为 toast：

```json
{"type":"async_export_task","presentation":"toast","payload":{"task_id":"export-1","status":"completed","message":"Export completed"}}
```

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| empty `user_id` in `CreateOrderCmd` | `CodeValidation`, safe message `user ID is required` |
| `POST /api/user/orders` request contains another `user_id` | ignored; owner must still come from current user |
| legacy `POST /api/orders` with another user's `user_id` by non-admin actor | `CodeForbidden`, safe message `cannot create order for another user` |
| `GET /api/user/orders` without authenticated actor | `CodeUnauthorized`, safe message `not logged in` |
| `GET /api/user/orders?status=<invalid>` | `CodeValidation`, safe message `invalid order status` |
| `GET /api/admin/orders` by non-admin actor | `CodeForbidden`, safe message `admin access is required` |
| legacy `GET /api/orders/user/:user_id` with another user's ID by non-admin actor | `CodeForbidden`, safe message `cannot view another user's orders` |
| `user_id` does not exist | `CodeValidation`, safe message `user not found`; no order inserted |
| `CreateOrderCmd.Items` empty | create a pending Creem ledger order |
| `CreateOrderCmd.Items` present | ignore legacy items for current Creem checkout flow; do not write `order_items` or reserve stock |
| pending order has `amount=0` | valid for Creem checkout ledger; provider product/price is authoritative |
| empty `order_id` in `CreateOrderPaymentCheckoutCmd` | `CodeValidation`, safe message `order ID is required` |
| missing payment channel/config/credential | safe usecase error; do not leak credential values |
| empty `order_id` in `PayOrderCmd` | `CodeValidation`, safe message `order ID is required` |
| order not found | `CodeNotFound`, safe message `order not found` |
| order status is `pending` | update to `paid`, publish `order.paid`, award points |
| order status is already `paid` | return current paid order, do not award duplicate points |
| order status is neither `pending` nor `paid` | `CodeConflict`, safe message `only pending orders can be paid` |
| `PATCH /api/orders/:id/status` with `paid` | `CodeValidation`, safe message `use pay order endpoint to mark order paid` |
| point award fails inside pay transaction | `CodeInternal`, order status rolls back |
| unauthenticated points/product/order management route | auth middleware returns unauthorized |
| WebSocket missing or invalid `access_token` | auth middleware returns unauthorized before connection upgrade |

### 5. Good/Base/Bad Cases

Good: Creem 下单页面调用 `createOrder({ product_id })` 创建 `pending` 台账订单，再调用 `createOrderPaymentCheckout(order.id)` 跳转 provider checkout；支付完成后由 webhook 触发 `PayOrder`，页面通过 WebSocket 收到 `points` + `refresh` envelope。

Base: 已存在的 pending 订单仍可从订单列表再次调用 `createOrderPaymentCheckout(order.id)` 获取 checkout URL；WebSocket 断开时，页面仍可通过 `getMyPoints()` HTTP 查询恢复当前积分。

Bad: 前端把本地 `products.price` 或 `orders.amount=0` 当成 Creem 实收金额展示或计算；真实收费金额只能来自 provider checkout/后续 webhook 归一化。

Bad: 前端或后台直接 `PATCH /api/orders/:id/status` 为 `paid`，这会绕过积分同步，必须被 usecase 拒绝。

### 6. Tests Required

* `go test ./...`：覆盖 `CreateOrder` 无商品创建台账、legacy items 不写入 `order_items`、missing user 不落单、`CreateOrderPaymentCheckout` 配置读取、`PayOrder` 支付送积分、重复支付幂等、积分失败回滚订单状态、普通状态接口拒绝 `paid`。
* `cd frontend && npm test`：覆盖 `createOrder` 只发送 `product_id`、`createOrderPaymentCheckout`、`payOrder`、`getMyPoints`、`getProducts`、`realtimeWebSocketURL` helper。
* `cd frontend && npm run build`：确认订单管理页面和 embed 产物可构建。

### 7. Wrong vs Correct

#### Wrong

```go
_ = usecase.UpdateOrderStatus(ctx, usecase.UpdateOrderStatusCmd{
    OrderID: orderID,
    Status:  "paid",
})
```

#### Correct

```go
order, err := usecase.PayOrder(ctx, usecase.PayOrderCmd{OrderID: orderID})
```

---

## Scenario: Site Settings Logo API

### 1. Scope / Trigger

Modify site settings, app logo upload/display, `app_settings`, `/api/settings/*`, or runtime OSS-backed static assets according to this section.

### 2. Signatures

Backend API:

```text
GET  /api/public/settings/site
POST /api/admin/settings/site/logo
GET  /api/public/settings/logo
```

Usecase:

```go
type SiteSettingsCo struct {
    LogoURL                     string
    LogoConfigured              bool
    LogoUpdatedAt               string
    LogoUploadAvailable         bool
    LogoUploadUnavailableReason string
}

type SaveSiteLogoCmd struct {
    Filename    string
    ContentType string
    Size        int64
    Body        io.Reader
}

func GetSiteSettings(ctx fwusecase.Context, qry SiteSettingsQry) (SiteSettingsCo, error)
func SaveSiteLogo(ctx fwusecase.Context, cmd SaveSiteLogoCmd) (SiteSettingsCo, error)
func GetSiteLogoObject(ctx fwusecase.Context, qry SiteLogoObjectQry) (SiteLogoObjectCo, error)
```

DB:

```sql
app_settings(setting_key TEXT PRIMARY KEY, value_json TEXT, created_at DATETIME, updated_at DATETIME)
```

Stored `site.logo` JSON:

```json
{"object_key":"settings/site-logo.png","content_type":"image/png","size":658,"updated_at":"2026-06-08T12:00:00.123456789Z","channel_code":"primary-r2","provider_code":"cloudflare_r2","adapter_key":"oss.cloudflare_r2.s3_compatible"}
```

### 3. Contracts

* `GET /api/public/settings/site` is safe to call before login and returns the internal success envelope.
* Default response when no logo is configured:

```json
{"logo_url":"/logo.png","logo_configured":false,"logo_updated_at":"","logo_upload_available":false,"logo_upload_unavailable_reason":"Primary OSS provider is not configured"}
```

* Configured response uses a cache-busting public image URL:

```json
{"logo_url":"/api/public/settings/logo?v=2026-06-08T12%3A00%3A00.123456789Z","logo_configured":true,"logo_updated_at":"2026-06-08T12:00:00.123456789Z","logo_upload_available":true,"logo_upload_unavailable_reason":""}
```

* `POST /api/admin/settings/site/logo` is admin-only and accepts multipart form data with a `logo` file field.
* Logo bytes are stored through `api/usecase/integrations/oss.Adapter`; DB stores only safe metadata, not raw image bytes.
* `GET /api/public/settings/logo` is unauthenticated because browser `<img>` requests cannot attach the app bearer token. It streams the configured object or redirects to `/logo.png` when no object is configured.
* Site-logo upload resolves the enabled primary OSS channel (`scenario=oss`, `enabled=1`, `is_primary=1`) and its enabled credential, then maps channel `config_json` plus credential JSON to `oss.ProviderConfig`.
* Site-logo storage must not use the local OSS adapter. `index.go` registers provider-backed OSS adapters such as `oss.cloudflare_r2.s3_compatible` and `oss.aliyun_oss.s3_compatible`; AWS SDK Go v2 configuration/signing details remain under `api/integrations/oss/<provider>`, and usecase depends only on the OSS port.
* Persisted `site.logo` metadata includes `channel_code`, `provider_code`, and `adapter_key`; public logo read uses that persisted provider/channel metadata. Legacy metadata without channel/adapter fields may fall back to the current primary OSS provider.
* Accepted logo formats are detected from file magic bytes: PNG, JPEG, and WebP.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| no `site.logo` setting | `GET /api/public/settings/site` returns `/logo.png`, `logo_configured:false`, and current upload availability fields |
| no enabled primary OSS provider | `GET /api/public/settings/site` returns `logo_upload_available:false`; `POST /api/admin/settings/site/logo` returns `CodeValidation`, safe message `primary OSS provider is not configured` |
| no public logo object | `GET /api/public/settings/logo` redirects to `/logo.png` |
| missing upload file | `CodeValidation`, safe message `logo file is required` |
| upload over 2 MiB | `CodeValidation`, safe message `logo file is too large` |
| unsupported magic bytes | `CodeValidation`, safe message `logo image type is not supported` |
| OSS adapter not registered | `CodeInternal`, safe message `logo storage is not configured` |
| object storage failure | `CodeInternal`, safe message `failed to store logo` or `failed to load logo` |

### 5. Good/Base/Bad Cases

Good: Settings page uploads `logo` as `FormData`; route builds `SaveSiteLogoCmd`; usecase validates bytes, resolves the primary OSS channel, calls the registered provider adapter, stores `site.logo` metadata, and returns a cache-busted `logo_url`.

Base: Fresh install has no `site.logo` row and no primary OSS provider; header displays `/logo.png`, while Settings upload is disabled through `logo_upload_available:false`.

Bad: Route writes uploaded bytes directly under `frontend/dist` or stores raw image bytes in SQLite; this bypasses the OSS port and breaks production/runtime separation.

Bad: The frontend sets `<img src="/api/admin/settings/site/logo">` behind `RequireAuth()`; browser image requests do not include bearer auth headers.

Bad: `index.go` registers a site-logo-only local adapter key and settings usecase hard-codes it; this bypasses Parameter-owned OSS provider configuration.

### 6. Tests Required

* `api/models/setting_test.go` covers `app_settings` upsert/get and single-row replacement.
* `api/models/integration_test.go` covers primary OSS channel config lookup and metadata channel lookup.
* `api/usecase/setting_test.go` covers default fallback, missing primary validation, primary OSS adapter invocation, persisted metadata, and object readback through persisted provider config.
* `api/routes/setting_test.go` covers internal envelope, upload availability DTO fields, route-local DTOs for settings read/upload, and missing-primary validation mapping.
* `api/integrations/oss/s3compatible/s3compatible_test.go` covers AWS SDK S3 client/presigner input mapping, key prefix handling, addressing-style defaults, and normalized provider errors.
* Frontend API tests cover `getSiteSettings()` response fields and `uploadSiteLogo()` path/method/FormData behavior.
* Run `go test ./...`, `cd frontend && npm test`, and `cd frontend && npm run build`.

### 7. Wrong vs Correct

#### Wrong

```go
func UploadSiteLogo(c echo.Context) error {
    file, _ := c.FormFile("logo")
    _ = os.WriteFile("frontend/dist/logo.png", fileBytes(file), 0644)
    return c.JSON(200, map[string]string{"logo_url": "/logo.png"})
}
```

#### Correct

```go
settings, err := usecase.SaveSiteLogo(ctx, usecase.SaveSiteLogoCmd{
    Filename:    fileHeader.Filename,
    ContentType: fileHeader.Header.Get(echo.HeaderContentType),
    Size:        fileHeader.Size,
    Body:        file,
})
return httpresponse.OK(c, toSiteSettingsResponse(settings))
```

---

## Tests Required

需要覆盖的测试方向：

* `api/framework/archguard/frontend_dto_boundary_test.go`：内部 route 不直接返回 model。
* `api/framework/archguard/layer_boundary_test.go`：内部 route 不直接 `c.JSON(...)`。
* DTO 单元测试：敏感字段不出现在 JSON，布尔字段类型正确。
* 前端 API client 测试：`frontend/src/api.test.js` unwrap `success/data`，保留 `error.message`。

常规命令：

```sh
go test ./...
cd frontend && npm test
```

---

## Wrong vs Correct

### Wrong

```go
return c.JSON(http.StatusOK, user)
```

### Correct

```go
return httpresponse.OK(c, ToUserResponse(user))
```

### Wrong

```go
var user models.User
if err := c.Bind(&user); err != nil {
    return httpresponse.BadRequest(c, "invalid request data")
}
```

### Correct

```go
var req CreateUserRequest
if err := c.Bind(&req); err != nil {
    return httpresponse.BadRequest(c, "invalid request data")
}
cmd := usecase.CreateUserCmd{Name: req.Name, Email: req.Email}
```

---

## Common Mistakes

* 把 `json:"-"` 当成唯一 API 边界保护。
* 为了省事直接返回 `models.*`。
* 在多个 route 中手写 envelope map。
* 让前端根据 ID 查字典或查名称，而不是后端 DTO 直接返回 `xxx_name`。
* 混用内部 `/api/*` DTO 和 `/open-api/v1/*` DTO。
