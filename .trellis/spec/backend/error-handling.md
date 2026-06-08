# Error Handling

> 本文是后端错误模型和安全错误响应的权威说明。HTTP envelope 的完整契约见 [API Contracts](./api-contracts.md) 和 [Open API Guidelines](./open-api-guidelines.md)。

---

## Overview

当前错误处理分三层：

* `models` 返回普通 Go error、`sql.ErrNoRows`、或 framework/model sentinel error。
* `usecase` 用 `fwusecase.E(code, message, cause)` 转成 typed error。
* `routes` 用 `httpresponse.InternalUsecaseError` 或 `httpresponse.OpenAPIUsecaseError` 映射成安全 HTTP 响应。

客户端只看到 safe message。内部原因保留在 error cause 中，并通过日志记录。

---

## Usecase Error Codes

`api/framework/usecase/errors.go` 定义了当前错误码：

| Code | Meaning |
| --- | --- |
| `CodeValidation` | 请求或业务输入不合法 |
| `CodeNotFound` | 资源不存在 |
| `CodeUnauthorized` | 未认证 |
| `CodeForbidden` | 已认证但无权限 |
| `CodeConflict` | 资源冲突，例如库存不足 |
| `CodeInternal` | 内部错误 |

usecase 应返回：

```go
return fwusecase.E(fwusecase.CodeValidation, "user ID is required", nil)
```

带内部原因时：

```go
return fwusecase.E(fwusecase.CodeInternal, "failed to load order", err)
```

`Message` 是客户端可见安全消息；`Cause` 是日志用内部原因。

---

## HTTP Mapping

内部 `/api/*` 使用 `httpresponse.InternalUsecaseError(c, err)`：

| Usecase code | Status | Error code |
| --- | --- | --- |
| `CodeValidation` | `400` | `validation` |
| `CodeUnauthorized` | `401` | `unauthorized` |
| `CodeForbidden` | `403` | `forbidden` |
| `CodeNotFound` | `404` | `not_found` |
| `CodeConflict` | `409` | `conflict` |
| default / internal | `500` | `internal_error` |

Open API 使用 `httpresponse.OpenAPIUsecaseError(c, err)`，状态码和错误码一致，但 envelope 由 Open API helper 构造。

---

## Model Error Rules

模型层只描述数据访问事实，不决定 HTTP 状态。

* 查询单条记录时，调用方需要区分 `sql.ErrNoRows`。
* 更新/删除后应检查 `RowsAffected()`，未命中时返回 `modelerror.ErrNotFound` 或领域内 sentinel error。
* 模型层遇到数据库失败时应保留上下文并 wrap error。
* 模型层不要返回 `fwusecase.Error`，也不要导入 `api/usecase`。

示例：

```go
result, err := exec.Exec(query, id)
if err != nil {
    return fmt.Errorf("delete user failed: %w", err)
}
rows, err := result.RowsAffected()
if err != nil {
    return fmt.Errorf("read affected rows failed: %w", err)
}
if rows == 0 {
    return modelerror.ErrNotFound
}
```

---

## Safe Message Rules

* 不要把 `err.Error()` 直接返回给客户端。
* 对 500 类错误，客户端消息必须是固定、安全、业务可理解的文本。
* 对安全敏感流程，例如忘记密码，无论资源是否存在，都返回相同成功消息。
* 日志可以记录 cause，但不要记录密码、token、API key、session cookie、请求体或响应体。

---

## Common Mistakes

* 在 route 中用字符串匹配 `err.Error()` 决定状态码。
* 在 model 中返回 usecase error。
* 在 usecase 中丢掉内部 cause，导致日志无法定位问题。
* 直接把 `sql.ErrNoRows` 暴露到 route。
* 500 响应把数据库错误、SQL、文件路径返回给前端。
