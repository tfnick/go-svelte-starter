# Refactor: Internal API Response Envelope

## Goal

Standardize all internal frontend-facing `/api/*` JSON responses so the
frontend receives a consistent envelope:

```json
{"success":true,"data":{}}
```

and all failures receive:

```json
{"success":false,"error":{"code":"snake_case","message":"safe message"}}
```

This speeds frontend development by making success/error handling predictable
across auth, user, order, admin, and other internal API routes.

## What I Already Know

* The user wants a refactor task.
* The user explicitly requires success responses to use
  `{"success":true,"data":...}`.
* The user explicitly requires failure responses to use
  `{"success":false,"error":{"code":"snake_case","message":"safe message"}}`.
* Existing specs distinguish internal frontend-facing `/api/*` from public
  `/open-api/*`.
* Open API endpoints already have a separate envelope contract and should not
  be changed by this internal API refactor.
* Current internal route helpers return compact shapes such as
  `{"message":"..."}` and `{"error":"..."}`.
* Current resource routes may return DTOs or arrays directly.
* Existing DTO boundaries must remain: routes return DTOs/CO-derived payloads,
  not model structs.

## Requirements

* Apply the response envelope to internal frontend-facing `/api/*` JSON
  endpoints.
* On success, return JSON shaped as:
  `{"success":true,"data":<payload>}`.
* On failure, return JSON shaped as:
  `{"success":false,"error":{"code":"snake_case","message":"safe message"}}`.
* Use stable snake_case error codes.
* Client-visible error messages must remain safe and must not expose raw
  model/db/usecase errors.
* Preserve route-local DTO mapping before wrapping payloads in `data`.
* Update framework HTTP response helpers so route handlers can use the envelope
  consistently.
* Update internal routes to use the new helpers/envelope shape.
* Add/update tests for helper response JSON and representative routes.
* Update backend specs after implementation.

## Acceptance Criteria

* [ ] Internal `/api/*` success resource responses are wrapped in
  `success/data`.
* [ ] Internal `/api/*` success message responses are wrapped in
  `success/data`.
* [ ] Internal `/api/*` error responses are wrapped in `success/error`.
* [ ] Error `code` values are snake_case.
* [ ] Server-side/internal error details are logged but not exposed to clients.
* [ ] Open API `/open-api/*` response contract remains unchanged.
* [ ] DTO boundary guard still prevents routes from returning models directly.
* [ ] Backend specs document the new internal response envelope contract.
* [ ] `go test ./...` passes.

## Definition of Done

* Framework response helpers support the new internal envelope.
* Internal routes consistently use the new helper/wrapper APIs.
* Tests cover envelope success and failure shapes.
* Backend specs are updated with executable response-contract guidance.
* Changes are committed.

## Technical Approach

* Keep the Open API helpers separate.
* Refactor `api/framework/http/response` internal helpers to return the new
  internal envelope.
* Add a generic success wrapper helper for DTO payloads.
* Keep route DTOs explicit; wrap DTOs only after mapping.
* Map usecase error codes to stable snake_case client codes.

## Decision (ADR-lite)

**Context**: Existing internal API responses mix direct DTOs, message-only
objects, and error-only objects. That makes frontend handling inconsistent.

**Decision**: Use one internal API envelope for every `/api/*` JSON response:
`success/data` for success and `success/error` for failure. Open API keeps its
separate public contract.

**Consequences**: Frontend callers must read payloads from `data`; backend
routes need a coordinated refactor; specs and tests need updates.

## Out of Scope

* Changing `/open-api/*` public response envelope.
* Changing frontend UI behavior beyond what is required by the backend API
  contract.
* Returning raw model structs.
* Introducing GraphQL, JSON:API, pagination metadata, or versioned API routing.

## Technical Notes

Relevant specs:

* `.trellis/spec/backend/api-contracts.md`
* `.trellis/spec/backend/route-handler-guidelines.md`
* `.trellis/spec/backend/error-handling.md`
* `.trellis/spec/backend/open-api-guidelines.md`

Relevant code:

* `api/framework/http/response`
* `api/routes`
* `api/framework/archguard`
* `api/usecase`

Existing dirty runtime files:

* `data/app.db-shm`
* `data/app.db-wal`
