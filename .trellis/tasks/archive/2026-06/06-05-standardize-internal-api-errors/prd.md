# Standardize Internal API Error Responses

## Goal

Remove raw internal error exposure from internal `/api/*` handlers and establish a small, repeatable internal API error response pattern. Server-side details should go to structured logs; clients should receive fixed safe messages.

## What I Already Know

* The previous architecture analysis identified internal API error responses as the highest-return next standardization item.
* The project already has file-backed request logging with request IDs and `surface` classification.
* Existing backend error-handling spec forbids returning `err.Error()` in client-facing 500 responses.
* Internal routes still expose raw errors in `api/routes/user.go`, `api/routes/order.go`, and `api/routes/admin.go`.
* Auth routes already mostly use fixed safe client messages.
* Open API routes already use a separate success/error envelope and are out of scope for this task.

## Requirements

* Replace internal `/api/*` responses that directly include `err.Error()` with fixed client-safe messages.
* Log the underlying server error using structured Zerolog fields before returning safe 500 responses.
* Preserve existing HTTP status codes unless a status is clearly unsafe or incorrect.
* Keep the internal API response shape simple: `{ "error": "message" }`.
* Avoid changing Open API response envelope behavior.
* Avoid broad DTO refactors in this task.
* Add tests covering representative safe-error behavior.
* Update error-handling spec with the concrete internal API server-error pattern.

## Acceptance Criteria

* [x] `rg "err\.Error\(\)" api/routes api/middleware` finds no internal API client response leaks.
* [x] Server-side failures in user/order/admin routes return fixed safe messages.
* [x] Server-side failures are logged with component, route context where available, and request ID when present.
* [x] Existing Open API envelope behavior remains unchanged.
* [x] Focused route tests verify that model/db errors do not leak to response bodies.
* [x] `go test ./...` passes.
* [x] `cd frontend && npm run build` passes if frontend artifacts are affected by the normal quality gate.
* [x] `.trellis/spec/backend/error-handling.md` documents the implemented pattern.

## Out of Scope

* Changing Open API error envelope contracts.
* Introducing a full internal API envelope.
* Refactoring all success DTOs.
* Adding a service layer.
* Reworking frontend API client behavior.
* Fixing unrelated database runtime files.

## Technical Approach

* Add route-level helpers in `api/routes` for internal error responses and logging.
* Use existing `middleware.GetRequestID(c)` to attach request ID to logs.
* Replace unsafe `err.Error()` responses in `user.go`, `order.go`, and `admin.go`.
* Add route tests that inject failing model/db conditions where practical or exercise helper behavior directly.

## Technical Notes

* Prior analysis: `.trellis/tasks/archive/2026-06/06-05-architecture-standardization-analysis/architecture-standardization-recommendations.md`
* Relevant specs:
  * `.trellis/spec/backend/error-handling.md`
  * `.trellis/spec/backend/logging-guidelines.md`
  * `.trellis/spec/backend/quality-guidelines.md`
* Candidate files:
  * `api/routes/user.go`
  * `api/routes/order.go`
  * `api/routes/admin.go`
  * `api/routes/auth.go`
  * `api/middleware/request_logging.go`
