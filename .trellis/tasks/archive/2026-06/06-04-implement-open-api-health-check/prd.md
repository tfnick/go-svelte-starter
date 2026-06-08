# Implement open-api health check endpoint

## Goal

Add a dedicated health check endpoint for the external open-api surface so integrators and infrastructure can verify that the `/open-api/v1` interface is reachable and functioning without depending on the authenticated `/account/me` flow.

## What I Already Know

- The project already has a distinct open-api route surface:
  - `/open-api/v1/...`
- The current open-api implementation includes:
  - API key middleware in `api/middleware/open_api_key.go`
  - one authenticated endpoint: `GET /open-api/v1/account/me`
- Open-api route handlers live in dedicated files under `api/routes/`.
- Current route registration is centralized in `index.go`.
- The open-api architecture intentionally keeps:
  - one deployable
  - no separate service layer
  - route -> models -> db structure

## Assumptions (Temporary)

- The health check endpoint should be narrow and stable.
- It should not depend on user session auth.
- It should likely be unauthenticated so external monitoring and partner integration checks can call it without API keys.

## Open Questions

- None

## Requirements

- Add one open-api health check endpoint under `/open-api/v1`
- Keep it separate from internal `/api` routes
- Do not route it through the authenticated account/me flow
- Do not introduce a service layer
- Use a dedicated open-api route file or extend the open-api route area cleanly
- Return a simple, stable JSON response
- Make the endpoint appropriate for uptime/integration probing

## Acceptance Criteria

- [x] A health check endpoint exists under `/open-api/v1`
- [x] The endpoint is reachable without session auth
- [x] The response is simple and stable enough for monitoring/integration checks
- [x] The endpoint does not expose internal account data
- [x] `go test ./...` passes after implementation

## Definition of Done

- Route is registered in the correct open-api namespace
- Response contract is documented in code/task notes
- No unintended coupling to account/authenticated partner flows
- Implementation follows current backend organization conventions

## Technical Approach

- Add a route such as `GET /open-api/v1/health`
- Prefer placing it in a dedicated route file like:
  - `api/routes/open_api_health.go`
- Keep the handler self-contained:
  - no DB query unless there is a deliberate requirement to validate DB connectivity
  - no API key requirement unless explicitly desired later
- Return a compact JSON payload, for example:
  - status
  - service
  - version placeholder or API surface indicator if useful

## Decision (ADR-lite)

**Context**: The external API surface already exists, but the only current endpoint depends on API-key-authenticated account context. A separate health endpoint provides a lower-friction operational contract for integration checks and monitoring.

**Decision**: Add a lightweight open-api health endpoint under `/open-api/v1`, keep it outside the API-key middleware path, and return a stable JSON response without coupling it to account/business reads.

**Consequences**:

- Pros:
  - simple monitoring target
  - clear separation between operational liveness and authenticated business APIs
  - no extra model/service complexity
- Cons:
  - it validates route-level availability more than deep dependency readiness unless we later add dependency checks

## Recommended Contract

- Method: `GET`
- Path: `/open-api/v1/health`

Suggested response:

```json
{
  "status": "ok",
  "surface": "open-api",
  "version": "v1"
}
```

Suggested HTTP status:

- `200 OK`

## Out of Scope

- Full readiness/deep dependency probing
- Partner/account validation
- Requiring API keys for health checks
- Internal `/api` health check redesign

## Technical Notes

- Current open-api route registration is in `index.go`
- Current authenticated open-api handler is `api/routes/open_api_account.go`
- This task should preserve the current open-api boundary style without widening business scope
