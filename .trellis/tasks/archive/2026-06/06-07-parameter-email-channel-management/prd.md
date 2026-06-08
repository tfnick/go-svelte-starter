# Parameter Email Channel Management

## Goal

在后台 `Parameter` 管理中新增 `Email` 渠道管理能力，与现有 `Payment`、`LLM`、`SMS` 渠道配置平级。目标是让管理员可以配置邮件服务渠道参数，默认覆盖 `aliyun` 和 `resend` 两类 provider，为后续邮件通知、验证码、营销/事务邮件等功能提供统一的集成参数来源。

## What I Already Know

* 用户希望在 `Parameter` 下新增 `Email` 渠道管理。
* `Email` 应与现有的 `Payment`、`LLM`、`SMS` 平级。
* 建议默认邮箱 provider 为 `aliyun` 与 `resend`。
* 当前 Parameter 管理已有 code-owned schema、动态表单、Advanced JSON、credential 字典、enable/disable 等能力。
* 当前后端 `normalizeIntegrationScenario` 只允许 `payment`、`llm`、`sms`。
* 当前前端 `Parameters.svelte` 的 tab/scenario 状态只包含 `payment`、`llm`、`sms`。
* 当前 schema registry 位于 `api/usecase/parameter_schema.go`，按 `adapter_key` 定义 `config_fields` 与 `credential_fields`。
* 当前 DB 表 `integration_channels` / `integration_credentials` 已能承载新 scenario，不需要新表。
* Aliyun 免费企业邮箱官方支持 SMTP 方式，认证方式是邮箱账号 + 客户端授权密码；推荐 SMTP 服务器 `smtp.qiye.aliyun.com`，SSL 端口 `465`。

## Assumptions (Temporary)

* 新增 `Email` 属于 integration channel scenario 扩展，而不是新增独立业务表。
* 后端应把 `email` 加入允许的 scenario，并提供至少两个 adapter schema：`email.aliyun.smtp` 和 `email.resend.api`。
* 前端应新增 `Email` tab，并复用现有 Parameter 页面表单、schema 动态渲染和不分页列表能力。
* 本任务聚焦渠道参数管理，不实现实际邮件发送 provider adapter。

## Open Questions

* None.

## Requirements (Evolving)

* `Parameter` 页面新增 `Email` tab，与 `Payment`、`LLM`、`SMS` 平级。
* Email tab 支持不分页列表、新增、编辑、enable/disable。
* 后端 `/api/parameters/integration-schemas?scenario=email` 返回 email 相关 adapter schema。
* 后端 `/api/parameters/integration-channels?scenario=email` 能管理 email scenario 的 channel。
* 默认 email providers 包含 `aliyun` 和 `resend`。
* 表单字段继续由 code-owned schema 驱动，并保留 Advanced JSON。
* 后端使用同一 schema 做 required、URL、敏感字段位置等校验。
* `integration_channels.scenario` 允许 `email`。
* Aliyun Email 使用 SMTP 企业邮箱账号 + 客户端授权密码认证，不使用 `api_key`。
* Credential type 字典新增 `smtp_password`，用于 SMTP username/password 凭证。
* `email.aliyun.smtp` schema：
  * Config fields: `smtp_host` required text default `smtp.qiye.aliyun.com`、`smtp_port` required number default `465`、`security` required select default `ssl`、`from_email` required text、optional `from_name` text。
  * Credential format: `json_object`。
  * Credential fields: `username` required secret/text、`password` required secret，且界面提示该值应填写邮箱客户端授权密码，不是账号登录密码。
  * Credential type: `smtp_password`。
* `email.resend.api` schema 暂按 Resend API key 管理：
  * Config fields: `base_url` required URL default `https://api.resend.com`、`from_email` required text、optional `from_name` text。
  * Credential format: `plain`。
  * Credential fields: `api_key` required secret。
  * Credential type: `api_key`。

## Acceptance Criteria (Evolving)

* [x] `Parameter` 页面显示 `Email` tab。
* [x] `Email` tab 下 adapter key 可选择默认的 `Aliyun SMTP` 与 `Resend API` schema。
* [x] 可以创建、编辑、启用/禁用 email integration channel。
* [x] Aliyun SMTP credential 以 username/password JSON object 保存，且 password 不出现在 `config_json`；界面提示 password 是客户端授权密码，不是账号登录密码。
* [x] Resend API credential 以 plain API key 保存，且 api_key 不出现在 `config_json`。
* [x] 后端拒绝非法 email scenario/schema mismatch/敏感字段放入 `config_json`。
* [x] 前端和后端测试覆盖 email scenario 和 schema。
* [x] 相关 backend/frontend spec 更新。

## Definition of Done

* Tests added/updated where appropriate.
* `go test ./...` passes.
* `cd frontend && npm test` passes.
* `cd frontend && npm run build` passes.
* Spec updated if API/DTO/schema behavior changes.

## Out of Scope

* 不实现实际邮件发送 usecase。
* 不接入真实 Aliyun/Resend SDK 或 HTTP API。
* 不新增邮件模板管理。
* 不新增邮件发送日志/回调/投递状态管理。

## Technical Notes

* 已检查文件：`api/usecase/parameter.go`、`api/usecase/parameter_schema.go`、`api/routes/parameter.go`、`api/models/integration.go`、`frontend/src/pages/Parameters.svelte`、`frontend/src/api.js`。
* 待更新 spec：`.trellis/spec/backend/api-contracts.md`、`.trellis/spec/frontend/svelte-vite-embed.md`。
* 预计后端改动：新增 `models.IntegrationScenarioEmail`、扩展 `normalizeIntegrationScenario`、新增 `email.aliyun.smtp` / `email.resend.api` schema、更新 `integration_credential_type` seed、更新 usecase/routes tests。
* 预计前端改动：`Parameters.svelte` 增加 `Email` scenario 状态和 tab、更新 `api.test.js` 覆盖 `scenario=email` helper path。

## Technical Approach

复用现有 Parameter integration channel 架构，不新增 DB 表。Email 只是新的 integration scenario：

* Backend: `IntegrationScenarioEmail` + `email` scenario validation + code-owned adapter schemas。
* Schema: Aliyun 使用 SMTP 账号密码，Resend 使用 API Key。
* Frontend: `Parameters.svelte` 增加 `Email` tab，动态表单继续由 schema 驱动。
* Spec: 更新 backend/frontend Parameter contract，明确 `email` scenario 和两个默认 adapter schema。
* Migration: baseline `002_seed.sql` 新增 `smtp_password`，并通过 `008_add_email_integration_seed.sql` 为已迁移数据库补种该字典值。

## Decision (ADR-lite)

**Context**: Email provider 的凭证形态不同。Aliyun 免费企业邮箱官方推荐 SMTP 账号密码；Resend 更典型的是 API Key。

**Decision**: Aliyun 使用 `email.aliyun.smtp` + `smtp_password` JSON object credential；Resend 使用 `email.resend.api` + `api_key` plain credential。

**Consequences**: Email tab 下会出现两种 credential format，但都复用现有 schema 动态表单与后端校验。实际邮件发送 adapter 仍不在本任务范围内。

## Implementation Summary

* 后端已新增 `email` integration scenario、`email.aliyun.smtp` 和 `email.resend.api` schema，并用同一 schema 校验 required、URL、number、options 与 credential format。
* 前端 `Parameter` 页面已新增 `Email` tab，复用现有动态 schema 表单、Advanced JSON、credential 遮罩和 enable/disable 交互。
* 测试已通过：`go test ./...`、`cd frontend && npm test`、`cd frontend && npm run build`。
