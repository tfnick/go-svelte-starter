# Logging Guidelines

> 本文是后端日志的权威说明。日志统一写入一个文件，通过结构化字段区分内部 API 与 Open API。

---

## Overview

当前项目使用 `github.com/rs/zerolog`。初始化逻辑位于 `api/framework/logging/logging.go`。

日志输出：

* stdout
* `logs/app.log`

所有环境都输出 JSON。开发模式只调整 level，不切换 pretty console writer。

---

## Logger API

```go
logging.Init(isDevelopment)
logging.For("component")
logging.IsDevelopment()
logging.Close()
```

默认文件：

```go
const DefaultLogPath = "logs/app.log"
```

启动时：

```go
if err := logging.Init(*isDevelopment); err != nil {
    panic(err)
}
logger := logging.For("main")
```

退出时关闭文件：

```go
defer func() {
    if err := logging.Close(); err != nil {
        logger.Error().Err(err).Msg("failed to close log file")
    }
}()
```

---

## Component Names

当前推荐 component：

| Component | Usage |
| --- | --- |
| `main` | 启动、关闭、server lifecycle |
| `db` | DB connect、migration、reopen |
| `http` | request logging、HTTP server error |
| `auth` | auth 相关开发辅助日志 |
| `events` | domain event validation、subscriber failure、durable delivery |

新增 component 前，先确认它代表稳定的能力边界。

---

## Request Logging

API route group 使用：

```go
api.Use(authMiddleware.RequestLogger("api"))
openAPI.Use(openAPIMiddleware.RequestLogger("open-api"))
```

每条 request log 至少包含：

* `component:"http"`
* `surface`
* `request_id`
* `method`
* `route`
* `path`
* `status`
* `duration`

`surface` 用来区分同一个日志文件中的来源：

| Surface | Meaning |
| --- | --- |
| `api` | 内部 Svelte-facing `/api/*` |
| `open-api` | 外部 partner-facing `/open-api/*` |

Open API 已认证请求可额外记录：

* `partner_id`
* `account_id`
* `environment`

---

## Request ID

`X-Request-ID` 规则：

* 请求带了 `X-Request-ID` 就保留。
* 请求没有则生成一个。
* 写入 Echo context。
* 写回响应 header。
* 写入 request log。
* server error log 中可带上同一个 request ID。

---

## Levels

| Level | Usage |
| --- | --- |
| `Debug` | 开发诊断，默认只在 dev level 可见 |
| `Info` | 正常生命周期和请求摘要 |
| `Error` | 可恢复失败 |
| `Fatal` | 启动或运行边界无法继续 |

开发模式 level 是 `debug`，生产模式 level 是 `info`。

---

## Sensitive Data

永远不要记录：

* password
* password hash
* raw API key
* session ID
* reset token
* request body / response body
* 完整用户或账户对象

当前允许的特例：开发模式下可记录本地 password reset URL，必须用 `logging.IsDevelopment()` 包裹。

---

## Tests Required

* `logging.Init()` 应创建 `logs/app.log` 并写入 JSON。
* request logging 应记录 `surface:"api"`，并返回 `X-Request-ID`。
* Open API request logging 应记录 `surface:"open-api"` 和安全 consumer 字段。
* dev/prod level 行为应有测试覆盖。

---

## Common Mistakes

* 使用 `fmt.Println` / `fmt.Printf` 输出内部日志。
* 在启动生命周期中使用 `echo.Logger.Fatal(...)`。
* 分裂出多个文件来区分 `api` / `open-api`，当前约定是一个文件加 `surface`。
* 打印 raw API key 或请求体。
* 开发模式切换到非 JSON 输出。
