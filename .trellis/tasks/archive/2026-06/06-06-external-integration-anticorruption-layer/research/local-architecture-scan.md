# Local Architecture Scan

## Current Repo Shape

The backend currently uses a lightweight layered architecture:

```text
routes -> usecase -> models -> db
```

`api/framework` holds business-agnostic capabilities. Business logic belongs in `api/usecase`, persistence in `api/models`, and HTTP adapter details in `api/routes`.

Relevant existing framework capabilities:

* `api/framework/http/*`: auth, middleware, context, response helpers.
* `api/framework/events`: durable DDD event facade.
* `api/framework/queue`: goqite-backed queue implementation. Raw goqite is restricted to this package.
* `api/framework/logging`: zerolog component logging and sensitive-field constraints.
* `api/framework/archguard`: import-boundary and architecture guard tests.

Existing external-facing surface:

* `/open-api/v1/*` is the public API surface for external consumers.
* Open API authentication is API-key based through middleware.
* Open API DTOs are route-local and separated from internal `/api/*` DTOs.

Existing eventual consistency foundation:

* Domain events are queue-first and durable.
* Fan-out is modeled through `domain_event_deliveries`.
* Business subscribers are expected to be idempotent.

## Design Constraints Inferred From Specs

* Do not put business-specific integration logic into `api/framework`.
* Do not let routes call models/db directly.
* Do not let models depend on events, routes, usecase, or HTTP framework packages.
* Do not expose raw provider DTOs to business usecases or frontend/internal DTOs.
* Do not log raw secrets, tokens, request bodies, or response bodies.
* If raw queue access is needed, it must stay behind `api/framework/queue`.
* New cross-layer patterns should be captured in `.trellis/spec/backend/directory-structure.md` and likely enforced through `api/framework/archguard`.

## Architectural Implications

External integrations should probably be split into three concern groups:

1. Framework primitives:
   business-agnostic HTTP client, auth strategies, signing helpers, callback verification helpers, retry/backoff policy types, redaction and logging helpers.

2. Provider anti-corruption adapters:
   scenario/channel-specific code that knows provider schemas, auth quirks, endpoint paths, callback payloads, and provider error codes.

3. Business ports and orchestration:
   usecase-owned interfaces and application workflows that speak business language, not provider language.

The likely direction is a ports-and-adapters style boundary:

```text
business usecase -> usecase-owned port interface -> provider adapter -> framework integration primitives
```

Callbacks should come through HTTP routes, verify provider-specific authenticity at the adapter edge, map payloads into stable internal commands/events, and then enter usecases/events.

## Open Design Choice

The main unresolved choice is how much to implement in this task:

* design-only ADR and directory plan,
* design plus framework skeleton and archguard rules,
* design plus one vertical provider example.
