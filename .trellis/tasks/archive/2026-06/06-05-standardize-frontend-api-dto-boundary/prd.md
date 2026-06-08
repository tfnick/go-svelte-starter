# Standardize Frontend API DTO Boundary

## Goal

Make DTOs the mandatory response boundary for every internal frontend-facing
`/api/*` resource payload, so route handlers never expose storage/business
models directly to the Svelte app. Preserve simple message/error helpers and
the separate public Open API contract.

## What I Already Know

* User/auth response DTOs already exist and are used for frontend-facing user
  payloads.
* Order response DTOs already exist and are used for order create/list/detail
  payloads.
* `api/routes/responses.go` intentionally returns simple `{ "message": ... }`
  and `{ "error": ... }` maps for helper responses.
* Open API routes already use a separate public envelope and DTO contract.
* There are currently no product HTTP routes.
* Admin reload is a message-only operation and does not need a resource DTO.

## Requirements

* Treat all internal frontend-facing `/api/*` resource responses as DTO-only.
* Do not return `models.*` structs directly from `c.JSON` in internal route
  handlers.
* Keep simple helper responses (`okMessage`, `badRequest`, `errorResponse`,
  etc.) as allowed map-based response helpers.
* Keep Open API routes governed by their existing public DTO/envelope rules.
* Prefer explicit per-resource mapper helpers in `api/routes`, using the
  existing `to<Resource>Response` and `to<Resource>Responses` naming pattern.
* Avoid reflection-based or generic "magic" DTO conversion.
* Add a lightweight guard test that fails when an internal route directly
  returns model variables or direct model constructor values through `c.JSON`.
* Update backend API contract docs to make the DTO boundary mandatory rather
  than conditional.

## Acceptance Criteria

* [x] Backend API contract docs state that internal `/api/*` resource responses
  must use DTOs.
* [x] Docs keep message/error helpers as an explicit exception.
* [x] Docs keep Open API routes as a separate public DTO/envelope contract.
* [x] Docs describe the recommended mapper helper naming convention.
* [x] A guard test exists for internal route model-return violations.
* [x] Existing user/auth/order DTO tests still pass.
* [x] `go test ./...` passes.

## Definition of Done

* Focused tests pass.
* Full Go test suite passes.
* Specs are updated with concrete contracts and wrong/correct examples.
* Task is committed, archived, and journaled.

## Technical Approach

* Add a static route test in `api/routes` that parses route source files with
  Go AST.
* Check internal route files only, excluding Open API files and test/helper
  files where model structs are expected.
* Fail when `c.JSON(...)` directly returns:
  * identifiers inferred from model-returning calls such as `user`, `users`,
    `order`, `orders`, `items`, or `account`;
  * direct `models.<Type>{...}` constructor values.
* Keep the guard intentionally narrow to avoid noisy false positives while
  catching the most common direct model exposure pattern.
* Strengthen `.trellis/spec/backend/api-contracts.md`.

## Decision (ADR-lite)

**Context**: Internal routes currently use lightweight handlers, and direct
model responses are easy to write accidentally as new endpoints are added.

**Decision**: Require explicit DTOs and per-resource mapper helpers for all
internal resource payloads. Use a focused AST guard test instead of a broad
reflection/generic mapper or runtime framework.

**Consequences**: New API surfaces require a little more upfront DTO code, but
frontend contracts become explicit, testable, and decoupled from model/schema
changes. The guard test prevents common violations without forcing all simple
message/error helpers into DTO structs.

## Out of Scope

* Adding product HTTP routes.
* Changing existing Open API response envelopes.
* Introducing a generic or reflection-based DTO mapper.
* Replacing existing message/error helper maps with DTO structs.
* Changing frontend behavior.
* Fixing unrelated database runtime files.

## Technical Notes

* Relevant files:
  * `api/routes/user_responses.go`
  * `api/routes/order_responses.go`
  * `api/routes/responses.go`
  * `api/routes/open_api_account.go`
  * `.trellis/spec/backend/api-contracts.md`
  * `.trellis/spec/backend/route-handler-guidelines.md`
* Repo scan shows current internal resource payloads already use DTO helpers:
  user/auth use user DTOs, and order create/list/detail use order DTOs.
* `data/app.db-shm` and `data/app.db-wal` are unrelated runtime files and must
  remain excluded from this task.
