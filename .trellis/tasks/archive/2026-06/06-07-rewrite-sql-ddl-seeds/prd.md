# Rewrite SQL DDL and Seeds

## Goal

Because the project has not gone live, replace the historical incremental SQL migrations with a clean baseline DDL and seed set. Seed rows that provide application IDs should use deterministic UUIDv7-style values so the seeded data follows the project's ordered-ID convention.

## Requirements

* Consolidate final `app` schema into a fresh baseline migration.
* Consolidate final `shared` schema into a fresh baseline migration.
* Delete obsolete historical migration SQL where the final schema can be represented directly.
* Keep the upstream `goqite` queue table isolated in `api/db/migrations/app/007_add_goqite.sql`.
* Do not bring retired `sessions`, `variables.purpose`, or `domain_event_executions` into the new baseline.
* Preserve all tables, columns, constraints, triggers, and indexes expected by current Go models, routes, usecases, and archguard tests.
* Rewrite seed SQL to use fixed UUIDv7-style IDs where seed rows have IDs.
* Update tests/docs/frontend API examples that intentionally refer to seeded user/product IDs.

## Acceptance Criteria

* [x] Fresh app DB migrates successfully from the new app SQL files.
* [x] Fresh shared DB migrates successfully from the new shared SQL files.
* [x] Seeded demo users/products/open API/dictionaries use UUIDv7-style IDs.
* [x] Go tests pass with the new migration layout.
* [x] Frontend API helper tests pass after seed/example ID updates.
* [x] Archguard still verifies `goqite` ownership boundaries.

## Definition of Done

* Run `go test ./...`.
* Run `cd frontend && npm test` because frontend API path tests are updated.
* Review whether `.trellis/spec/` needs any durable lesson from the SQL rewrite.
* Commit the work as a coherent task commit before archiving.

## Technical Approach

* Replace `api/db/migrations/app/001_init.sql` through the historical app increments with:
  * `001_schema.sql` for all project-owned app tables.
  * `002_seed.sql` for demo users, demo Open API key, and dictionaries.
  * existing `007_add_goqite.sql` for the component-owned queue table only.
* Keep `api/db/migrations/shared/001_schema.sql` and `002_seed.sql` as placeholders while shared has no project-owned tables; products live in the app baseline.
* Use deterministic UUIDv7-style values such as `019ea0c1-0001-7000-8000-000000000001` for stable tests and docs.

## Decision (ADR-lite)

**Context**: The project is pre-launch, so preserving historical migration replay compatibility is less valuable than a readable baseline schema.

**Decision**: Collapse project-owned app/shared schema into clean baseline migrations while keeping `007_add_goqite.sql` isolated because existing specs and archguard treat it as a component-owned boundary.

**Consequences**: Fresh DB setup becomes simpler. Existing local developer DB files that have already recorded old migration names should be recreated if they need the new exact baseline.

## Out of Scope

* Building a live migration path for already-deployed databases.
* Changing Go model APIs or frontend behavior beyond seed/example ID references.
* Changing the upstream goqite table ID default, since it is component-owned and not seed data.

## Technical Notes

* Relevant specs: `.trellis/spec/backend/database-guidelines.md`, `.trellis/spec/backend/uuid-generation.md`, `.trellis/spec/backend/eventing-guidelines.md`, `.trellis/spec/frontend/svelte-vite-embed.md`.
* `api/framework/archguard/layer_boundary_test.go` requires `goqite` creation to stay in a file ending with `007_add_goqite.sql`.
* `integration_credentials` must keep legacy storage columns because current model writes `ciphertext`, `key_version`, `masked_value`, and `value_text`.
