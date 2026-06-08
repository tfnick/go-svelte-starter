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
* `GET /api/auth/me` 返回 `{user:{id,name,email,email_verified}}`。
* User detail/create/update/delete 返回 `UserResponse` 或空响应；User 管理列表使用分页 envelope。
* `GET /api/users?page=1&page_size=10` 返回 `UsersResponse`，其中 `items` 为 `[]UserResponse`，`pagination` 遵守 [Pagination Contract](#pagination-contract)。
* `PATCH /api/users/:id/active` 用于启用/禁用用户，request body 为 `{active:boolean}`，返回更新后的 `UserResponse`。
* 禁用用户沿用 `users.is_active=0` 语义；登录和受保护 API 已通过 auth/usecase/middleware 拒绝 disabled user。

敏感字段永远不能出现在 DTO 中：

* `password`
* `password_hash`
* `session_id`
* raw JWT signing secret
* reset token
* raw API key

---

## Order Product Admin Contract

订单相关内部响应当前规则：

* `CreateOrderResponse` 是 create route 的 wrapper，包含 `message` 和 `order`。
* `PayOrderResponse` 是 pay route 的 wrapper，包含 `message` 和 `order`。支付成功必须通过 `POST /api/orders/:id/pay`，不能通过 `PATCH /api/orders/:id/status` 直接设置 `paid`。
* `OrderDetailResponse` 是 detail route 的 wrapper，包含 `order` 和 `items`。
* 订单 list 使用 `GET /api/orders/user/:user_id?page=1&page_size=10`，返回 `UserOrdersResponse`，其中 `items` 为 `[]OrderResponse`，`pagination` 遵守 [Pagination Contract](#pagination-contract)。
* `UpdateOrderStatus` 返回 `data.message`。

产品列表通过 `GET /api/products` 返回 `[]ProductResponse`，用于下单页面展示可选商品。产品 DTO 只暴露 `id`、`name`、`description`、`price`、`stock`，不返回 `models.Product`。

积分相关内部响应当前规则：

* `GET /api/points/me` 返回 `PointsResponse`，字段为 `user_id` 和 `balance`。
* `GET /api/points/sse?access_token=<jwt>` 是 SSE stream endpoint，不使用 HTTP JSON envelope；连接成功后以 `data: ...` 推送 realtime envelope，例如 `{"type":"points","presentation":"refresh","payload":{"user_id":"...","client_id":"...","balance":10}}`。
* `POST /api/notifications/test-export-toast` 是登录态验证入口，返回 `data.message`，并向当前用户发布 `async_export_task` + `toast` realtime envelope。

Admin 当前只提供 `POST /api/admin/reload-shared-db`，返回 message。未来如果 admin 返回资源数据，再在 `api/routes/admin.go` 中新增明确 DTO。

---

## Scenario: User Management API

### 1. Scope / Trigger

修改 User 管理页面、`GET /api/users` 分页契约、`users.is_active` 启禁用语义、或前端 `listUsers` / `setUserActive` helper 时，遵守本节。

### 2. Signatures

Backend API:

```text
GET /api/users?page=1&page_size=10
PATCH /api/users/:id/active
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

* `GET /api/users` uses [Pagination Contract](#pagination-contract) and returns `UsersResponse`.
* `UsersResponse.items` is `[]UserResponse` with `id`, `name`, `email`, `email_verified`, `is_active`, `created_at`, and `updated_at`.
* `UserResponse` must never include `password_hash`, reset tokens, sessions, or JWT details.
* User list ordering must be stable: `ORDER BY created_at DESC, id DESC`.
* `PATCH /api/users/:id/active` only mutates `users.is_active` and `updated_at`; it does not delete rows or change profile fields.
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

Good: User page calls `GET /api/users?page=1&page_size=10`, then calls `PATCH /api/users/:id/active` from a row action and refreshes the current page.

Base: Re-enabling a disabled user returns the updated `UserResponse` with `is_active:true`; the page refreshes the row/list.

Bad: Use `DELETE /api/users/:id` to disable a user; this removes the row and breaks account history and foreign-key relationships.

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
GET /api/events?page=1&page_size=10
GET /api/events/:id/deliveries
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

* `GET /api/events` uses [Pagination Contract](#pagination-contract) and returns `DomainEventsResponse`.
* `DomainEventsResponse.items` is `[]DomainEventResponse` with `id`, `topic`, `aggregate_type`, `aggregate_id`, `payload_json`, `metadata_json`, `occurred_at`, and `created_at`.
* `GET /api/events/:id/deliveries` returns `[]DomainEventDeliveryResponse` with `id`, `event_id`, `subscriber`, `message_id`, `status`, `attempts`, `last_error`, `created_at`, and `updated_at`.
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

Good: Event page calls `GET /api/events?page=1&page_size=10`, then lazy-loads `GET /api/events/:id/deliveries` when the user selects an event.

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

修改 Notification Center、`notifications` 台账、业务通知创建 usecase、SSE notification envelope、或 admin `/api/notifications` 查询接口时，遵守本节。

Notification Center 的定位是业务通知入口与台账，不替代外部渠道 port/provider adapter。业务场景需要发送用户通知时默认调用 `usecase.CreateNotification(...)`；也可以由领域事件 subscriber 调用该 usecase。纯技术实时刷新、渠道连通性测试、provider 防腐层内部逻辑，才直接调用对应 port 或 realtime primitive。

### 2. Signatures

Admin query API:

```text
GET /api/notifications?page=1&page_size=10&type=sse&email=ada@example.com&phone=138
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
dictionary_values(value_code='sse'|'sms'|'email'|'wechat_official_account')
```

### 3. Contracts

* Notification type is dictionary-managed through `notification_type`; usecases validate against enabled dictionary values.
* Notification status is code-owned, not dictionary-managed. Allowed values are `pending`, `sent`, `failed`, and `skipped`; DB must enforce them with `CHECK`.
* There is no admin HTTP create API for MVP. Creation is usecase-only so business usecases and event subscribers share the same entry.
* `GET /api/notifications` is admin-only and uses the standard pagination contract.
* Filter parameters are `type`, `email`, and `phone`; email/phone are substring filters over `recipient_email` and `recipient_phone`.
* List DTO includes both `notification_type` and `notification_type_label`.
* Non-SSE types are ledger-only in the MVP and must be stored as `skipped`; they do not call SMS, email, or WeChat provider ports yet.
* SSE notifications must create the ledger record first, publish realtime message type `notification`, then update status to `sent` or `failed`.
* SSE realtime payload is a safe presentation snapshot only: `id`, `title`, `summary`, `source_type`, and `source_id`. Do not push raw `payload_json` or provider/channel secrets.
* `payload_json` must be a JSON object and remains an internal ledger payload.

Successful list response:

```json
{
  "items": [
    {
      "id": "notification-id",
      "notification_type": "sse",
      "notification_type_label": "SSE",
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
| SSE without `user_id` | `CodeValidation`, safe message `user ID is required for SSE notification` |
| invalid or non-object `payload_json` | `CodeValidation` |
| realtime publish fails | ledger status becomes `failed`, `last_error` stores a safe message, caller receives `CodeInternal` |
| invalid pagination query | `CodeValidation` -> internal `400` envelope |
| non-admin list request | middleware returns `403` before handler |

### 5. Good/Base/Bad Cases

Good: Order payment usecase calls `CreateNotification` with `notification_type=sse`, `source_type=order`, `source_id=...`, and a safe title/summary; user receives a toast and admin can inspect the ledger row.

Base: A future SMS notification creates a `skipped` ledger row until the SMS delivery chain is implemented behind Notification Center.

Bad: Business usecase calls SMS provider port directly and separately inserts a notification row; this splits status and audit behavior.

### 6. Tests Required

* `api/usecase/notification_test.go` covers SSE create/publish/status, non-SSE skipped ledger, dictionary type validation, and list filters.
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
    NotificationType: usecase.NotificationTypeSSE,
    UserID: userID,
    Title: "Order paid",
    Summary: "Your points have been awarded",
})
```

---

## Scenario: Dictionary Management API

### 1. Scope / Trigger

Modify Dictionary management, `/api/dictionary*` routes, `/api/dictionaries` lookup behavior, `dictionary_types` / `dictionary_values` DB schema, or frontend dictionary helpers according to this section. Dictionaries provide selectable values for UI and business forms; they are not credential or secret storage.

### 2. Signatures

Lookup API:

```text
GET /api/dictionaries?types=product_category,region
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

* Existing `GET /api/dictionaries?types=...` response shape stays `{dictionaries:{type:[{value,label}]}}`.
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

Modify `Parameter` management, external integration channel configuration, credential write boundaries, adapter schema registry, `/api/parameters/*` routes, or frontend parameter helpers according to this section. This scenario is the management-surface slice of the external integration anti-corruption layer.

### 2. Signatures

Backend API:

```text
GET   /api/parameters/integration-schemas?scenario=payment|llm|sms|email
GET   /api/parameters/integration-channels?scenario=payment|llm|sms|email
POST  /api/parameters/integration-channels
PUT   /api/parameters/integration-channels/:id
PATCH /api/parameters/integration-channels/:id/enabled
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
integration_channels(id, scenario, channel_code, provider_code, adapter_key, environment, enabled, priority, credential_id, policy_id, webhook_enabled, config_json, metadata_json)
integration_webhook_receipts(id, scenario, channel_id, channel_code, provider_code, provider_event_id, idempotency_key, status, attempts, message_id)
integration_credentials(id, credential_type, value_text, enabled, rotated_at)
dictionary_types(type_key='integration_environment')
dictionary_values(value_code='test'|'production')
dictionary_types(type_key='integration_credential_type')
dictionary_values(value_code='payment_bundle'|'api_key'|'smtp_password')
```

`smtp_password` is present in baseline `002_seed.sql` and is also backfilled for already-migrated databases by `008_add_email_integration_seed.sql`.

### 3. Contracts

#### Adapter Schema Contract

* `GET /api/parameters/integration-schemas` 返回 code-owned adapter schema DTO；该 DTO 用于前端动态渲染，也用于后端保存时校验。
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

* API only manages `integration_channels + integration_credentials`; it does not manage `integration_operation_configs`, `integration_model_options`, policy, webhook receipts, invocation raw data, provider request/response, prompt, or stream chunks.
* Parameter APIs are protected internal admin configuration APIs. Routes must run behind `RequireAuth()` and `RequireAdmin()`; admin access is represented by `users.is_admin=1`.
* `scenario` only allows `payment`, `llm`, `sms`, and `email`.
* `credential_type` uses the `integration_credential_type` dictionary. Current seeded values are `payment_bundle`, `api_key`, and `smtp_password`; save usecases reject values that are not enabled dictionary values.
* `config_json` and `metadata_json` must be JSON objects and may only contain non-sensitive config. Obvious sensitive keys such as `api_key`, `secret`, `password`, `token`, and `private_key` are rejected recursively, including inside arrays.
* create requires `credential_value`; update with empty `credential_value` preserves the existing credential value.
* `credential_value` is stored as administrator-managed configuration in `integration_credentials.value_text`; it is not encrypted by this module. Legacy `ciphertext/key_version/masked_value` columns may exist for migration compatibility, but they are not part of the current API contract.
* Responses return `credential_type` and `credential_value` for the protected admin page. They must never return `credential_plaintext`, `ciphertext`, `key_version`, or `masked_value`.
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
| update/toggle missing channel | `CodeNotFound`, safe message `integration channel not found` |
| credential storage or database failure | `CodeInternal` with safe message and internal cause |

### 5. Good/Base/Bad Cases

Good: Create a `payment.creem.hosted_checkout` channel with non-sensitive `base_url/product_id/success_url` in `config_json`, and `api_key/webhook_secret` in admin-managed `credential_value`.

Good: Create an `email.aliyun.smtp` channel with SMTP host/port/security/from fields in `config_json`, and username/password in admin-managed `credential_value`.

Base: Edit an LLM channel's `priority` and `metadata_json` with empty `credential_value`; backend preserves the previous credential value.

Bad: Store `api_key` inside `config_json`, or expose legacy `ciphertext/masked_value` through route DTO. This breaks the external integration anti-corruption credential boundary.

### 6. Tests Required

* `api/models/integration_test.go` covers channel/credential CRUD, admin credential value view, and duplicate conflict.
* `api/usecase/dictionary_test.go` covers seeded `integration_credential_type` values, including `smtp_password`.
* `api/usecase/parameter_test.go` covers schema listing, credential value persistence, empty value preservation, non-empty value update, sensitive JSON key rejection including arrays, schema required field validation, URL validation, plain vs JSON object credential formats, and enable/disable.
* `api/routes/parameter_test.go` covers schema route DTO, internal envelope, DTO without legacy plaintext/ciphertext/masked fields, create response, and enable/disable response.
* `frontend/src/api.test.js` covers parameter helper paths, HTTP methods, encoded IDs, and bodies.
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

## Scenario: Variable Management API

### 1. Scope / Trigger

Modify Variable management, typed application configuration, `/api/variables*` routes, `variables` DB schema, or frontend variable helpers according to this section. Variables are app-level configuration records used as global parameters or logic-control inputs; they are not encrypted credentials and should not store secrets.

### 2. Signatures

Backend API:

```text
GET   /api/variables
POST  /api/variables
PUT   /api/variables/:id
PATCH /api/variables/:id/enabled
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

订单支付、积分余额、商品列表和积分 SSE 属于跨层契约，涉及 route/usecase/model/db/frontend。任何扩展支付、积分或订单管理页面时，都必须先对齐本节。

### 2. Signatures

后端 API：

```text
POST /api/orders/:id/pay
GET  /api/points/me
GET  /api/points/sse?access_token=<jwt>
POST /api/notifications/test-export-toast
GET  /api/products
```

usecase：

```go
type PayOrderCmd struct {
    OrderID string
}

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

`POST /api/orders/:id/pay` 成功：

```json
{"success":true,"data":{"message":"order paid","order":{"id":"...","user_id":"...","user_name":"...","amount":1000,"status":"paid","created_at":"..."}}}
```

`GET /api/points/me` 成功：

```json
{"success":true,"data":{"user_id":"u001","balance":10}}
```

`GET /api/products` 成功：

```json
{"success":true,"data":[{"id":"p001","name":"iPhone 15","description":"...","price":699900,"stock":100}]}
```

`POST /api/notifications/test-export-toast` 成功：

```json
{"success":true,"data":{"message":"export notification sent"}}
```

`GET /api/points/sse` SSE `data:` message：

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
| empty `order_id` in `PayOrderCmd` | `CodeValidation`, safe message `order ID is required` |
| order not found | `CodeNotFound`, safe message `order not found` |
| order status is `pending` | update to `paid`, publish `order.paid`, award points |
| order status is already `paid` | return current paid order, do not award duplicate points |
| order status is neither `pending` nor `paid` | `CodeConflict`, safe message `only pending orders can be paid` |
| `PATCH /api/orders/:id/status` with `paid` | `CodeValidation`, safe message `use pay order endpoint to mark order paid` |
| point award fails inside pay transaction | `CodeInternal`, order status rolls back |
| unauthenticated points/product/order management route | auth middleware returns unauthorized |
| SSE missing or invalid `access_token` | auth middleware returns unauthorized before stream starts |

### 5. Good/Base/Bad Cases

Good: 下单页面调用 `createOrder(...)` 创建 `pending` 订单，再调用 `payOrder(order.id)` 完成支付，页面通过 SSE 收到 `points` + `refresh` envelope。

Base: SSE 断开时，页面仍可通过 `getMyPoints()` HTTP 查询恢复当前积分。

Bad: 前端或后台直接 `PATCH /api/orders/:id/status` 为 `paid`，这会绕过积分同步，必须被 usecase 拒绝。

### 6. Tests Required

* `go test ./...`：覆盖 `PayOrder` 支付送积分、重复支付幂等、积分失败回滚订单状态、普通状态接口拒绝 `paid`。
* `cd frontend && npm test`：覆盖 `createOrder`、`payOrder`、`getMyPoints`、`getProducts`、`pointsSSEURL` helper。
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
