# 修复 Knowledge base 文档索引缺少 embedding 配置

## Goal

修复 Knowledge base 菜单下构建文档索引 / reindex 时报错的问题：

`embedding config missing: integration channel not found: not found — configure an embedding provider in Settings > Integrations (scenario=embedding, operation=create)`

让管理员能在后台正确配置 embedding provider 通道，使知识库索引正常工作。

## 现状

- 配置入口定位：当前代码里的 integration channel 管理在 `Parameter` 菜单，不在 `Setting` 菜单。
- 前端入口：`frontend/src/router.js` 将 `/app/parameters` 标为 `Parameter`，描述是 `Integration settings`。
- 前端页面：`frontend/src/pages/Parameters.svelte` 的 `scenarios` 已包含 `embedding` tab，默认 adapter 是 `embedding.deepseek.openai_compatible`。
- 后端 API：`frontend/src/api.js` 调用 `/api/admin/parameters/integration-channels` 和 `/api/admin/parameters/integration-schemas`。
- 后端 schema：`api/usecase/parameter_schema.go` 已定义 embedding schema，字段为 `base_url` 和 `api_key`。
- 后端校验：`api/usecase/parameter.go` 的 `normalizeIntegrationScenario` 已允许 `IntegrationScenarioEmbedding`。
- Knowledge base 索引：`api/usecase/kb_retriever.go` 调用 `models.GetEnabledEmbeddingConfig`，使用 `scenario=embedding` 和 `operation=embedding_create`。
- 配置加载链路：`GetEnabledEmbeddingConfig` 会先查 `integration_operation_configs`，取默认 `channel_code/model_code`，再查 enabled `integration_channels`，最后查 enabled `integration_model_options`。
- 报错中的 `Settings > Integrations` 是误导性的旧文案；当前产品实际配置位置应是 `Parameter > Embedding` / integration channels。

## 初步判断

当前报错不再像是“没有 UI schema”或“scenario 被拒绝”，而更可能是以下配置数据缺口之一：

- 没有启用的 `embedding` integration channel。
- `integration_operation_configs` 中 `scenario=embedding`、`operation=embedding_create` 的记录不存在，或指向的 `channel_code` 不存在 / 未启用。
- 对应 channel 下没有启用的 `integration_model_options`，或 model code 不匹配。
- 文案仍指向 `Settings > Integrations`，需要改成 `Parameter > Embedding`。

## Requirements

- 管理员可以在 Parameter 菜单下配置 embedding 通道。
- Knowledge base 文档索引 / reindex 能加载 embedding 配置并正常执行。
- 错误提示应指向真实配置入口：Parameter > Embedding，而不是 Settings > Integrations。
- 修复缺失的 operation/model/channel 配置链路，确保 `scenario=embedding`、`operation=embedding_create` 可解析到启用的 channel 和 model。

## Technical Approach

1. 核查 seed / migration 是否需要补充 embedding 的 operation config 与 model option。
2. 核查创建 embedding channel 后是否需要自动创建或引导创建 `integration_operation_configs` / `integration_model_options`。
3. 修正 Knowledge base 索引失败文案中的配置入口。
4. 增加或更新测试覆盖 `GetEnabledEmbeddingConfig` 的 embedding happy path 和缺失配置错误。

## Out of Scope

- 其他 embedding provider（OpenAI、Jina 等）

## Acceptance Criteria

- [ ] Admin 可以在 Parameter 页面看到 Embedding tab 和 DeepSeek embedding provider
- [ ] 可以创建 DeepSeek embedding 通道，填写 API URL 和 API Key
- [ ] `scenario=embedding` + `operation=embedding_create` 能解析到启用 channel 和 model
- [ ] 知识库索引功能可以加载该配置并正常工作
- [ ] 错误提示不再指向 Settings > Integrations
- [ ] `go build ./...` + `go test ./...` 通过
