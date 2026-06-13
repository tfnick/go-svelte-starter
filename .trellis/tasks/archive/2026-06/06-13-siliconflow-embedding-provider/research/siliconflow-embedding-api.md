# SiliconFlow Embedding API Research

## Source

* Official docs: https://api-docs.siliconflow.cn/docs/api/embeddings-post

## API Contract

* Endpoint: `POST https://api.siliconflow.cn/v1/embeddings`
* Auth: `Authorization: Bearer <API_KEY>`
* Headers: `Content-Type: application/json`
* Request body:
  * `model`: required provider model id.
  * `input`: required string or string array. The current project embedding port already supports batch text input, so use a string array.
  * `encoding_format`: optional; use `float` for direct `[]float32` vector storage.
  * `dimensions`: optional and only supported by Qwen3 embedding models.
* Response shape is OpenAI-style:
  * `data[].embedding`
  * `data[].index`
  * `model`
  * `usage.prompt_tokens`
  * `usage.total_tokens`

## Model Notes

SiliconFlow lists multiple embedding models, including BAAI/bge and Qwen3-Embedding families. The project currently stores vectors in sqlite-vec with a fixed `float[64]` table, so the MVP should avoid models whose output dimension cannot be configured to 64.

Recommended MVP model:

* `Qwen/Qwen3-Embedding-0.6B`
  * Free model in SiliconFlow model catalog.
  * Supports the `dimensions` parameter.
  * Can be configured with `dimensions=64` to match the current `kb_chunk_embedding_vec float[64]` schema.

Other free models may be exposed as selectable Parameter options later only if their output dimensions are compatible with the vector schema, or after a separate vector-dimension migration.

## Repo Constraints

* Existing embedding adapter key: `embedding.deepseek.openai_compatible`.
* Existing local fallback adapter key: `embedding.local_hash_64`.
* KB index/reindex already routes through:
  * `models.GetEnabledEmbeddingConfig`
  * `registeredEmbeddingAdapter`
  * `embeddingProviderConfig`
  * adapter `Embed`
* Current vector table is `kb_chunk_embedding_vec embedding float[64]`; mismatched provider dimensions will fail or produce unusable search.
* Therefore, the SiliconFlow provider should use OpenAI-compatible request/response semantics but should be represented as its own adapter key, schema, and default seeded channel/model.

## Proposed Provider Contract

* Adapter key: `embedding.siliconflow.openai_compatible`
* Provider code: `siliconflow`
* Default API URL: `https://api.siliconflow.cn`
* Default endpoint path: `/v1/embeddings`
* Default model: `Qwen/Qwen3-Embedding-0.6B`
* Default params: `{"dimensions":64,"encoding_format":"float"}`

## Migration Direction

* Remove DeepSeek embedding from Parameter schemas.
* Disable or remove seeded DeepSeek embedding config from active lookup paths.
* Seed SiliconFlow embedding credential/channel/model/operation config.
* Ensure `embedding_create` operation points to SiliconFlow when it is currently empty, local default, or legacy DeepSeek.
* Keep local hash embedding available only if product still wants an offline/dev fallback; otherwise remove it from Parameter menu in a separate explicit decision.
