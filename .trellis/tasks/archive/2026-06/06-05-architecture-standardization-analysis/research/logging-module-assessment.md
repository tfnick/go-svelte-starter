# Logging Module Assessment

## Scope

Assess whether the current logging module needs improvement, and identify which upgrades are worth doing now versus later.

This is analysis only. It does not change `api/logging`, route code, middleware, or `.trellis/spec/`.

## Current Implementation

Primary file:

* `api/logging/logging.go`

Current public API:

* `logging.Init(isDevelopment bool)`
* `logging.For(component string) zerolog.Logger`
* `logging.IsDevelopment() bool`

Current behavior:

* Uses `github.com/rs/zerolog`.
* Emits JSON logs to stdout.
* Adds timestamps.
* Adds `component` fields through `logging.For`.
* Uses `debug` level in development and `info` in production.
* Keeps development output JSON instead of switching to pretty console output.
* Stores dev mode and base logger behind a mutex.

Current call sites:

* `index.go`: startup, database initialization failures, cleanup failure, server start/stop.
* `api/db/db.go`: database connection, migrations, migration skip/apply, reload lifecycle.
* `api/routes/auth.go`: development-only password reset URL log.

Current spec:

* `.trellis/spec/backend/logging-guidelines.md` already documents the current Zerolog contract.
* Echo HTTP access/request logging is explicitly out of scope today.

## Assessment

The current module is a good foundation. It is small, consistent, and appropriate for the current system size.

The project does not need:

* A new logging library.
* A separate logging framework.
* Pretty console logs in development.
* A logging abstraction that hides Zerolog completely.
* Distributed tracing or metrics as part of this immediate standardization task.

The project does need clearer observability conventions as the API surface grows.

The user has also clarified a target requirement: logs should be persisted to one file, while still making it easy to distinguish internal `/api` events from `/open-api` events. That makes file logging and API-surface classification part of the recommended future logging design, not just an optional operational enhancement.

## Strengths

### 1. Simple Shared Logger

`logging.For("component")` gives every subsystem a stable component field without forcing large abstractions.

This is enough for current components:

* `main`
* `db`
* `auth`

### 2. JSON Everywhere

JSON logs in both dev and production keep runtime output machine-readable and avoid environment-specific parsing differences.

### 3. Correct Startup Boundary Logging

Startup failures use `Fatal()` in `index.go`, which fits unrecoverable process boundaries.

### 4. DB Lifecycle Logs Are Structured

DB connection and migration logs include useful fields like `database`, `path`, and `migration`.

### 5. Sensitive Development Helper Is Guarded

Password reset URL logging is wrapped in `logging.IsDevelopment()`, which matches the current logging spec.

## Gaps

### 1. No File Log Storage

The current logger writes only to stdout.

Impact:

* Runtime logs disappear when the process exits unless the parent environment captures stdout.
* Local Windows execution through scripts or a built executable has no stable log files for later inspection.
* API support/debugging becomes harder because there is no durable log trail.

Recommended future direction:

* Write JSON logs to one file under a dedicated ignored directory such as `logs/app.log`.
* Keep stdout in development if desired, but persist to files for both development and production runs.
* Decide whether production should write stdout plus files, or files only. For this repo, stdout plus files is usually the safer default because it preserves console visibility.
* Add `logs/` to `.gitignore`.
* Define what happens when log file creation fails. Recommended: fail startup for production if configured log files cannot be opened; allow explicit fallback only if documented.

Suggested file:

* `logs/app.log` for startup, database lifecycle, internal `/api/*`, and public `/open-api/*` logs.

Distinguish sources with structured fields:

* `surface: "app"` for startup, database, and general internal logs.
* `surface: "api"` for internal `/api/*` request/error logs.
* `surface: "open-api"` for public `/open-api/*` request/error logs.

Priority: P1 because the user explicitly wants durable logs that can distinguish API surfaces.

### 2. No API Surface Classification Field

Internal `/api/*` and partner-facing `/open-api/*` currently share the same logger behavior.

Impact:

* Partner-facing support/debugging can get mixed with internal UI traffic inside the single log stream.
* Open API logs need distinct safe field rules even when they share the same file.
* Public API logs should carry partner/account context, while internal API logs should carry user/session-safe context.

Recommended future direction:

* Add a structured `surface` field, for example `surface: "api"` or `surface: "open-api"`.
* Keep all logs in `logs/app.log`.
* Add route group helpers or middleware so `/api/*` events consistently use `surface: "api"` and `/open-api/*` events consistently use `surface: "open-api"`.
* Keep Open API logs free of raw API keys and secrets.
* If Open API traffic later needs separate retention or access control, split files can be a future migration. Start with one file because that is the requested shape.

Priority: P1.

### 3. No Request Correlation

There is no request ID in logs or response headers.

Impact:

* When one API call fails, it is hard to connect the client-visible failure with server logs.
* Once internal handlers stop returning `err.Error()`, logs become more important for diagnosis.
* Partner-facing Open API support will need a way to trace support requests without exposing secrets.

Recommended future direction:

* Accept incoming `X-Request-ID` when present.
* Generate a request ID when missing.
* Set `X-Request-ID` on responses.
* Add `request_id` to request, middleware, and route-level error logs.

Priority: P1 when implementing file/surface-separated logs, because request IDs make separated files useful.

### 4. No Standard Route-Level Error Logging Pattern

Existing handlers often return safe fixed messages in auth routes, but several user/order/admin handlers still return raw `err.Error()`.

Once those are fixed, internal error details must move into logs. There is not yet a standard for:

* Which handler errors should be logged.
* Which fields to include.
* Which fields must never be included.
* Whether route handlers create their own component loggers.

Recommended future direction:

Define a route error logging pattern:

```go
logger.Error().
    Err(err).
    Str("request_id", requestID).
    Str("route", c.Path()).
    Str("method", c.Request().Method).
    Msg("failed to create order")
```

This is illustrative only; do not implement until the handler/response convention is approved.

Priority: P1 after API error contract work begins.

### 5. No HTTP Access Logging Decision

The logging spec currently says Echo HTTP request/access logging is out of scope.

That was fine before the file/surface classification requirement. With a single file, lightweight request/access logging becomes the natural way to add distinguishable `api` and `open-api` entries. The project should still keep payloads out, but it should define the access-log shape.

* request start/end logs,
* duration,
* status code,
* route pattern,
* method,
* request ID,
* error summary.

Recommended future direction:

Add lightweight request logging with method, route, status, duration, request ID, and surface. Avoid request/response bodies by default.

Priority: P1 for API surface logs.

### 6. Open API Logging Is Not Yet Defined

Open API middleware resolves a consumer context with:

* `KeyID`
* `PartnerID`
* `AccountID`
* `Scopes`
* `Environment`

The safe logging rules for these fields are not documented.

Recommended future direction:

Allow:

* `partner_id`
* `account_id`
* `environment`
* route name
* request ID

Avoid:

* raw API keys
* session IDs
* password reset tokens
* plaintext passwords
* password hashes
* full user/account objects

For `key_id`, document whether it is safe enough to log. If unsure, treat it as sensitive and avoid logging it by default.

Priority: P1 before adding separated Open API file logs.

### 7. Logger Lifecycle Is Fine, but Initialization Ordering Should Stay Explicit

`api/db/db.go` creates a package-level logger through `logging.For("db")`. This happens before `logging.Init()` runs, but `logging.For` returns a logger derived from the then-current `baseLogger`.

In this repo, the default `baseLogger` already writes JSON with timestamps to stdout. After `logging.Init`, newly created loggers get the configured level. Existing package-level loggers may keep the logger state captured when they were created.

Current practical impact:

* DB logs are `Info`, so this is unlikely to break current behavior.
* If a package-level logger expected dev-only `Debug` after `logging.Init`, that could be surprising.

Recommended future direction:

Document one of these conventions:

* Keep package-level loggers only for stable `Info`/`Error` components.
* Prefer creating loggers after `logging.Init()` for components that need environment-dependent debug behavior.
* Or update `logging.For`/logger storage in a future refactor if dev debug behavior becomes important across package-level loggers.

Priority: P3, because current usage is not harmed.

## Recommended Upgrade Sequence

### Step 1: Document File Logging and API Surface Classification

Future spec update:

* Update `.trellis/spec/backend/logging-guidelines.md`.
* Cross-link from route handler guidelines.
* Include Open API safe logging fields.

Deliverable:

* No code required.
* Make log directory, file name, request ID, API surface classification, route error logging, access logging, and sensitive field rules explicit.

### Step 2: Add File Sink Support

Future implementation:

* Ensure `logs/` exists on startup.
* Open one file sink for `logs/app.log`.
* Keep JSON output.
* Add `logs/` to `.gitignore`.
* Consider whether log rotation is needed now or should be a follow-up.

Suggested file:

* `logs/app.log`

### Step 3: Add Request ID Middleware

Future implementation:

* Add middleware that reads or generates `X-Request-ID`.
* Store request ID in Echo context.
* Set response header.
* Provide a small helper such as `middleware.GetRequestID(c)` if needed.

Suggested fields:

* Header: `X-Request-ID`
* Context key: `request_id`
* Log field: `request_id`

### Step 4: Add API Surface Request Logging

Future implementation:

* Add middleware for `/api/*` that writes request summaries with `surface: "api"`.
* Add middleware for `/open-api/*` that writes request summaries with `surface: "open-api"`.
* Include method, route pattern, status, duration, request ID, surface, and safe identity fields.
* Exclude bodies and secrets.

### Step 5: Standardize Handler Error Logs

Future implementation:

* When replacing client-facing `err.Error()` responses, add server-side logs in the same change.
* Use component-specific loggers for stable route groups.
* Include request ID, route, method, and safe entity identifiers.

### Step 6: Decide on Rotation and Retention

Future decision:

* Decide whether to use an external process manager/log shipper, time-based file names, or a Go rotation library.
* For a starter app, simple files plus a documented cleanup policy may be enough.
* For production, define retention for `logs/app.log`.
* If Open API traffic later requires different access control, consider splitting files in a separate future task.

## Suggested Future Spec Content

Add these sections to `.trellis/spec/backend/logging-guidelines.md`:

### File Log Storage

* Write logs to `logs/`.
* Keep log files out of git.
* Use JSON lines.
* Suggested file:
  * `logs/app.log`
* Define startup behavior when files cannot be opened.

### API Surface Classification

* Internal `/api/*` request logs use `surface: "api"`.
* Public `/open-api/*` request logs use `surface: "open-api"`.
* General startup/database lifecycle logs use `surface: "app"` or omit `surface` if the component already identifies them.
* All logs go to `logs/app.log`.

### Request Correlation

* Use `X-Request-ID`.
* Preserve incoming values when present.
* Generate values when missing.
* Return the final request ID in the response header.
* Include `request_id` in route error logs.

### Route Error Logging

* Log unexpected server-side errors before returning safe client messages.
* Include `request_id`, route pattern, method, and safe identifiers.
* Do not log request bodies by default.
* Do not log secrets.

### Open API Logging

* Allow partner/account/environment identifiers if documented as safe.
* Never log raw API keys.
* Include request ID for supportability.

### Access Logging Decision

* For classified API logs, log method, route, status, duration, request ID, and safe API surface.
* Do not log request or response bodies by default.

### Rotation and Retention

* Start simple if this remains a local/starter app.
* Add rotation when logs can grow beyond manual cleanup.
* Treat Open API entries as operational data that may later need stricter retention/access rules.

## Recommendation

The earlier recommendation was to avoid immediate logging changes. With the clarified requirement, the recommended next logging work should become a dedicated implementation task:

The next useful logging work should be:

1. Document file logging and API surface classification.
2. Add file sink support for `logs/app.log`.
3. Add request ID middleware.
4. Add internal API and Open API request logging middleware with `surface` fields.
5. Add route-level server error logs when raw `err.Error()` client responses are removed.
6. Decide rotation/retention once expected deployment mode is clear.

This keeps the logging system lightweight while preparing it for real operational debugging.
