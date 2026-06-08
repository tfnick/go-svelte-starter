# Architecture Standardization Recommendations

## Executive Summary

The current system is at the right stage for architecture standardization, but not for a broad architecture rewrite.

The project should keep its lightweight shape:

* Echo route handlers in `api/routes`.
* Business logic and database access in `api/models`.
* Named database and migration infrastructure in `api/db`.
* Auth and Open API authentication in `api/middleware`.
* Svelte pages using a centralized frontend API client.

The highest-return next work is to make repeated feature patterns explicit. Future features will move faster if the team standardizes API response contracts, handler flow, DTO boundaries, Open API rules, frontend API client usage, and a cross-layer testing checklist.

This task recommends spec updates only. It does not update `.trellis/spec/` and does not change application code.

## Architecture Snapshot

### Backend

Current shape:

* `index.go` performs startup, DB initialization, middleware setup, route grouping, and frontend route registration.
* `api/routes/*` handlers bind requests, validate input, call model functions, and return JSON.
* `api/models/*` contains structs, business rules, CRUD functions, database access, and transaction use.
* `api/db/db.go` owns named DB registration, SQLite/Postgres configuration, migration execution, transactions, and reopen support.
* `api/middleware/auth.go` owns session auth and user context injection.
* `api/middleware/open_api_key.go` owns Open API key auth and public consumer context injection.

This matches the existing backend specs: flat layers, no service layer, route validation before model calls, and model-level business/data logic.

### Frontend

Current shape:

* `frontend/src/api.js` wraps `fetch`, credentials, JSON parsing, and error throwing.
* `frontend/src/router.js` provides SPA path normalization, navigation, and route titles.
* Svelte pages consume API helpers and display API errors.
* Vite proxies `/api` in development; Go embeds `frontend/dist` in production.

The frontend has a good central API starting point, but the convention is not yet broad enough for future domains beyond auth.

### Public API Boundary

The repo now has two API surfaces:

* Internal Svelte-facing `/api/*`.
* Partner-facing `/open-api/v1/*`.

Open API already has stronger public-contract patterns than internal routes:

* Explicit response DTOs in `api/routes/open_api_account.go`.
* `success`, `data`, and `error` envelope shape.
* API key auth in `api/middleware/open_api_key.go`.

This difference is reasonable, but it needs to be documented so future endpoints do not blur internal and public contracts.

## Priority Recommendations

### P1: Standardize Internal API Errors and Responses

Problem:

Internal `/api/*` routes currently use mixed response styles:

* Ad hoc `map[string]string` responses.
* Ad hoc `map[string]interface{}` responses.
* Direct model struct responses.
* Direct `err.Error()` exposure in some routes.

Evidence:

* `api/routes/user.go:40`, `api/routes/user.go:52`, `api/routes/user.go:92`, and `api/routes/user.go:104` return `err.Error()`.
* `api/routes/order.go:70`, `api/routes/order.go:92`, `api/routes/order.go:115`, and `api/routes/order.go:152` return `err.Error()`.
* `api/routes/admin.go:14` includes `err.Error()` in a client response.
* `api/routes/auth.go` uses safer fixed client messages for most auth failures.
* `.trellis/spec/backend/error-handling.md` already forbids exposing internal 500 errors.

Recommendation:

Define an internal API response/error convention:

* Keep simple success payloads if the internal API should stay lightweight.
* Use a common client-safe error shape, for example `{ "error": "<message>" }`, unless the team chooses an envelope.
* Never return raw `err.Error()` to clients for server-side failures.
* Log internal errors server-side and return a fixed safe message.
* Use explicit response DTOs when payloads cross a frontend boundary or contain fields the UI should not know about.

Suggested future spec work:

* Add `.trellis/spec/backend/api-contracts.md`.
* Update `.trellis/spec/backend/error-handling.md`.
* Update `.trellis/spec/frontend/svelte-vite-embed.md` with frontend error-consumption expectations.

Classification:

* Document-only first.
* Small refactor later if the team adds helper functions such as `clientError`, `serverError`, or `jsonError`.

Why this speeds future work:

Every future endpoint can follow one response rule. Frontend components can handle errors predictably, and AI-generated handlers are less likely to leak implementation details.

### P1: Define a Route Handler Template

Problem:

Route handlers mostly follow the same informal flow, but the pattern is not yet captured as an executable checklist.

Evidence:

* `api/routes/auth.go` has local request DTOs and separate binding helpers.
* `api/routes/order.go` has a local `CreateOrderRequest` and inline validation.
* `api/routes/user.go` binds model structs directly for create/update.
* `.trellis/spec/backend/quality-guidelines.md` already says request types belong in route files and validation happens before model calls.

Recommendation:

Document a standard handler flow:

1. Define request DTOs near the handler.
2. Bind request using the route's accepted input mode.
3. Normalize input values.
4. Validate before calling model functions.
5. Read auth context from middleware helpers when needed.
6. Call model functions.
7. Log internal failures with safe, structured context.
8. Return a standardized success or error response.

Suggested future spec work:

* Add `.trellis/spec/backend/route-handler-guidelines.md`.
* Link it from `.trellis/spec/backend/index.md`.

Classification:

* Document-only.
* Small refactor later if common response helpers are introduced.

Why this speeds future work:

New features start from a shared skeleton instead of rediscovering validation, logging, auth context, and response shape each time.

### P1: Create an Open API Contract Spec

Problem:

Open API endpoints need stricter public consistency than internal UI endpoints, and the surface has started to grow.

Evidence:

* `index.go:94` registers `/open-api/v1/health`.
* `index.go:98` applies `RequireOpenAPIKey()` to protected Open API routes.
* `api/routes/open_api_account.go` defines `OpenAPIAccountEnvelope`, `OpenAPIErrorEnvelope`, and `OpenAPIErrorBody`.
* `api/middleware/open_api_key.go:63` defines a separate `openAPIError` helper with a similar map-based shape.
* `OpenAPIConsumerContext` includes `Scopes`, but scope enforcement conventions are not documented yet.

Recommendation:

Document Open API-specific rules:

* Public routes live under `/open-api/v1`.
* Public route files and model files use `open_api_*` naming.
* Public responses use a stable envelope with `success`, `data`, and `error`.
* Public error codes use stable snake_case identifiers.
* API key auth accepts Bearer and `X-API-Key` as currently implemented.
* Scope checks should use one named helper pattern before new scoped endpoints are added.
* Public DTOs must not return internal model structs directly.

Suggested future spec work:

* Add `.trellis/spec/backend/open-api-guidelines.md`.
* Link it from `.trellis/spec/backend/index.md`.

Classification:

* Document-only first.
* Small refactor later if shared Open API response helpers are consolidated.

Why this speeds future work:

Partner-facing endpoints need predictable contracts. A dedicated spec prevents future public endpoints from copying internal route shortcuts.

### P2: Standardize DTO Boundary Rules

Problem:

Some routes return model structs directly, while public Open API routes define explicit DTOs. The repo needs a rule for when each is acceptable.

Evidence:

* `api/routes/user.go:44`, `api/routes/user.go:55`, `api/routes/user.go:76`, and `api/routes/user.go:96` return user model data directly.
* `api/routes/open_api_account.go` maps model data into `OpenAPIAccountResponse`.
* `api/models/user.go` protects `PasswordHash` with `json:"-"`, which reduces risk but does not replace explicit API contracts.

Recommendation:

Use this future rule:

* Open API endpoints always return explicit response DTOs.
* Internal API endpoints may return model structs only when the struct is stable and has no sensitive or internal-only fields.
* Auth, account, admin, and partner-facing endpoints should prefer explicit DTOs.
* Response DTOs should be named by API surface and resource, such as `OpenAPIAccountResponse` or `CurrentUserResponse`.

Suggested future spec work:

* Include DTO boundary rules in `.trellis/spec/backend/api-contracts.md`.

Classification:

* Document-only first.
* Small refactor later if existing internal responses are cleaned up.

Why this speeds future work:

Developers can decide payload shape quickly and avoid accidental frontend coupling to database structs.

### P2: Standardize Frontend API Client Usage

Problem:

The frontend API client is centralized but under-specified for future feature growth.

Evidence:

* `frontend/src/api.js:1` defines one `request` wrapper.
* `frontend/src/api.js:22` through `frontend/src/api.js:60` exposes auth helpers and `getUser`.
* `frontend/src/router.js` centralizes route helpers.
* Existing frontend spec requires relative `/api/*` URLs and daisyUI UI patterns.

Recommendation:

Document frontend API client rules:

* Components should use API helper functions, not direct `fetch`, for app API calls.
* API helpers should use relative `/api/*` URLs.
* API helpers should preserve server-provided safe error messages.
* Keep `api.js` as the single module while small.
* Split into domain modules only when it becomes hard to scan, for example `api/auth.js`, `api/users.js`, `api/orders.js`.
* Frontend UI should not depend on raw database field assumptions when a backend DTO would be clearer.

Suggested future spec work:

* Update `.trellis/spec/frontend/svelte-vite-embed.md`.
* Optionally add `.trellis/spec/frontend/api-client-guidelines.md` once frontend API usage grows.

Classification:

* Document-only.

Why this speeds future work:

Future pages get a predictable data-loading and error-handling pattern, reducing UI-specific copies of backend assumptions.

### P2: Add a Per-Feature Cross-Layer Checklist

Problem:

Existing specs include useful quality checks, but feature work would benefit from a compact cross-layer checklist.

Evidence:

* `.trellis/spec/backend/quality-guidelines.md` includes route/model testing expectations.
* `.trellis/spec/frontend/svelte-vite-embed.md` requires `npm run build`, `go test ./...`, and packaging checks for frontend-serving changes.
* `static_test.go` already tests SPA fallback and API path behavior.

Recommendation:

Document a feature checklist:

* Route changed: validate success, client error, auth error, and server error behavior.
* Model/database changed: verify DB helper choice, transaction boundaries, migrations, and rollback behavior.
* Frontend changed: run `cd frontend && npm run build`.
* Cross-layer JSON changed: verify frontend consumes the actual backend response shape.
* Production frontend serving changed: run `go test ./...`; run `build.bat` and `verify-build.bat` for packaging changes.
* Public Open API changed: verify envelope, error code, auth header behavior, and public DTOs.

Suggested future spec work:

* Update `.trellis/spec/backend/quality-guidelines.md`.
* Update `.trellis/spec/frontend/svelte-vite-embed.md`.

Classification:

* Document-only.

Why this speeds future work:

It tells developers and AI agents which checks matter for each feature type instead of relying on memory.

### P1: Add File-Based API Surface Classification

Problem:

The logging module is clean and sufficient for startup and database lifecycle logs, but it currently writes only to stdout. The desired direction is durable file logging in one file, while distinguishing internal `/api/*` logs from partner-facing `/open-api/*` logs through structured fields.

Evidence:

* `api/logging/logging.go` centralizes Zerolog setup with JSON output, timestamps, component fields, and dev/prod log levels.
* `index.go` logs startup, DB initialization failures, cleanup failures, and server start/stop events.
* `api/db/db.go` logs database connection, migration, and reload lifecycle events.
* `api/routes/auth.go` logs password reset URLs only in development mode.
* `.trellis/spec/backend/logging-guidelines.md` explicitly keeps Echo HTTP access/request logging out of scope today.
* API handlers are expected to return safe client messages, but there is not yet a standard for where internal handler errors are logged once client-safe errors replace `err.Error()` responses.
* See `research/logging-module-assessment.md` for the detailed logging module assessment.

Recommendation:

Keep the current Zerolog wrapper. It is not worth replacing. Add a file sink and API-surface classification around it:

* Persist structured JSON logs under an ignored `logs/` directory.
* Use one log file: `logs/app.log`.
* Add `logs/` to `.gitignore` in the future implementation task.
* Add or standardize request IDs, preferably by accepting an incoming `X-Request-ID` or generating one per request.
* Include request ID in response headers and route/middleware logs.
* Add `surface: "api"` for `/api/*` request/error logs.
* Add `surface: "open-api"` for `/open-api/*` request/error logs.
* Use `surface: "app"` or component fields for startup, database, and general internal logs.
* Define route-level error logging rules so internal errors are logged server-side while clients receive safe messages.
* Add public API context fields carefully, such as `partner_id`, `account_id`, and endpoint name, but never log raw API keys, session IDs, reset tokens, passwords, or password hashes.
* Keep log output JSON in both dev and production, matching the current spec.
* Decide rotation and retention after deployment expectations are clear; simple files are acceptable for the first implementation if log growth is bounded.

Suggested future spec work:

* Update `.trellis/spec/backend/logging-guidelines.md`.
* Cross-link logging expectations from future `.trellis/spec/backend/route-handler-guidelines.md`.
* Include Open API logging constraints in future `.trellis/spec/backend/open-api-guidelines.md`.

Classification:

* Document-only in this analysis task.
* Dedicated implementation task later for one file sink, request ID middleware, API-surface request logging, and `.gitignore`.

Why this speeds future work:

Once handlers stop returning raw internal errors, logs become the main way to diagnose production failures. File-backed logs with a `surface` field keep storage simple while still making internal UI traffic and partner-facing Open API traffic easy to filter.

### P3: Define Route Registration Growth Threshold

Problem:

`index.go` currently registers all routes directly. This is still acceptable, but route growth will eventually make startup harder to scan.

Evidence:

* `index.go:64` through `index.go:100` registers auth, user, order, admin, Open API health, and protected Open API account routes.

Recommendation:

Document a threshold rather than refactoring immediately:

* Keep direct registration in `index.go` while route count remains easy to scan.
* Move to `routes.RegisterInternalAPI(api)` and `routes.RegisterOpenAPI(router)` when route groups grow enough that startup concerns and feature route lists obscure each other.
* Do not introduce registration abstractions per endpoint.

Suggested future spec work:

* Update `.trellis/spec/backend/directory-structure.md`.

Classification:

* Larger future decision.

Why this speeds future work:

This avoids premature structure while giving future growth a clear refactor target.

## Recommended Future Spec Set

Create these future spec files:

* `.trellis/spec/backend/api-contracts.md`
* `.trellis/spec/backend/route-handler-guidelines.md`
* `.trellis/spec/backend/open-api-guidelines.md`

Update these existing spec files:

* `.trellis/spec/backend/index.md`
* `.trellis/spec/backend/directory-structure.md`
* `.trellis/spec/backend/error-handling.md`
* `.trellis/spec/backend/logging-guidelines.md`
* `.trellis/spec/backend/quality-guidelines.md`
* `.trellis/spec/frontend/svelte-vite-embed.md`

Optional later:

* `.trellis/spec/frontend/api-client-guidelines.md`

## Proposed Feature Development Template

Use this as the future per-feature planning checklist.

### 1. Classify the API Surface

* Internal `/api/*`.
* Public `/open-api/v1/*`.
* Frontend-only.
* Database/model-only.
* Build/runtime infrastructure.

### 2. Define the Cross-Layer Contract

* Request DTO.
* Validation rules.
* Auth requirement.
* Model function contract.
* Database changes or migrations.
* Response DTO.
* Error cases and HTTP status codes.
* Frontend API helper and UI state.

### 3. Pick the Standard Handler Flow

* Bind.
* Normalize.
* Validate.
* Read middleware context.
* Call model.
* Log internal error if needed.
* Return safe standardized response.

### 4. Check the Boundary Rules

* Do not expose raw internal errors.
* Do not return sensitive model fields.
* Do not mix internal and Open API DTOs.
* Do not call `fetch` directly from Svelte components when an API helper belongs in `api.js`.
* Do not add a service layer unless repeated complexity proves it is needed.

### 5. Run the Right Verification

* Backend-only: `go test ./...`.
* Frontend change: `cd frontend && npm run build`, then `go test ./...`.
* Production embed or packaging change: `build.bat` and `verify-build.bat`.
* DB change: migration and rollback-oriented model tests where practical.
* Open API change: envelope, error code, and auth behavior tests.

## Non-Recommendations

Do not do these as immediate architecture work:

* Do not add a service layer just to look more layered.
* Do not move all route registration out of `index.go` yet.
* Do not force the internal `/api/*` surface to match the Open API envelope unless the team explicitly wants that consistency.
* Do not rewrite existing handlers in bulk before the response and DTO conventions are documented.
* Do not split `frontend/src/api.js` until real growth makes it hard to scan.

## Suggested Follow-Up Tasks

1. Write backend API contract and handler guideline specs.
2. Write Open API guideline spec.
3. Update frontend API client expectations in the frontend spec.
4. Refactor internal route error responses to stop exposing `err.Error()`.
5. Consolidate Open API response helper types after the public contract is documented.
6. Add file-backed logging to `logs/app.log`, including request IDs, `surface` classification, and safe Open API fields.

The first three and the logging guideline update are documentation tasks. The logging file-sink work and other implementation tasks should happen only after the conventions are approved.
