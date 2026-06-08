# Diagnose Architecture and Plan open-api Upgrade

## Goal

Diagnose the current backend architecture of this project and produce a concrete upgrade direction for exposing a subset of third-party-facing APIs under `/open-api`, while intentionally staying a single deployable and avoiding a new standalone service layer.

## What I Already Know

- The project is a single Echo-based Go web app with one deployable entrypoint in `index.go`.
- Current API surface is grouped under `/api`.
- Authentication is session-cookie based and implemented in `api/middleware/auth.go`.
- Route handlers are organized under `api/routes/` and call `api/models/*` directly.
- There is no separate service layer today; business logic and data access are both placed in `api/models/`.
- Current models already mix different responsibilities:
  - user CRUD and query logic in `api/models/user.go`
  - auth/session/reset flows in `api/models/auth.go`
  - order creation + transaction coordination in `api/models/order.go`
- Current route handlers often bind directly into internal model types or return internal model structs directly.
- The current protected API area uses `RequireAuth()` middleware based on user session cookies.
- Your stated preferences are:
  - stay single deployable for now
  - separate internal API and external API at route, auth, and application boundaries
  - do not introduce a standalone service layer
  - reorganize `models` for clearer responsibilities
  - make `/open-api` a distinct route surface
  - use API key middleware for `/open-api`
  - introduce dedicated DTOs for external API responses instead of exposing internal models directly

## Assumptions (Temporary)

- This task is primarily an architecture diagnosis and upgrade plan task, not immediate implementation.
- `/open-api` will likely begin with a read-heavy subset of data rather than broad write access.
- Partner identity / API key ownership / scope rules should be planned now even if only minimally implemented later.

## Open Questions

- None

## Requirements (Evolving)

- Analyze and summarize the current architecture characteristics.
- Identify current strengths, bottlenecks, and coupling risks relevant to opening third-party APIs.
- Propose an upgrade path for `/open-api` that:
  - keeps the app as one deployable
  - separates internal vs external API entrypoints
  - uses dedicated API key middleware
  - avoids introducing a new service layer
  - reorganizes `models` into clearer internal/open-api responsibilities
  - introduces external DTOs instead of returning internal entities directly
- Explain what should change now versus what can be deferred.
- Cover routing, authentication, model organization, DTO boundaries, versioning readiness, and partner-management readiness.
- Treat account / user-like profile read APIs as the first `/open-api/v1` candidate slice.

## Acceptance Criteria (Evolving)

- [ ] Current architecture is documented in concrete repo terms.
- [ ] Risks of exposing `/open-api` on top of the current structure are explicitly identified.
- [ ] A staged upgrade plan exists that matches the stated preferences.
- [ ] The plan distinguishes immediate refactors from later-phase improvements.
- [ ] The plan includes a recommended `/open-api` package/file layout.
- [ ] The plan explains how account/profile reads should avoid exposing internal user fields directly.

## Definition of Done (Team Quality Bar)

- Diagnosis reflects the actual current code layout
- Recommended upgrade path is consistent with repo conventions
- Scope boundaries and out-of-scope items are explicit
- Trellis artifacts are sufficient for later implementation planning

## Technical Approach

- Inspect the current route registration, middleware boundaries, and models layout.
- Describe the current architecture as a layered-but-collapsed structure:
  - route layer
  - middleware layer
  - model layer containing both domain logic and persistence logic
  - db layer
- Identify the main coupling issues for third-party exposure:
  - internal and external consumer concerns are currently mixed under one `/api` surface
  - auth middleware is user-session oriented, not partner/API-key oriented
  - internal model structs are too close to HTTP response shape
  - model files combine internal operational concerns that should not automatically become partner contracts
- Recommend a staged architecture upgrade that preserves current simplicity while creating hard boundaries in the right places.

## Decision (ADR-lite)

**Context**: The project wants to expose a subset of APIs to third-party consumers but prefers not to split into services yet and does not want to introduce a dedicated service layer.

**Decision**: Keep a single deployable and preserve the current route → models → db style, but harden the boundaries by splitting internal and external API surfaces, adding dedicated API-key authentication for `/open-api`, reorganizing models by responsibility, and introducing external DTOs so internal model shapes do not become public contracts.

**Consequences**:

- Pros:
  - low operational complexity
  - clear path to expose external APIs without premature microservice work
  - better contract isolation at the route and DTO boundary
  - model responsibilities become easier to reason about without a service-layer detour
- Cons:
  - models layer remains a broad layer and must be curated carefully
  - some duplication between internal and external response shaping is intentional
  - future service extraction is easier but not free

**Initial open-api slice**: Start with account / user-like profile read endpoints rather than orders or products. This gives the architecture a realistic privacy-sensitive contract first, forcing clear DTO and field-boundary discipline early.

## Current Architecture Diagnosis

### Strengths

- Simple single-binary deployment
- Clear top-level request flow from `index.go`
- Existing route files are already separated by domain (`auth`, `user`, `order`, `admin`)
- Existing auth middleware is isolated in `api/middleware/auth.go`
- Existing `db` package already centralizes DB registration and migration behavior

### Current Structural Characteristics

- Route handlers perform input parsing, validation, and direct `models` invocation.
- `models` acts as a combined domain + application + persistence layer.
- `models` returns internal entities directly to route handlers in many cases.
- Middleware is currently consumer-specific only for browser/user-session flows.
- Public contract boundaries are weak because internal model structs often double as response payloads.

### Risks If `/open-api` Is Added Naively

- Reusing `/api` handlers would couple partner-facing contracts to internal handler assumptions.
- Reusing session-auth middleware would mix user-login semantics with machine-to-machine access.
- Returning internal `User` / `Order` structs directly would make internal schema evolution harder.
- Putting open-api logic into the same route files would blur internal vs external policy and validation rules.
- Adding open-api-specific behavior directly into existing model files without file-level separation would increase responsibility sprawl.

## Recommended Upgrade Path

### Phase 1: Separate External Entry Surface

Add a distinct route namespace:

- `/open-api/v1/...`

Recommended route file layout:

- `api/routes/open_api_auth.go` (if needed later)
- `api/routes/open_api_account.go`
- `api/routes/open_api_orders.go`
- `api/routes/open_api_keys.go` (admin/internal management endpoints if exposed through HTTP later)

Do not place `/open-api` handlers into current `auth.go`, `user.go`, or `order.go` route files.

### Phase 2: Add Dedicated API Key Middleware

Add separate middleware for partner-facing machine auth:

- `api/middleware/open_api_key.go`

Responsibilities:

- read API key from header
- validate key existence / active state
- load partner/app identity into context
- reject using partner-oriented error responses

Do not reuse current session-cookie middleware for `/open-api`.

### Phase 3: Reorganize models Without Adding a Service Layer

Keep `api/models`, but organize by responsibility.

Recommended direction:

- `api/models/user.go`
- `api/models/order.go`
- `api/models/auth.go`
- `api/models/open_api_keys.go`
- `api/models/open_api_account_read.go`
- `api/models/open_api_order_read.go`

Principle:

- internal operational models stay separate from external-read / partner-contract support logic
- external-use-case-oriented data fetch logic can live in `models`, but should be grouped into dedicated `open_api_*` files instead of being scattered into unrelated internal files

### Phase 4: Introduce External DTOs at Route Boundary

For `/open-api`, route handlers should define dedicated request/response DTOs.

Examples:

- `OpenAPIAccountResponse`
- `OpenAPIOrderSummaryResponse`
- `OpenAPIOrderDetailResponse`

Do not return internal `models.User`, `models.Order`, or future account entities directly.

This is the most important anti-coupling measure in the recommended plan.

For the initial account/profile slice, DTOs should be explicitly allow-listed. Example categories:

- safe identity fields
- partner-visible account status fields
- public/profile metadata intended for third-party consumption

Do not expose:

- password-related fields
- internal activation / moderation details unless intentionally part of contract
- session/auth internals
- raw internal persistence shape just because it is convenient

### Phase 5: Versioning and Partner Management Readiness

Not everything needs to be implemented immediately, but the architecture should leave space for:

- `/open-api/v1/...` versioned prefix from day one
- partner / app / API key entities in models and DB
- future key rotation, revocation, and scope fields
- future per-partner DTO evolution without breaking internal API shape

For the first account/profile slice, plan for partner-specific field policy from the start, even if v1 uses a single default contract.

## Recommended File / Package Direction

### Immediate New Boundaries

- `api/routes/open_api_*.go`
- `api/middleware/open_api_key.go`
- `api/models/open_api_*.go`

Suggested first files:

- `api/routes/open_api_account.go`
- `api/models/open_api_keys.go`
- `api/models/open_api_account_read.go`

### Keep As-Is for Now

- single `index.go` deployable
- `api/db`
- no separate `service/` package

## Implementation Blueprint

### 1. First `/open-api/v1/account` Route Draft

Recommended first endpoint shape:

- `GET /open-api/v1/account/me`

Purpose:

- return the account/profile view for the current partner-authenticated principal
- establish the external auth, DTO, and route boundary without introducing broad list/search complexity first

Optional near-following endpoint once the first one is stable:

- `GET /open-api/v1/accounts/:id`

Use this only if the partner model truly requires cross-account reads. If partner access is mostly self-scoped, start with `/me` only.

Recommended route file:

- `api/routes/open_api_account.go`

Recommended route responsibilities:

- parse partner-facing parameters only
- read authenticated partner/app/account context from middleware
- call `models` open-api read function(s)
- map internal read result into external DTO
- return partner-safe error payloads

### 2. API Key Middleware Design

Recommended new file:

- `api/middleware/open_api_key.go`

Recommended header contract:

- `Authorization: Bearer <api-key>` or
- `X-API-Key: <api-key>`

Recommendation:

- prefer `Authorization: Bearer <api-key>` as the primary contract
- optionally support `X-API-Key` during early development only if clearly documented

Suggested middleware responsibilities:

1. Read the API key from request headers
2. Reject missing credentials with partner-safe `401`
3. Call `models` lookup logic in `open_api_keys.go`
4. Validate:
   - key exists
   - key is active
   - key is not revoked
   - key is allowed for open-api usage
5. Store partner/app identity in context
6. Optionally store a limited auth context struct instead of the full DB row

Suggested context keys:

- `OpenAPIConsumerContextKey`
- `OpenAPIKeyContextKey`

Suggested context struct:

```go
type OpenAPIConsumerContext struct {
    KeyID       string
    PartnerID   string
    AccountID   string
    Scopes      []string
    Environment string
}
```

Why this helps:

- route handlers do not need to understand raw key persistence schema
- future rotation / scope evolution stays behind middleware + model lookup

### 3. `models` Reorganization Blueprint

Keep `api/models`, but group by use-case and consumer boundary.

Recommended near-term shape:

- `api/models/user.go`
  - internal user CRUD and internal user query flows
- `api/models/auth.go`
  - session + password reset flows
- `api/models/order.go`
  - internal order workflows
- `api/models/open_api_keys.go`
  - API key lookup
  - active/revoked checks
  - partner/app/account identity loading
- `api/models/open_api_account_read.go`
  - partner-facing account/profile read queries
  - only the data assembly needed for external account/profile responses

Guideline:

- internal writes stay in existing internal model files
- external read composition goes into `open_api_*` files
- avoid mixing partner auth concerns into `auth.go`, because current `auth.go` is session-oriented

### 4. DTO Blueprint for the First Account/Profile Slice

Route-level DTOs should be defined in `api/routes/open_api_account.go`.

Suggested first DTO:

```go
type OpenAPIAccountResponse struct {
    ID          string `json:"id"`
    ExternalRef string `json:"external_ref,omitempty"`
    Name        string `json:"name"`
    Email       string `json:"email,omitempty"`
    Status      string `json:"status"`
    CreatedAt   string `json:"created_at,omitempty"`
}
```

This is only a blueprint, not a required final field set.

Field strategy:

- allow-list only
- fields must be justified as partner-facing contract
- avoid reusing internal `models.User` directly even if field names happen to match

Recommended response shaping flow:

1. `models.open_api_account_read` returns an internal read model or query result struct
2. route maps that result into `OpenAPIAccountResponse`
3. route returns the DTO only

### 5. Partner-Safe Error Contract

Do not reuse browser/user-oriented auth error payloads blindly.

Suggested error shape:

```json
{
  "error": {
    "code": "unauthorized",
    "message": "invalid api key"
  }
}
```

Recommended starter codes:

- `unauthorized`
- `forbidden`
- `not_found`
- `invalid_request`
- `internal_error`

### 6. Routing Registration Blueprint

Recommended future registration shape in `index.go`:

```go
openAPI := router.Group("/open-api/v1")
openAPI.Use(openAPIMiddleware.RequireAPIKey())
{
    openAPI.GET("/account/me", routes.GetOpenAPIAccountMe)
}
```

Why this is preferred:

- hard route namespace boundary
- hard auth boundary
- easy version prefix from day one

### 7. Suggested DB / Persistence Readiness

Not all of this must be implemented immediately, but the architecture should expect:

- API key storage table
- partner/app identity table or mapping
- optional account-to-partner linkage
- future scopes / revocation / expiry fields

Suggested starter persistence concepts:

- `open_api_keys`
- `open_api_partners` or equivalent owner entity

Even if the first implementation is minimal, name these concepts explicitly now to avoid reusing session tables or user tables in awkward ways.

### 8. Recommended Phased Rollout

#### Phase A: Boundary Setup

- add `/open-api/v1` route group
- add API key middleware
- add starter persistence model for keys

#### Phase B: First Read Endpoint

- implement `GET /open-api/v1/account/me`
- add route DTO
- add `open_api_account_read.go`

#### Phase C: Contract Hardening

- error contract normalization
- field allow-list review
- partner scope placeholder or minimal scope validation

#### Phase D: Expansion

- add more account/profile endpoints
- add partner management and key rotation
- only then consider order/profile cross-resource expansion

## Concrete Recommendation

If the team wants the lowest-risk path that still meaningfully upgrades architecture, the first practical implementation should be:

1. add `api/middleware/open_api_key.go`
2. add `api/models/open_api_keys.go`
3. add `api/models/open_api_account_read.go`
4. add `api/routes/open_api_account.go`
5. register `GET /open-api/v1/account/me`
6. return a dedicated `OpenAPIAccountResponse`

This sequence gives you a real external API boundary without prematurely widening scope.

## `/open-api/v1/account/me` Contract Draft

### 1. Endpoint

- Method: `GET`
- Path: `/open-api/v1/account/me`

### 2. Purpose

- Return the profile/account view for the account represented by the authenticated API key context.
- Avoid arbitrary account lookup in the first version.
- Prove the route boundary, auth boundary, DTO boundary, and consumer boundary in one small slice.

### 3. Authentication Contract

Primary request header:

```http
Authorization: Bearer <api-key>
```

Optional early-development fallback if explicitly supported:

```http
X-API-Key: <api-key>
```

Recommendation:

- Document `Authorization: Bearer <api-key>` as the official contract
- Treat `X-API-Key` as optional compatibility only, not the long-term primary form

### 4. Request Contract

- No request body
- No query parameters in v1
- Account identity comes only from API key context, not from caller-supplied ID

### 5. Success Response Contract

Suggested success status:

- `200 OK`

Suggested response shape:

```json
{
  "data": {
    "id": "acct_123",
    "external_ref": "partner-user-42",
    "name": "Example Account",
    "email": "owner@example.com",
    "status": "active",
    "created_at": "2026-06-04 10:00:00"
  }
}
```

Suggested route DTO:

```go
type OpenAPIAccountResponse struct {
    ID          string `json:"id"`
    ExternalRef string `json:"external_ref,omitempty"`
    Name        string `json:"name"`
    Email       string `json:"email,omitempty"`
    Status      string `json:"status"`
    CreatedAt   string `json:"created_at,omitempty"`
}

type OpenAPIAccountEnvelope struct {
    Data OpenAPIAccountResponse `json:"data"`
}
```

### 6. Field Allow-List

Allowed starter fields:

- `id`
  - stable external-facing account identifier
- `external_ref`
  - partner-visible reference if the business model needs one
- `name`
  - human-readable display name
- `email`
  - only if partner visibility is acceptable for the contract
- `status`
  - normalized external-facing status string such as `active`, `inactive`, `suspended`
- `created_at`
  - optional lifecycle metadata

### 7. Explicitly Forbidden Fields

Do not expose these from internal user/account state:

- `password_hash`
- session IDs
- password reset state
- internal auth flags that are not part of public contract
- internal moderation/admin-only notes
- raw `email_verified` / `is_active` integer storage flags unless mapped into stable external enum semantics
- any future internal audit/security fields

### 8. Internal-to-External Mapping Rule

Do not serialize `models.User` directly.

Recommended flow:

1. middleware resolves `OpenAPIConsumerContext`
2. route calls `models.GetOpenAPIAccountByConsumer(...)` or equivalent
3. model returns an internal read struct
4. route maps that read struct into `OpenAPIAccountResponse`
5. route returns envelope DTO

### 9. Suggested `models` Read Shape

Recommended internal read struct in `api/models/open_api_account_read.go`:

```go
type OpenAPIAccountReadModel struct {
    AccountID    string `db:"id"`
    ExternalRef  string `db:"external_ref"`
    Name         string `db:"name"`
    Email        string `db:"email"`
    IsActive     int    `db:"is_active"`
    CreatedAt    string `db:"created_at"`
}
```

Then map to external DTO status:

- `is_active = 1` -> `active`
- `is_active = 0` -> `inactive`

This preserves the separation between persistence shape and public contract shape.

### 10. Error Contract

Suggested error envelope:

```json
{
  "error": {
    "code": "unauthorized",
    "message": "invalid api key"
  }
}
```

Suggested status/code matrix:

| HTTP Status | `error.code` | Meaning |
|------------|--------------|---------|
| `401` | `unauthorized` | missing or invalid API key |
| `403` | `forbidden` | valid key but not allowed for this account/scope |
| `404` | `not_found` | authenticated consumer context has no visible account |
| `500` | `internal_error` | unexpected backend failure |

### 11. Handler Responsibilities

Recommended handler in `api/routes/open_api_account.go`:

- no body binding
- no account ID parsing from caller input
- read consumer context from middleware
- call single-purpose model read function
- map result to DTO
- return normalized partner-safe error envelope

### 12. Middleware Responsibilities for This Endpoint

For `/account/me`, middleware must provide enough context to avoid another lookup just to know who the caller is:

- `PartnerID`
- `AccountID`
- `KeyID`
- optional `Scopes`

If `AccountID` is absent in auth context, this endpoint should likely fail with `403` or `404` depending on business semantics, not silently infer too much.

### 13. Versioning Rule

Even if the initial payload is small, v1 should assume fields are contract-stable once published.

Rules:

- additive fields are OK in v1 if documented
- field renames/removals require a new version
- internal schema changes must be absorbed by DTO mapping, not leaked to partners

### 14. Concrete Recommendation for the First Implementation

For the first real implementation, prefer this narrow shape:

- only `GET /open-api/v1/account/me`
- only API-key-derived identity
- only allow-listed DTO fields
- only read path
- no list/search endpoint yet

This keeps the first external contract small, privacy-aware, and architecture-validating.

## Code Skeleton Blueprint

### 1. `api/middleware/open_api_key.go`

Recommended purpose:

- machine-to-machine authentication for `/open-api`
- no coupling to browser session cookie auth

Suggested skeleton:

```go
package middleware

import (
    "net/http"
    "strings"

    "github.com/labstack/echo/v4"
    "github.com/tfnick/go-svelte-starter/api/models"
)

const OpenAPIConsumerContextKey = "open_api_consumer"

type OpenAPIConsumerContext struct {
    KeyID       string
    PartnerID   string
    AccountID   string
    Scopes      []string
    Environment string
}

func RequireOpenAPIKey() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            apiKey := readAPIKey(c)
            if apiKey == "" {
                return c.JSON(http.StatusUnauthorized, openAPIError("unauthorized", "missing api key"))
            }

            consumer, err := models.ResolveOpenAPIConsumer(apiKey)
            if err != nil {
                return c.JSON(http.StatusUnauthorized, openAPIError("unauthorized", "invalid api key"))
            }

            c.Set(OpenAPIConsumerContextKey, consumer)
            return next(c)
        }
    }
}

func GetOpenAPIConsumer(c echo.Context) *OpenAPIConsumerContext {
    consumer, ok := c.Get(OpenAPIConsumerContextKey).(*OpenAPIConsumerContext)
    if !ok {
        return nil
    }
    return consumer
}

func readAPIKey(c echo.Context) string {
    auth := c.Request().Header.Get("Authorization")
    if strings.HasPrefix(auth, "Bearer ") {
        return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
    }
    return c.Request().Header.Get("X-API-Key")
}
```

Suggested helper for partner-safe errors:

```go
func openAPIError(code, message string) map[string]interface{} {
    return map[string]interface{}{
        "error": map[string]string{
            "code":    code,
            "message": message,
        },
    }
}
```

### 2. `api/models/open_api_keys.go`

Recommended purpose:

- API key lookup and auth context resolution
- keep partner-auth persistence logic out of `auth.go`

Suggested starter structs:

```go
package models

type OpenAPIKey struct {
    ID          string `db:"id"`
    PartnerID   string `db:"partner_id"`
    AccountID   string `db:"account_id"`
    TokenHash   string `db:"token_hash"`
    Status      string `db:"status"`
    Scopes      string `db:"scopes"`
    Environment string `db:"environment"`
}
```

Suggested functions:

```go
func ResolveOpenAPIConsumer(rawKey string) (*middleware.OpenAPIConsumerContext, error)
func GetOpenAPIKeyByTokenHash(tokenHash string) (*OpenAPIKey, error)
func ValidateOpenAPIKey(key *OpenAPIKey) error
```

Implementation note:

- avoid raw token storage
- hash the presented key before lookup, similar in spirit to reset token handling

### 3. `api/models/open_api_account_read.go`

Recommended purpose:

- partner-facing account/profile read composition
- no internal write logic here

Suggested read model:

```go
package models

type OpenAPIAccountReadModel struct {
    AccountID   string `db:"id"`
    ExternalRef string `db:"external_ref"`
    Name        string `db:"name"`
    Email       string `db:"email"`
    IsActive    int    `db:"is_active"`
    CreatedAt   string `db:"created_at"`
}
```

Suggested functions:

```go
func GetOpenAPIAccountByConsumerAccountID(accountID string) (*OpenAPIAccountReadModel, error)
func GetOpenAPIAccountStatus(isActive int) string
```

Behavior rule:

- if this file must join multiple tables later, keep that join logic here instead of leaking it into the route

### 4. `api/routes/open_api_account.go`

Recommended purpose:

- external route-only DTOs
- external error envelope usage
- no direct use of internal `models.User` in response

Suggested DTOs:

```go
package routes

type OpenAPIAccountResponse struct {
    ID          string `json:"id"`
    ExternalRef string `json:"external_ref,omitempty"`
    Name        string `json:"name"`
    Email       string `json:"email,omitempty"`
    Status      string `json:"status"`
    CreatedAt   string `json:"created_at,omitempty"`
}

type OpenAPIAccountEnvelope struct {
    Data OpenAPIAccountResponse `json:"data"`
}
```

Suggested handler:

```go
func GetOpenAPIAccountMe(c echo.Context) error {
    consumer := middleware.GetOpenAPIConsumer(c)
    if consumer == nil {
        return c.JSON(http.StatusUnauthorized, openAPIError("unauthorized", "missing consumer context"))
    }

    account, err := models.GetOpenAPIAccountByConsumerAccountID(consumer.AccountID)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, openAPIError("internal_error", "failed to load account"))
    }
    if account == nil {
        return c.JSON(http.StatusNotFound, openAPIError("not_found", "account not found"))
    }

    resp := OpenAPIAccountEnvelope{
        Data: OpenAPIAccountResponse{
            ID:          account.AccountID,
            ExternalRef: account.ExternalRef,
            Name:        account.Name,
            Email:       account.Email,
            Status:      models.GetOpenAPIAccountStatus(account.IsActive),
            CreatedAt:   account.CreatedAt,
        },
    }

    return c.JSON(http.StatusOK, resp)
}
```

### 5. `index.go` Registration Skeleton

Suggested future registration:

```go
import openAPIMiddleware "github.com/tfnick/go-svelte-starter/api/middleware"
import routes "github.com/tfnick/go-svelte-starter/api/routes"

openAPI := router.Group("/open-api/v1")
openAPI.Use(openAPIMiddleware.RequireOpenAPIKey())
{
    openAPI.GET("/account/me", routes.GetOpenAPIAccountMe)
}
```

Key rule:

- do not place `/open-api` handlers under the existing `/api` group

### 6. Error Helper Placement Recommendation

For the first implementation, keep open-api-specific error shaping local to `api/routes/open_api_account.go` or a small open-api routes helper file, for example:

- `api/routes/open_api_errors.go`

Suggested helper:

```go
func openAPIError(code, message string) map[string]interface{} {
    return map[string]interface{}{
        "error": map[string]string{
            "code":    code,
            "message": message,
        },
    }
}
```

This prevents route files from drifting back into internal/browser response semantics.

### 7. Suggested First DB Migration Concepts

If implementation proceeds, the first migration concepts likely needed are:

- `open_api_partners`
- `open_api_keys`

Suggested field categories:

- partner identity
- account linkage
- token hash
- status
- scopes
- environment
- created_at / revoked_at / expires_at

No need to finalize the full schema in this diagnosis task, but implementation should not reuse `sessions` or `users.password_hash` patterns directly without separate tables.

### 8. Minimal End-to-End Flow

The first end-to-end request should look like:

1. request hits `/open-api/v1/account/me`
2. `RequireOpenAPIKey()` extracts and validates API key
3. middleware stores `OpenAPIConsumerContext`
4. route reads context
5. route calls `models.GetOpenAPIAccountByConsumerAccountID(...)`
6. route maps result to `OpenAPIAccountEnvelope`
7. route returns `200` JSON response

This is the minimum slice that proves the target architecture.

## Immediate vs Deferred Work

### Do Now

- split route surface
- add API key middleware
- create external DTOs
- split models by internal/open-api responsibility
- adopt `/open-api/v1/...` pathing
- design the first account/profile read contract with strict field allow-listing

### Defer

- service extraction
- multi-deployable split
- complex partner admin console/workflows
- advanced rate limiting / quotas / per-partner policy engines
- broader version negotiation beyond path versioning

## Out of Scope (Explicit)

- Immediate microservice decomposition
- Independent deployables
- Introducing a generic standalone service layer
- Full partner platform design beyond the first practical boundary cuts

## Technical Notes

- Route registration is centralized in `index.go`
- Current auth middleware is in `api/middleware/auth.go`
- Current route handlers return internal data models directly in several places
- Current structure already supports file-level reorganization without package explosion
