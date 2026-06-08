# Open API Guidelines

> 本文是 `/open-api/v1/*` 公开接口的权威说明。内部 `/api/*` 不使用本文的成功响应规则。

---

## Overview

Open API 是给外部 partner/consumer 使用的公开契约，路径固定在：

```text
/open-api/v1/*
```

它与内部 Svelte `/api/*` 分开维护：

* route 文件使用 `open_api_*` 前缀。
* 认证使用 API key middleware。
* usecase context 使用 `fwcontext.OpenAPIUsecaseContext(c)`。
* 错误 envelope 使用 `httpresponse.ErrorResponse(...)`。
* 公开 DTO 放在对应 `api/routes/open_api_*.go`。

---

## Current Routes

当前公开 route：

| Route | Auth | Handler |
| --- | --- | --- |
| `GET /open-api/v1/health` | 无需 API key | `GetOpenAPIHealth` |
| `GET /open-api/v1/account/me` | `RequireOpenAPIKey()` | `GetOpenAPIAccountMe` |

受保护 Open API route 在 `index.go` 中单独注册，并使用 `RequestLogger("open-api")`。

---

## Authentication

API key 支持两种传入方式：

* `Authorization: Bearer <api-key>`
* `X-API-Key: <api-key>`

同时存在时优先使用 Bearer。middleware 解析后设置 `OpenAPIConsumerContext`，包含：

* `PartnerID`
* `AccountID`
* `Scopes`
* `Environment`
* `Authenticated`

raw API key 不能写入日志，也不能传入 usecase。

---

## Envelope Contract

成功响应：

```json
{"success":true,"data":{}}
```

失败响应：

```json
{"success":false,"error":{"code":"snake_case","message":"safe message"}}
```

错误 helper：

```go
httpresponse.ErrorResponse("unauthorized", "missing api key")
httpresponse.OpenAPIUsecaseError(c, err)
```

Open API route 可以直接 `c.JSON(status, envelope)`，因为它不受内部 `/api/*` response helper guard 约束。

---

## DTO Rules

* 公开响应 DTO 不复用内部 `/api/*` DTO。
* 公开 DTO 不返回 `models.*`。
* 公开字段要稳定，不能因为内部模型新增字段而自动暴露。
* route 从 usecase 获取 `XxxCo` 后映射为 `OpenAPI<Xxx>Response`。
* usecase 可根据 `fwusecase.Context.Consumer` 做业务一致性校验，例如账号只能访问自己的 account。

---

## Error Matrix

| Condition | Status | Code | Message |
| --- | --- | --- | --- |
| missing API key | `401` | `unauthorized` | `missing api key` |
| invalid API key | `401` | `unauthorized` | `invalid api key` |
| missing consumer context | `401` | `unauthorized` | `missing consumer context` |
| account context mismatch | `403` | `forbidden` | `account context mismatch` |
| account not found | `404` | `not_found` | `account not found` |
| internal failure | `500` | `internal_error` | safe message |

新增公开错误码必须是稳定 `snake_case`。

---

## Logging

Open API request log 使用：

```go
RequestLogger("open-api")
```

允许字段：

* `surface: "open-api"`
* `partner_id`
* `account_id`
* `environment`
* `request_id`
* `method`
* `route`
* `path`
* `status`
* `duration`

禁止记录：

* raw API key
* password
* session ID
* reset token
* request body / response body

---

## Tests Required

* API key middleware 测试 missing/invalid key envelope。
* route error 测试公开错误 envelope。
* request logging 测试 `surface:"open-api"` 和安全 consumer 字段。
* 修改 Open API 后运行 `go test ./...`。

---

## Common Mistakes

* 把内部 `/api/*` DTO 或 helper 复制到 Open API。
* 新增公开 route 但没有 `open_api_*` 文件边界。
* Open API 直接返回 model。
* 在日志里打印 raw API key。
* 在 usecase 中重新解析 header，而不是使用 `fwusecase.Context.Consumer`。
