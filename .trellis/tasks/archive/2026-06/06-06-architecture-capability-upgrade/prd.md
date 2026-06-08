# Architecture Capability Upgrade

## Goal

Upgrade the backend architecture into a clearer, easier-to-use application framework for future business development. The target is not only to satisfy the current layered architecture rules, but also to make common business needs such as API/open API separation, authentication and authorization, transaction ownership, cross-module events, ID-to-name translation, and browser realtime refresh predictable and reusable.

## What I already know

* The project currently uses `routes -> usecase -> models -> db`, with reusable infrastructure under `api/framework`.
* `routes` already owns Echo handlers, route-local DTOs, middleware integration, and response conversion.
* `usecase` already owns `Qry`/`Cmd` inputs, `XxxCo` returns, orchestration, app transaction helpers, and some cross-usecase calls.
* `models` already owns sqlx-backed data access and batch lookup helpers.
* `api/framework/usecase` already provides contextual identity/surface information and app transaction lifecycle helpers.
* `api/framework/events` already provides an in-memory event bus with synchronous transaction handlers and after-commit async dispatch.
* `api/framework/realtime` already provides a hub that publishes messages to subscribers grouped by `userID`.
* `api/framework/data/namelookup` already provides registry-based batch ID-to-name lookup, but the developer-facing usage still needs to be validated and made easier across normal business scenarios.
* A previous decision says app transactions apply only to the `app` database, not `shared`.

## Requirements

* Architecture boundaries must remain explicit:
  * `routes` provides page-facing APIs and open APIs.
  * Page API DTOs and open API DTOs are maintained independently and are not forced to share request/response structs.
  * `routes` is responsible for unified authentication and authorization through middleware and HTTP adapter code.
  * User login must use JWT-based authentication instead of cookie-based authentication.
  * Page APIs and WebSocket connections must authenticate with JWT and must not rely on login cookies.
  * `usecase` is responsible for business logic reuse, orchestration, and transaction control.
  * `usecase` methods keep the shape `XxxQry`/`XxxCmd` as inputs and `XxxCo` as return objects.
  * `models` remains the data access layer and exposes query/write operations to usecases.
* Cross-module coupling should be supported through DDD-like application/domain events:
  * Usecases can publish events to decouple modules.
  * Transaction-sensitive handlers must participate in the app transaction where required.
  * Post-commit side effects must not run before the transaction commits.
* ID-to-name translation must be framework-level enough for developers:
  * Batch lookup must avoid N+1 queries.
  * Reusable resources such as `user.nickname` or `user.name` should work across multiple scenarios, not just one list page.
  * The required code for a common business scenario should be small, obvious, and hard to misuse.
* WebSocket/realtime capability must support the product model that each logged-in browser tab/window is a client:
  * A logged-in browser can receive dynamic data refresh messages, such as points balance updates.
  * The backend can target updates by user and, where needed, by browser client/session.
  * The route/middleware/usecase ownership of realtime authentication and publish behavior must be clear.
* Improve readability and maintainability:
  * Reduce the amount of boilerplate needed for future business development.
  * Keep abstractions small and aligned with existing package responsibilities.
  * Preserve or strengthen architecture guard tests for layer boundaries.

## Acceptance Criteria

* [ ] `api/README.md` accurately documents the final responsibility of `routes`, `usecase`, `models`, and `framework` subpackages.
* [ ] Page API and open API examples show independent route DTOs calling the same usecase where business behavior overlaps.
* [ ] Authentication and authorization entry points are visibly owned by `routes`/middleware, not hidden in models.
* [ ] User login issues JWT credentials and authenticated API calls no longer depend on cookies.
* [ ] WebSocket authentication uses JWT-compatible client identity propagation instead of cookie-based session lookup.
* [ ] At least one transaction-owned usecase demonstrates app transaction control and event publication without leaking db internals upward.
* [ ] Event propagation supports same-transaction handlers and post-commit side effects in a documented and tested way.
* [ ] ID-to-name translation has at least two representative examples and tests proving batch loading rather than N+1 behavior.
* [ ] Realtime/WebSocket support identifies browser clients and can refresh points balance for logged-in clients.
* [ ] Architecture guard tests prevent business-agnostic framework code from drifting into `routes`, `usecase`, or `models`, and prevent layer violations.
* [ ] `go test ./...` passes.
* [ ] `go vet ./...` passes.

## Definition of Done

* Tests added or updated for architecture-critical behavior.
* Lint/type checks pass where applicable.
* Documentation or Trellis specs are updated if new architecture rules are established.
* Code remains organized by layer responsibility and avoids broad unrelated refactors.
* Rollback risk is considered for any framework-level API change.

## Out of Scope

* Replacing the current Go web framework.
* Forcing page API and open API DTOs to share structs.
* Keeping cookie-based login/session authentication.
* Moving transaction ownership into `routes` or `models`.
* Making `shared` database writes participate in app transactions.
* Building a multi-node realtime fanout system unless explicitly added later.
* Implementing a durable distributed event bus unless explicitly added later.

## Technical Notes

* Current layered architecture guide: `api/README.md`.
* Usecase context and transaction helpers: `api/framework/usecase/context.go`, `api/framework/usecase/transaction.go`.
* Existing event bus: `api/framework/events/events.go`.
* Existing realtime hub: `api/framework/realtime/realtime.go`.
* Existing points websocket route: `api/routes/points.go`.
* Existing ID-to-name framework: `api/framework/data/namelookup/namelookup.go`.
* Existing event-driven points example: `api/usecase/points_events.go`.

## Technical Approach

* Implement JWT as an HTTP framework concern:
  * `routes` issues access tokens after successful login/register.
  * `middleware` authenticates `Authorization: Bearer <token>` for HTTP APIs.
  * WebSocket authentication uses `access_token` query parameter because browser-native `WebSocket` cannot set custom authorization headers.
  * `usecase` continues to work with `fwusecase.Context.Actor` and does not parse HTTP headers or cookies.
* Keep page API DTOs and open API DTOs independent. Shared business behavior is reused through usecase entry points, not shared request/response structs.
* Keep eventing in `api/framework/events` and use sync transaction handlers for strongly consistent same-app-transaction side effects.
* Keep ID-to-name translation in `api/framework/data/namelookup` plus `api/usecase/translate` registrations. Use helper APIs to collect IDs from typed business rows and batch-load once per resource key.
* Model each logged-in browser WebSocket connection as a realtime client with `client_id`, while preserving user-level broadcast for data such as points balance.
* Update architecture guard tests so retired cookie/session auth symbols do not reappear in production code.

## Open Questions

* Should this architecture optimization task include implementation immediately, or should it first produce a more detailed technical design with staged PRs?
* JWT MVP uses access tokens only, with a 7-day lifetime. Refresh-token rotation is out of scope unless later requested.
* Frontend MVP stores the access token in `localStorage`; this is simple for the starter app but can be revisited for higher-security production scenarios.
* WebSocket MVP transports the token in `access_token` query parameter.
