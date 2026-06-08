# Standardize Frontend API Client Usage

## Goal

Make the existing frontend API access pattern explicit and slightly more robust so future Svelte pages use one predictable client boundary for `/api/*` calls, credentials, JSON parsing, and safe server error messages.

## What I Already Know

* The architecture analysis identified frontend API client usage as a remaining P2 standardization item.
* `frontend/src/api.js` already centralizes `fetch`, credentials, JSON parsing, and error throwing.
* Current Svelte components import API helpers from `api.js`; no component directly calls `fetch`.
* Existing frontend spec requires relative `/api/*` URLs and describes the auth status contract.
* Internal API errors now use simple safe `{ "error": "message" }` responses.

## Requirements

* Keep `frontend/src/api.js` as the single frontend API boundary while the API surface is small.
* Preserve current exported auth/user helper function names and call sites.
* Make the shared request helper robust for:
  * JSON request bodies.
  * FormData or non-JSON request bodies when a future endpoint needs them.
  * Empty responses such as `204 No Content`.
  * Safe server-provided error messages from `error` or `message`.
* Do not introduce a new frontend dependency or split domain modules in this task.
* Add focused frontend unit tests for request helper behavior.
* Update frontend spec docs with API client rules and future split threshold.

## Acceptance Criteria

* [x] Components continue to use API helper functions instead of direct `fetch`.
* [x] API helpers continue to use relative `/api/*` URLs.
* [x] Shared request helper preserves credentials and safe error messages.
* [x] Shared request helper does not force `Content-Type: application/json` for `FormData` or explicitly provided headers.
* [x] Shared request helper handles empty success responses.
* [x] Focused frontend tests pass.
* [x] `cd frontend && npm run build` passes.
* [x] `go test ./...` passes.
* [x] Frontend spec documents API client conventions and split threshold.

## Out of Scope

* Splitting `frontend/src/api.js` into domain modules.
* Adding TypeScript.
* Changing backend API response shapes.
* Changing UI rendering or page flows.
* Adding a frontend state-management library.
* Fixing unrelated database runtime files.

## Technical Approach

* Export the shared `request` helper from `frontend/src/api.js` so tests and future domain helpers can reuse it.
* Set JSON content type only when the request body is JSON-like and callers did not provide their own content type.
* Return `null` for `204 No Content` or empty successful responses.
* Keep existing helper exports for auth and user endpoints.
* Add a small Node test runner setup because the repo does not already have frontend tests and Node 22 supports the needed browser fetch primitives.
* Update `.trellis/spec/frontend/svelte-vite-embed.md` with concrete API client conventions.

## Technical Notes

* Prior analysis: `.trellis/tasks/archive/2026-06/06-05-architecture-standardization-analysis/architecture-standardization-recommendations.md`
* Relevant files:
  * `frontend/src/api.js`
  * `frontend/src/App.svelte`
  * `frontend/src/pages/*.svelte`
  * `.trellis/spec/frontend/svelte-vite-embed.md`
* Repo inspection:
  * `rg -n "fetch\\(" frontend/src` shows only `frontend/src/api.js` calls `fetch`.
  * `rg -n "/api/" frontend/src` shows API paths are centralized in `api.js`, aside from a literal explanatory route string in `Dashboard.svelte`.
