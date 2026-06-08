# Directory Structure

> 本文是后端目录和分层边界的权威说明。API envelope、DTO、事务、日志等细节只在对应 spec 中展开。

---

## Current Architecture

后端依赖方向固定为：

```text
routes -> usecase -> models -> db
```

`routes` 是 HTTP adapter，负责 Echo、request/response DTO、middleware context、HTTP response helper，负责认证鉴权。
`usecase` 负责应用流程、业务校验、事务边界、跨模型编排，以及返回给 route 的 `XxxCo`，是业务逻辑复用的关键边界。
`models` 负责数据库读写、SQL、model-layer sentinel error、batch loader。
`db` 负责连接、migration、transaction-aware executor。
`framework` 负责业务无关的基础能力。

---

## Scenario: Marketing Pages And App Route Boundary

### 1. Scope / Trigger

Use this section when changing public marketing pages, `static.go`, `marketing.go`, or the route boundary between server-rendered pages, the Svelte app, and `/api/*`.

### 2. Signatures

```text
GET /                         -> Go html/template marketing page
GET /features                 -> Go html/template marketing page
GET /pricing                  -> Go html/template marketing page
GET /robots.txt               -> text/plain
GET /sitemap.xml              -> application/xml
GET /marketing/assets/*       -> embedded marketing asset
GET /app, /app/*              -> embedded Svelte index.html
GET /assets/*, /logo.png      -> embedded Svelte dist asset
GET /api/* missing route      -> API 404, never frontend HTML
```

Code ownership:

```text
marketing.go                  -> marketing templates, SEO endpoints, product CTA view model
marketing/templates/*.html    -> replaceable public page templates
marketing/assets/*            -> replaceable public page assets
static.go                     -> Svelte dist embed, /app app shell, legacy redirects, /api hard boundary
```

### 3. Contracts

* Public SEO pages are server-rendered with `html/template`; do not serve the Svelte SPA at `/`.
* The authenticated SaaS workspace starts at `/app`. New app pages should live under `/app/...`.
* Legacy app URLs such as `/login`, `/orders`, and `/products` may redirect to `/app/login`, `/app/orders`, and `/app/products`, but they should not become new marketing routes.
* `/api/*` remains an API namespace. Missing API paths must return API 404 rather than embedded `index.html`.
* Marketing pages may read enabled products through usecase-layer functions and build CTAs as `/app/checkout?product_id=<id>`.
* Marketing pages may use public site settings such as logo URL, but should fall back safely when settings storage is unavailable.
* `APP_PUBLIC_BASE_URL`, when set, is the canonical marketing origin and OAuth public origin.
* Production build still embeds both `frontend/dist` and `marketing/*` into the single Go executable.

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| request `/` | server-rendered marketing HTML with SEO tags, no `<div id="app"></div>` |
| request `/app/login` | embedded Svelte `index.html` |
| request `/pricing` with enabled Creem products | product cards include `/app/checkout?product_id=<id>` |
| request `/api/unknown` | 404 API response, not Svelte index |
| marketing product lookup fails | page still renders with safe empty-offer state |
| legacy `/login?redirect=...` | redirect to `/app/login?redirect=...` preserving query |

### 5. Good/Base/Bad Cases

Good: `/pricing` renders product cards from enabled products and links to `/app/checkout?product_id=...`.

Base: Fresh install with no products renders marketing pages and an empty pricing state.

Bad: Add a catch-all `router.GET("/*")` that serves Svelte `index.html`; this breaks SEO pages and hides missing public routes.

### 6. Tests Required

* Root route test asserts marketing HTML and canonical metadata.
* `/app/*` route test asserts embedded Svelte index.
* `/api/*` route test asserts API 404 is not swallowed by the frontend.
* `robots.txt` and `sitemap.xml` tests assert canonical URLs.
* Run `go test ./...`, `cd frontend && npm test`, and `cd frontend && npm run build`.

### 7. Wrong vs Correct

#### Wrong

```go
router.GET("/*", func(c echo.Context) error {
	return serveEmbeddedFrontend(c, dist)
})
```

#### Correct

```go
registerMarketingRoutes(router)
registerFrontendRoutes(router, isDevelopment, frontendDevURL) // owns /app and frontend assets
```

---

## Directory Layout

```text
index.go
marketing.go
marketing/
  templates/
  assets/
static.go
api/
  db/
    db.go
    tx.go
    migrations/
      app/*.sql
      shared/*.sql
  framework/
    archguard/
    data/
      modelerror/
      namelookup/
    database/
    events/
    http/
      auth/
      context/
      middleware/
      response/
    logging/
    usecase/
  models/
  routes/
  usecase/
    translate/
frontend/
  src/
  dist/
```

---

## Module Responsibilities

| Path | Responsibility |
| --- | --- |
| `index.go` | 程序入口、flag、DB 初始化、Echo route group、middleware 注册 |
| `static.go` | `frontend/dist` embed、SPA fallback、开发模式前端重定向 |
| `api/db` | DB manager、SQLite PRAGMA、migration、事务 executor |
| `api/framework/archguard` | 架构守卫测试，约束分层导入、DTO 边界、domain event 边界 |
| `api/framework/http/auth` | JWT 签发、解析、token claims 等 HTTP 认证基础能力 |
| `api/framework/http/context` | Echo context 到 `fwusecase.Context` 的转换 |
| `api/framework/http/middleware` | auth、Open API key、request logging |
| `api/framework/http/response` | 内部 API envelope helper、Open API error helper、usecase error mapper |
| `api/framework/usecase` | usecase context、typed error、`WithAppTx`、after-commit hook |
| `api/framework/events` | Durable DDD event facade，通过 `api/framework/queue` 使用 goqite，不让业务直接依赖 raw queue |
| `api/framework/data/*` | 模型层可复用的数据 helper，例如 `modelerror`、`namelookup` |
| `api/framework/logging` | Zerolog 初始化、文件 sink、component logger |
| `api/models` | sqlx-backed structs、查询/写入函数、model-layer validation |
| `api/usecase` | `XxxCmd`、`XxxQry`、`XxxCo`、业务流程、事务边界 |
| `api/usecase/translate` | ID/name lookup key 与 model batch loader 的绑定 |
| `api/routes` | Echo handler、request DTO、response DTO、`Co -> DTO` mapper |

---

## Import Boundaries

架构守卫测试位于 `api/framework/archguard`，核心边界是：

* `api/routes` 不能导入 `api/models` 或 `api/db`。
* `api/usecase` 不能导入 `api/routes`、`api/db` 或 `api/framework/http`。
* `api/models` 不能导入 `api/routes`、`api/usecase` 或 `api/framework/http`。
* `api/models` 可以导入 `api/framework/data/*`。
* raw `github.com/asaskevich/EventBus` 不允许出现在任何生产代码。
* raw `maragu.dev/goqite` 只能出现在 `api/framework/queue`。
* `api/models` 不能导入 `api/framework/events`。
* `api/usecase` 不能导入 `api/framework/http/auth`，JWT 是 route/middleware 的 HTTP concern。

---

## Naming Conventions

* Go package 名使用小写单词，尽量与目录一致。
* Go 文件名使用 `snake_case.go`。
* Open API 相关 route/model 文件使用 `open_api_*` 前缀。
* 请求 DTO 和响应 DTO 放在使用它的 `api/routes/*.go` 文件中。
* usecase 输入使用 `XxxCmd` 或 `XxxQry`，返回给 route 的对象使用 `XxxCo`。
* model struct 放在 `api/models`，不直接作为内部前端 API 响应。
* JSON 和 SQL 字段使用 `snake_case`。
* ID 当前使用 `TEXT` UUID，由 `github.com/google/uuid` 生成。

---

## Scenario: External Integration Boundary

### 1. Scope / Trigger

新增或修改外部系统接入能力时适用，包括 LLM、SMS、Payment、HRM、WeChat、Work WeChat 等 provider/channel adapter，以及它们的 DB 配置、凭证、调用记录。

### 2. Signatures

Recommended package boundaries:

```text
api/framework/integrations/<primitive>
api/usecase/integrations/<scenario>
api/integrations/<scenario>/<provider>
api/models/integration*.go
api/db/migrations/app/*_add_integrations.sql
```

Core DB tables:

```text
integration_channels
integration_credentials
integration_policies
integration_model_options
integration_operation_configs
integration_invocations
```

### 3. Contracts

* `api/framework/integrations/*` 只能放 provider-agnostic 基础能力，例如 credential encryption、normalized provider error、signing/auth/stream primitive。
* `api/usecase/integrations/<scenario>` 定义业务稳定 port 和 DTO，例如 LLM `Adapter`、`ProviderConfig`、`GenerateRequest`、`GenerateResult`，或 OSS `Adapter`、`ProviderConfig`、`PutObjectRequest`、`PresignObjectRequest`。
* `api/integrations/<scenario>/<provider>` 实现具体 provider adapter，只负责 provider DTO mapping、HTTP/SDK 调用、provider error normalization。
* `api/usecase` 通过 registry 或 bootstrap 注入 adapter，不导入 `api/integrations`。
* `index.go` 或启动 bootstrap 负责把 provider adapter 注册到 usecase 可见的 registry。
* 业务功能如果需要选择 channel/model，必须使用 DB 里的稳定 alias，例如 `integration_operation_configs.channel_code` 和 `model_code`；不要让产品用户传 raw provider model ID。
* 凭证明文只能出现在 framework credential 边界和 provider call 的极短运行时路径中，不进入 DTO、日志、事件或普通业务表。
* `integration_invocations` 只记录安全 metadata，例如 `channel_code`、`model_code`、`provider_request_id`、`usage_json`、`duration_ms`；不要保存 prompt、raw request body、raw response body。

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| `api/usecase` imports `api/integrations` | archguard fails |
| `api/routes` imports `api/integrations` or `api/models` | archguard fails |
| `api/integrations` imports `api/db`, `api/models`, `api/routes`, or `api/framework/http` | archguard fails |
| missing enabled channel/model/operation config | usecase returns `CodeInternal` with safe message |
| provider returns raw error/body | adapter maps it to normalized provider error before crossing into usecase |
| credential decrypt fails | usecase returns safe internal error and records failed invocation when possible |

### 5. Good/Base/Bad Cases

Good: route calls `usecase.SummarizeTextWithLLM`, usecase resolves `text_summary` config from DB, decrypts credential through `framework/integrations/credentials`, invokes registered DeepSeek adapter through the LLM port, and records `integration_invocations`.

Good: future artifact/PPT usecase resolves an enabled OSS channel from DB, decrypts `s3_access_key`, invokes a registered OSS adapter through `api/usecase/integrations/oss`, and keeps Aliyun/R2 SDK details under `api/integrations/oss/<provider>`.

Base: If `integration_operation_configs` is absent for an operation, resolver may fall back to enabled channel/model priority for MVP compatibility, but production operations should be explicitly configured.

Bad: route accepts `{provider_model_id:"deepseek-..."}` from product users and passes it to provider adapter; this bypasses admin/backend policy and DB-managed configuration.

### 6. Tests Required

* model tests cover `integration_*` config lookup and encrypted credential storage.
* usecase tests use fake adapters to assert alias-to-provider-model mapping, decrypted credential injection, validation, and invocation status.
* route tests assert internal envelope and route-local DTOs.
* provider adapter tests use fake HTTP clients and assert provider request mapping, response mapping, and normalized provider errors.
* Run `go test ./...` and archguard tests after adding or moving integration packages.

### 7. Wrong vs Correct

#### Wrong

```go
import "github.com/tfnick/go-svelte-starter/api/integrations/llm/deepseek"

func Summarize(...) {
    client := deepseek.NewAdapter(nil)
    // business usecase now depends on provider implementation
}
```

#### Correct

```go
// index.go
appusecase.RegisterLLMAdapter("llm.deepseek.openai_compatible", deepseek.NewAdapter(nil))

// usecase
adapter, ok := registeredLLMAdapter(config.Channel.AdapterKey)
```

---

## Retired Patterns

* `api/types` 已退役，不要新增共享 request/response struct。
* 不要把通用 helper、guard test、response helper 放进 `routes`、`usecase`、`models`。
* 不要让 route 直接调用 model/db。
* 不要把 JWT、API key、Echo context 检查放进 usecase。
* cookie/session 登录已退役，不要新增 `session_id` cookie、session model 函数或基于 cookie 的登录态。
