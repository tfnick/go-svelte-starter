# Current Architecture Standardization Opportunities

## Purpose

Capture repo-backed observations about where the current working system can become more standardized, with the goal of speeding up later feature development.

## Current Architecture Snapshot

The project is intentionally lightweight:

* `index.go` owns process setup, database initialization, middleware, and route registration.
* `api/routes/*` handlers bind and validate requests, call `api/models/*`, and return JSON.
* `api/models/*` owns data structures, CRUD, business rules, and transaction use.
* `api/middleware/*` owns session auth and Open API key auth.
* `api/db/db.go` owns named database management, migrations, transaction helpers, and reopen support.
* `frontend/src/api.js` centralizes browser API calls.
* `frontend/src/router.js` centralizes simple SPA route helpers.

This shape is appropriate for the current project size. The best next standardization work is mostly about making feature patterns explicit, not adding new architectural layers immediately.

## High-Impact Standardization Candidates

### 1. Internal API response and error contract

Current observations:

* Internal `/api/*` handlers often return ad hoc `map[string]string` or `map[string]interface{}` responses.
* Some routes return model structs directly.
* Some handlers return `err.Error()` to clients, especially in `api/routes/user.go` and `api/routes/order.go`.
* The existing backend error-handling spec already says 500 responses should not expose internal errors.
* Open API routes already use a structured `success`, `data`, and `error` envelope.

Recommendation:

Define a standard for internal API success and error responses. This does not have to copy the Open API envelope exactly; internal Svelte-facing APIs can remain simpler if that is intentional. The key is to make the choice explicit and provide helper functions or examples.

Classification: small refactor if helpers are added; document-only if only a spec is written.

Spec target:

* Add or update `.trellis/spec/backend/error-handling.md`.
* Consider a new `.trellis/spec/backend/api-contracts.md`.
* Update `.trellis/spec/frontend/svelte-vite-embed.md` with frontend error expectations if the internal API shape changes.

Why it accelerates future work:

New handlers can follow one response template, frontend error handling can stay predictable, and future AI-generated handlers are less likely to leak internal errors.

### 2. Handler implementation template

Current observations:

* Handlers follow a repeated informal flow: bind, validate, call model, return JSON.
* Auth handlers include form and JSON binding compatibility helpers.
* Other handlers mostly use direct `c.Bind`.
* Validation is inline and route-specific.
* Logging is present in auth reset flow, middleware, db, and main, but handler-level logging expectations are not fully standardized.

Recommendation:

Document a handler template:

1. Define request DTO near the handler.
2. Bind request using the agreed helper/pattern.
3. Normalize and validate input.
4. Read auth context if required.
5. Call model functions.
6. Log server-side failures with safe context.
7. Return standardized success or error response.

Classification: document-only first; small refactor later if response or binding helpers are introduced.

Spec target:

* New `.trellis/spec/backend/route-handler-guidelines.md`.

Why it accelerates future work:

Every new endpoint starts from the same skeleton, which reduces decisions and avoids inconsistent validation and response behavior.

### 3. DTO boundary rules

Current observations:

* Some internal routes return model structs directly.
* Open API routes define explicit response DTOs, such as `OpenAPIAccountResponse`.
* Public-facing API boundaries are already separated from internal model types.

Recommendation:

Standardize when direct model returns are acceptable and when response DTOs are required. A practical rule could be:

* Open API: always explicit DTOs and envelope.
* Internal API: explicit DTOs when the model contains sensitive, unstable, or unused fields; direct model return is acceptable only for simple demo/internal entities with no sensitive fields.
* Frontend API client docs should name expected payload shapes for each endpoint.

Classification: document-only initially.

Spec target:

* New `.trellis/spec/backend/api-contracts.md`.
* Update frontend API contract section in `.trellis/spec/frontend/svelte-vite-embed.md`.

Why it accelerates future work:

Prevents accidental exposure and makes cross-layer contracts easier to reason about before adding UI.

### 4. Open API public contract

Current observations:

* Open API middleware accepts Bearer tokens or `X-API-Key`.
* Open API account route uses a public envelope and explicit DTO.
* Open API errors are generated in both middleware and route code, with similar but separate helper shapes.
* Scope data exists in `OpenAPIConsumerContext`, but scope enforcement conventions are not yet visible.

Recommendation:

Create a focused Open API contract spec covering:

* Response envelope.
* Error code naming.
* Authentication header precedence.
* Scope check pattern.
* Versioning rules under `/open-api/v1`.
* Public DTO naming.
* Separation from internal `/api` handlers and models.

Classification: document-only first; small refactor if shared Open API response helpers are added.

Spec target:

* New `.trellis/spec/backend/open-api-guidelines.md`.

Why it accelerates future work:

Future partner-facing endpoints need tighter consistency than internal UI endpoints, and this keeps that standard visible before the Open API surface grows.

### 5. Route registration organization

Current observations:

* `index.go` currently registers all routes directly.
* The route list is still manageable, but internal API and Open API routes are now distinct surfaces.

Recommendation:

Keep `index.go` as the application bootstrap for now, but define the threshold and preferred pattern for moving route registration into `api/routes/register.go` or feature-specific registration functions.

Classification: larger future decision; small refactor only when route count or grouping complexity justifies it.

Spec target:

* Update `.trellis/spec/backend/directory-structure.md`.

Why it accelerates future work:

Avoids premature abstraction while giving future route growth a known landing pattern.

### 6. Frontend API client conventions

Current observations:

* `frontend/src/api.js` wraps `fetch`, credentials, JSON parsing, and error throwing.
* It currently exposes auth and `getUser`, not the full internal API surface.
* Svelte pages rely on thrown `Error` messages for UI alerts.

Recommendation:

Document a frontend API module convention:

* All API calls use relative `/api/*` URLs.
* New endpoints get named functions in `api.js` or domain-specific API modules once the file grows.
* Errors should preserve the server-provided client-safe message.
* Components should not call `fetch` directly unless the task explicitly justifies it.

Classification: document-only initially; small refactor if `api.js` is split by domain later.

Spec target:

* Update `.trellis/spec/frontend/svelte-vite-embed.md`.
* Consider a new `.trellis/spec/frontend/api-client-guidelines.md` if frontend API usage grows.

Why it accelerates future work:

Keeps Svelte pages focused on UI state and makes cross-layer API changes easier to audit.

### 7. Per-feature testing matrix

Current observations:

* Existing specs list build checks such as `cd frontend && npm run build` and `go test ./...`.
* Backend DB spec lists integration tests for named database behavior.
* There is not yet a concise per-feature test checklist that combines route, model, frontend, and embed concerns.

Recommendation:

Add a standard checklist for feature work:

* Model/database changes: migration test or integration coverage.
* Route changes: handler tests or API-level tests for success and error cases.
* Frontend changes: `npm run build`.
* Cross-layer changes: verify backend JSON shape matches frontend consumption.
* Production-serving changes: run embed-specific tests and packaging scripts.

Classification: document-only.

Spec target:

* Update `.trellis/spec/backend/quality-guidelines.md`.
* Update `.trellis/spec/frontend/svelte-vite-embed.md`.

Why it accelerates future work:

Makes "what should I run?" obvious for each type of feature and reduces missed cross-layer regressions.

## Recommended Priority

1. API response/error contract and handler template.
2. Open API public contract.
3. DTO boundary rules.
4. Frontend API client conventions.
5. Per-feature testing matrix.
6. Route registration organization threshold.

This order focuses first on the patterns that affect almost every future feature.

## Suggested Spec Changes

Create:

* `.trellis/spec/backend/api-contracts.md`
* `.trellis/spec/backend/route-handler-guidelines.md`
* `.trellis/spec/backend/open-api-guidelines.md`

Update:

* `.trellis/spec/backend/index.md`
* `.trellis/spec/backend/directory-structure.md`
* `.trellis/spec/backend/error-handling.md`
* `.trellis/spec/backend/quality-guidelines.md`
* `.trellis/spec/frontend/svelte-vite-embed.md`

Optional later:

* `.trellis/spec/frontend/api-client-guidelines.md`

## Recommended Scope for This Task

Recommended MVP:

* Complete the architecture analysis.
* Add or update specs for API contracts, route handler guidelines, Open API guidelines, and cross-layer frontend API conventions.
* Do not implement code refactors yet, except possibly tiny helper additions if the user explicitly wants executable conventions in the same task.

Recommended out of scope:

* Introducing a service layer.
* Moving route registration out of `index.go`.
* Reworking all existing handlers.
* Changing endpoint response shapes without a separate migration plan.
