# Standardize Route Handler Template

## Goal

Turn the repeated internal route handler pattern into a concrete convention and lightweight helper implementation so future feature endpoints can be written faster and more consistently.

## What I Already Know

* The architecture analysis identified route handler templates as a high-return standardization item.
* Internal `/api/*` routes currently repeat simple `{"error": "..."}`
  and `{"message": "..."}` response maps in many handlers.
* The previous task added safe server-error helpers in `api/routes/internal_errors.go`.
* Open API routes already have their own public envelope and must stay separate.
* The project intentionally keeps a lightweight routes -> models -> db shape with no service layer.

## Requirements

* Add route-level helpers for simple internal API client errors and message responses.
* Keep internal response shapes unchanged: `{"error":"..."}` and `{"message":"..."}`.
* Reuse the existing safe server-error helpers for logged model/db failures.
* Refactor representative internal routes to use the helper/template pattern.
* Do not change Open API envelope behavior.
* Add focused tests for the response helper behavior.
* Add a backend route handler guideline spec and link it from the backend index.

## Acceptance Criteria

* [x] Internal route helpers exist for common client error/status responses.
* [x] User/order/auth/admin simple error or message responses use the shared helpers where practical.
* [x] Server-side failures continue to log through `internalServerError(...)` or `notFoundError(...)`.
* [x] Open API response envelope files are unchanged.
* [x] Focused route helper tests pass.
* [x] `go test ./...` passes.
* [x] `cd frontend && npm run build` passes if the normal quality gate is run.
* [x] `.trellis/spec/backend/route-handler-guidelines.md` documents the standard handler flow.
* [x] `.trellis/spec/backend/index.md` links the new route handler guide.

## Out of Scope

* Adding a service layer.
* Refactoring all success DTOs.
* Changing Open API public response envelopes.
* Changing frontend API client behavior.
* Reworking route registration in `index.go`.
* Fixing unrelated database runtime files.

## Technical Approach

* Add a small `api/routes/responses.go` helper file for internal API response shapes.
* Keep helper names package-private so this remains a route-layer convention.
* Update `internal_errors.go` to use the shared response shape helper.
* Refactor existing internal routes where the response body is a simple error/message map.
* Add tests that exercise helper status codes and response JSON shape.
* Add a backend route handler guideline spec with the canonical flow and examples.

## Technical Notes

* Prior analysis: `.trellis/tasks/archive/2026-06/06-05-architecture-standardization-analysis/architecture-standardization-recommendations.md`
* Relevant specs:
  * `.trellis/spec/backend/error-handling.md`
  * `.trellis/spec/backend/logging-guidelines.md`
  * `.trellis/spec/backend/quality-guidelines.md`
* Candidate files:
  * `api/routes/responses.go`
  * `api/routes/internal_errors.go`
  * `api/routes/auth.go`
  * `api/routes/user.go`
  * `api/routes/order.go`
  * `api/routes/admin.go`
  * `.trellis/spec/backend/route-handler-guidelines.md`
  * `.trellis/spec/backend/index.md`
