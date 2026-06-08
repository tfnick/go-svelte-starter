# LLM Streaming Research

## Sources

* OpenAI official streaming guide: https://developers.openai.com/api/docs/guides/streaming-responses
* Anthropic official streaming guide: https://platform.claude.com/docs/en/build-with-claude/streaming

## Observations

* Mainstream LLM providers commonly support HTTP streaming with SSE.
* OpenAI Responses streaming emits typed semantic events such as lifecycle, output delta, completion, and error events.
* Anthropic streaming also uses named SSE events and includes lifecycle events, content block deltas, ping events, errors, and unknown future event types that clients should tolerate.
* Streaming can be consumed incrementally or accumulated into a final message.
* Streaming can fail after partial output has already been emitted, so retry semantics are different from normal request/response APIs.
* Tool calls and structured JSON arguments may arrive as deltas, so provider adapters must handle partial fragments and final aggregation.

## Impact On Anti-Corruption Layer

LLM integration cannot be modeled only as:

```text
request -> provider response
```

It needs at least two invocation shapes:

```text
request -> final response
request -> normalized event stream -> optional final response
```

Some future providers/features may also need a duplex or session-shaped model:

```text
client input stream <-> provider realtime session <-> output event stream
```

The anti-corruption layer should normalize provider events into business-level stream events instead of leaking raw provider event names and payloads.

Candidate normalized event categories:

* `started`
* `text_delta`
* `tool_call_delta`
* `tool_call_done`
* `reasoning_delta`
* `usage_delta`
* `completed`
* `failed`
* `heartbeat`

## Design Consequences

* LLM ports should expose both non-streaming and streaming methods.
* Streaming adapter interfaces should support context cancellation.
* The business layer should choose whether to pass stream events to a client, accumulate them into a final response, or run the request as a background job.
* Live token streaming should not be forced through durable queue processing because queue latency and retry semantics conflict with low-latency user-facing streams.
* Background LLM jobs can still use queue/event processing when the product does not require live token streaming.
* Partial output persistence is a separate policy decision: for live streams, storing all chunks may be unnecessary or sensitive; for auditable/background jobs, store a final safe result and selected metadata.
* Logs must not contain prompt content, raw chunks, tool arguments, credentials, or provider payload bodies by default.

## Directory Implication

Business-scenario-first organization still works, but the `llm` scenario needs streaming-specific ports and event DTOs:

```text
api/usecase/integrations/llm/
  ports.go              # LLMClient / LLMStream interfaces in business language
  stream_events.go      # normalized stream event DTOs
  adapters/
    openai/
    anthropic/
```

If shared stream primitives are later needed, they should be business-agnostic and narrow, for example cancellation/backpressure helpers or SSE parsing utilities under a framework integration package. Provider-specific event mapping stays in adapters.
