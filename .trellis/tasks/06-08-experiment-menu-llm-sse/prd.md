# Experiment Menu: LLM and SSE Demo

## Goal

Add an authenticated Experiment menu page for functional research and demos. The page should expose two right-side tabs, `LLM` and `SSE`, using the same daisyUI lifted-tab style as the Parameter page.

## Requirements

* Add an `Experiment` menu route in the logged-in app shell.
* Render a new Experiment page with a left overview/control area and right-side `tabs tabs-lift` content.
* Add two tabs: `LLM` and `SSE`.
* Move the existing header action that triggers the async export completion notification into the `SSE` tab.
* The `SSE` tab must show the current SSE connection state and a small realtime event log/toast-style output when messages arrive.
* The `LLM` tab must provide a chat-window style summary demo.
* The LLM form input must include original text and a requirement prompt.
* Submitting the LLM form must call the existing DeepSeek-backed LLM channel through the internal LLM summary API.
* The generated summary must be appended to the chat window together with request metadata.

## Acceptance Criteria

* [ ] `/experiments` is visible in the app sidebar for logged-in users and has a `/experiments.html` alias.
* [ ] Header no longer shows the `trigger export completed` action after login.
* [ ] The Experiment page compiles and its tabs use the Parameter page lifted radio-tab pattern.
* [ ] The SSE tab can trigger `/api/notifications/test-export-toast`; the result is delivered through the existing `/api/points/sse` stream.
* [ ] The LLM tab calls `/api/llm/summaries` via `frontend/src/api.js`, not direct component fetch.
* [ ] The LLM backend accepts a requirement prompt in addition to raw text while preserving existing dimensions-based clients.
* [ ] Frontend tests cover the new route and API helper.
* [ ] Backend tests cover the new prompt field mapping into the LLM request.

## Definition of Done

* `go test ./...` passes.
* `cd frontend && npm test` passes.
* `cd frontend && npm run build` passes.

## Technical Approach

* Reuse existing backend LLM route/usecase/provider registry and extend the summary command/request DTO with `prompt`.
* Keep `dimensions` backward compatible. The Experiment UI will use a single `summary` dimension and pass the user prompt to the usecase.
* Reuse existing realtime helper and `pointsSSEURL()` for EventSource authentication.
* Add a new `frontend/src/pages/Experiments.svelte` page and wire it through `router.js` and `App.svelte`.

## Decision (ADR-lite)

Context: The repo already has DeepSeek LLM integration, an internal LLM summary endpoint, realtime envelopes, and a test export notification endpoint.

Decision: Build the Experiment page as a thin frontend demo over existing capabilities, with only a small backend extension for prompt-aware summarization.

Consequences: This keeps the demo aligned with production integration settings under Parameter. The LLM result remains JSON-summary based instead of free-form streaming chat, which is acceptable for this MVP and preserves the current adapter abstraction.

## Out of Scope

* Streaming LLM token output.
* A new SSE endpoint separate from `/api/points/sse`.
* Storing chat history in the database.
* Adding a new LLM provider or bypassing Parameter channel configuration.

## Technical Notes

* Relevant files inspected: `frontend/src/pages/Parameters.svelte`, `frontend/src/pages/Dashboard.svelte`, `frontend/src/components/Header.svelte`, `frontend/src/api.js`, `api/routes/llm.go`, `api/usecase/llm_summary.go`, `api/routes/points.go`.
* Relevant specs: `.trellis/spec/frontend/svelte-vite-embed.md`, `.trellis/spec/backend/route-handler-guidelines.md`, `.trellis/spec/backend/api-contracts.md`, `.trellis/spec/backend/error-handling.md`.
