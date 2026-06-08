# LLM Provider Webhook and Extension Notes

Date: 2026-06-08

## Question

Do LLM providers generally require webhook configuration, and does the current project architecture need changes before adding new LLM providers?

## Provider Findings

### OpenAI / Azure OpenAI

OpenAI documents first-class API webhooks for event delivery, with official examples for validating webhook signatures and handling events. The documented use cases are provider lifecycle / asynchronous events such as background responses and fine-tuning style events, not ordinary synchronous chat completion calls.

Azure OpenAI also documents webhooks as a way to receive notifications when asynchronous operations complete, which reinforces that webhook demand belongs to async job completion rather than the baseline synchronous generation path.

Sources:

- https://platform.openai.com/docs/guides/webhooks
- https://learn.microsoft.com/en-us/azure/ai-foundry/openai/how-to/webhooks

### Anthropic

Anthropic supports asynchronous Message Batches. Official documentation describes creating batches and retrieving result files / batch state. I did not find a first-class Anthropic webhook configuration page in the official docs during this pass. For the architecture decision, this still means async LLM work may need polling or a callback later, but it should not be modeled as part of the synchronous `Generate` contract.

Sources:

- https://docs.anthropic.com/en/api/creating-message-batches
- https://docs.anthropic.com/en/docs/build-with-claude/batch-processing

### Google Gemini

Gemini has a Batch API for processing many requests asynchronously. The official docs describe batch workflows and status/result retrieval. Google AI Studio can also run batch jobs and supports webhook notifications in that UI/workflow context. As with OpenAI, this is an async/batch concern, not a synchronous text generation requirement.

Sources:

- https://ai.google.dev/gemini-api/docs/batch-api
- https://ai.google.dev/gemini-api/docs/openai
- https://ai.google.dev/gemini-api/docs/text-generation

### DeepSeek

DeepSeek's official API docs are OpenAI-compatible for chat completion style calls. The existing project adapter is aligned with that model. I did not find an official DeepSeek webhook requirement in the docs during this pass.

Sources:

- https://api-docs.deepseek.com/
- https://api-docs.deepseek.com/api/create-chat-completion

## Current Repo Findings

- Stable LLM port is `api/usecase/integrations/llm/ports.go`.
- Current adapter contract is synchronous: `Adapter.Generate(ctx, cfg, req)`.
- DeepSeek implementation lives in `api/integrations/llm/deepseek/deepseek.go` and performs an OpenAI-compatible chat completions request.
- `SummarizeTextWithLLM` loads enabled LLM channel/model config, records an invocation, calls the registered adapter, parses JSON, and records usage.
- Parameter schema currently exposes only DeepSeek `base_url` and `api_key`, with no LLM webhook credential/config fields.
- Backend spec already defines the intended integration boundary:
  - `api/usecase/integrations/<scenario>` owns stable ports/DTOs.
  - `api/integrations/<scenario>/<provider>` owns provider DTO mapping, HTTP/SDK calls, and error normalization.
  - usecases select provider/channel/model through DB configuration and registry, not direct provider imports.

## Synthesis

1. Webhook support is real for some LLM providers, but mostly for asynchronous/background/batch/lifecycle workflows.
2. Synchronous text summary generation does not need webhook config.
3. Adding webhook fields to every LLM provider schema now would create misleading configuration surface.
4. Future LLM webhook ingress should be operation-specific and optional:
   - provider callback route receives the event
   - route verifies provider signature
   - integration boundary normalizes the provider event
   - usecase updates job/invocation state or emits a domain event
5. The current architecture is good enough for adding another synchronous LLM provider. The main improvement before broad provider expansion is to make provider config less DeepSeek/OpenAI-compatible specific when a non-compatible provider actually lands.

