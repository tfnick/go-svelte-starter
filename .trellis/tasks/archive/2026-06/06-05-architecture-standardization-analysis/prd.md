# Architecture Standardization Opportunity Analysis

## Goal

Analyze the current system from an architecture perspective and identify the next standardization opportunities now that the basic features are working. The output should become a concrete set of project conventions that makes future feature development faster, more predictable, and easier for AI assistants and developers to follow.

## What I Already Know

* The repository is a single Go + Svelte/Vite project.
* Backend code is organized around `api/db`, `api/middleware`, `api/models`, and `api/routes`.
* The current backend convention intentionally has no separate service layer; route handlers validate input, call model functions, and return JSON.
* Frontend code lives in `frontend/` and uses Svelte, Vite, Tailwind CSS, and daisyUI.
* Production embeds `frontend/dist` into the Go executable; development uses Vite for HMR and proxies `/api` to Go.
* Existing Trellis specs cover backend directory structure, database usage, error handling, quality, logging, and frontend embed contracts.
* The system now has both internal `/api/*` endpoints and partner-facing `/open-api/v1/*` endpoints.
* Open API responses already use a success/data/error envelope, while many internal API routes still return ad hoc JSON maps or direct model structs.
* Some internal handlers still return `err.Error()` to clients, which conflicts with the existing error-handling guideline.
* Frontend API access is centralized in `frontend/src/api.js`, but currently covers only part of the backend surface.

## Requirements

* Review the existing architecture and implementation patterns across backend, frontend, database, routing, authentication, Open API, and build/runtime boundaries.
* Identify places where repeated feature work would benefit from a stronger standard or template.
* Separate recommendations into:
  * conventions that can be documented immediately,
  * small refactors that would unlock future speed,
  * larger architectural decisions that should remain optional until feature pressure justifies them.
* Prioritize standardization opportunities by impact on future feature velocity, consistency, correctness, and implementation risk.
* Produce concrete guidance that can be added to `.trellis/spec/` rather than a vague architecture essay.
* Implement the approved logging first step: durable single-file logging plus API surface classification.
* Preserve the current lightweight architecture unless the analysis finds a clear reason to introduce an extra layer.

## Analysis Scope

* Backend route registration and handler shape.
* Request DTO, validation, and response DTO conventions.
* Internal API response and error response contracts.
* Open API response, authentication, scopes, and versioning contracts.
* Model-layer database access patterns and transaction boundaries.
* Migration, seed data, and multi-database conventions.
* Logging and observability conventions in routes, middleware, and models.
* Frontend API client conventions, error handling, auth state, and route/page patterns.
* Cross-layer contracts between Go JSON responses and Svelte consumers.
* Build, dev, and production embed workflow expectations.
* Testing expectations for new features.

## Candidate Standardization Areas

* API response contracts:
  define whether internal `/api/*` should keep simple JSON responses or move toward a common success/error envelope.
* Error helpers:
  introduce or document helper functions for safe client errors so handlers do not repeat maps or leak `err.Error()`.
* Handler templates:
  define a standard flow for bind, validate, authorize, call model, log, and respond.
* DTO boundaries:
  clarify when routes may return model structs directly and when response structs are required.
* Open API contract:
  standardize envelope types, error codes, scope checks, versioning, and public DTO naming.
* Route registration:
  consider moving route registration out of `index.go` into feature-specific registration functions if route count keeps growing.
* Frontend API client:
  define a consistent module pattern for all backend endpoints, typed-ish payload naming, and UI error propagation.
* Auth/session contract:
  standardize cookie settings, auth context keys, optional auth behavior, and frontend auth state refresh.
* Database usage:
  reinforce named database rules, transaction boundaries, migration naming, seed data ownership, and shared DB reload behavior.
* Logging storage:
  implement durable single-file logging with structured `surface` fields to distinguish general app, internal `/api/*`, and public `/open-api/*` log entries.
* Testing matrix:
  create a per-feature checklist for backend tests, frontend build checks, API contract checks, and production embed checks.
* Spec structure:
  decide which new Trellis spec documents should exist so later feature work can load the right guidance quickly.

## Acceptance Criteria

* [x] A written architecture analysis exists under this task, with findings grouped by domain and priority.
* [x] The analysis cites concrete files or patterns from the current repo.
* [x] The analysis recommends a minimal set of new or updated `.trellis/spec/` documents.
* [x] Each recommendation states why it helps future feature development.
* [x] Each recommendation is classified as document-only, small refactor, or larger future decision.
* [x] The final output includes a proposed feature-development checklist or template.
* [x] The final output explicitly names which spec updates are recommended for future tasks.
* [x] Runtime logs are persisted to `logs/app.log`.
* [x] `logs/` is ignored by git.
* [x] Internal `/api/*` request logs include `surface: "api"`.
* [x] Public `/open-api/*` request logs include `surface: "open-api"`.
* [x] Request logs include method, route, status, duration, and request ID.
* [x] Request IDs are returned in `X-Request-ID`.
* [x] Open API logs do not include raw API keys.

## Definition of Done

* Task PRD is agreed.
* Architecture analysis artifact is written in the task directory.
* Logging implementation is complete and verified.
* Relevant `.trellis/spec/` logging guidance is updated after implementation.
* Existing unrecognized dirty files are not committed or modified.

## Out of Scope

* Rewriting the application architecture by default.
* Adding a service layer unless the analysis produces a clear, concrete reason.
* Implementing all recommended refactors in this task.
* Changing product behavior or endpoint semantics without separate approval.
* Replacing Svelte, Echo, sqlx, Tailwind, or daisyUI.

## Technical Notes

* Relevant existing specs:
  * `.trellis/spec/backend/index.md`
  * `.trellis/spec/backend/directory-structure.md`
  * `.trellis/spec/backend/error-handling.md`
  * `.trellis/spec/backend/database-guidelines.md`
  * `.trellis/spec/frontend/index.md`
  * `.trellis/spec/frontend/svelte-vite-embed.md`
  * `.trellis/spec/guides/cross-layer-thinking-guide.md`
  * `.trellis/spec/guides/code-reuse-thinking-guide.md`
* Representative files inspected:
  * `index.go`
  * `api/routes/auth.go`
  * `api/routes/user.go`
  * `api/routes/order.go`
  * `api/routes/open_api_account.go`
  * `api/middleware/auth.go`
  * `api/middleware/open_api_key.go`
  * `api/db/db.go`
  * `api/logging/logging.go`
  * `frontend/src/api.js`
  * `frontend/src/router.js`
* Analysis artifacts:
  * `.trellis/tasks/06-05-architecture-standardization-analysis/architecture-standardization-recommendations.md`
  * `.trellis/tasks/06-05-architecture-standardization-analysis/research/current-architecture-standardization.md`
  * `.trellis/tasks/06-05-architecture-standardization-analysis/research/logging-module-assessment.md`
* Current task directory:
  * `.trellis/tasks/06-05-architecture-standardization-analysis`
