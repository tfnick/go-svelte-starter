# Standardize Internal API DTO Boundary

## Goal

Reduce coupling between internal `/api/*` responses and database/model structs by introducing explicit user response DTOs for the highest-impact internal user/auth endpoints.

## What I Already Know

* The architecture analysis identified DTO boundary rules as a remaining P2 standardization item.
* Open API endpoints already use explicit DTOs.
* Internal user routes still return `models.User` directly for create/get/update/list.
* `models.User` currently protects `PasswordHash` with `json:"-"`, but direct model responses still couple clients to DB fields such as numeric `email_verified` and `is_active`.
* Frontend `Dashboard.svelte` displays the `/api/users/:id` JSON result as formatted text and does not depend on numeric field types.
* Auth status and current user already build small ad-hoc user maps.

## Requirements

* Introduce explicit internal user response DTOs in `api/routes`.
* Replace direct `models.User` JSON responses in user routes with DTO mapping helpers.
* Keep existing endpoint response shapes as compatible as practical:
  * user list/detail/create/update still return user objects or arrays, not envelopes.
  * auth current/status still return their existing wrapper objects.
* Prefer boolean DTO fields for user-facing account state where handlers already expose boolean semantics.
* Avoid changing Open API DTOs.
* Add focused tests for DTO mapping and sensitive field exclusion.
* Add/extend backend API contract docs with DTO boundary rules.

## Acceptance Criteria

* [x] Internal user routes no longer return `models.User` directly.
* [x] User DTOs do not expose `password_hash`.
* [x] User account state is exposed through explicit DTO fields.
* [x] Auth current/status responses use shared user summary DTO helpers instead of ad-hoc maps.
* [x] Open API DTO behavior remains unchanged.
* [x] Focused route/DTO tests pass.
* [x] `go test ./...` passes.
* [x] `cd frontend && npm run build` passes.
* [x] Backend API contract/spec docs describe DTO boundary rules.

## Out of Scope

* Introducing a full internal API envelope.
* Refactoring order/product DTOs.
* Changing frontend rendering logic.
* Changing Open API response DTOs.
* Adding a service layer.
* Fixing unrelated database runtime files.

## Technical Approach

* Add internal user DTO structs and mapping helpers in `api/routes`.
* Use the DTO helper in `CreateUser`, `GetAllUsers`, `GetUser`, and `UpdateUser`.
* Use a compact user summary DTO in auth current/status responses.
* Add route-level tests for DTO JSON behavior.
* Add `.trellis/spec/backend/api-contracts.md` and link it from the backend index.

## Technical Notes

* Prior analysis: `.trellis/tasks/archive/2026-06/06-05-architecture-standardization-analysis/architecture-standardization-recommendations.md`
* Relevant specs:
  * `.trellis/spec/backend/route-handler-guidelines.md`
  * `.trellis/spec/backend/open-api-guidelines.md`
  * `.trellis/spec/backend/quality-guidelines.md`
* Candidate files:
  * `api/routes/user.go`
  * `api/routes/auth.go`
  * `api/routes/user_responses.go`
  * `.trellis/spec/backend/api-contracts.md`
  * `.trellis/spec/backend/index.md`
