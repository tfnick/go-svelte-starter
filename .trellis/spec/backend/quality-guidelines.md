# Quality Guidelines

> 本文是后端质量总则和检查清单。具体规则以对应主题 spec 为准，本文不重复展开 envelope、事务、日志等细节。

---

## Overview

后端开发优先保持三件事：

* 分层边界清楚：`routes -> usecase -> models -> db`。
* 对外契约稳定：内部 API、Open API、DTO、错误响应都有明确 helper。
* 变更可验证：新增规则尽量有单元测试、集成测试或 archguard 覆盖。

---

## Required Patterns

### Request Binding

请求 DTO 放在对应 `api/routes/*.go` 文件。认证相关接口当前同时支持 JSON 和 form 输入，通过 route-local bind helper 判断 `Content-Type`。

```go
func bindLoginRequest(c echo.Context) (LoginRequest, error) {
    var req LoginRequest
    if wantsJSON(c) {
        if err := c.Bind(&req); err != nil {
            return req, err
        }
        return req, nil
    }

    req.Email = c.FormValue("email")
    req.Password = c.FormValue("password")
    return req, nil
}
```

普通 JSON 接口可以直接 `c.Bind(&req)`，但响应仍必须走 `httpresponse` helper。

### Validation

* transport-level bind failure 在 route 中处理。
* 业务字段是否为空、状态是否合法、资源是否存在等由 usecase 负责，返回 `fwusecase.E(...)`。
* route 不应绕过 usecase 直接调用 model 来做业务判断。

### DTO Boundary

所有给前端的内部 `/api/*` 响应必须返回 route-local DTO，不直接暴露 `models.*`。详见 [API Contracts](./api-contracts.md)。

### Authentication

认证由 middleware 和 route context helper 处理：

* 内部 API 使用 `RequireAuth()`、`OptionalAuth()`、`fwcontext.InternalUsecaseContext(c)`。
* Open API 使用 `RequireOpenAPIKey()`、`fwcontext.OpenAPIUsecaseContext(c)`。
* usecase 接收 `fwusecase.Context`，不要依赖 Echo、cookie、header 或 raw API key。

### Dynamic SQL

动态 SQL 使用 `db.DynamicExecutorFor(ctx, name)` 和 `#[ ... ]` 条件片段。不要手写字符串拼接 WHERE 条件。

```go
eng, err := db.DynamicExecutorFor(ctx, "app")
if err != nil {
    return nil, err
}
err = eng.Select(&users, `
    SELECT * FROM users
    WHERE 1=1
        #[ AND id = :id ]
        #[ AND email LIKE :email ]
`, query)
```

---

## Forbidden Patterns

| Pattern | Reason |
| --- | --- |
| route 直接导入 `api/models` 或 `api/db` | 破坏分层边界，archguard 会失败 |
| usecase 导入 `api/db` 或 `api/framework/http` | usecase 应只表达应用流程 |
| model 导入 `api/usecase` 或 `api/routes` | model 不应依赖上层 |
| 内部 route 直接 `c.JSON(...)` | 会绕过统一 envelope helper |
| API 响应直接返回 `models.*` | 会暴露数据库结构 |
| raw EventBus 出现在任何生产代码 | EventBus 已退役；使用 `api/framework/events` durable facade |
| raw goqite 出现在 `api/framework/queue` 之外 | goqite 必须隔离在 framework queue 边界 |
| 创建 `domain_event_executions` 表，或把项目状态字段塞进 `goqite` 表 | durable event 使用 `domain_events` / `domain_event_deliveries`，但 `goqite` 仍是 component-owned 表 |
| 记录密码、token、API key、session cookie | 敏感信息不能进入日志 |

---

## Testing Requirements

* 常规后端变更运行 `go test ./...`。
* 修改内部 API envelope 或前端 API 客户端时运行 `cd frontend && npm test`。
* 修改前端源代码或 embed 相关行为时运行 `cd frontend && npm run build`。
* 修改分层规则、DTO 边界、domain event 边界时更新或运行 `api/framework/archguard` 测试。
* 修改事务逻辑时覆盖成功提交、失败回滚、nested transaction、after-commit 行为。

---

## Code Review Checklist

* [ ] 新 route 是否只处理 HTTP concern，并调用 usecase？
* [ ] 内部 `/api/*` 是否全部使用 `httpresponse` helper？
* [ ] 返回给前端的数据是否经过 route-local DTO？
* [ ] usecase 是否返回 `fwusecase.E(...)`，并保留内部 cause？
* [ ] model 是否使用 `db.ExecutorFor` 或 `db.DynamicExecutorFor`？
* [ ] UPDATE/DELETE 是否检查 `RowsAffected()`？
* [ ] 事务边界是否只在 usecase 中使用 `fwusecase.WithAppTx()`？
* [ ] 日志是否包含必要上下文，并避开敏感字段？
* [ ] Open API 是否仍保持独立 envelope、route 文件和认证规则？
* [ ] 相关测试是否已经运行？
