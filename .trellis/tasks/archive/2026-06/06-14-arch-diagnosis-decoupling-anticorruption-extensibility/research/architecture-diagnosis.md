# Architecture Diagnosis

## Scope

本报告基于当前代码和 `.trellis/spec/`，从解耦、防腐、可扩展性三个角度审计项目架构。审计重点是发现后续增长中的结构性风险，并给出可拆分的改进路线；本任务不直接进行大规模重构。

## Executive Summary

当前项目已经不是“缺少架构”的状态。后端有清晰的 `routes -> usecase -> models -> db` 分层，外部集成已经采用 `api/usecase/integrations/*` ports + `api/providers/*` adapters，事件机制也已经从 raw queue/EventBus 收敛到 `api/framework/events` durable facade。`api/framework/archguard` 已经覆盖了一批关键规则，并且 `go test ./api/framework/archguard` 通过。

主要优化空间不在于推翻现有架构，而在于控制增长后的集中化压力：

* `index.go` 同时承担启动、provider 注册、事件注册、queue runner、路由表，随着模块继续增加会变成 bootstrap 的中心化瓶颈。
* `frontend/src/api.js` 是唯一 API client 边界，这是好事；但它已经承载所有领域 helper，后续会成为前端模块扩展的单文件膨胀点。
* 部分新业务 usecase 文件开始变成长流程聚合，例如 `api/usecase/kb_chat.go` 同时承担 RAG、conversation、lead capture、admin console 查询等职责。
* provider/config/invocation 模式已经成型，但不同场景仍有局部重复，例如 LLM、embedding、payment、OSS 都有各自的 provider config 解析、adapter lookup、invocation failure completion 逻辑。
* archguard 对 Go 后端边界较强，对前端“页面只能通过 API helper 调用 API”的边界目前主要靠约定和测试，没有等价的静态守卫。

## What Is Already Strong

### Backend Layering

Evidence:

* `.trellis/spec/backend/directory-structure.md` 定义了 `routes -> usecase -> models -> db`。
* `api/framework/archguard/layer_boundary_test.go` 检查：
  * `api/routes` 不能导入 `api/models`、`api/db`、`api/providers`
  * `api/usecase` 不能导入 `api/routes`、`api/db`、`api/providers`、`api/framework/http`
  * `api/models` 不能导入 `api/routes`、`api/usecase`、`api/framework/http`
  * `api/providers` 不能导入 `api/db`、`api/models`、`api/routes`、`api/framework/http`
* `api/framework/archguard/frontend_dto_boundary_test.go` 防止 route 直接返回 `models.*`。
* `api/framework/archguard/layer_boundary_test.go` 防止内部 route 直接 `c.JSON(...)`，要求走 `api/framework/http/response` envelope helper。

Assessment:

分层边界已经有可执行守卫，属于当前架构最稳的部分。后续不建议引入重型 Clean Architecture/DDD 框架；保持当前轻量分层更符合项目体量。

### Integration Anti-Corruption

Evidence:

* `api/usecase/integrations/payment/ports.go`
* `api/usecase/integrations/oss/ports.go`
* `api/usecase/integrations/llm/ports.go`
* `api/usecase/integrations/embedding/ports.go`
* `api/providers/payment/creem/creem.go`
* `api/providers/oss/s3compatible/s3compatible.go`
* `api/providers/llm/deepseek/deepseek.go`
* `api/providers/embedding/deepseek/deepseek.go`
* `index.go` 通过 `RegisterLLMAdapter`、`RegisterEmbeddingAdapter`、`RegisterPaymentAdapter`、`RegisterOSSAdapter` 注册 adapter。

Assessment:

第三方 SDK/HTTP 细节基本被隔离在 `api/providers/*`，usecase 依赖的是 port 和 registry，不直接 import provider。`api/framework/integrations/providererror` 也给 provider error normalization 提供了统一出口。这是良好的防腐层形态。

### Eventing Boundary

Evidence:

* `.trellis/spec/backend/eventing-guidelines.md`
* `api/framework/events`
* `api/framework/queue`
* `api/usecase/events`
* `api/framework/archguard/layer_boundary_test.go` 中的 `TestDomainEventsStayQueueBacked`、`TestBusinessRealtimePublishingStaysBehindNotificationBoundary`、`TestDurableEventTablesStayProjectOwned`

Assessment:

事件机制已经有明确边界：业务只发布 domain event，raw goqite 只在 framework queue，realtime publishing 通过 notification boundary。该部分不需要重构，后续重点是保证新增事件继续使用 typed payload、idempotent subscriber 和 archguard。

### Frontend API Boundary

Evidence:

* `frontend/src/api.js` 是唯一直接调用 `fetch(...)` 的前端源码文件。
* `rg 'fetch\(' frontend/src` 只命中 `frontend/src/api.js`。
* `.trellis/spec/frontend/svelte-vite-embed.md` 明确 Svelte component 必须通过 API helper，不直接 fetch `/api/*`。

Assessment:

单一 API client 边界是正确方向。问题不是边界缺失，而是单文件继续增长后的维护成本。

## Risks And Recommendations

### P1: Split Bootstrap Responsibilities Out Of `index.go`

Evidence:

`index.go` 目前集中承担：

* runtime env loading
* DB manager 初始化和 migrations
* queue manager 初始化
* provider adapter registration
* domain event handler registration
* queue runner startup
* Echo middleware
* internal API route table
* public support routes
* Open API routes
* marketing/frontend route registration

Risk:

新增 provider、queue、route、event subscriber 都要修改 `index.go`。文件会成为 merge conflict 和 accidental coupling 的热点。它也让“新增一个业务模块”的改动表面上跨越太多概念，降低可扩展性。

Recommendation:

分阶段抽出启动注册函数，不改变运行行为：

* `api/bootstrap/providers.go` 或根层 `bootstrap_providers.go`：集中 provider adapter registration。
* `api/bootstrap/events.go` 或根层 `bootstrap_events.go`：集中 event handler registration。
* `api/bootstrap/queues.go` 或根层 `bootstrap_queues.go`：集中 queue runner startup。
* `api/routes/register.go`：按领域拆分 route registration，例如 `RegisterAuthRoutes`、`RegisterOrderRoutes`、`RegisterAdminRoutes`、`RegisterIntegrationWebhookRoutes`。

Suggested follow-up task:

* “Refactor bootstrap registration out of index.go without behavior changes”

Validation:

* `go test ./...`
* Smoke route registration by existing route tests.

### P1: Split `frontend/src/api.js` Into Domain API Modules While Preserving One Public Boundary

Evidence:

`frontend/src/api.js` contains auth, users, dictionaries, orders, products, notifications, scheduler, events, messages, parameter integration channels, settings, variables, realtime, tasks, support chat, KB, support console helpers.

Risk:

The file is a good boundary but an increasingly poor unit of change. Every new frontend-facing feature edits the same file and the same `api.test.js`, which increases conflict probability and makes ownership unclear.

Recommendation:

Keep `request(...)`, auth token handling, envelope unwrap, and shared helpers in `frontend/src/api/request.js` or equivalent. Move domain helpers into small modules:

```text
frontend/src/api/
  request.js
  auth.js
  orders.js
  parameters.js
  settings.js
  support.js
  kb.js
  index.js
```

`frontend/src/api.js` can remain as a compatibility barrel during migration, re-exporting from the domain modules.

Suggested follow-up task:

* “Modularize frontend API helpers behind existing api.js compatibility export”

Validation:

* `cd frontend && npm test`
* `cd frontend && npm run build`
* Add/adjust tests so pages still import stable helpers.

### P1: Extract Support Chat Usecase Responsibilities

Evidence:

`api/usecase/kb_chat.go` is one of the largest usecase files and currently combines:

* public support chat message handling
* RAG embedding and retrieval
* LLM answer generation
* support conversation start/list/detail
* lead capture intent detection
* support lead creation
* support console admin list/detail logic
* rate limiting helpers

Risk:

The file mixes multiple cohesive areas. This does not violate the current layer boundary, but it weakens internal usecase modularity and makes future changes risky: RAG tuning, lead capture, admin console, and conversation persistence can trip over each other.

Recommendation:

Split by usecase responsibility while staying inside `api/usecase`:

```text
api/usecase/support_chat.go          public chat flow and message send
api/usecase/support_rag.go           RAG prompt/retrieval/generation orchestration
api/usecase/support_lead.go          lead capture detection and lead creation
api/usecase/support_console.go       admin console queries
api/usecase/support_rate_limit.go    rate limit helpers
```

Do not introduce a new architectural layer. This is a cohesive-file split.

Suggested follow-up task:

* “Split support chat usecase by responsibility without API changes”

Validation:

* Existing support chat, KB, and route tests.
* `go test ./...`

### P2: Standardize Provider Invocation Lifecycle

Evidence:

* `api/usecase/llm_summary.go` manually creates and completes `integration_invocations`.
* `api/usecase/payment.go` manually creates and completes `integration_invocations`.
* `api/usecase/payment_webhook.go` has its own receipt lifecycle.
* Config parsing appears separately in `llmProviderConfig`, `embeddingProviderConfig`, `paymentProviderConfig`, `siteLogoProviderFromChannel`.

Risk:

The provider port pattern is sound, but invocation recording and failure mapping are repeated enough that behavior can drift between scenarios. For example, duration, retryable flag, provider request id, and provider error category can be handled inconsistently.

Recommendation:

Add a small framework/usecase helper only after a third or fourth scenario needs identical lifecycle behavior. Candidate shape:

```text
api/framework/integrations/invocation
```

or, if it needs app model access, keep it in `api/usecase` as an integration helper. Avoid moving business operation semantics into framework. The helper should own only lifecycle boilerplate:

* start invocation
* complete success
* complete provider failure
* normalize providererror metadata
* capture duration

Suggested follow-up task:

* “Evaluate and extract integration invocation lifecycle helper”

Validation:

* LLM/payment tests should assert invocation rows before/after extraction.

### P2: Add Frontend Static Guard For Direct API Calls

Evidence:

* Current convention is in `.trellis/spec/frontend/svelte-vite-embed.md`.
* `rg 'fetch\(' frontend/src` currently only finds `frontend/src/api.js`.
* There is no archguard-equivalent test preventing future Svelte files from calling `fetch('/api/...')` directly.

Risk:

A future page can accidentally bypass `request(...)`, losing auth header injection, envelope unwrap, safe error handling, and FormData behavior.

Recommendation:

Add a lightweight frontend test or Node script that fails if `fetch(` appears in `frontend/src` outside allowed API boundary files. This is low risk and mirrors the backend archguard philosophy.

Suggested follow-up task:

* “Add frontend API boundary guard test”

Validation:

* `cd frontend && npm test`

### P2: Convert Large Admin Pages Gradually Into Local Components

Evidence:

Largest frontend pages include:

* `frontend/src/pages/Parameters.svelte`
* `frontend/src/pages/Dashboard.svelte`
* `frontend/src/pages/Dictionary.svelte`
* `frontend/src/pages/Scheduler.svelte`
* `frontend/src/pages/Experiments.svelte`

Risk:

Large page files make UI behavior, form state, table state, API calls, and formatting harder to reason about. This is a frontend maintainability issue, not an architectural breach.

Recommendation:

Only split when touching the page for feature work. Prefer page-local components under a domain folder, for example:

```text
frontend/src/pages/parameters/
  Parameters.svelte
  ChannelForm.svelte
  ChannelTable.svelte
  SchemaField.svelte
```

Avoid a generic component library unless repetition appears across multiple pages.

Suggested follow-up task:

* “Extract Parameters page local components during next Parameter feature”

### P2: Document Public Support Chat Boundary In Specs

Evidence:

Recent code added:

* `api/routes/support_chat.go`
* `api/routes/support_console.go`
* `api/models/support.go`
* `api/usecase/kb_chat.go`
* `frontend/src/components/SupportChat.svelte`
* support chat helpers in `frontend/src/api.js`

Risk:

Support chat crosses public anonymous API, rate limiting, privacy hashing, RAG, LLM, KB, lead capture, and admin console. Current specs mention many scenarios, but support chat does not yet appear as a dedicated scenario in the backend/frontend specs. Future changes may scatter rules between KB, LLM, support, and public API without a single contract.

Recommendation:

Add a dedicated spec section for public support chat and lead capture. It should define:

* public route signatures
* visitor token/IP hashing rules
* rate limit boundary
* RAG/LLM provider boundary
* lead capture required fields
* admin console DTO boundary
* frontend helper/component expectations

Suggested follow-up task:

* “Add support chat architecture contract to Trellis specs”

## Anti-Corruption Review

### Provider DTOs

The provider DTO boundary is generally healthy. Provider-facing request/result structs live under `api/usecase/integrations/<scenario>`, while raw provider mapping stays under `api/providers/<scenario>/<provider>`.

Concern:

Some usecase provider config structs are scenario-local and repeated:

* `paymentChannelConfig`
* `llmChannelConfig`
* embedding channel config in `kb_embedding_config.go`
* OSS channel config in `setting.go`

This is acceptable today because each scenario has different semantics. Do not prematurely abstract all config parsing into one generic mechanism. Instead, extract only repeated lifecycle and validation mechanics when drift appears.

### Secrets And Raw Payloads

The architecture has the right direction:

* credentials live behind integration credential records and provider config conversion
* webhook raw payloads are encrypted in `payment_webhook.go`
* `providererror` prevents raw provider details from crossing the boundary as user-visible messages

Concern:

`parameter.go` and tests show `value_text` compatibility behavior for credentials. This may be intentional, but it should be treated as a migration/compatibility boundary and documented clearly. For long-term hardening, prefer one canonical secret storage path.

## Extensibility Review

### Adding A New Provider

Current flow:

1. Add port implementation under `api/providers/<scenario>/<provider>`.
2. Register adapter in `index.go`.
3. Add parameter schema in `api/usecase/parameter_schema.go`.
4. Seed/update integration channel dictionaries or migrations if needed.
5. Add tests.

Assessment:

This is workable, but concentrated registration/schema edits are a friction point. Splitting bootstrap registration and keeping schema definitions discoverable would help.

### Adding A New Business Module

Current flow:

1. Add model file.
2. Add usecase file.
3. Add route file.
4. Register routes in `index.go`.
5. Add frontend API helper in `frontend/src/api.js`.
6. Add page/component and router entry.
7. Add spec/archguard/tests.

Assessment:

The data flow is clear, but both backend and frontend have central files that every module touches. This is the main scalability bottleneck.

### Adding A New Event Subscriber

Current flow is healthy:

* event payload and handler live under `api/usecase/events`
* register in startup
* publish through `api/framework/events`
* subscriber idempotency covered by tests/spec

Recommendation:

Keep current event model. The only improvement is to split event registration out of `index.go` as part of bootstrap modularization.

## Proposed Roadmap

### Phase 1: Guard And Document

Low-risk tasks:

* Add frontend API boundary guard test.
* Add support chat architecture contract to `.trellis/spec/`.
* Add a short bootstrap ownership note to backend directory spec: `index.go` should stay thin; provider/event/route registration can be delegated.

### Phase 2: Reduce Central File Pressure

Behavior-preserving refactors:

* Split provider/event/queue/route registration out of `index.go`.
* Modularize `frontend/src/api.js` behind a compatibility export.
* Split `api/usecase/kb_chat.go` by responsibility.

### Phase 3: Normalize Integration Lifecycle

Only after Phase 2:

* Extract provider invocation lifecycle helper if repeated behavior remains obvious.
* Review credential storage compatibility path and document or migrate to one canonical shape.

## Recommended Next Trellis Tasks

1. `frontend-api-boundary-guard`: Add test preventing direct `fetch('/api/*')` outside the frontend API boundary.
2. `support-chat-architecture-spec`: Capture support chat public API, RAG, lead capture, privacy, and admin console boundaries in `.trellis/spec/`.
3. `bootstrap-registration-split`: Move provider, event, queue, and route registration out of `index.go` without behavior change.
4. `frontend-api-modularization`: Split `frontend/src/api.js` into domain modules while preserving current imports.
5. `support-chat-usecase-split`: Split `api/usecase/kb_chat.go` into cohesive files without API changes.

## Verification

Ran:

```sh
go test ./api/framework/archguard
```

Result:

```text
ok github.com/tfnick/go-svelte-starter/api/framework/archguard
```

No full test suite was run because this diagnostic report did not change production code.
