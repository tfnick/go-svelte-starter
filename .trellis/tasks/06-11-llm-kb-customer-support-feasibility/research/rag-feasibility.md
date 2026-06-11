# RAG Feasibility Notes

## Repo Context

Current system already has several useful building blocks:

* Backend is Go + Echo with SQLite app DB and embedded migrations.
* LLM integration exists for chat-completion style generation:
  * `api/usecase/integrations/llm/ports.go`
  * `api/providers/llm/deepseek/deepseek.go`
  * `api/usecase/llm_summary.go`
  * `POST /api/llm/summaries`
* LLM channels/models/credentials are admin-managed through `integration_channels`, `integration_model_options`, `integration_operation_configs`, and `integration_invocations`.
* Parameter management already has an LLM provider schema for DeepSeek OpenAI-compatible chat completion.
* There is a demo page at `/experiments` for LLM summary and realtime delivery.
* Durable queue/retry infrastructure exists through `api/framework/queue` and goqite.
* OSS provider configuration exists, but current user-facing upload flow is limited to site logo.

Missing pieces for a knowledge-base customer support system:

* No knowledge-base source, document, or chunk tables.
* No document ingestion or chunking pipeline.
* No embedding provider port/adapter.
* No vector index or vector similarity search.
* No chat conversation/session/message model.
* No citation, feedback, confidence, or answer audit contract.

## External Research

### Vector Store Options

* sqlite-vec: official GitHub describes it as a SQLite vector search extension that stores/query float, int8, and binary vectors in `vec0` virtual tables and runs anywhere SQLite runs. It is also explicitly pre-v1, so breaking changes are expected: https://github.com/asg017/sqlite-vec
* pgvector: official GitHub describes it as open-source vector similarity search for Postgres with exact/approximate nearest neighbor search and distance metrics such as cosine, inner product, and L2: https://github.com/pgvector/pgvector
* Qdrant: official docs describe it as an AI-native vector search / semantic search engine for extracting meaningful information from unstructured data, with local/self-hosted and cloud options: https://qdrant.tech/documentation/

### Embedding Provider Options

* OpenAI official embeddings docs define embeddings as vectors whose distance measures relatedness, and note embeddings requests are billed by input tokens: https://developers.openai.com/api/docs/guides/embeddings
* DeepSeek official docs emphasize OpenAI/Anthropic-compatible chat/model API access and API-key bearer authentication. I did not find a clear official embeddings endpoint in the surfaced DeepSeek API docs, so embeddings should be treated as a separate provider capability until confirmed: https://api-docs.deepseek.com/
* Jina AI markets embedding models for search and RAG systems, and can be considered for multilingual retrieval if the project is willing to add another provider: https://jina.ai/embeddings/

## Feasible Approaches

### Approach A: SQLite-first RAG MVP (Recommended)

How it works:

* Store KB sources, documents, chunks, embeddings, conversations, and feedback in the app SQLite DB.
* Add an embedding port under `api/usecase/integrations/llm` or a new `embedding` scenario.
* Start with small-to-medium KB retrieval by loading candidate vectors and computing cosine similarity in Go, or use `sqlite-vec` later after validating driver/extension packaging.
* Use existing LLM chat-completion adapter for final answer generation.

Pros:

* Fits current single-binary / SQLite deployment.
* Minimal operational complexity.
* Works well for self-use and small internal KB.
* Keeps future Qdrant/pgvector migration possible through a retrieval abstraction.

Cons:

* Brute-force vector search will not scale indefinitely.
* `sqlite-vec` is pre-v1 and loadable-extension packaging may be non-trivial with `modernc.org/sqlite`.
* Needs careful batching and reindex jobs to avoid slow admin requests.

### Approach B: Qdrant Sidecar

How it works:

* Keep source documents and conversations in app SQLite.
* Store chunk vectors and metadata in Qdrant.
* Backend calls Qdrant for top-k retrieval before generating answers.

Pros:

* Purpose-built vector search.
* Scales better and supports metadata filtering.
* Clearer path for larger KB and hybrid retrieval.

Cons:

* Adds a new runtime service, config, backups, and deployment checks.
* More moving pieces for local Windows/Linux exe use.
* Needs consistency handling between SQLite source records and Qdrant points.

### Approach C: PostgreSQL + pgvector

How it works:

* Move or optionally deploy app data on Postgres.
* Store embeddings next to document chunks using pgvector.
* Use SQL for retrieval and normal relational joins.

Pros:

* Strong data consistency and mature database operations.
* Good long-term fit if the app is already moving toward Postgres.

Cons:

* Current app defaults to SQLite.
* A DB migration/deployment strategy would be a larger prerequisite than the customer support feature itself.

## Recommendation

For a self-use customer support system, start with Approach A:

* Keep all core data in SQLite.
* Add clean boundaries: `KnowledgeBase`, `EmbeddingAdapter`, `Retriever`, `SupportChat`.
* Compute vector search in Go for the first MVP, with a feature boundary that can later switch to sqlite-vec or Qdrant.
* Keep DeepSeek for answer generation if already configured, but use a separate embedding-capable provider/channel for embeddings unless DeepSeek embedding support is officially confirmed.

