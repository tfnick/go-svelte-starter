# Parameter Webhook Format Help Text

## Goal

在 `Parameter` 的 create/edit 表单中，为 `Webhook` 属性增加问号 help text，提示管理员第三方平台 webhook callback URL 的格式。当前主要用于 Payment / Creem 配置，但格式设计为跨 `payment`、`sms`、`email`、`llm` 场景一致。

## What I Already Know

* 用户明确要求在 Parameter 的 Payment add/edit 页面给 webhook 属性增加格式提示，用于指示如何配置到 Creem。
* 用户偏好使用现有字段 help text 风格：问号图标，鼠标移上去显示。
* 用户指出回调格式可能对 `payment`、`sms`、`email`、`llm` 都一致，应避免做成 Payment-only UI 模式。
* 前端页面是 `frontend/src/pages/Parameters.svelte`，`Webhook` 目前只是一个 toggle，没有说明。
* Payment 默认 adapter 是 `payment.creem.hosted_checkout`，schema 在 `api/usecase/parameter_schema.go` 中定义。
* Creem webhook ingress route 已在 backend spec 中记录为 `/api/integrations/payment/<channel_code>/webhooks/creem`。
* Parameter schema field 已支持 `help_text` tooltip，但 `Webhook` toggle 是固定表单字段，不属于 schema field。
* 当前代码里实际注册的 provider webhook route 只有 payment/creem：`/api/integrations/payment/:channel_code/webhooks/creem`。

## Requirements

* 在 create/edit 表单的 `Webhook` 属性 label 旁展示问号 help text 图标，风格与 schema field 的 `help_text` 一致。
* 鼠标移入问号时显示 webhook callback URL 通用格式：`https://<public-domain>/api/integrations/<scenario>/<channel_code>/webhooks/<provider_code>`。
* help text 应根据当前表单的 `scenario`、`channel_code`、`provider_code` 动态代入示例；字段为空时保留 `<channel_code>` / `<provider_code>` 占位。
* 对当前 Payment / Creem，help text 应自然显示或包含：`https://<public-domain>/api/integrations/payment/<channel_code>/webhooks/creem`。
* 提示只做说明，不改变保存 payload、后端 schema、校验规则或 webhook 处理逻辑。
* 页面文案保持简短，适合 tooltip 展示，不做额外弹窗或复杂向导。

## Acceptance Criteria

* [ ] 打开 Parameter 页面时，Webhook label 旁有问号 help text 图标。
* [ ] 鼠标移到问号上时，tooltip 显示通用 webhook callback URL 格式。
* [ ] 修改 Channel code 或 Provider code 后，tooltip 中的对应 segment 使用当前表单值；为空时显示占位。
* [ ] Payment / Creem 场景的 tooltip 能指示 Creem 应配置到 `/api/integrations/payment/<channel_code>/webhooks/creem`。
* [ ] `cd frontend && npm test` 通过。
* [ ] `cd frontend && npm run build` 通过。

## Definition of Done

* 变更范围小，优先只改 `frontend/src/pages/Parameters.svelte` 和必要测试。
* 不引入新 UI 库，不改变 API helper。
* 保持 tooltip 内容在桌面和窄宽度下可读，不撑破表单布局。

## Out of Scope

* 自动生成部署域名或读取后端 public URL 配置。
* 在 Creem 后台自动配置 webhook。
* 改动 webhook 接收 route、签名校验、队列或支付处理逻辑。
* 为 SMS/Email/LLM 新增真实 webhook 接收 route。

## Technical Approach

* 在 `Parameters.svelte` 中复用现有 tooltip / `?` help text 视觉样式，为固定 `Webhook` label 增加帮助图标。
* 新增 helper 根据当前 `form.scenario`、`form.channel_code`、`form.provider_code` 生成 tooltip 文案。
* URL 格式统一为 `https://<public-domain>/api/integrations/<scenario>/<channel_code>/webhooks/<provider_code>`。
* 如需测试，优先通过现有 frontend build/test 捕获 Svelte 编译和 helper 回归。

## Technical Notes

* Existing UI: `frontend/src/pages/Parameters.svelte` renders `Webhook` at the fixed Enabled/Webhook toggle grid.
* Existing backend contract: `POST /api/integrations/payment/:channel_code/webhooks/creem`.
* Existing spec: `.trellis/spec/frontend/svelte-vite-embed.md`, Scenario: Parameter Integration Management UI.
