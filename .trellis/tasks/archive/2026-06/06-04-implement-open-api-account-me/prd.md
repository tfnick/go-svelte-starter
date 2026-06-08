# Implement open-api account me endpoint

## Goal

Implement the first external-facing machine-to-machine API endpoint, `GET /open-api/v1/account/me`, using a dedicated `/open-api` route surface, API key middleware, external DTOs, and `models` reorganization that preserves the current single-deployable architecture.

## What I Already Know

- The architecture diagnosis task [`06-04-diagnose-architecture-open-api-upgrade`](../06-04-diagnose-architecture-open-api-upgrade/prd.md) recommends:
  - keep a single deployable
  - keep the route → models → db structure
  - do not add a standalone service layer
  - split `/open-api/v1/...` from internal `/api`
  - add a dedicated API key middleware
  - use route-level DTOs for external contracts
- Current internal auth is session-cookie based in `api/middleware/auth.go`.
- Current route registration is centralized in `index.go`.
- Current models layer already mixes persistence and application logic, so the upgrade path is to add dedicated `open_api_*` files inside `api/models`.
- The first external slice should be account / user-like profile read, not orders or products.

## Assumptions (Temporary)

- This implementation can introduce the route, middleware, model files, and any required starter persistence support.
- The endpoint should remain narrow: `/account/me` only, no list/search or arbitrary account lookup.
- If schema support is missing, we may need a starter migration for API key / partner linkage concepts.

## Open Questions

- None

## Requirements

- Add route namespace `/open-api/v1/...`
- Add `GET /open-api/v1/account/me`
- Add dedicated API key middleware for `/open-api`
- Do not reuse current session auth middleware for this endpoint
- Add dedicated route DTOs for the external response
- Do not return `models.User` directly
- Add `api/models/open_api_keys.go`
- Add `api/models/open_api_account_read.go`
- Keep implementation within the current single deployable
- Do not add a standalone `service/` layer
- Use partner-safe error responses for this endpoint

## Acceptance Criteria

- [x] `GET /open-api/v1/account/me` is registered under a separate `/open-api/v1` route group
- [x] The endpoint is protected by dedicated API key middleware
- [x] Successful responses return an external DTO envelope, not internal model structs
- [x] Internal `/api` auth/session behavior remains unchanged
- [x] The implementation follows the architecture blueprint from the diagnosis task
- [x] `go test ./...` passes

## Definition of Done

- New route, middleware, and models files are added as needed
- Any required schema/migration support is included if necessary
- External contract is explicitly bounded by DTO mapping
- Error responses are partner-safe
- Specs or task notes are updated if implementation reveals new constraints

## Technical Approach

- Add `/open-api/v1` route registration in `index.go`
- Create `api/middleware/open_api_key.go`
  - extract API key from `Authorization: Bearer <api-key>`
  - optionally support `X-API-Key` only if useful during early implementation
  - resolve a limited consumer context
- Create `api/models/open_api_keys.go`
  - key lookup
  - validity / active checks
  - consumer context resolution
- Create `api/models/open_api_account_read.go`
  - account/profile read logic for the authenticated consumer account
- Create `api/routes/open_api_account.go`
  - route DTOs
  - `GetOpenAPIAccountMe`
  - partner-safe error envelope
- If persistence support is missing, add starter migration concepts for API keys / partner linkage

## Decision (ADR-lite)

**Context**: We want the first real external API slice without prematurely broadening scope or changing the project into a service-oriented architecture.

**Decision**: Implement only `/open-api/v1/account/me` as the first vertical slice, with a hard route boundary, dedicated API-key middleware, route-level DTO mapping, and `models`-level responsibility split.

**Consequences**:

- Pros:
  - validates the target architecture with minimal scope
  - protects internal user/session model details
  - keeps future expansion path clean
- Cons:
  - introduces some intentional duplication around DTO mapping
  - requires adding new persistence/auth concepts before broad partner features exist

## Out of Scope

- `/open-api` order endpoints
- partner admin console/workflows
- broad account search/list endpoints
- service extraction
- multi-deployable split
- advanced scope/rate-limit management beyond what is minimally needed for this endpoint

## Technical Notes

- Architecture source task: `../06-04-diagnose-architecture-open-api-upgrade/prd.md`
- Current internal auth middleware: `api/middleware/auth.go`
- Current route registration: `index.go`
- Current user model and internal account-like data source: `api/models/user.go`
