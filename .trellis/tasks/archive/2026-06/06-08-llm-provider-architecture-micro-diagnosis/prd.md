# LLM Provider Architecture Micro Diagnosis

## Goal

Diagnose two architecture questions before expanding LLM integrations:

- Whether LLM providers need webhook configuration.
- Whether the current architecture needs optimization before adding new LLM providers.

The output of this task is an architecture decision note, not an implementation change.

## What I Already Know

- Current LLM request flow is synchronous:

```text
routes -> usecase.SummarizeTextWithLLM -> llm.Adapter.Generate -> provider HTTP API
```

- The stable port lives at `api/usecase/integrations/llm/ports.go`.
- Provider implementation lives under `api/integrations/llm/deepseek`.
- `index.go` registers DeepSeek with adapter key `llm.deepseek.openai_compatible`.
- Parameter schema currently exposes `base_url` and `api_key` for LLM, with no webhook secret/config.
- The backend integration spec already prefers:
  - stable scenario ports under `api/usecase/integrations/<scenario>`
  - provider DTO/HTTP/error mapping under `api/integrations/<scenario>/<provider>`
  - usecase registry/bootstrap instead of direct provider imports

## Research References

- [`research/llm-provider-webhook-and-extension-notes.md`](research/llm-provider-webhook-and-extension-notes.md) - official provider documentation pass for OpenAI, Azure OpenAI, Anthropic, Gemini, and DeepSeek.

## Diagnosis

### Final Recheck

The current codebase still matches the diagnosis:

- `api/usecase/integrations/llm/ports.go` exposes one synchronous `Adapter.Generate` port.
- `api/usecase/llm_registry.go` keeps provider registration out of business usecases.
- `api/usecase/parameter_schema.go` exposes DeepSeek LLM `base_url` and `api_key`, with no generic LLM webhook secret.
- Payment webhooks are already modeled separately as provider ingress, which is the right pattern to reuse only if a concrete async LLM operation needs it.

### 1. Does an LLM provider need webhook configuration?

Not for the current synchronous text summary feature.

Webhook support does exist in the LLM ecosystem, but it mainly appears around asynchronous/background/batch/lifecycle workflows:

- background response completion
- batch processing completion
- fine-tuning or job lifecycle events
- file/vector/eval lifecycle events, depending on provider

So the project should not add generic LLM webhook fields such as `webhook_secret` to the base LLM provider schema yet. Doing so would make the Parameters UI imply that every LLM provider needs a webhook, which is false for DeepSeek-style synchronous chat completions.

Recommended rule:

```text
Synchronous generation: no webhook config.
Async provider job with real callback support: add provider-specific webhook config for that operation.
```

### 2. Does the current architecture need optimization before adding new LLM providers?

No large architecture change is needed for another synchronous LLM provider. The current boundary is mostly healthy:

- usecase owns business orchestration and invocation recording
- integration port owns the stable LLM contract
- provider package owns provider-specific HTTP/SDK mapping
- channel/model selection goes through DB config

The places to optimize are small and should be done when the next provider proves the need:

- `ProviderConfig` is currently biased toward OpenAI-compatible providers with `BaseURL`, `APIKey`, and `ProviderModelID`.
- `GenerateRequest` models chat/text generation only.
- Streaming, embeddings, tool calling, multimodal requests, and batch jobs should not all be forced into one `Generate` method.
- The DeepSeek fallback behavior should become more explicit or provider-neutral if multiple providers become first-class.

## Recommended Architecture

Keep the current sync path as-is:

```text
third request
  -> routes
  -> usecase
  -> usecase/integrations/llm.Adapter
  -> integrations/llm/<provider>
  -> provider API
```

If a future provider/operation requires callbacks, add a separate async ingress path:

```text
provider webhook
  -> routes/integrations/llm/<provider> webhook endpoint
  -> verify provider signature
  -> integrations/llm/<provider> normalizes event
  -> usecase updates async job/invocation state
  -> optional domain event if a real domain fact occurred
```

Do not add webhook methods to the base `Adapter` interface until there is a concrete async LLM use case.

## Scenario: Prompt-to-PPT Generation

If the next feature is "generate a PPT from a user prompt", treat it as a new product workflow that uses LLM, not as a new LLM provider capability.

The clean split is:

```text
routes/presentations
  -> usecase.CreatePresentationGenerationJob
  -> models presentation job/artifact rows
  -> queue presentation-generation job
  -> usecase.HandlePresentationGenerationJob
  -> llm.Adapter.Generate produces a structured slide plan
  -> presentation renderer creates .pptx
  -> artifact storage records downloadable file metadata
```

Recommended boundaries:

- LLM integration: only generate a safe structured slide plan, for example title, theme hints, slide list, speaker notes, image prompts/placeholders.
- PPT rendering: separate local renderer boundary, not part of `api/integrations/llm`.
- Artifact storage: separate model/storage boundary for generated files, ownership, MIME type, path, size, checksum, expiry, and download authorization.
- Job state: separate presentation generation task table or artifact job table; do not overload `integration_invocations` because invocation rows are provider call audit metadata, not user-visible file lifecycle.

Why this matters:

- PPT generation may take longer than a normal summary request.
- File generation needs retry/status/download state.
- The generated `.pptx` is a product artifact, while the LLM request is only one step inside that workflow.
- A future provider switch should not affect the PPT renderer or artifact download API.

Suggested minimal MVP flow:

```text
POST /api/presentations/generations
  -> returns {generation_id, status}

GET /api/presentations/generations/:id
  -> returns status, title, artifact_id/download_url when ready

GET /api/artifacts/:id/download
  -> streams the .pptx if current user can access it
```

For a very small prototype, the first endpoint can run synchronously and return the artifact immediately, but the architecture should still use job/artifact concepts in the model. That keeps the API compatible with an async implementation once generation becomes slow or needs retries.

Expected new files for MVP:

- `api/routes/presentation.go`
- `api/usecase/presentation_generation.go`
- `api/models/presentation_generation.go`
- `api/models/artifact.go`
- `api/usecase/presentations/renderer.go` or `api/usecase/presentation_renderer.go`
- `api/framework/storage/local.go` if file storage needs a reusable filesystem boundary
- `api/db/migrations/app/*_add_presentation_generation.sql`
- `frontend/src/pages/Presentations.svelte`
- frontend API helpers/routes/sidebar entries as needed

Expected modified files:

- `api/usecase/integrations/llm/ports.go` - add a new operation constant such as `presentation_outline` or `presentation_plan`; avoid adding PPT-specific fields to the base provider config.
- `api/usecase/llm_summary.go` - no direct change unless shared LLM config helpers are extracted.
- `api/usecase/llm_registry.go` - likely unchanged.
- `index.go` - register a queue runner for presentation generation if async.
- `api/framework/queue/queue.go` - add a queue name if using a dedicated queue.
- `api/usecase/parameter_schema.go` / integration seed data - add operation config/model capability if PPT generation should use a different model.

Do not put the PPT renderer under `api/integrations/llm/<provider>`. Provider packages should not know what a PPT is.

## Expected Files for Adding a New Synchronous LLM Provider

Example: adding OpenAI, Anthropic, Gemini, or Azure OpenAI for text summary.

Likely new files:

- `api/integrations/llm/<provider>/<provider>.go`
- `api/integrations/llm/<provider>/<provider>_test.go`

Likely modified files:

- `index.go` - register the adapter key.
- `api/usecase/parameter_schema.go` - expose provider-specific config and credential fields.
- `api/db/migrations/app/*` - seed or update provider/channel/model options when needed.
- `api/usecase/integrations/llm/ports.go` - only if the provider requires config shape or response fields the current port cannot represent.
- `api/usecase/llm_summary_test.go` - only if provider selection, fallback behavior, or usecase-level behavior changes.

Expected impact:

- Low impact for another OpenAI-compatible provider.
- Medium impact for Anthropic/Gemini/Azure if credential/config shape differs.
- High impact only if adding a new operation family such as streaming, embeddings, multimodal, batch jobs, or provider webhook callbacks.

## Proposed Optimization Boundaries

### Keep Now

- Keep `Adapter.Generate` for synchronous text generation.
- Keep provider implementations under `api/integrations/llm/<provider>`.
- Keep webhook config out of the base LLM schema.
- Keep invocation records metadata-only; do not store raw prompts or raw provider payloads.

### Add Later When Needed

- Provider-neutral config bundle, for example typed common fields plus provider-specific config JSON.
- Separate operation ports:
  - `TextGenerator`
  - `StreamingTextGenerator`
  - `EmbeddingGenerator`
  - `BatchSubmitter`
  - `WebhookEventNormalizer`
- Async job tables/state if batch/background operations become product requirements.
- Provider-specific webhook signature verification only for providers that actually send callbacks.

## Decision (ADR-lite)

Context:

LLM providers are expanding beyond synchronous chat completion, and some already support async lifecycle/webhook patterns. The current product feature is only synchronous text summary.

Decision:

Keep the current synchronous LLM adapter architecture. Do not add LLM webhook configuration by default. Treat LLM webhooks as an optional future async-operation ingress, separate from `Adapter.Generate`.

Consequences:

- Adding another synchronous LLM provider remains small and localized.
- Parameters UI stays accurate and does not ask for irrelevant webhook secrets.
- Future async/batch provider support has a clean place to land without bloating the basic generation interface.
- When adding non-OpenAI-compatible providers, `ProviderConfig` may need a small generalization.

## Requirements

- Capture the webhook-vs-sync LLM distinction in the task record.
- Identify the current extension points for adding new LLM providers.
- List likely files impacted by a future provider addition.
- Keep the diagnosis scoped to architecture and planning; do not change runtime code in this task.

## Acceptance Criteria

- [x] A Trellis task exists for the LLM provider architecture diagnosis.
- [x] Provider webhook research is persisted under `research/`.
- [x] `prd.md` answers both user questions.
- [x] `prd.md` recommends whether to add generic LLM webhook config now.
- [x] `prd.md` describes expected file additions/modifications for future LLM providers.
- [x] Current LLM port, registry, and Parameter schema were rechecked before finishing.

## Definition of Done

- Task materials are written under `.trellis/tasks/06-08-llm-provider-architecture-micro-diagnosis`.
- No business code is changed by this diagnosis task.
- Follow-up implementation can use this PRD as the design note.

## Out of Scope

- Implementing a new LLM provider.
- Adding webhook routes or signature verification.
- Adding streaming, batch, embeddings, multimodal, or tool-calling support.
- Changing the Parameters UI.
- Migrating existing DeepSeek config.
