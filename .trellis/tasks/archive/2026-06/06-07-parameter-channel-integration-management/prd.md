# Parameter Channel Integration Management

## Goal

新增登录后菜单 `Parameter`，集中管理外部渠道对接参数。页面右侧使用 daisyUI `radio tabs-lift + tab content` 风格展示 `Payment`、`LLM`、`SMS` 三个 Tab。每个 Tab 下提供对应 scenario 的渠道集成参数管理：不分页列表、新增、编辑、enable/disable。

## Confirmed MVP Scope

本任务管理 `integration_channels + integration_credentials`：

* `integration_channels` 保存渠道启停、优先级、环境、adapter 选择、非敏感 `config_json` 和 `metadata_json`。
* `integration_credentials` 保存管理员配置的 `credential_value`，当前不做 DB 加密；前端编辑时使用普通输入遮罩和显示/隐藏按钮保护视觉展示。
* `Payment`、`LLM`、`SMS` 只作为 scenario 维度展示和过滤。
* `policy`、`operation_config`、LLM `model_options`、callback receipt、invocation inspection 暂不进入本任务。

## Anti-Corruption Layer Check

对照 `.trellis/tasks/06-06-external-integration-anticorruption-layer`，本设计遵守以下边界：

* 管理页面只维护 DB-backed channel/credential 配置，不引入 provider SDK、provider DTO 或 provider webhook payload。
* `adapter_key` 只选择 code-owned adapter，DB 不存可执行逻辑。
* `config_json` 和 `metadata_json` 只允许非敏感 provider 参数，例如 `base_url`、`product_id`、`success_url`。
* credential value 只用于 `users.is_admin=1` 的后台管理员配置页面；不进入日志、事件或普通用户页面状态。
* route 返回 route-local DTO，不直接返回 `models.*`，响应遵守内部 API envelope。
* usecase 负责事务和业务校验；model 只做 SQL 数据访问；route 不直接访问 model/db。
* 本任务不暴露 callback raw payload、provider request/response body、prompt、stream chunk 或 invocation raw data。

## Requirements

* 新增登录后侧边菜单 `Parameter`，路由为 `/parameters`，兼容 alias `/parameters.html`，仅管理员可见。
* `Parameter` 页面右侧包含 `Payment`、`LLM`、`SMS` 三个 Tab。
* Tab 使用 daisyUI `tabs tabs-lift` + radio input pattern，并提供每个 tab 的 content 区域。
* 每个 Tab 加载对应 scenario 的不分页渠道列表。
* 每个渠道支持新增、编辑、enable/disable。
* 列表展示 `channel_code`、`provider_code`、`adapter_key`、`environment`、`priority`、`enabled`、`callback_enabled`、`credential_type`、configured 状态、`updated_at`。
* 表单支持 `channel_code`、`provider_code`、`adapter_key`、`environment`、`priority`、`enabled`、`callback_enabled`、`config_json`、`metadata_json`、`credential_type`、`credential_value`。
* 编辑时后端 admin DTO 回填 `credential_value`；前端对 schema secret 字段默认 password mask，并提供显示/隐藏按钮。
* 后端 DTO 不包含 legacy `ciphertext`、`key_version`、`masked_value` 或 `credential_plaintext`。
* `config_json`、`metadata_json` 必须是合法 JSON object，且不得包含明显敏感 key。

## Schema Upgrade

* Add a code-owned adapter schema registry keyed by `adapter_key`.
* Expose `GET /api/parameters/integration-schemas?scenario=payment|llm|sms` so the UI can render config and credential fields dynamically.
* Keep Advanced JSON for adapters without schema and for extra non-sensitive config/metadata fields.
* Backend save usecases must validate schema-managed fields with the same code-owned schema, including required fields, URL format, numeric/boolean types, provider/scenario/credential type matching, and credential placement.
* Payment Creem schema stores `base_url`, `product_id`, optional `success_url`, and optional `units` in `config_json`; `api_key` and `webhook_secret` stay in `credential_value` as a JSON bundle.
* DeepSeek LLM schema stores `base_url` in `config_json`; its `api_key` credential remains a plain `credential_value` string to match the current adapter.
* SMS tab exposes a reserved `sms.aliyun.adapter` schema so the UI can render an adapter dropdown; the real SMS provider adapter remains out of scope.
* Dictionary capability is used for code-like selects such as `integration_environment`; provider URLs remain schema fields/options instead of dictionary values because dictionary values are normalized codes, not arbitrary URLs.
* `credential_type` uses the `integration_credential_type` dictionary with seeded values `payment_bundle` and `api_key`; the Parameter form renders it as a select, and the backend save usecase rejects values outside enabled dictionary options.

## Acceptance Criteria

* [ ] 登录后侧边菜单出现 `Parameter`，点击进入 `/parameters`。
* [ ] `/parameters` 页面有 `Payment`、`LLM`、`SMS` 三个 lifted radio tabs。
* [ ] 每个 tab 通过 API 加载对应 scenario 的不分页渠道列表。
* [ ] 可以新增渠道并创建 credential value。
* [ ] 可以编辑渠道非敏感字段；credential value 默认 mask，点击显示按钮可查看和修改。
* [ ] 可以 enable/disable 渠道。
* [ ] 前端 DTO 和页面不暴露 legacy credential ciphertext/plaintext/masked fields；列表不展示具体 credential value。
* [ ] 后端 route/usecase/model 测试覆盖 list/create/update/enable 行为、schema 校验和敏感字段边界。
* [ ] 前端 api/router 测试覆盖新增 helper 与菜单路由。
* [ ] 更新后端和前端 spec，记录 Parameter 管理 API/UI 契约。
* [ ] `go test ./...`、`cd frontend && npm test`、`cd frontend && npm run build` 通过。

## Data Flow

```text
Parameters.svelte form
  -> frontend/src/api.js helper
  -> /api/parameters/integration-schemas
  -> /api/parameters/integration-channels/*
  -> route request/response DTO
  -> usecase ParameterIntegration*Cmd/Qry
  -> models integration channel/credential CRUD
  -> app DB transaction
  -> usecase IntegrationChannelCo with admin credential_value
  -> route DTO with admin credential_value and without legacy credential fields
  -> frontend table/form
```

## Out of Scope

* 不实现 SMS provider adapter。
* 不实现 policy 管理 UI。
* 不实现 `integration_operation_configs` 管理 UI。
* 不实现 LLM `integration_model_options` 管理 UI。
* 不实现 invocation/callback receipt 运维页面。
* 不实现复杂 credential rotation 历史，只提供当前 credential 更新。

## Technical Notes

* Backend specs: `.trellis/spec/backend/index.md`、`directory-structure.md`、`route-handler-guidelines.md`、`api-contracts.md`、`database-guidelines.md`、`error-handling.md`、`logging-guidelines.md`、`quality-guidelines.md`。
* Frontend specs: `.trellis/spec/frontend/index.md`、`svelte-vite-embed.md`。
* Cross-layer guide: `.trellis/spec/guides/cross-layer-thinking-guide.md`。
* Design source: `.trellis/tasks/06-06-external-integration-anticorruption-layer/prd.md` and `info.md`。
