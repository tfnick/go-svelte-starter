# Event management

## Goal

Add an Event management UI so operators can inspect domain events and the delivery records for each event. The feature should expose paginated event queries, add an Event entry in the left navigation, and allow viewing `domain_event_deliveries` rows associated with a selected event.

## What I already know

* The user wants a new left-side menu item named Event.
* The Event page must support paginated event queries.
* Each event row must allow viewing that event's delivery records from `domain_event_deliveries`.
* The project already has a standard pagination contract from the order list work.
* Domain event storage already exists in `api/models/domain_event.go` and `api/db/migrations/app/009_add_domain_event_delivery.sql`.
* `domain_events` columns: `id`, `topic`, `aggregate_type`, `aggregate_id`, `payload_json`, `metadata_json`, `occurred_at`, `created_at`.
* `domain_event_deliveries` columns: `id`, `event_id`, `subscriber`, `message_id`, `status`, `attempts`, `last_error`, `created_at`, `updated_at`.
* Authenticated backend routes are registered in `index.go` under the protected `/api` group.
* The logged-in app shell is driven by `frontend/src/router.js`, `frontend/src/App.svelte`, and `frontend/src/components/AppSidebar.svelte`.
* `frontend/src/pages/Scheduler.svelte` is the closest existing management-page reference for read-only history tables and durable queue inspection.

## Assumptions (temporary)

* This is an internal/admin-style management view, not a customer-facing workflow.
* The first MVP can be read-only: list events and inspect each event's delivery records.
* Existing authentication/navigation conventions should be reused.
* Event delivery records should be fetched on demand when the user opens an event detail/record view.
* Delivery records for one event do not need their own pagination in the MVP because there is one row per subscriber.

## Open Questions

* None.

## Requirements (evolving)

* Add a left navigation entry for Event.
* Add an Event management page.
* Backend exposes a standard paginated query endpoint for domain events.
* Backend exposes a way to fetch `domain_event_deliveries` records for a selected event ID.
* Frontend consumes the backend APIs and renders responsive pagination consistent with the order list.
* Event management is read-only in this task.
* MVP includes pagination only; no event list filters in this task.
* Event list ordering should be stable, likely `ORDER BY created_at DESC, id DESC`.
* Delivery records should be ordered by `created_at ASC` or `subscriber ASC` so fan-out handling is easy to scan.

## Acceptance Criteria (evolving)

* [x] Event menu item appears in the left navigation.
* [x] Event page lists events with pagination using the standard backend pagination response shape.
* [x] Pagination remains usable on narrow/mobile widths.
* [x] Selecting an event shows that event's `domain_event_deliveries` records.
* [x] Event rows expose enough context to identify topic, aggregate, occurrence time, and payload/metadata preview or detail.
* [x] Delivery rows expose subscriber, status, attempts, message id, last error, created/updated timestamps.
* [x] Backend validates pagination parameters consistently with the shared pagination contract.
* [x] Tests cover backend pagination/query behavior and frontend API/UI behavior where practical.

## Definition of Done (team quality bar)

* Tests added/updated where appropriate.
* Go tests pass.
* Frontend tests/build pass.
* Docs/spec notes updated if a reusable event-management or pagination convention emerges.
* Runtime database files are not committed.

## Out of Scope (explicit)

* Mutating event state, replaying events, or manually triggering retries unless explicitly included after scope confirmation.
* Changing the domain event publishing/consumption architecture.
* Replacing goqite or changing queue internals.
* Editing payload or metadata.
* Building a generalized audit-log system.
* Event list filters such as `topic`, `aggregate_id`, `status`, or time range.

## Technical Approach

Build the feature as a read-only internal management surface:

* Backend model layer adds count/list queries for `domain_events` and a list query for `domain_event_deliveries` by `event_id`.
* Backend usecase layer normalizes pagination via `fwusecase.PageQuery`, returns `{items, pagination}`, and exposes delivery lookup for a selected event.
* Backend route layer parses pagination via `fwrequest.PageQuery(c)` and returns route-local DTOs through `httpresponse.OK`.
* Frontend adds `/events` to the logged-in app shell, calls API helpers from `frontend/src/api.js`, renders a responsive paginated event table, and lazy-loads delivery records when an event is selected.

## Decision (ADR-lite)

**Context**: Event management could start with filters/replay controls, but the immediate need is to inspect events and per-event delivery records.

**Decision**: Implement only pagination and per-event `domain_event_deliveries` viewing in this task.

**Consequences**: The first version is smaller and safer. Filtering, replay, retry, and mutation workflows remain future enhancements.

## Implementation Plan

* Backend: add domain event query model/usecase/route and register protected routes.
* Frontend API/router: add Event helpers, `/events` menu route, and route tests.
* Frontend page: add responsive Event page with pagination and delivery-record panel.
* Verification: add/update backend and frontend tests, then run Go tests, frontend tests, and frontend build.

## Technical Notes

* Reuse pagination helpers under `api/framework/usecase` and `api/framework/http/request`.
* Likely backend additions:
  * `api/models/domain_event.go`: count/list event queries and list deliveries by event ID.
  * `api/usecase/domain_event.go`: read-only event management usecases.
  * `api/routes/domain_event.go`: route DTOs and handlers.
  * `index.go`: protected routes.
* Likely backend API:
  * `GET /api/events?page=1&page_size=10`
  * `GET /api/events/:id/deliveries`
* Likely frontend additions:
  * `frontend/src/api.js`: `listEvents({ page, pageSize })` and `listEventDeliveries(eventId)`.
  * `frontend/src/router.js`: `/events` alias/menu/title.
  * `frontend/src/App.svelte`: mount new Event page.
  * `frontend/src/pages/Events.svelte`: responsive list + delivery record panel.
* Existing specs to follow:
  * `.trellis/spec/backend/api-contracts.md`
  * `.trellis/spec/backend/eventing-guidelines.md`
  * `.trellis/spec/frontend/svelte-vite-embed.md`
* Verification completed:
  * `go test ./...`
  * `cd frontend && npm test`
  * `cd frontend && npm run build`
