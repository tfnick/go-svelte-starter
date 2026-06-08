# Introduce Zerolog and Standardize Logging

## Goal

Introduce `zerolog` into the project and replace the current ad-hoc logging approach with a consistent logging pattern that is easier to extend, safer for production use, and aligned across startup, database, and request-related flows.

## What I Already Know

- The project currently does not use a structured logging library.
- Current logging is spread across:
  - `fmt.Println()` / `fmt.Printf()`
  - `router.Logger.Fatal(...)`
- Existing runtime log locations include:
  - startup and server boot logs in `index.go`
  - migration / database lifecycle logs in `api/db/db.go`
  - development-only password reset link printing in `api/routes/auth.go`
- The current spec file `.trellis/spec/backend/logging-guidelines.md` still describes the old non-structured approach.

## Assumptions (Temporary)

- Zerolog should become the default application logger for new and migrated logs.
- We should preserve current developer-visible behavior where it still matters, but express it through a standardized logger API.
- This task likely affects runtime code plus backend logging specs.

## Open Questions

- None

## Requirements (Evolving)

- Add Zerolog to the project.
- Standardize existing logging call sites behind a clear project convention.
- Scope is limited to application/internal logs only.
- Do not include Echo HTTP access/request logging in this task.
- Use JSON log output in both development and production.
- Keep development-only logs distinguishable from production-safe logs.
- Avoid logging sensitive authentication data in production.

## Acceptance Criteria (Evolving)

- [ ] Zerolog is integrated into the project.
- [ ] Existing core application logs use the new logging convention.
- [ ] Development and production both emit JSON logs.
- [ ] Logging behavior is documented in Trellis specs.

## Technical Approach

- Introduce a project logging package or shared logger entrypoint based on Zerolog.
- Replace current `fmt.Println` / `fmt.Printf` application logs and startup `echo.Logger.Fatal(...)` usage with the shared Zerolog pattern where appropriate.
- Standardize structured fields for internal logs such as component, operation, and error context.
- Keep dev-only messages as logs with explicit fields or levels instead of switching to a separate pretty-print format.

## Decision (ADR-lite)

**Context**: The project needs a single standardized logging approach after moving away from ad-hoc `fmt` logging. A key open decision was whether development should use a human-friendly console formatter or whether all environments should emit the same structured format.

**Decision**: Use Zerolog JSON output in both development and production.

**Consequences**:

- Pros:
  - one format everywhere
  - simpler spec and implementation
  - easier future integration with log collection or parsing
- Cons:
  - local development logs are less human-friendly than pretty console output
  - developers will rely more on structured fields than visual formatting

## Definition of Done (Team Quality Bar)

- Tests added/updated if behavior requires them
- Lint / typecheck / CI green
- Docs/notes updated if behavior changes
- Rollout/rollback considered if risky

## Out of Scope (Explicit)

- Echo HTTP access/request logging
- Full observability platform work
- Remote log shipping
- Metrics/tracing unless required by the chosen logging design

## Technical Notes

- Current startup logging is in `index.go`
- Current DB/migration logging is in `api/db/db.go`
- Current logging spec is `.trellis/spec/backend/logging-guidelines.md`
