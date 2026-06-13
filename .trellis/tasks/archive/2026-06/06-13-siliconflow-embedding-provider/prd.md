# Replace DeepSeek Embedding With SiliconFlow Provider

## Goal

将系统 Embedding 能力从 DeepSeek embedding 配置切换到硅基流动 SiliconFlow provider，并确保知识库的 index/reindex 都基于新的 SiliconFlow provider 能力生成向量。Parameter 菜单下不再展示 DeepSeek Embedding 配置，改为展示 SiliconFlow Embedding 配置。

## What I Already Know

* 用户要求移除 Parameter 菜单下的 DeepSeek embedding 配置。
* 用户要求增加 Parameter 菜单下的硅基流动 embedding 配置。
* 用户要求知识库的索引与 reindex 都基于新的硅基流动 provider 能力实现。
* 官方 SiliconFlow embedding API 是 `POST https://api.siliconflow.cn/v1/embeddings`，Bearer API key，OpenAI-style response。
* SiliconFlow 文档列出多个 embedding 模型，其中 Qwen3 Embedding 系列支持 `dimensions` 参数。
* 当前项目 KB 向量表是 `kb_chunk_embedding_vec embedding float[64]`，所以默认模型必须输出 64 维向量，或通过 `dimensions=64` 降维。
* 当前 KB index/reindex 已统一走 `models.GetEnabledEmbeddingConfig`、embedding adapter registry、`embeddingProviderConfig` 和 adapter `Embed`，无需在前端 KB 页面单独接 provider。
* 当前项目已有 `embedding.local_hash_64` 本地 provider 作为默认 embedding 配置，也有 `embedding.deepseek.openai_compatible` DeepSeek embedding provider/schema。

## Research References

* [`research/siliconflow-embedding-api.md`](research/siliconflow-embedding-api.md) — SiliconFlow API、模型维度与本项目 KB `float[64]` 约束。

## Requirements

* Parameter 菜单的 Embedding schema 不再展示 `embedding.deepseek.openai_compatible`。
* Parameter 菜单新增 `embedding.siliconflow.openai_compatible` schema。
* SiliconFlow schema 使用 API key 凭证。
* SiliconFlow schema 默认 API URL 为 `https://api.siliconflow.cn`。
* SiliconFlow schema 默认 endpoint path 为 `/v1/embeddings`。
* SiliconFlow schema 支持配置模型，默认 `Qwen/Qwen3-Embedding-0.6B`。
* SiliconFlow schema 默认请求参数包含 `dimensions=64` 和 `encoding_format=float`，以匹配当前 sqlite-vec `float[64]` 存储。
* 新增 SiliconFlow embedding provider adapter，按官方 OpenAI-style `/v1/embeddings` contract 发起批量 embedding 请求。
* Adapter 将 provider response 中的 `data[].embedding` 映射到项目 `embedding.EmbedResult.Vectors`。
* Adapter 处理 provider error status，并映射到现有 `providererror` 分类。
* 启动时注册 `embedding.siliconflow.openai_compatible` adapter。
* 新 migration seed SiliconFlow embedding channel/model/operation config。
* legacy DeepSeek embedding active config 应被禁用或从默认 operation 中移除，避免 KB index/reindex 继续走 DeepSeek。
* KB index/reindex 不新增特殊分支，仍通过统一 embedding config lookup 选择 SiliconFlow provider。
* 保留 `embedding.local_hash_64` 作为本地/dev fallback，只移除 DeepSeek embedding。

## Acceptance Criteria

* [ ] `ListParameterIntegrationSchemas(scenario=embedding)` 返回 Local Hash 和 SiliconFlow Embedding，不返回 DeepSeek Embedding。
* [ ] 前端 Parameter > Embedding 可选择并创建/编辑 SiliconFlow embedding channel。
* [ ] SiliconFlow adapter 对 `/v1/embeddings` 发起包含 `model`、`input`、`encoding_format`、`dimensions` 的请求。
* [ ] SiliconFlow adapter 能解析 OpenAI-style embedding response。
* [ ] 启动注册包含 `embedding.siliconflow.openai_compatible`。
* [ ] 默认或 migrated `embedding_create` operation 使用 SiliconFlow embedding config，而不是 DeepSeek embedding。
* [ ] KB `IndexDocument` 和 `ReindexKBDocument` 使用新的 SiliconFlow provider config 生成并保存向量。
* [ ] DeepSeek embedding provider/schema 不再作为 Parameter 可配置项出现。
* [ ] 单元测试覆盖 schema、model fallback、adapter request/response、KB indexing config。
* [ ] `go test ./...` 通过。
* [ ] `cd frontend && npm test` 通过。
* [ ] `cd frontend && npm run build` 通过。

## Decision (ADR-lite)

**Context**: SiliconFlow 有多个免费 embedding 模型，但当前 KB vector schema 固定为 64 维。如果直接使用无法配置维度的模型，会引入 DB schema migration、旧向量兼容与 reindex 风险。

**Decision**: MVP 默认使用 `Qwen/Qwen3-Embedding-0.6B`，通过 `dimensions=64` 保持与当前 `float[64]` schema 兼容；新建独立 adapter key `embedding.siliconflow.openai_compatible`，而不是复用 DeepSeek adapter；保留 `embedding.local_hash_64` 作为本地/dev fallback。

**Consequences**: 可以较小范围替换 provider，并保证 KB index/reindex 立即可用。其他 SiliconFlow 免费 embedding 模型暂不默认启用，后续如要支持非 64 维模型，应单独做向量维度迁移与全量 reindex 方案。

## Open Questions

* None.

## Out of Scope

* 迁移 `kb_chunk_embedding_vec` 到可配置维度或多维度并存。
* 支持所有 SiliconFlow embedding 模型的自动模型目录同步。
* 自动调用 SiliconFlow 真实 API 做端到端在线测试。
* 移除 DeepSeek LLM 配置；本任务只处理 DeepSeek Embedding。
* 改造 KB 页面 UI 或路由。

## Technical Notes

Likely impacted files:

* `api/usecase/parameter_schema.go`
* `api/models/integration.go`
* `api/providers/embedding/deepseek/*`
* new `api/providers/embedding/siliconflow/*`
* `index.go`
* `api/db/migrations/app/*.sql`
* `api/usecase/kb_embedding_config.go`
* `api/usecase/kb_retriever.go`
* tests under `api/usecase`, `api/models`, `api/providers/embedding`, `api/routes`
* `frontend/src/pages/Parameters.svelte` only if the default embedding metadata needs changing from `local` to `siliconflow`; most UI is schema-driven.

Quality gates:

* `go test ./...`
* `cd frontend && npm test`
* `cd frontend && npm run build`
* `git diff --check`

## Implementation Summary

Status: implemented and checked.

Completed scope:

* Parameter Embedding schemas now expose `embedding.siliconflow.openai_compatible` and `embedding.local_hash_64`; DeepSeek Embedding is no longer exposed as a Parameter schema.
* Startup registers the SiliconFlow embedding adapter under `embedding.siliconflow.openai_compatible`; Local Hash remains registered as fallback.
* SiliconFlow adapter sends OpenAI-style `POST /v1/embeddings` requests with `model`, `input`, `dimensions`, and `encoding_format`, and maps `data[].embedding` plus provider errors into the shared embedding port.
* Migration `020_add_siliconflow_embedding_provider.sql` seeds SiliconFlow channel/model/operation config, lowers Local Hash priority, and disables legacy DeepSeek embedding channels/model options.
* KB indexing/reindexing still uses the unified embedding config lookup and adapter registry; it now passes SiliconFlow model/dimension/encoding settings through `embeddingProviderConfig`.
* Chunk embedding metadata uses the actual adapter result model metadata, so a configured Qwen3 model override is recorded correctly.
* Backend spec was updated for Parameter Embedding and KB embedding provider contracts.
* Follow-up fix: support chat query embedding falls back to `embedding.local_hash_64` only when the selected external embedding config is incomplete because `base_url` or credential value is missing, preserving existing local/dev KB search before a SiliconFlow API key is configured. The same fix also adds the sqlite-vec `v.k = ?` KNN constraint and carries `document_id` through retrieved chunks so support chat citations persist successfully.

Quality gates run:

* `go test ./...` passed.
* `cd frontend && npm test` passed.
* `cd frontend && npm run build` passed; existing `KnowledgeBase.svelte` a11y warnings remain unrelated to this task.
* `git diff --check` passed.
