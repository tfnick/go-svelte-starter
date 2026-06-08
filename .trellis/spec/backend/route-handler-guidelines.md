# Route Handler Guidelines

> 本文说明内部 `/api/*` route handler 的标准执行流程。响应 envelope 和 DTO 细节见 [API Contracts](./api-contracts.md)。

---

## Overview

Route handler 只处理 HTTP concern：

* 读取 path/query/body。
* 执行 transport-level bind。
* 读取 middleware 已设置的用户或 consumer context。
* 构建 `fwusecase.Context`。
* 构建 `usecase.XxxCmd` / `usecase.XxxQry`。
* 调用 usecase。
* 将 `XxxCo` 映射成 route-local DTO。
* 使用 `httpresponse` helper 返回结果。

Route 不直接访问 `models` 或 `db`，不创建事务，不解析 cookie/API key 的业务语义。

---

## Standard Handler Flow

新增或修改内部 `/api/*` handler 时按这个顺序：

1. 在当前 route 文件定义 request DTO。
2. 使用 `c.Bind(&req)` 或 route-local bind helper 读取输入。
3. 读取 path/query 参数并做必要 normalize。
4. 对 bind failure 返回 `httpresponse.BadRequest(c, "invalid request data")`。
5. 从 middleware helper 读取当前用户或 session 状态。
6. 使用 `fwcontext.InternalUsecaseContext(c)` 创建 usecase context。
7. 构建 `usecase.XxxCmd` 或 `usecase.XxxQry`。
8. 调用 usecase entry point。
9. 用 `httpresponse.InternalUsecaseError(c, err)` 映射 typed usecase error。
10. 将 usecase `Co` 映射为 route DTO。
11. 使用 `httpresponse.OK`、`Created`、`OKMessage` 或 `OKEmpty` 返回。

---

## Current Helpers

内部 API response helper 位于 `api/framework/http/response`：

```go
httpresponse.OK(c, data)
httpresponse.Created(c, data)
httpresponse.OKMessage(c, message)
httpresponse.OKEmpty(c)
httpresponse.BadRequest(c, message)
httpresponse.Unauthorized(c, message)
httpresponse.Forbidden(c, message)
httpresponse.InternalUsecaseError(c, err)
```

usecase context helper 位于 `api/framework/http/context`：

```go
fwcontext.InternalUsecaseContext(c)
fwcontext.OpenAPIUsecaseContext(c)
```

Open API route 不使用内部 API helper 的成功 envelope，详见 [Open API Guidelines](./open-api-guidelines.md)。

Provider webhook endpoints are also an explicit exception to the internal frontend envelope. Routes such as `/api/integrations/<scenario>/<channel>/webhooks/<provider>` are external provider ingress, so successful ACK may use the provider-required status/body instead of `httpresponse.OK(...)`; Creem payment webhooks must ACK with `c.NoContent(http.StatusOK)`. Provider-specific webhook authentication is endpoint-level behavior, not global middleware: the route reads the raw body and may forward request headers as a generic map, then delegates provider-specific header interpretation and signature verification to usecase/adapter and maps errors through safe response helpers. Do not hard-code provider signature header names such as `creem-signature` into route-to-usecase command contracts.

---

## Validation Matrix

| Condition | Route behavior |
| --- | --- |
| bind failure | `httpresponse.BadRequest(c, "invalid request data")` |
| 未登录访问受保护内部 API | middleware 或 route 返回 `unauthorized` |
| usecase validation error | `httpresponse.InternalUsecaseError(c, err)` -> `400` |
| usecase not found | `httpresponse.InternalUsecaseError(c, err)` -> `404` |
| usecase conflict | `httpresponse.InternalUsecaseError(c, err)` -> `409` |
| usecase internal error | `httpresponse.InternalUsecaseError(c, err)` -> `500`，日志记录 cause |
| 只返回消息 | `httpresponse.OKMessage(c, "...")` |
| 无 payload 成功 | `httpresponse.OKEmpty(c)` |
| 返回资源 | `Co -> DTO` 后 `httpresponse.OK/Created` |

---

## Auth Handling

内部 API 当前在 `index.go` 中按 group 注册：

```go
api := router.Group("/api")
api.Use(authMiddleware.RequestLogger("api"))

protected := api.Group("")
protected.Use(authMiddleware.RequireAuth())
```

规则：

* 登录、注册、忘记密码、重置密码、auth status、dictionaries 是未登录也可访问或可选登录的内部 API。
* 受保护 route 通过 `RequireAuth()` 保证用户已认证。
* 内部 API 登录态使用 JWT：普通 HTTP 请求必须携带 `Authorization: Bearer <access_token>`。
* 浏览器原生 `EventSource` 不能设置自定义 `Authorization` header，因此 SSE stream 使用 `access_token` query parameter。
* route 需要当前用户时使用 `middleware.GetCurrentUser(c)`。
* admin-only 后台配置接口使用 `RequireAuth()` 后再叠加 `RequireAdmin()`；当前 admin 标识来自 `users.is_admin`。
* `POST /api/auth/login` 和 `POST /api/auth/register` 在 route 层签发 JWT；usecase 只返回认证后的 `AuthCo/UserCo`。
* usecase 不直接读取 Echo context、JWT、cookie 或 header。

## Scenario: JWT Auth for Internal API

### 1. Scope / Trigger

修改登录、注册、受保护内部 API、SSE 认证或 `frontend/src/api.js` 时，必须对齐本节。

### 2. Signatures

```text
POST /api/auth/login
POST /api/auth/register
POST /api/auth/logout
GET  /api/auth/status
GET  /api/points/sse?access_token=<jwt>
```

```go
func RequireAuth() echo.MiddlewareFunc
func OptionalAuth() echo.MiddlewareFunc
func IssueUserToken(userID string) (auth.Token, error)
func ParseUserToken(raw string) (auth.Claims, error)
```

### 3. Contracts

* HTTP API token 传递：`Authorization: Bearer <access_token>`。
* SSE token 传递：`access_token` query parameter。
* JWT secret：可选环境变量 `APP_JWT_SECRET`。未配置时进程内生成临时 secret，重启后旧 token 失效。
* JWT MVP 只包含 access token，不包含 refresh token。
* `routes` 签发 token，`middleware` 解析 token 并设置 current user，`usecase` 不依赖 HTTP auth 包。

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| missing token on protected route | `401` internal envelope, code `unauthorized`, message `not logged in` |
| invalid or expired token | `401` internal envelope, message `login token is invalid or expired` |
| token subject user not found | `401` internal envelope, message `user does not exist` |
| disabled user | `403` internal envelope, message `account is disabled` |
| legacy `session_id` cookie only | ignored; request remains unauthenticated |

### 5. Good/Base/Bad Cases

Good: frontend login stores `access_token`, later API requests add `Authorization: Bearer ...`, SSE uses `?access_token=...`.

Base: logout is stateless on the server and tells frontend to discard the local access token.

Bad: route reads `session_id` cookie or usecase parses `Authorization` header.

### 6. Tests Required

* `api/framework/http/auth/jwt_test.go`: issue, parse, expired, tampered token.
* `api/framework/http/middleware/auth_test.go`: bearer token, query token, missing token, legacy cookie ignored.
* `frontend/src/api.test.js`: token storage, authorization header, SSE query token.
* `go test ./...` and `cd frontend && npm test`.

### 7. Wrong vs Correct

#### Wrong

```go
cookie, _ := c.Cookie("session_id")
```

#### Correct

```go
user := middleware.GetCurrentUser(c)
```

---

## Good and Bad Cases

Good:

```go
ctx := fwcontext.InternalUsecaseContext(c)
order, err := usecase.CreateOrder(ctx, cmd)
if err != nil {
    return httpresponse.InternalUsecaseError(c, err)
}
return httpresponse.Created(c, CreateOrderResponse{
    Message: "order created",
    Order:   ToOrderResponse(order),
})
```

Bad:

```go
order, err := models.GetOrderByID(c.Request().Context(), id)
return c.JSON(http.StatusOK, order)
```

---

## Tests Required

* 修改 handler 流程时运行 `go test ./...`。
* 修改 envelope 或 API client 相关行为时运行 `cd frontend && npm test`。
* 新增内部 route 时确认 archguard 不失败：
  * route 不导入 `api/models` 或 `api/db`。
  * route 不直接调用 `c.JSON(...)`。
  * route 不直接返回 `models.*`。

---

## Common Mistakes

* 在 route 中做跨模型业务编排。
* 在 route 中创建事务。
* 在 route 中根据 `err.Error()` 判断 HTTP status。
* 手写 `map[string]interface{}{"success": ...}`。
* 把 Open API 的 public envelope 复制到内部 API。
