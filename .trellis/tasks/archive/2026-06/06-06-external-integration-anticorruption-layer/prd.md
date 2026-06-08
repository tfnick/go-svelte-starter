# Design External Integration Anti-Corruption Layer

## Goal

Design an architecture for integrating many external systems and channels while keeping provider-specific schemas, authentication, callback behavior, and policy quirks outside core business logic. The design should clarify the anti-corruption layer, framework responsibilities, business responsibilities, and the recommended project directory layout.

## What I Already Know

* The project is a Go backend plus Svelte frontend.
* Current backend layering is `routes -> usecase -> models -> db`.
* `api/framework` is reserved for business-agnostic infrastructure.
* Existing public external API surface is `/open-api/v1/*` with API-key authentication.
* Existing durable domain event capability is queue-first through `api/framework/events` and `api/framework/queue`.
* Raw goqite is restricted to `api/framework/queue`.
* Existing specs emphasize route-local DTOs, usecase-owned `Cmd/Qry/Co`, safe logging, and strict import boundaries.
* The requested design must handle multiple external scenarios, for example payment, HRM systems, SMS, LLMs, WeChat, and Work WeChat.
* The design must support many channels, potentially 20+.
* Different channels may use different authentication methods.
* Some channels have blacklist/whitelist constraints.
* Some channels support callback notifications.
* LLM providers may use streaming request/response flows with partial output, tool-call deltas, usage updates, and late errors.
* SMS usage needs confirmed-failure manual resend, with the option to resend on the original channel or switch to another channel, without leaking provider details into business callers.
* HRM integration needs to support department and employee pull sync plus provider callbacks, while updating internal canonical department/employee models instead of leaking provider entities into the business model.
* LLM usage includes text multi-dimension summarization where business code may request a specific channel and model.
* LLM usage also includes long-running intelligent PPT generation where a user submits a topic and parameters, and the system generates an outline and slide content asynchronously.
* First LLM implementation should integrate DeepSeek through DB-managed channel/model configuration.

## Research References

* [`research/local-architecture-scan.md`](research/local-architecture-scan.md) - current repo constraints and initial architecture implications.
* [`research/llm-streaming.md`](research/llm-streaming.md) - LLM streaming affects ports, adapter contracts, cancellation, partial output, and persistence.

## Assumptions

* This is primarily an architecture design task, not a feature implementation task yet.
* The solution should align with DDD / ports-and-adapters / anti-corruption-layer thinking.
* Business usecases should not depend on raw provider DTOs, SDKs, webhook payloads, or transport concerns.
* Provider callbacks may need eventual consistency and idempotent processing, likely using the existing durable event and queue foundation.
* Secrets and raw provider payloads must be treated as sensitive by default.

## Open Questions

* No blocking questions remain for this design task.
* Follow-up implementation decisions are tracked in [`info.md`](info.md#open-follow-up-decisions).

## Requirements (Evolving)

* This task is design-only: produce an architecture review/design document and do not implement code skeletons or real channels in this round.
* Primary organization should be business-scenario-first, for example payment, HRM, SMS, LLM, WeChat, and Work WeChat.
* Provider/channel-specific details should live under the owning business scenario as adapters, not as the top-level business boundary.
* Channel configuration should be database-managed first: channel enablement, auth configuration, whitelist/blacklist policy, callback settings, priority, environment, and provider-specific metadata should be represented as manageable configuration.
* Credentials should be encrypted in the project database for this design round.
* Credential design must include ciphertext storage, secret redaction, safe audit fields, and key-rotation planning; plaintext credentials must not appear in logs, errors, events, or provider callback records.
* Provider callbacks should use persistent receipt plus queue-based asynchronous processing: the HTTP route verifies and records the callback, then quickly acknowledges while business processing runs asynchronously and idempotently.
* Callback receipt tables should store raw provider payloads in encrypted ciphertext fields, plus hashes and safe snapshots. Raw payloads must not be exposed through ordinary DTOs, logs, domain events, frontend pages, or feature business tables.
* This design should only plan backend architecture. Integration management API and frontend UI are future tasks.
* LLM streaming must be represented as a first-class integration shape, separate from ordinary request/response and provider callback flows.
* LLM adapters should normalize provider-specific streaming events into stable business-level stream events and support cancellation, timeout, partial failure handling, and optional final aggregation.
* Live LLM streaming should not be forced through durable queues; background LLM jobs may still use queue/event processing when live token streaming is not required.
* LLM channel and model selection should be DB-configured through business-visible aliases such as `channel_code` and `model_code`; business usecases should not depend on raw provider model IDs.
* LLM channel/model selection should be backend/admin configured only in the first implementation. Product users should not directly choose provider channels or models.
* LLM product features such as multi-dimension summarization and intelligent PPT generation should own prompts, output schemas, task state, authorization, and persistence. Provider adapters only call providers and normalize results/events.
* Long-running LLM product workflows should use project-owned task state plus queue execution by default, optionally reporting progress through SSE or domain events.
* Intelligent PPT generation should be protected by submission rate limits, concurrent task limits, queue depth limits, model/channel quota checks, and explicit product task states.
* Product task status, such as PPT generation progress, should be stored in product-owned tables. The integration layer may record provider invocations, but should not own feature task lifecycle.
* SMS should not use system-level automatic channel failover by default. Provider adapters send through one channel; they should not secretly switch channels.
* SMS manual resend and channel switching should be initiated explicitly after confirmed failure by an operator or approved business workflow.
* SMS delivery attempts should be explicitly recorded so resend decisions, duplicate risk, selected channel, operator action, and provider callbacks remain auditable.
* HRM department and employee sync should support both pull sync and callback sync, converging into the same idempotent internal apply usecases.
* HRM external department/employee IDs should be mapped to internal model IDs through explicit integration entity mappings; external IDs should not become canonical model IDs by default.
* HRM employee lifecycle changes, such as termination, should update the internal employee model first. Follow-up actions such as disabling a login user should be separate business policies or event subscribers.
* Define the boundary between framework responsibilities and business/provider responsibilities.
* Propose a project directory layout for external integrations.
* Cover outbound integration calls and inbound provider callbacks.
* Cover multiple scenarios, multiple channels, different authentication types, blacklist/whitelist constraints, and callback notification support.
* Preserve current repo boundaries: `framework` stays business-agnostic; `usecase` owns business orchestration; `routes` owns HTTP adapter concerns.
* Describe how provider-specific DTOs map into stable business commands/events.
* Describe how errors, retries, idempotency, logging, and observability should be handled at architecture level.
* Identify likely archguard/spec updates needed to keep the boundary enforceable.

## Acceptance Criteria (Evolving)

* [x] Scope is confirmed as design-only; no code skeleton or provider implementation in this task.
* [x] Primary directory organization is confirmed as business-scenario-first.
* [x] Channel configuration mode is confirmed as database-managed first.
* [x] Credential storage direction is confirmed as encrypted DB storage.
* [x] Callback processing direction is confirmed as persistent receipt plus queue-based async processing.
* [x] Management API/UI is excluded from this task.
* [x] Design distinguishes framework primitives from provider adapters and business usecase ports.
* [x] Design includes a concrete directory layout proposal.
* [x] Design covers outbound calls and inbound callbacks.
* [x] Design covers LLM streaming as an online stream flow with normalized stream events.
* [x] Design validates LLM channel/model selection through DB-backed aliases rather than provider model IDs.
* [x] First LLM implementation provider is confirmed as DeepSeek with DB-managed channel/model configuration.
* [x] Design validates text multi-dimension summarization and long-running intelligent PPT generation against the LLM anti-corruption-layer boundaries.
* [x] Design states that intelligent PPT generation requires rate limiting, queued execution, capped worker concurrency, and project-owned task status.
* [x] Design covers at least payment, HRM, SMS, LLM, WeChat, and Work WeChat examples.
* [x] Design validates HRM department and employee pull sync plus callback sync against the anti-corruption-layer boundaries.
* [x] Design covers 20+ channel scale without requiring a giant switch/case in business usecases.
* [x] Design covers channel auth strategies, blacklist/whitelist policy, callback verification, retries, idempotency, and safe logging.
* [x] Design covers SMS confirmed-failure manual resend and manual channel switching as scenario orchestration concerns.
* [x] Design aligns with existing DDD event / queue capabilities.
* [x] Design lists implementation phases or small PR slices.
* [x] Vertical slice rollout order is confirmed as LLM, then SMS, then Payment, then HRM.

## Definition of Done

* A clear architecture proposal is recorded in the task.
* Directory layout and ownership rules are explicit.
* Trade-offs and alternatives are documented.
* If implementation is included, relevant tests and archguard/spec updates are added.
* If implementation is not included, the follow-up implementation plan is concrete enough to start a later task.

## Out of Scope (Tentative)

* Adding or modifying production Go code.
* Adding framework skeleton packages or archguard tests in this task.
* Designing or implementing integration management APIs.
* Designing or implementing frontend management pages.
* Designing or implementing intelligent PPT generation APIs, task tables, workers, or frontend pages.
* Implementing all providers/channels.
* Choosing vendor-specific SDKs for every scenario.
* Building a full admin UI for integration configuration.
* Persisting actual credentials before a concrete secret-management design is approved.

## Technical Notes

* Existing directory spec: `.trellis/spec/backend/directory-structure.md`.
* Existing eventing spec: `.trellis/spec/backend/eventing-guidelines.md`.
* Existing Open API spec: `.trellis/spec/backend/open-api-guidelines.md`.
* Existing quality/spec constraints: `.trellis/spec/backend/quality-guidelines.md`, `.trellis/spec/backend/logging-guidelines.md`, `.trellis/spec/backend/error-handling.md`.
* Candidate design direction: ports-and-adapters with provider-specific anti-corruption adapters and business-owned ports.
* LLM streaming research points to three invocation shapes: request/final response, request/normalized event stream/final aggregation, and possible future realtime session-style streams.
* Formal architecture design: [`info.md`](info.md).

## Decision Log

### Directory Organization

**Context**: External integrations may include many scenarios and 20+ provider channels. If top-level code is organized by provider, business usecases can start depending on provider vocabulary and DTOs.

**Decision**: Use business-scenario-first organization. Each scenario owns stable business ports and DTOs; provider/channel adapters live under that scenario boundary.

**Consequences**: Business semantics stay clear and DDD-friendly. Some providers that support multiple scenarios may have adapter code in multiple scenario folders; shared low-level HTTP/signing primitives can still live in framework-level packages when they are truly business-agnostic.

### Channel Configuration

**Context**: There may be 20+ channels with different auth methods, restrictions, callback behavior, environments, and operational enablement needs.

**Decision**: Use database-managed channel configuration as the primary model.

**Consequences**: Runtime configuration can be managed without redeploying code, but the design must include auditability, validation, sensitive-value handling, and safe fallback behavior. The adapter implementation still remains code-owned; DB configuration selects and parameterizes adapters rather than storing executable logic.

### Credential Storage

**Context**: Different channels need secrets such as API keys, client secrets, private keys, webhook secrets, app secrets, and signing keys.

**Decision**: Store credentials encrypted in the project database for this design.

**Consequences**: The design must include an encryption boundary, master-key source, key rotation, redaction, and audit rules. DB rows should store ciphertext plus non-sensitive metadata such as credential type, key version, masked display value, enabled state, and timestamps. Plaintext is only available inside a narrow runtime decryption boundary and must never cross into logs, business events, error messages, or frontend DTOs.

### Callback Processing

**Context**: Payment, WeChat, Work WeChat, HRM, and other providers may retry callbacks, send duplicates, or expect fast acknowledgement.

**Decision**: Use persistent callback receipt plus queue-based asynchronous processing.

**Consequences**: Callback routes should authenticate/verify source, enforce channel policy, compute idempotency keys, persist callback receipt metadata plus encrypted raw payload, payload hash, and safe snapshot, enqueue processing, and return quickly. Raw payload access must be restricted, audited, and excluded from ordinary logs, events, responses, frontend DTOs, and scenario feature tables. Business processing runs in a durable worker and must be idempotent. This aligns with the existing goqite-backed event/queue capabilities and avoids coupling provider HTTP timeouts to core business state transitions.

### Management Surface

**Context**: Database-managed channel configuration eventually needs operational APIs and UI for enablement, credentials, policy, callback records, and audit review.

**Decision**: Exclude management API and frontend UI from this design task.

**Consequences**: This task can focus on backend architecture boundaries, directory layout, data concepts, and processing flows. Management APIs/pages should be split into later tasks after the architecture is accepted.

### LLM Streaming

**Context**: LLM integrations are not always short request/response calls. Many providers emit streaming chunks, typed lifecycle events, tool-call deltas, usage updates, and late errors while the user is waiting.

**Decision**: Treat streaming as a first-class adapter contract in the LLM scenario.

**Consequences**: The `llm` scenario needs business-level stream event DTOs and ports that can return a normalized stream, not just a final response. Provider adapters translate raw stream events into stable internal stream events. Live streaming should use online transport/cancellation/backpressure semantics rather than durable queue delivery. Durable queues remain appropriate for background LLM jobs where live token streaming is not required.

### LLM Channel And Model Selection

**Context**: Business features may need to choose a specific LLM channel and model, for example using one model for fast summaries and another model for high-quality PPT generation. For the first implementation, product users should not directly choose providers or model IDs.

**Decision**: Model selection should be DB-configured through business-visible aliases. Business/admin configuration may set `channel_code` and `model_code`, and the resolver maps them to enabled channels, encrypted credentials, adapter implementations, provider model IDs, default parameters, and allowed capabilities. Product users do not directly choose channel/model in the first implementation.

**Consequences**: Business code remains free of raw provider model IDs and SDK types. Operators can enable, disable, or re-map models through configuration. Policies can restrict which operations may use which channels/models. Product features still own prompts, structured output schemas, task persistence, and user authorization.

### First LLM Provider

**Context**: The first implementation slice should validate the LLM anti-corruption layer with one real provider.

**Decision**: Integrate DeepSeek first. Channel, credentials, model options, and policy remain DB-managed. The adapter should use DeepSeek's OpenAI-compatible API shape.

**Consequences**: The first LLM implementation can focus on one concrete adapter while still proving the provider-agnostic port, DB-backed model aliasing, encrypted credential boundary, and invocation recording. Additional providers can be added later under the same `llm` scenario without changing business usecases.

### Vertical Slice Rollout Order

**Context**: The design can be validated through multiple bounded scenarios. LLM stresses model selection, streaming, long-running tasks, rate limiting, and provider invocation recording. SMS validates outbound delivery and manual resend. Payment validates strong idempotency and state transitions. HRM validates pull sync plus callback convergence.

**Decision**: Implement vertical slices in this order: LLM first, then SMS, then Payment, then HRM.

**Consequences**: The first implementation work should prioritize the shared integration configuration, credential boundary, model options, invocation records, DeepSeek LLM adapter, LLM ports, and resource-controlled PPT task pipeline. Later slices can reuse the same foundation while adding delivery attempts, payment callback idempotency, and external entity mappings.
