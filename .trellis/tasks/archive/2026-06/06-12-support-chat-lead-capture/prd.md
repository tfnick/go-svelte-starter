# Support Chat Lead Capture MVP

## Goal

Implement a self-use public consultation and lead-capture chat capability based on the archived feasibility PRD. The first version should let admins maintain a small knowledge base, expose a public floating assistant for product consultation, answer from SQLite-backed RAG, and capture visitor contact details when follow-up intent appears.

## Source PRD

* Archived PRD: `.trellis/tasks/archive/2026-06/06-11-llm-kb-customer-support-feasibility/prd.md`
* Archived research: `.trellis/tasks/archive/2026-06/06-11-llm-kb-customer-support-feasibility/research/rag-feasibility.md`

## What I Already Know

* Backend is Go + Echo + SQLite with embedded migrations.
* Existing internal API shape follows `routes -> usecase -> models -> db`.
* Existing LLM chat generation exists through:
  * `api/usecase/integrations/llm/ports.go`
  * `api/providers/llm/deepseek/deepseek.go`
  * `api/usecase/llm_summary.go`
  * `api/routes/llm.go`
* Existing LLM/provider configuration uses `integration_channels`, `integration_model_options`, `integration_operation_configs`, credentials, and `integration_invocations`.
* Existing async task/queue infrastructure exists through `api/framework/queue`, `api/models/async_task.go`, and migration `015_add_async_tasks.sql`.
* Existing frontend is Svelte + Vite; app navigation is centralized in `frontend/src/router.js` and sidebar icons in `frontend/src/components/AppSidebar.svelte`.
* Current worktree has an unrelated dirty `frontend/package-lock.json`; do not include it in this task unless the user explicitly asks.

## MVP Scope

Build the first usable version, not the full future support platform.

Included:

* Admin-managed knowledge base.
* Manual content, Markdown upload, and single URL import.
* SQLite-stored source text, chunks, embeddings, conversations, citations, and leads.
* SQLite vector retrieval using `sqlite-vec` via `modernc.org/sqlite/vec`.
* Provider ports for answer generation and embedding generation.
* DeepSeek-compatible chat generation as the first chat adapter.
* A separate embedding adapter/config path; if DeepSeek embeddings are unavailable, implementation may provide another OpenAI-compatible embedding adapter without changing the support chat workflow.
* Public floating chat widget for product consultation.
* Same-browser anonymous visitor session persistence.
* Lead capture after meaningful product, quote, demo, trial, purchase, or follow-up intent.
* Admin read-only support console for conversations and leads.

Excluded:

* Human agent handoff and ticket assignment.
* CRM integration/export.
* Lead status workflow and CSV export.
* Cross-device visitor identity or visitor profile enrichment.
* CAPTCHA provider integration.
* Fine-tuning.
* Generic ungrounded LLM fallback answers outside retrieved KB content.
* Full-site crawling, sitemap crawling, recurring web crawling.
* Word, PDF, PPT, XLS ingestion.
* Requiring Qdrant, pgvector, Elasticsearch, or another external retrieval service.
* Requiring local filesystem or OSS storage for KB uploads.

## Requirements

### Backend

* Add migrations and models for:
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
* Add admin KB APIs/usecases:
  * list knowledge sources/documents
  * create/update manual document
  * upload Markdown document
  * import one online URL
  * enable/disable source or document
  * enqueue reindex automatically after content changes
  * inspect indexing status and last error
* Add indexing workflow:
  * parse manual, Markdown, or URL content
  * store original upload/extracted text metadata in SQLite
  * chunk text
  * generate embeddings
  * write searchable vectors to `sqlite-vec` `vec0`
  * mark status/errors
  * skip stale jobs by comparing document version or content hash
* Add provider boundaries:
  * `ChatGenerator` for answer generation
  * `EmbeddingGenerator` for embeddings
  * both resolve channel/model/credential configuration through existing integration patterns
  * provider failures should be recorded without leaking secrets
* Add retrieval and answer generation:
  * embed visitor question
  * query top-k chunks from SQLite vector storage
  * filter disabled/stale/incompatible chunks
  * generate answer only from retrieved context
  * store citations pointing at chunk/source IDs
  * refuse safely when no suitable chunks or low confidence
* Add public support chat APIs:
  * create/resume anonymous visitor conversation
  * accept visitor message
  * return assistant answer, citations, and lead-capture prompt state
  * enforce IP-level and per-session lightweight limits
  * store source page/referrer
* Add lead capture APIs/usecases:
  * require phone or email
  * require need description
  * allow optional name and company
  * link lead to conversation and source page
  * store detected intent and conversation summary
* Add admin support console APIs:
  * conversation list/detail
  * lead list/detail
  * message and citation inspection

### Frontend

* Add admin Knowledge Base menu/page:
  * list records without pagination for MVP
  * create/edit manual document
  * upload Markdown file
  * import one URL
  * enable/disable source or document
  * display index status and last error
* Add admin Support Console menu/page:
  * conversation list/detail
  * lead list/detail
  * show contact, need, source page, detected intent, summary, original conversation, and citations
* Add public floating support widget:
  * visible on public/marketing/product pages when enabled
  * compact launcher plus chat panel
  * desktop/mobile friendly
  * persists anonymous session token in localStorage
  * includes source page/referrer in requests
  * asks for contact details after follow-up intent or no-match opportunity
  * shows safe fallback instead of fabricated answers

## Acceptance Criteria

* [ ] Admin can create/edit/enable/disable manual KB content.
* [ ] Admin can upload Markdown content and index it.
* [ ] Admin can import one URL and index extracted text.
* [ ] KB updates enqueue or run reindexing without blocking normal page use.
* [ ] SQLite stores sources, documents, chunks, embeddings, conversations, messages, citations, and leads.
* [ ] Retrieval uses `sqlite-vec` via `modernc.org/sqlite/vec` with a clear retriever boundary.
* [ ] Answer generation refuses when retrieval has no suitable context.
* [ ] Public widget can start/resume a same-browser conversation.
* [ ] Public widget can submit visitor messages and display assistant replies.
* [ ] Widget captures phone or email plus need description when lead capture is triggered.
* [ ] Admin can view lead detail and original conversation.
* [ ] Admin can view answer citations or source snippets.
* [ ] Abuse limits are present for anonymous public chat.
* [ ] Provider configuration errors are surfaced safely.
* [ ] Backend tests cover core model/usecase behavior.
* [ ] Frontend tests/build pass after adding routes/pages/widget.

## Technical Approach

Use the archived PRD's selected approach: SQLite-first RAG MVP with `sqlite-vec` through `modernc.org/sqlite/vec`, provider ports, and DeepSeek-compatible chat generation.

Implementation should follow existing project layering:

* `api/db/migrations/app/*` for schema.
* `api/models/*` for DB persistence and sentinel model errors.
* `api/usecase/*` for validation, orchestration, provider calls, indexing, retrieval, and lead capture.
* `api/routes/*` for Echo handlers and DTO mapping.
* `frontend/src/api.js` for client functions.
* `frontend/src/router.js` and `frontend/src/components/AppSidebar.svelte` for menus.
* New Svelte pages/components for Knowledge Base, Support Console, and the public widget.

The implementation can be phased inside one task:

1. Persistence and provider ports.
2. KB admin APIs and pages.
3. Indexing/chunking/retrieval.
4. Public widget and chat APIs.
5. Lead capture and support console.
6. Tests, build, and cleanup.

## Open Questions

* None. User selected one complete MVP task, with internal phased implementation and optional child-task split only if the task becomes too large.

## Decision

Implement as one complete MVP task. Keep implementation and commits internally phased: DB/provider foundation, KB admin, indexing/retrieval, public widget/chat API, lead capture/support console, then verification. If the first pass becomes too large after the DB/provider foundation, split child tasks for widget/console polish rather than weakening backend contracts.

## Recommendation

Start with one implementation task but keep commits internally phased. If the first pass becomes too large after the DB/provider foundation, split child tasks for widget/console polish rather than watering down the backend contracts.

## Definition Of Done

* New task PRD is confirmed.
* Implementation follows the archived feasibility decisions.
* `go test ./...` passes.
* `cd frontend && npm test` passes.
* `cd frontend && npm run build` passes.
* Task changes are committed before finish-work.
