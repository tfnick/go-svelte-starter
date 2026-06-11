# LLM Knowledge Base Customer Support Feasibility

## Goal

Research whether the current backend can support a self-use customer support system that combines LLM generation with a knowledge base, and define a realistic MVP path if it is feasible.

## What I Already Know

* User wants a new task to study feasibility, not to implement the feature immediately.
* The system already has a backend LLM integration path for chat-completion style summarization through DeepSeek/OpenAI-compatible APIs.
* Existing LLM configuration is managed through Parameter/integration tables: channels, model options, operation configs, credentials, and invocation logs.
* Existing backend stack is Go + Echo + SQLite, with migrations embedded into the executable.
* Existing async/durable queue infrastructure can support background ingestion/reindex jobs.
* Existing frontend has an `/experiments` page that demonstrates LLM summary and realtime message delivery.
* The system does not yet have knowledge-base documents, chunks, embeddings, vector retrieval, support conversations, citations, or feedback.
* There is another active planning task, `06-11-optimize-marketing-website`, which this task should not modify.

## Research References

* [`research/rag-feasibility.md`](research/rag-feasibility.md) - Repository context, vector store options, embedding provider options, and recommended architecture.

## Feasibility Summary

Feasible, with a staged implementation. The current LLM and integration foundation is enough for answer generation and provider configuration, but the system needs a new RAG layer before it can behave like a reliable support assistant.

The main missing capabilities are:

* Knowledge-base source/document/chunk storage.
* Chunking and indexing pipeline.
* Embedding provider abstraction and embedding generation.
* Vector retrieval or similarity search.
* Support chat session/message storage.
* Citation, confidence, feedback, and audit contracts.

## Assumptions

* The MVP targets a public customer-facing chat assistant, not only an internal/admin assistant.
* The public assistant is mainly for product consultation and customer lead capture.
* Initial knowledge base can be manually maintained, uploaded as Markdown documents, or imported from online URLs by an admin.
* Uploaded knowledge files should be stored in SQLite for MVP, together with extracted text and metadata. Local filesystem/OSS storage can be added later if file sizes or retention needs grow.
* Chinese answers are required or strongly preferred.
* The first version should prioritize correctness, traceability, abuse resistance, and lead quality over large-scale retrieval performance.
* Human agent handoff and full ticketing workflow are not part of the first feasibility MVP unless the user chooses otherwise.
* MVP must implement usable RAG Q&A on top of SQLite storage. External vector databases are future extensions, not first-version requirements.
* MVP provider strategy starts from DeepSeek / DeepSeek-compatible configuration where possible, but the system must depend on abstract ports so chat generation and embedding generation can be switched to other providers later.

## Open Questions

* None. User selected research/PRD completion only; do not start implementation from this task.

## Requirements

* Produce a feasibility assessment for building an LLM + knowledge-base support assistant on the current system.
* Identify existing reusable modules and missing backend/frontend capabilities.
* Compare at least two retrieval/vector storage approaches against current repo constraints.
* Recommend an MVP architecture and phased implementation plan.
* MVP includes a public website chat assistant for product consultation.
* MVP public chat entry is an embedded site widget, shown as a floating assistant on marketing/product pages.
* Widget should preserve the recent conversation for the same browser so visitors can refresh or navigate pages and continue consulting.
* MVP includes an admin knowledge-base management UI so administrators can create, edit, enable/disable, and reindex knowledge content used by the assistant.
* MVP knowledge-base sources include manual content, Markdown uploads, and online URLs.
* Uploaded document MVP format is Markdown (`.md`). Word document (`.doc` / `.docx`) ingestion is deferred to a later phase.
* Uploaded source file storage MVP: store original small file bytes or normalized file text in SQLite, plus extracted text and metadata. Define conservative file size limits.
* URL source MVP imports one specified page URL at a time. Full-site crawling is out of scope unless explicitly added later.
* Knowledge-base indexing strategy: after admin creates, edits, uploads, or imports knowledge content, the system automatically enqueues an indexing job.
* SQLite is the required first-version retrieval substrate: knowledge chunks, embeddings, retrieval metadata, conversations, citations, and leads must be persisted in SQLite and support end-to-end RAG answer generation without Qdrant/pgvector.
* Vector storage/search decision: use `sqlite-vec` through the current `modernc.org/sqlite/vec` CGO-free integration for MVP vector KNN. Do not use the separate `sqlite-vector` extension for the first implementation.
* Provider integration decision: use provider ports and registries. Default answer generation can use the existing DeepSeek OpenAI-compatible chat adapter; embedding generation must be a separate port/adapter so it can use DeepSeek-compatible embeddings if available, or another embedding provider without changing the RAG workflow.
* MVP uses "chat first, lead later" capture: visitors can ask product questions first, and the assistant asks for contact details after detecting meaningful product, quote, demo, trial, purchase, or follow-up intent.
* MVP lead required fields are contact method and need description:
  * contact method: phone or email, at least one required
  * need description: required
  * name and company are optional
* MVP stores source page, detected intent, and conversation summary with each lead.
* Admin lead management MVP is read-only list/detail: administrators can view contact method, need description, source page, detected intent, conversation summary, and original conversation. Status workflow and export are out of scope.
* Public chat must include basic abuse controls such as rate limits, session limits, and safe refusal behavior.
* MVP abuse control is lightweight: anonymous visitor session, IP-level rate limit, and per-session message cap. CAPTCHA is out of scope unless abuse becomes blocking.
* If retrieval confidence is too low or no suitable KB chunks are found, the assistant must not fabricate an answer. It should safely say it did not find accurate information and invite the visitor to leave contact details for follow-up.
* Preserve future migration paths to stronger vector infrastructure such as Qdrant or pgvector.
* Ensure support answers can cite knowledge chunks or sources used to generate the answer.
* Ensure low-confidence or no-match retrieval can refuse to answer rather than hallucinate.
* Track LLM/retrieval invocations for debugging, cost awareness, and quality review.

## Acceptance Criteria

* [ ] PRD documents whether the feature is feasible on the current architecture.
* [ ] PRD lists reusable existing modules and newly required modules.
* [ ] Research notes compare SQLite-first retrieval, Qdrant, and pgvector.
* [ ] Research notes cover embedding-provider implications and current DeepSeek limitation uncertainty.
* [ ] MVP scope is explicit as a public product-consultation chat assistant with lead capture.
* [ ] Public chat entry is defined as an embedded floating site widget.
* [ ] Widget preserves recent conversation for the same browser.
* [ ] MVP includes admin knowledge-base create/edit/enable-disable/reindex functionality.
* [ ] MVP knowledge-base source types include manual entry, Markdown upload, and online URL import.
* [ ] Document and URL ingestion constraints are explicit.
* [ ] Uploaded knowledge file storage is scoped to SQLite for MVP.
* [ ] Knowledge-base updates automatically enqueue reindex jobs.
* [ ] SQLite-first RAG is explicit: first version can retrieve chunks and generate grounded answers using SQLite-backed storage.
* [ ] Vector extension decision is explicit: MVP uses `sqlite-vec` via `modernc.org/sqlite/vec`, with a fallback retriever boundary if vec0 packaging or behavior changes.
* [ ] Provider abstraction is explicit: support chat generation and embedding generation through ports, with DeepSeek-compatible defaults and provider-swapping capability.
* [ ] Public chat risk controls are listed, including rate limiting, prompt-injection handling, and low-confidence refusal.
* [ ] Low-confidence/no-match answers safely refuse and trigger lead-capture guidance.
* [ ] MVP abuse controls are explicit as lightweight limits without CAPTCHA.
* [ ] Lead capture fields and trigger timing are confirmed.
* [ ] Admin lead management is scoped to read-only list/detail for MVP.
* [ ] Out-of-scope items are explicit.
* [ ] A recommended technical approach is selected before implementation begins.

## Recommended MVP

### Backend

* Add KB tables:
  * `kb_sources`
  * `kb_documents`
  * `kb_source_files`
  * `kb_chunks`
  * `kb_chunk_embeddings`
  * `support_conversations`
  * `support_messages`
  * `support_answer_citations`
  * `support_feedback`
  * `support_leads`
* Add admin KB management usecases:
  * list knowledge sources/documents
  * create new knowledge document
  * update existing knowledge document
  * upload Markdown document source
  * import content from a single online URL
  * enable/disable source or document
  * automatically enqueue reindex after content changes
  * inspect indexing status and last error
* Add an embedding provider port and one adapter/provider configuration path.
* Keep answer-generation and embedding-generation provider ports separate:
  * chat answer port can reuse/extend existing LLM chat-completion adapter patterns
  * embedding port returns vector values plus model/dimension metadata
  * both resolve provider credentials through Parameter/integration configuration
* Add async indexing jobs:
  * parse manually edited content, uploaded document content, or fetched URL content
  * store original upload/extracted text metadata in SQLite
  * chunk text
  * generate embeddings
  * mark index status and errors
  * ignore duplicate/stale jobs when a newer document version exists
* Add retrieval service:
  * embed user question
  * retrieve top-k chunks
  * apply metadata filters such as enabled source / category
  * return source snippets and similarity scores
  * classify no-match/low-confidence retrieval before answer generation
* Add answer generation service:
  * build prompt from user question plus retrieved chunks
  * instruct model to answer only from provided context
  * return citations
  * refuse answer when retrieval confidence is too low
  * when refusing, guide the visitor to leave contact details for human follow-up
  * default to the configured DeepSeek-compatible chat provider if available
* Add public chat session service:
  * create anonymous visitor session
  * store visitor messages and assistant replies
  * enforce per-session and per-IP limits
  * reject or throttle requests that exceed limits with a safe message
  * summarize conversation intent for lead records
  * allow same-browser visitors to resume the recent conversation through an anonymous session token
* Add lead capture service:
  * collect contact fields after product/follow-up intent is detected
  * require phone or email
  * require need description
  * allow optional name and company
  * link lead to conversation and source page
  * store detected intent and conversation summary
  * record lead status for later admin follow-up
* Record safe metadata in `integration_invocations`; avoid storing raw provider secrets or sensitive prompts in integration logs.

### Frontend

* Add public chat widget:
  * embedded on marketing/product pages as a floating assistant
  * opens from a compact launcher button
  * preserves anonymous visitor session across page navigation
  * stores only the anonymous session token/client-side session reference in localStorage
  * records source page URL/referrer for lead attribution
  * can be enabled/disabled through admin/site settings
  * works on desktop and mobile
  * supports product consultation Q&A
  * asks for contact details after meaningful product/follow-up intent is detected
  * shows safe fallback when the assistant cannot answer
* Add admin Knowledge Base menu:
  * source/document list
  * create/edit manual document
  * upload document source
  * import from online URL
  * structured fields for title, source type, category/tags, content/body, source URL, file metadata, and enabled state
  * enable/disable source
  * indexing status and last error
* Add admin Support Console:
  * conversation list
  * lead list
  * lead detail view with contact, need, source page, detected intent, conversation summary, and original conversation
  * cited source snippets
  * feedback action
  * "no answer found" state

## Feasible Approaches

### Approach A: SQLite-first RAG MVP (Recommended)

Store KB, chunks, embeddings, conversations, and feedback in the existing app SQLite DB. Compute similarity from SQLite-backed vectors in Go for the first MVP, then keep a retrieval boundary that can later switch to sqlite-vec or Qdrant.

Best fit when:

* KB size is small or moderate.
* Deployment simplicity matters.
* First version must run RAG Q&A without an external vector service.

Main risk:

* Vector search performance will eventually need a stronger index if the KB grows.

### Approach A.1: sqlite-vec through modernc.org/sqlite/vec (Selected)

Use `modernc.org/sqlite/vec`, which exposes a CGO-free version of `sqlite-vec` in the current driver dependency. Store searchable embeddings in `vec0` virtual tables and query them with `MATCH ... ORDER BY distance LIMIT k`.

Why selected:

* Current project already uses `modernc.org/sqlite` and v1.49.1 includes `modernc.org/sqlite/vec`.
* Local verification confirmed `CREATE VIRTUAL TABLE ... USING vec0(...)` and KNN query work in this repository.
* Keeps Windows/Linux single-binary deployment simpler than shipping a separate dynamic extension.
* Avoids adding CGO.

Risks:

* `sqlite-vec` is still pre-v1, so table/query details may break on dependency upgrades.
* Implementation should keep a plain SQLite/Go cosine fallback path or at least a retriever interface so vec0 can be replaced.

### Approach A.2: sqlite-vector extension (Not selected for MVP)

`sqlite-vector` is not selected for the first implementation.

Reasons:

* It would add a separate external extension packaging question for Windows/Linux deployments.
* The current repo already has a working CGO-free sqlite-vec path through `modernc.org/sqlite/vec`.
* The MVP does not need a second vector-extension dependency before proving the product workflow.

### Approach B: Qdrant Sidecar

Keep app records in SQLite, but store vectors and metadata in Qdrant.

Best fit when:

* KB grows large.
* Search latency and metadata filtering become important.
* Running an extra service is acceptable.

Main risk:

* Adds operational complexity and data consistency concerns.

### Approach C: PostgreSQL + pgvector

Use Postgres as the app database and store embeddings through pgvector.

Best fit when:

* The project is already planning a Postgres deployment.
* Strong DB consistency and SQL retrieval matter.

Main risk:

* Too large as a prerequisite for a first support-assistant MVP because the current app is SQLite-first.

## Recommendation

If/when implementation is approved later, choose Approach A for the first implementation. This is now a product constraint, not just a preference: the first version should deliver usable RAG Q&A on SQLite.

Reasoning:

* It matches the current single-binary / SQLite deployment model.
* It can reuse existing LLM channel/model/credential patterns.
* It avoids introducing Qdrant/Postgres before proving the product workflow.
* A clean retrieval interface can preserve future migration paths.
* It satisfies the requirement that the first version can perform RAG retrieval and answer generation on SQLite.
* The selected vector implementation is `sqlite-vec` via `modernc.org/sqlite/vec`.
* It keeps DeepSeek as the preferred first provider while avoiding hard provider lock-in through ports.

## Decisions

* Lead capture timing: chat first, lead later. Visitors can consult freely first; after the assistant detects meaningful product, quote, demo, trial, purchase, or follow-up intent, it asks for contact details in the conversation.
* Lead required fields: phone or email is required, need description is required; name and company are optional.
* SQLite-first RAG: the first implementation must persist chunks and embeddings in SQLite and perform retrieval from that SQLite data before answer generation. Qdrant/pgvector remain optional future migrations.
* Vector extension: use `sqlite-vec` through `modernc.org/sqlite/vec` for MVP KNN vector search. Do not choose `sqlite-vector` for the first implementation.
* Provider abstraction: default provider family is DeepSeek / DeepSeek-compatible where available, but RAG code depends on `ChatGenerator` and `EmbeddingGenerator` style ports. If DeepSeek embedding support is unavailable or insufficient, only the embedding adapter/config changes.
* Public chat entry: use an embedded floating site widget for MVP, not a standalone chat page.
* Indexing strategy: admin knowledge-base changes automatically enqueue reindex jobs. Manual reindex can be considered later, but is not required for MVP.
* Abuse control: use lightweight limits only for MVP: anonymous session, IP rate limit, and per-session message cap. Do not add CAPTCHA in the first version.
* Low-confidence behavior: no-match or low-score retrieval must produce safe refusal plus lead-capture guidance, not generic LLM fallback.
* Admin lead handling: MVP only supports read-only lead list and detail. No status workflow or CSV export in first version.
* Visitor session persistence: same-browser visitors can resume their recent conversation using an anonymous session token stored client-side. Full visitor profile aggregation is out of scope.
* File storage: uploaded KB files are stored in SQLite for MVP. Do not require local path storage or OSS storage in the first version.

## Expansion Sweep

Future evolution:

* Visitor identity enrichment, lead scoring, and CRM export.
* Human handoff / ticket creation when the assistant cannot answer.
* Multi-source ingestion from Markdown, HTML pages, PDFs, order/product docs, and system FAQs.
* Reranking, hybrid keyword + vector search, and answer quality dashboards.

Related scenarios:

* Reuse Parameter integration patterns for embedding providers.
* Reuse scheduler/queue patterns for indexing and retryable reindex jobs.
* Reuse notification/realtime patterns for long-running import progress.

Failure and edge cases:

* Provider timeout, rate limit, invalid API key, or model mismatch.
* DeepSeek chat configuration may exist while DeepSeek-compatible embedding configuration is absent; the system should surface this as "embedding provider is not configured" without breaking unrelated LLM chat features.
* Empty KB, disabled source, stale embeddings, and chunk/model dimension mismatch.
* SQLite brute-force vector search may become slow as KB size grows; MVP should define practical limits and keep the retriever replaceable.
* Admin updates KB content while indexing is running; indexing should be idempotent and leave the previous usable index until a new version succeeds.
* Multiple rapid admin edits can enqueue multiple indexing jobs; worker must skip stale jobs by comparing document version/content hash.
* Uploaded documents may be too large, malformed, unsupported, duplicate, or fail text extraction.
* Storing files in SQLite can bloat the DB; MVP should enforce conservative upload size limits and store extracted text separately from raw bytes.
* URL import may fail because of timeout, non-HTML content, robots/access restrictions, redirects, encoding issues, or pages with little meaningful text.
* Prompt injection inside KB content.
* User question asks for data outside the KB or outside the current user's permissions.
* Low retrieval confidence should produce a safe refusal.
* No-match questions may still represent sales opportunities; the widget should ask whether the visitor wants follow-up rather than ending the conversation abruptly.
* Anonymous visitors may spam the widget or attempt prompt injection.
* Anonymous session tokens can be lost when localStorage is cleared; widget should start a new session gracefully.
* IP-only rate limits can affect visitors behind shared networks; responses should degrade gracefully rather than blocking the whole site.
* Floating widget can obscure important page controls on mobile; design must respect viewport size and safe spacing.
* Lead contact fields may contain invalid, duplicate, or low-quality data.

## Out Of Scope

* Human agent handoff and ticket assignment.
* CRM integration/export.
* Lead status workflow such as new/contacted/invalid/won.
* Lead CSV export.
* Cross-device visitor identity, visitor profile enrichment, and multi-session visitor aggregation.
* CAPTCHA provider integration.
* Fine-tuning models.
* Generic LLM fallback answers that are not grounded in retrieved KB content.
* Voice/chatbot integrations such as WeChat/WhatsApp.
* Full-site crawling, sitemap crawling, and recurring web crawling.
* Word `.doc` / `.docx` ingestion unless added as a later phase.
* PDF/PPT/XLS ingestion unless added as a later phase.
* Multi-tenant customer support permissions.
* Migrating the whole app to Postgres.
* Requiring Qdrant, pgvector, Elasticsearch, or another external retrieval service for the first RAG version.
* Requiring local filesystem or OSS storage for KB uploads in the first version.

## Technical Notes

Likely backend areas:

* `api/db/migrations/app/*`
* `api/models/*`
* `api/usecase/integrations/llm/ports.go` or a new embedding integration package
* `api/usecase/llm_summary.go` as a pattern for invocation recording
* `api/usecase/parameter_schema.go` for provider schema registration
* `api/framework/queue` for background indexing
* `api/routes/*`
* new admin KB route/usecase/model files following existing routes -> usecase -> models layering

Likely frontend areas:

* `frontend/src/router.js`
* `frontend/src/App.svelte`
* `frontend/src/api.js`
* new `KnowledgeBase.svelte`
* new `SupportAssistant.svelte`
* new public `SupportWidget.svelte` or equivalent shared widget component mounted in the public site shell

Possible operations:

* `llm.chat_answer`
* `embedding.create`
* `kb.reindex_document`
* `support.answer_question`

SQLite RAG MVP mechanics:

* Import `modernc.org/sqlite/vec` so the SQLite driver registers sqlite-vec/`vec0`.
* Store searchable chunk embeddings in a `vec0` virtual table, with row IDs that map back to `kb_chunks`.
* Query top-k chunks with vector `MATCH`, `distance`, and `LIMIT`.
* Keep enough relational metadata in normal SQLite tables to filter enabled source/document/version status before or after vector search.
* Keep a fallback/replacement boundary: if `vec0` behavior changes, the retriever can fall back to loading embeddings from normal SQLite rows and computing cosine similarity in Go.
* Chunk rows must include document/source enabled state, content hash/version, token/character counts, and embedding model/dimension metadata so stale or incompatible vectors can be excluded.
* The answer should include citation records referencing chunk IDs, source titles, and source URLs.
* The retriever interface should hide the concrete storage/search implementation so SQLite brute-force retrieval can later be replaced with sqlite-vec, Qdrant, or pgvector.

Provider ports MVP:

* `ChatGenerator` port:
  * input: system prompt, conversation messages, retrieved context chunks, generation params
  * output: answer text, usage, provider request ID
  * first adapter: DeepSeek OpenAI-compatible chat completion
* `EmbeddingGenerator` port:
  * input: one or more text chunks plus operation metadata
  * output: embedding vectors, model code, provider model ID, dimensions, usage, provider request ID
  * first adapter: DeepSeek-compatible embeddings if confirmed/configured; otherwise another OpenAI-compatible embedding provider can be registered without changing RAG flow
* Both ports should use integration channel/model/credential resolution rather than hard-coded provider IDs.

Admin KB MVP fields:

* `title`
* `source_type` (`manual`, `file`, `url`)
* `content`
* `category` or `tags`
* `source_url`
* `file_name`
* `file_mime_type`
* `file_size`
* `file_content_blob` or extracted/raw text storage reference in SQLite
* `enabled`
* `index_status`
* `last_indexed_at`
* `last_index_error`

Source ingestion MVP:

* Manual source: admin edits `title` and `content` directly.
* Markdown file source: store uploaded file data/text in SQLite, extract plain Markdown text, and preserve headings as chunk metadata where practical.
* Word file source: deferred to a later phase after Markdown and URL RAG are stable.
* URL source: fetch exactly one URL supplied by admin, extract readable page text, store canonical URL and fetched timestamp, then index extracted text.

## Definition Of Done

* PRD captures feasibility, approach options, MVP recommendation, risks, and resolved decisions.
* Research notes include external references and repo-specific constraints.
* User confirms this remains a research/PRD task and implementation is not started now.
* No code implementation is started until the task is explicitly moved from planning to in progress.
