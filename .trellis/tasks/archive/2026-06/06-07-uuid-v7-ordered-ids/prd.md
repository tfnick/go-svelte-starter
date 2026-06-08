# Use UUID v7 for Project UUID Generation

## Goal

Replace project UUID generation with `uuid.NewV7()` so newly generated UUIDs are time-ordered, consistent across runtime paths, and friendlier to database index locality and operational tracing.

## Requirements

* Replace every current `uuid.New().String()` and `uuid.NewString()` call with UUID v7 generation.
* Preserve existing public APIs and persisted ID string format.
* Apply the convention to both persistent IDs and non-persistent runtime IDs, such as request/client/task/execution identifiers.
* Do not change table schemas, migrations, or query semantics.
* Capture UUID v7 as a project-level backend spec so future UUID generation uses the same convention.

## Acceptance Criteria

* [ ] `rg "uuid\\.New\\(|uuid\\.NewString\\(" api` returns no production call sites.
* [ ] New UUID strings are generated through `uuid.Must(uuid.NewV7()).String()`.
* [ ] Backend spec documents the project-level UUID v7 convention.
* [ ] Existing tests pass with `go test ./...`.

## Definition of Done

* Backend specs consulted before code changes.
* Code is formatted with `gofmt`.
* Quality checks are run and any failures caused by this task are fixed.

## Out of Scope

* Backfilling or rewriting existing UUIDs.
* Changing database schemas or index definitions.
* Introducing a new UUID library.

## Technical Notes

* `github.com/google/uuid v1.6.0` is already in `go.mod`.
* Initial search found `uuid.New()` and `uuid.NewString()` call sites across `api/models`, `api/usecase`, `api/framework`, and `api/routes`.
