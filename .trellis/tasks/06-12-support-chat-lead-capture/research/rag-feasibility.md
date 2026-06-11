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
* modernc.org/sqlite v1.49.1 includes a CGO-free version of sqlite-vec under `modernc.org/sqlite/vec`. Local verification in this repo confirmed `CREATE VIRTUAL TABLE ... USING vec0(...)` and KNN `MATCH ... ORDER BY distance LIMIT k` work after importing `_ "modernc.org/sqlite/vec"`.
* sqlite-vector: considered but not selected for MVP because it would introduce a separate extension packaging question, while the current Go SQLite driver already has a working CGO-free sqlite-vec integration.
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
* Use `sqlite-vec` via `modernc.org/sqlite/vec` for first-version KNN vector search.
* Keep a fallback/replacement retriever boundary that can load candidate vectors from normal SQLite rows and compute cosine similarity in Go if sqlite-vec behavior changes.
* Use existing LLM chat-completion adapter for final answer generation.

Pros:

* Fits current single-binary / SQLite deployment.
* Minimal operational complexity.
* Works well for product-consultation KBs with controlled content volume.
* Keeps future Qdrant/pgvector migration possible through a retrieval abstraction.
* Meets the product constraint that first-version RAG must run on SQLite.

Cons:

* Brute-force SQLite-backed vector search will not scale indefinitely.
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

For the public product-consultation assistant, start with Approach A:

* Keep all core data in SQLite.
* Add clean boundaries: `KnowledgeBase`, `EmbeddingAdapter`, `Retriever`, `SupportChat`.
* Use SQLite-backed vector retrieval for the first MVP, with `sqlite-vec` through `modernc.org/sqlite/vec` as the selected KNN implementation and a retriever boundary for future replacement.
* Keep DeepSeek for answer generation if already configured, but use a separate embedding-capable provider/channel for embeddings unless DeepSeek embedding support is officially confirmed.

Updated vector decision:

* Use `sqlite-vec` through `modernc.org/sqlite/vec` for MVP vector KNN.
* Do not use the separate `sqlite-vector` extension in the first implementation.
* Still keep the retriever interface replaceable for future Qdrant, pgvector, sqlite-vector, or plain-Go cosine fallback.
* MVP ingestion formats are manual content, Markdown upload, and single URL import. Word/PDF/PPT/XLS ingestion should be treated as later-phase work.

## SQLite RAG Mechanics

Recommended first implementation:

* Store source/document/chunk rows in SQLite.
* Store chunk embeddings in a `vec0` virtual table and maintain relational chunk/source/document metadata in normal SQLite tables.
* Store embedding model code, provider model ID, dimensions, content hash, and source/document version.
* On question:
  * create an embedding for the visitor question
  * query sqlite-vec/`vec0` for top-k nearest chunk vectors
  * filter by enabled source/document/version status
  * take top-k chunks above a minimum score
  * generate an answer from the retrieved chunks only
  * persist citations that reference chunk IDs, source titles, and source URLs
* Define a practical MVP limit for total indexed chunks so brute-force retrieval remains acceptable.
* Keep `Retriever` as an interface so the storage/search implementation can later move to sqlite-vec, Qdrant, or pgvector without rewriting the support chat flow.
