# Implement DeepSeek LLM Vertical Slice

## Goal

Implement the first external-integration vertical slice with DeepSeek while preserving the anti-corruption-layer boundaries from the design task. The slice should prove DB-managed channel/model configuration, encrypted credentials, provider adapter isolation, stable LLM usecase ports, and provider invocation recording.

## Design Inputs

* Archived design: `.trellis/tasks/archive/2026-06/06-06-external-integration-anticorruption-layer/info.md`
* DeepSeek API research: `research/deepseek-api.md`
* First rollout order is LLM, then SMS, then Payment, then HRM.

## Confirmed Decisions

* First LLM provider is DeepSeek.
* Channel, credential, policy, and model option configuration must be database-managed.
* Product users must not directly choose channel/model in the first implementation.
* Backend/admin configuration may select `channel_code` and `model_code` per operation.
* DeepSeek adapter should use the OpenAI-compatible API shape.
* Current provider model IDs should be configured as `deepseek-v4-flash` and `deepseek-v4-pro`.

## MVP Scope

* Add backend integration directory boundary for provider adapters.
* Add framework integration primitives only where needed for this slice.
* Add DB migrations/models for:
  * `integration_channels`
  * `integration_credentials`
  * `integration_policies`
  * `integration_model_options`
  * `integration_invocations`
* Add encrypted credential storage boundary with safe masking/redaction.
* Add LLM usecase port and DTOs for final or structured generation.
* Add DeepSeek adapter under `api/integrations/llm/deepseek`.
* Add resolver that loads channel/model/credential from DB using backend-configured aliases.
* Add provider invocation recording with safe metadata and usage counters.
* Add a minimal text multi-dimension summary usecase and internal API route for validation.

## Out Of Scope

* Intelligent PPT generation task table/worker.
* SSE progress for PPT generation.
* SMS, Payment, and HRM slices.
* Full integration management UI.
* Letting product users choose provider channel or model.
* Storing prompts or provider raw request/response bodies in generic integration logs.

## Initial API Shape

The validation route can be internal and protected:

```text
POST /api/llm/summaries
```

Request:

```json
{
  "text": "long text",
  "dimensions": ["key_points", "risks", "actions"]
}
```

Response:

```json
{
  "summary": {
    "key_points": "...",
    "risks": "...",
    "actions": "..."
  },
  "model_code": "summary-fast",
  "channel_code": "deepseek-prod",
  "invocation_id": "..."
}
```

Route DTOs must remain route-local. Usecase returns `Co` types, not provider DTOs.

## Acceptance Criteria

* [x] DeepSeek adapter is isolated under `api/integrations/llm/deepseek`.
* [x] Business usecases do not import provider SDKs or provider DTOs.
* [x] Channel/model/credential/policy data is loaded from DB.
* [x] Product users cannot pass arbitrary provider model IDs.
* [x] Credentials are encrypted or otherwise protected at rest for this slice and never returned in DTOs/logs.
* [x] LLM provider invocations are recorded in `integration_invocations` with safe metadata.
* [x] Text summary route works through the LLM port and DeepSeek adapter.
* [x] Tests cover resolver/model behavior, usecase validation, route envelope, and provider adapter mapping with a fake HTTP client.
* [x] Existing backend tests pass.

## Open Implementation Questions

* Master key source: credential encryption reads `APP_INTEGRATION_MASTER_KEY`; when absent, the process uses an in-memory fallback key suitable only for tests/local throwaway data.
* First summary endpoint: implemented as normal protected internal API under `POST /api/llm/summaries`; it relies on backend/admin DB configuration for provider/channel/model selection.
* Seed data: no real DeepSeek credential or production config is seeded in migration. Tests insert encrypted fake config. Real channel/model/credential rows should be inserted by admin/setup tooling.

## Implementation Notes

* Added `integration_operation_configs` so backend/admin configuration can bind operations such as `text_summary` to stable `channel_code` and `model_code` aliases.
* `index.go` registers `llm.deepseek.openai_compatible` during bootstrap; usecases resolve adapters through the registry and never import provider packages.
* Generic provider invocation records intentionally exclude prompts, source text, provider request bodies, and provider response bodies.
