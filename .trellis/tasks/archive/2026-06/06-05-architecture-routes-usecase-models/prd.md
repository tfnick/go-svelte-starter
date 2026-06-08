# Routes -> Usecase -> Models Architecture Upgrade

## Goal

Upgrade the backend architecture from the current lightweight
`routes -> models -> db` shape to `routes -> usecase -> models -> db`.

The goal is to make route handlers protocol adapters for page-facing `/api/*`
and third-party-facing `/open-api/*`, move reusable business use cases into a
dedicated `api/usecase` layer, and keep `api/models` focused on sqlx-backed
data access and persistence operations.

## What I Already Know

- The current backend is an Echo Go app.
- Current internal routes live in `api/routes`.
- Current persistence/business structs and DB functions live in `api/models`.
- Earlier local work introduced `api/dto` for API payload structs/mappers; this has since been retired in favor of route-local DTOs
  and `api/helpers/*` for reusable helper utilities.
- Current order routes call models directly and perform route-level
  orchestration such as product validation, order creation, and display-name
  assembly.
- Current `models.CreateOrder` does more than a simple write: it reserves
  shared product stock, writes order/order_items in the app DB transaction, and
  compensates stock on failure.
- Current Open API routes under `/open-api/v1/*` are separate from internal
  `/api/*` routes and use their own public envelope rules.
- The requested target shape is:
  - `routes`: Open API endpoints for third-party apps plus internal API
    endpoints for pages.
  - `usecase`: reusable use-case layer, receives `Qry` / `Cmd` inputs and
    returns client objects named `XxxCo`.
  - `models`: query/write operations for the usecase layer, built on sqlx and
    DB helpers.

## Requirements

- Add a new `api/usecase` layer.
- Route handlers must call usecase functions for business flows instead of
  orchestrating those flows directly through models.
- Usecase entry points are organized by use case, not by transport route.
- Usecase entry points receive a route-built `fwusecase.Context` as their first
  argument.
  - Internal `/api/*` routes pass an internal surface context with the
    authenticated actor populated from auth middleware when available.
  - Open API routes pass an OpenAPI surface context with the authenticated
    consumer populated from OpenAPI middleware.
  - Authentication and authorization remain route/middleware concerns. Usecase
    code can rely on the context representing the already-authenticated caller,
    and may enforce business-level context consistency such as account
    mismatch checks.
  - Horizontal usecase-to-usecase calls must forward the same context.
  - Models receive standard `context.Context`, not `fwusecase.Context`.
- Usecase inputs use explicit command/query types:
  - `XxxCmd` for state-changing operations.
  - `XxxQry` for read/query operations.
- Usecase outputs use explicit client-object types:
  - `XxxCo` naming convention.
  - `Co` values represent route-independent application results that routes
    can map into internal DTOs or Open API DTO/envelopes.
- `Co -> DTO` and `Co -> Open API DTO/envelope` mapping lives in the
  corresponding route files.
  - `routes` should call explicit route-local mapping helpers.
  - `usecase` should not return route/protocol DTOs directly.
- Different route surfaces that perform the same business operation must reuse
  the same usecase entry point.
  - Example: page-facing and third-party-facing order creation routes should
    both call the same order creation usecase.
- Usecase layer may orchestrate multiple model operations and helper calls.
- Usecase methods normally call their own module's model functions for
  module-internal data access.
- Usecase methods may call other usecase methods horizontally for
  cross-module reuse.
  - Example: an order usecase can call a user/account/auth-related usecase when
    it needs cross-module application behavior rather than duplicating logic.
  - Horizontal usecase calls should remain acyclic and intentional.
  - Usecase-to-usecase calls should reuse `Cmd` / `Qry` / `Co` contracts rather
    than reaching into another module's models directly.
- Usecase layer owns a reusable typed error/code system.
  - Example codes: validation, not found, unauthorized, forbidden, conflict,
    internal.
  - Routes map usecase error codes to internal `/api/*` error responses or
    Open API error envelopes.
  - Routes must not inspect `error.Error()` strings to determine HTTP status.
- Models layer should be narrowed toward DB access and persistence operations.
  - Models can expose query/write functions needed by usecases.
  - Models should not own route-specific response shaping.
  - Models should not own cross-route application usecase orchestration unless
    it is purely a persistence transaction concern.
- Keep route layer responsibilities narrow:
  - bind HTTP/path/query/body values,
  - read authenticated/current consumer context,
  - rely on middleware for authentication and authorization,
  - build a `fwusecase.Context` from the authenticated route context,
  - validate transport-level presence/format where appropriate,
  - build `Qry` / `Cmd`,
  - call usecase with `fwusecase.Context`,
  - map `Co` to internal DTO or Open API response shape,
  - return HTTP status/errors.
- Keep Open API and page API route contracts separate even when they call the
  same usecase.
- Preserve the no-N+1 ID/name pattern introduced by the dictionary/id-name
  normalization task.
- MVP scope is full migration of order, user CRUD, auth/session/password-reset,
  and Open API account read flows.
- Any route touched by those flows should call `api/usecase` instead of
  orchestrating model calls directly.
- Update backend specs to reflect the new architecture.

## Acceptance Criteria

- [x] `api/usecase` exists and contains migrated use cases for order, user,
      auth, and Open API account flows.
- [x] Order creation is migrated behind a reusable usecase entry point.
- [x] User CRUD routes call user usecases.
- [x] Auth/session/password-reset routes call auth usecases.
- [x] Open API account read route calls an Open API/account usecase.
- [x] At least two route surfaces can reuse the same usecase entry point, or
      the PRD explicitly documents why only one route surface exists in the
      MVP while preserving the reusable signature.
- [x] Usecase command/query/client-object naming follows `XxxCmd`, `XxxQry`,
      and `XxxCo`.
- [x] Routes pass a `fwusecase.Context` into usecase entry points, and the
      context carries different internal actor vs OpenAPI consumer metadata.
- [x] Authn/authz remains in middleware/routes; usecase receives
      already-authenticated caller context and may enforce business consistency
      checks.
- [x] Usecase forwards standard `context.Context` to models.
- [x] Usecase errors use explicit typed codes, and routes map those codes to
      the correct internal API / Open API response shapes.
- [x] No migrated route uses string matching on `error.Error()` to choose HTTP
      status.
- [x] `api/routes` owns route-local mapping from `usecase.XxxCo` values to
      internal response DTOs and Open API response DTOs/envelopes.
- [x] `api/usecase` does not import `api/routes`.
- [x] Cross-module business reuse happens through usecase-to-usecase calls, not
      by one module reaching into another module's model internals.
- [x] Usecase-to-usecase calls do not introduce package import cycles.
- [x] Routes do not call models directly for migrated usecase flows.
- [x] Models remain responsible for sqlx/database operations.
- [x] Existing internal `/api/*` behavior continues to work.
- [x] Existing `/open-api/*` behavior continues to work.
- [x] Backend tests cover the migrated usecase and at least one route using it.
- [x] `go test ./...` passes.
- [x] Frontend tests/build pass if any page-facing API response changes.
- [x] `.trellis/spec/backend/*` documents the new layer responsibilities.

## Definition of Done

- Tests added or updated for usecase behavior and route integration.
- Backend and affected frontend checks pass.
- Specs updated for the new architecture and naming conventions.
- Rollback path is clear: routes can temporarily call old model functions if a
  migration is incomplete, but new usecase-backed flows must be isolated and
  tested.
- Implementation style is a broad one-pass migration across all target flows;
  final code must not leave order/user/auth/OpenAPI account flows half-migrated.

## Out of Scope

- Large framework adoption or dependency injection framework unless separately
  approved.
- Introducing unrelated domain rewrites outside order, user, auth, and Open API
  account flows.
- Changing database schema unless needed by a migrated usecase.
- Changing public Open API response contracts unless explicitly required.
- Reworking frontend UI unless API response shape changes require it.

## Technical Approach (Evolving)

MVP: migrate the main backend flows in one architecture upgrade.

- Create `api/usecase` files grouped by business area, such as:
  - `order.go`
  - `user.go`
  - `auth.go`
  - `open_api_account.go`
- Define command/query/client-object types for migrated flows:
  - `fwusecase.Context`
  - `CreateOrderCmd`
  - `OrderDetailQry`
  - `UserOrdersQry`
  - `CreateUserCmd`
  - `UpdateUserCmd`
  - `UserDetailQry`
  - `RegisterCmd`
  - `LoginCmd`
  - `ForgotPasswordCmd`
  - `ResetPasswordCmd`
  - `AuthStatusQry`
  - `OpenAPIAccountQry`
  - `OrderCo`
  - `OrderItemCo`
  - `OrderDetailCo`
  - `UserCo`
  - `AuthSessionCo`
  - `AuthStatusCo`
  - `OpenAPIAccountCo`
- Define a usecase error contract, such as:
  - `type ErrorCode string`
  - `type Error struct { Code ErrorCode; Message string; Cause error }`
  - `func E(code ErrorCode, message string, cause error) error`
  - `func CodeOf(err error) ErrorCode`
  - constants for validation/not found/unauthorized/forbidden/conflict/internal.
- Move reusable order orchestration into usecase:
  - validate business-level order item quantities,
  - load products/prices,
  - call model write/query functions,
  - batch-load display names,
  - return `Co`.
- Keep HTTP-specific errors/status codes in routes.
- Keep DB-specific operations in models.
- Routes map:
  - internal `/api/*`: `Co -> route-local DTO` response,
  - Open API `/open-api/*`: `Co -> public Open API envelope/DTO`.
- Each route file owns explicit `Co` mapping helpers for its HTTP contract, such as:
  - `func ToOrderResponse(co usecase.OrderCo) OrderResponse`
  - `func ToOrderDetailResponse(co usecase.OrderDetailCo) OrderDetailResponse`
  - `func ToOpenAPIAccountEnvelope(co usecase.OpenAPIAccountCo) OpenAPIAccountEnvelope`
- Dependency direction:
  - `routes -> usecase`
  - `usecase -> models/helpers`
  - `fwusecase.Context -> context.Context -> models`
  - `usecase -> usecase` is allowed for cross-module application reuse
  - never `usecase -> routes`
- Default usecase behavior is module-internal `usecase -> models` access.
  Horizontal `usecase -> usecase` calls are reserved for cross-module behavior
  reuse and should avoid cycles.
- Dictionary routes can remain direct/static in this task unless implementation
  touches DB-backed dictionary behavior.

## Scope Decision

- MVP scope: migrate order, user CRUD, auth/session/password-reset, and Open
  API account read flows in the same task.
- Usecase error strategy: use the framework-owned typed error/code system from `api/framework/usecase`.
  Routes map usecase error codes to HTTP statuses and to the correct internal
  API or Open API response shape.
- CO mapping strategy: corresponding `api/routes` files own
  `Co -> DTO/OpenAPI DTO` mapping helpers. Routes remain thin callers of
  usecase and route-local mapping functions.
- Implementation style: perform one broad architecture change across all
  target flows rather than staging order, user, auth, and OpenAPI account as
  separate incremental slices.
- Usecase call topology: default to module-internal `usecase -> models`, and
  allow intentional horizontal `usecase -> usecase` calls for cross-module
  reuse.

## Technical Notes

- Existing route files inspected:
  - `api/routes/order.go`
  - `api/routes/user.go`
  - `api/routes/auth.go`
  - `api/routes/open_api_account.go`
  - `api/routes/dictionaries.go`
- Existing model files inspected:
  - `api/models/order.go`
  - `api/models/user.go`
  - `api/models/open_api_account.go`
  - `api/models/open_api_key.go`
- Existing support packages:
  - `api/helpers/httpresponse`
  - `api/helpers/idname`
  - `api/helpers/orderdisplay`
- Relevant specs to update during implementation:
  - `.trellis/spec/backend/directory-structure.md`
  - `.trellis/spec/backend/route-handler-guidelines.md`
  - `.trellis/spec/backend/api-contracts.md`
  - `.trellis/spec/backend/database-guidelines.md`
  - `.trellis/spec/backend/open-api-guidelines.md`

## Expansion Sweep

Future evolution:

- More route surfaces may reuse the same usecase, especially internal page APIs
  and third-party Open API flows.
- Usecase interfaces may later become stable testing seams for background jobs,
  CLI commands, or async workflows.

Related scenarios:

- Internal API and Open API should keep separate response contracts even when
  they call the same usecase.
- DTO and CO should remain distinct: DTO is protocol-facing, CO is
  route-independent usecase output.

Failure and edge cases:

- Usecases need clear error categories so routes can map errors to proper HTTP
  statuses without string matching.
- Order creation crosses app/shared DB concerns; the design must preserve
  compensation behavior and test it.

## DTO directory retirement

- pi/dto is retired. Request/response DTO structs and explicit Co -> DTO mapping helpers live in the corresponding pi/routes file to keep each HTTP contract close to its handler.
- Reusable non-route helper code still belongs under pi/helpers/*; usecase remains route-independent and must not import pi/routes.
