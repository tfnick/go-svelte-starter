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

## Assumptions (Temporary)

* The MVP targets a public customer-facing chat assistant, not only an internal/admin assistant.
* The public assistant is mainly for product consultation and customer lead capture.
* Initial knowledge base can be manually maintained, uploaded as documents, or imported from online URLs by an admin.
* Chinese answers are required or strongly preferred.
* The first version should prioritize correctness, traceability, abuse resistance, and lead quality over large-scale retrieval performance.
* Human agent handoff and full ticketing workflow are not part of the first feasibility MVP unless the user chooses otherwise.

## Open Questions

* Should the first implementation be a research task only, or should it proceed into implementation after PRD confirmation?

## Requirements (Evolving)

* Produce a feasibility assessment for building an LLM + knowledge-base support assistant on the current system.
* Identify existing reusable modules and missing backend/frontend capabilities.
* Compare at least two retrieval/vector storage approaches against current repo constraints.
* Recommend an MVP architecture and phased implementation plan.
* MVP includes a public website chat assistant for product consultation.
* MVP includes an admin knowledge-base management UI so administrators can create, edit, enable/disable, and reindex knowledge content used by the assistant.
* MVP knowledge-base sources include uploaded documents and online URLs.
* Uploaded document MVP formats include Markdown (`.md`) and Word document (`.doc` / `.docx`) if extraction tooling is available; otherwise Word extraction may be phased after Markdown and URL ingestion.
* URL source MVP imports one specified page URL at a time. Full-site crawling is out of scope unless explicitly added later.
* MVP uses "chat first, lead later" capture: visitors can ask product questions first, and the assistant asks for contact details after detecting meaningful product, quote, demo, trial, purchase, or follow-up intent.
* MVP lead required fields are contact method and need description:
  * contact method: phone or email, at least one required
  * need description: required
  * name and company are optional
* MVP stores source page, detected intent, and conversation summary with each lead.
* Public chat must include basic abuse controls such as rate limits, session limits, and safe refusal behavior.
* Preserve future migration paths to stronger vector infrastructure such as Qdrant or pgvector.
* Ensure support answers can cite knowledge chunks or sources used to generate the answer.
* Ensure low-confidence or no-match retrieval can refuse to answer rather than hallucinate.
* Track LLM/retrieval invocations for debugging, cost awareness, and quality review.

## Acceptance Criteria (Evolving)

* [ ] PRD documents whether the feature is feasible on the current architecture.
* [ ] PRD lists reusable existing modules and newly required modules.
* [ ] Research notes compare SQLite-first retrieval, Qdrant, and pgvector.
* [ ] Research notes cover embedding-provider implications and current DeepSeek limitation uncertainty.
* [ ] MVP scope is explicit as a public product-consultation chat assistant with lead capture.
* [ ] MVP includes admin knowledge-base create/edit/enable-disable/reindex functionality.
* [ ] MVP knowledge-base source types include document upload and online URL import.
* [ ] Document and URL ingestion constraints are explicit.
* [ ] Public chat risk controls are listed, including rate limiting, prompt-injection handling, and low-confidence refusal.
* [ ] Lead capture fields and trigger timing are confirmed.
* [ ] Out-of-scope items are explicit.
* [ ] A recommended technical approach is selected before implementation begins.

## Recommended MVP

### Backend

* Add KB tables:
  * `kb_sources`
  * `kb_documents`
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
  * upload Markdown or Word document source
  * import content from a single online URL
  * enable/disable source or document
  * trigger reindex after content changes
  * inspect indexing status and last error
* Add an embedding provider port and one adapter/provider configuration path.
* Add async indexing jobs:
  * parse manually edited content, uploaded document content, or fetched URL content
  * chunk text
  * generate embeddings
  * mark index status and errors
* Add retrieval service:
  * embed user question
  * retrieve top-k chunks
  * apply metadata filters such as enabled source / category
  * return source snippets and similarity scores
* Add answer generation service:
  * build prompt from user question plus retrieved chunks
  * instruct model to answer only from provided context
  * return citations
  * refuse answer when retrieval confidence is too low
* Add public chat session service:
  * create anonymous visitor session
  * store visitor messages and assistant replies
  * enforce per-session and per-IP limits
  * summarize conversation intent for lead records
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
  * embedded on marketing/product pages
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
  * indexing status and reindex action
* Add admin Support Console:
  * conversation list
  * lead list
  * cited source snippets
  * feedback action
  * "no answer found" state

## Feasible Approaches

### Approach A: SQLite-first RAG MVP (Recommended)

Store KB, chunks, embeddings, conversations, and feedback in the existing app SQLite DB. Compute similarity in Go for the first MVP, then keep a retrieval boundary that can later switch to sqlite-vec or Qdrant.

Best fit when:

* KB size is small or moderate.
* Deployment simplicity matters.
* Feature is for internal/self use first.

Main risk:

* Vector search performance will eventually need a stronger index if the KB grows.

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

Choose Approach A for the first implementation after research is approved.

Reasoning:

* It matches the current single-binary / SQLite deployment model.
* It can reuse existing LLM channel/model/credential patterns.
* It avoids introducing Qdrant/Postgres before proving the product workflow.
* A clean retrieval interface can preserve future migration paths.

## Decisions

* Lead capture timing: chat first, lead later. Visitors can consult freely first; after the assistant detects meaningful product, quote, demo, trial, purchase, or follow-up intent, it asks for contact details in the conversation.
* Lead required fields: phone or email is required, need description is required; name and company are optional.

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
* Empty KB, disabled source, stale embeddings, and chunk/model dimension mismatch.
* Admin updates KB content while indexing is running; indexing should be idempotent and leave the previous usable index until a new version succeeds.
* Uploaded documents may be too large, malformed, unsupported, duplicate, or fail text extraction.
* URL import may fail because of timeout, non-HTML content, robots/access restrictions, redirects, encoding issues, or pages with little meaningful text.
* Prompt injection inside KB content.
* User question asks for data outside the KB or outside the current user's permissions.
* Low retrieval confidence should produce a safe refusal.
* Anonymous visitors may spam the widget or attempt prompt injection.
* Lead contact fields may contain invalid, duplicate, or low-quality data.

## Out Of Scope (Draft)

* Human agent handoff and ticket assignment.
* CRM integration/export.
* CAPTCHA provider integration unless abuse becomes blocking.
* Fine-tuning models.
* Voice/chatbot integrations such as WeChat/WhatsApp.
* Full-site crawling, sitemap crawling, and recurring web crawling.
* PDF/PPT/XLS ingestion unless added as a later phase.
* Multi-tenant customer support permissions.
* Migrating the whole app to Postgres.

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

Possible operations:

* `llm.chat_answer`
* `embedding.create`
* `kb.reindex_document`
* `support.answer_question`

Admin KB MVP fields:

* `title`
* `source_type` (`manual`, `file`, `url`)
* `content`
* `category` or `tags`
* `source_url`
* `file_name`
* `file_mime_type`
* `file_size`
* `enabled`
* `index_status`
* `last_indexed_at`
* `last_index_error`

Source ingestion MVP:

* Manual source: admin edits `title` and `content` directly.
* Markdown file source: extract plain Markdown text, preserve headings as chunk metadata where practical.
* Word file source: extract text from `.doc` / `.docx` if a reliable library/tooling path is selected; otherwise explicitly defer after Markdown and URL.
* URL source: fetch exactly one URL supplied by admin, extract readable page text, store canonical URL and fetched timestamp, then index extracted text.

## Definition Of Done

* PRD captures feasibility, approach options, MVP recommendation, risks, and open questions.
* Research notes include external references and repo-specific constraints.
* User confirms the MVP boundary before implementation starts.
* No code implementation is started until the task is explicitly moved from planning to in progress.
