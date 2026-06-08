# Standardize Open API Contract

## Goal

Make the public `/open-api/v1/*` response contract explicit and shared so future partner-facing endpoints do not copy internal API shortcuts or duplicate envelope helpers.

## What I Already Know

* Open API routes already use a `success`, `data`, and `error` envelope.
* Account route defines `OpenAPIErrorEnvelope` and `OpenAPIErrorBody`.
* API key middleware currently builds a similar envelope with `map[string]interface{}`.
* Open API request logging already uses `surface: "open-api"` and safe consumer fields.
* Internal `/api/*` response helpers are intentionally separate and must not be reused for Open API.

## Requirements

* Centralize Open API error envelope types/helpers in `api/routes` or another stable shared package.
* Update Open API route and API key middleware to use the same typed error envelope.
* Preserve existing response JSON shape and status codes.
* Keep `/open-api/v1` versioning unchanged.
* Do not change internal `/api/*` response helpers.
* Add tests for Open API auth error envelope shape.
* Add a backend Open API guideline spec and link it from the backend index.

## Acceptance Criteria

* [x] Open API route and middleware errors use a single typed envelope helper.
* [x] Missing and invalid API key responses keep `success:false` plus `error.code/message`.
* [x] Account endpoint errors keep `success:false` plus `error.code/message`.
* [x] Internal `/api/*` helper behavior is unchanged.
* [x] Focused Open API envelope tests pass.
* [x] `go test ./...` passes.
* [x] `cd frontend && npm run build` passes if the normal quality gate is run.
* [x] `.trellis/spec/backend/open-api-guidelines.md` documents the public contract.
* [x] `.trellis/spec/backend/index.md` links the Open API guide.

## Out of Scope

* Adding new Open API endpoints.
* Implementing scope enforcement for new endpoints.
* Changing response envelope fields.
* Changing API key storage or hashing.
* Refactoring internal `/api/*` responses.
* Fixing unrelated database runtime files.

## Technical Approach

* Move Open API error envelope structs/helper into a shared route-level file.
* Import the route helper from Open API key middleware for auth failures.
* Keep success DTOs in their route files unless they become shared.
* Add middleware tests for missing/invalid key envelope shape.
* Add `.trellis/spec/backend/open-api-guidelines.md` with versioning, auth, envelope, DTO, and logging rules.

## Technical Notes

* Prior analysis: `.trellis/tasks/archive/2026-06/06-05-architecture-standardization-analysis/architecture-standardization-recommendations.md`
* Relevant files:
  * `api/routes/open_api_account.go`
  * `api/routes/open_api_health.go`
  * `api/middleware/open_api_key.go`
  * `api/middleware/request_logging.go`
  * `.trellis/spec/backend/logging-guidelines.md`
  * `.trellis/spec/backend/route-handler-guidelines.md`
