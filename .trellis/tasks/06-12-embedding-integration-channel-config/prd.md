# 添加 Embedding 集成通道配置支持

## Goal

让管理员能在后台 Settings > Integrations 中配置 embedding provider 通道，使知识库索引正常工作。

## 现状

- LLM DeepSeek 通道已在 `parameter_schema.go` 中定义，Parameter 菜单下可正常配置
- `api/usecase/integrations/embedding/ports.go` — embedding adapter 接口已定义
- `api/usecase/embedding_registry.go` — embedding adapter 注册已实现
- `api/providers/embedding/deepseek/deepseek.go` — DeepSeek embedding 适配器已实现
- `api/models/integration.go` — `IntegrationScenarioEmbedding` 常量已定义，`GetEnabledEmbeddingConfig` 已实现
- **缺失**：`parameter_schema.go` 中没有 embedding schema → admin UI 无法创建 embedding 通道
- **缺失**：`parameter.go` 中 `normalizeIntegrationScenario` 没有包含 embedding → 即使有 schema 也会被拒绝

## Requirements

- 管理员可以在 Parameter > Integration Channels 中创建 embedding 通道
- 支持 DeepSeek OpenAI-compatible embedding API
- 配置项：API URL（默认 https://api.deepseek.com）、API Key

## Technical Approach

1. `api/usecase/parameter_schema.go`: 添加 embedding 的 integration schema 定义（参考 LLM DeepSeek 的 pattern）
2. `api/usecase/parameter.go`: `normalizeIntegrationScenario` 中加入 `IntegrationScenarioEmbedding`

## Out of Scope

- 其他 embedding provider（OpenAI、Jina 等）

## Acceptance Criteria

- [ ] Admin 可以在 Integration Channels 中看到 Embedding 类型的 provider
- [ ] 可以创建 DeepSeek embedding 通道，填写 API URL 和 API Key
- [ ] 知识库索引功能可以加载该配置并正常工作
- [ ] `go build ./...` + `go test ./...` 通过
