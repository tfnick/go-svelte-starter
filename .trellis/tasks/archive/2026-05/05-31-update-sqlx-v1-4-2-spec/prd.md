# PRD: Update Trellis Spec for sqlx v1.4.2 API Changes

## Summary

Update `.trellis/spec/backend/database-guidelines.md` to reflect the project's current `github.com/tfnick/sqlx v1.4.2` usage after the upgrade and code migration work.

The current spec still documents several `v1.4.1` assumptions that are now stale, especially around `Engine` method signatures and the project's `DBManager` usage. This task captures the new executable contract so future implementation work follows the real API and avoids reintroducing old patterns.

## Problem

The existing backend database spec is outdated in ways that can cause incorrect code generation or regressions:

1. It still describes the project as using `sqlx v1.4.1`.
2. It still shows `Engine` calls that pass `context.Background()`, while `v1.4.2` `Engine` methods in this project are now used as:
   - `eng.Exec(sql, arg)`
   - `eng.Get(&dest, sql, arg)`
   - `eng.Select(&dest, sql, arg)`
3. It contains guidance around `sqlx.NewManager()` that no longer matches the simplified local `DBManager` implementation.
4. Its "Common Mistakes" section still warns about forgetting `context.Background()` for `Engine`, which is now the opposite of the correct behavior in this repo.

## Goals

1. Update the backend database spec to describe the actual `sqlx v1.4.2` API used in this project.
2. Clarify the three supported access patterns in executable, copy-paste-safe examples:
   - `db.GetDB(name)` for direct `*sqlx.DB` access
   - `db.GetEngine(name)` for dynamic SQL via `LazyEngine`
   - `db.WithTransaction(name, fn)` for transaction boundaries
3. Document the project's current `DBManager` contract:
   - named DB registry
   - `Open`, `GetDB`, `GetEngine`, `WithTransaction`, `AutoMigrate`, `Reopen`, `Close`
4. Remove or replace outdated examples that teach pre-`v1.4.2` `Engine` calling style.

## Non-Goals

1. Changing runtime database behavior again.
2. Refactoring application code outside of spec updates.
3. Creating new migration files or changing schema.
4. Rewriting unrelated backend spec documents unless a small cross-reference update is clearly needed.

## Target Spec Files

Primary target:

- `.trellis/spec/backend/database-guidelines.md`

Optional secondary target if needed:

- `.trellis/spec/backend/index.md`

## Required Spec Updates

The updated spec should include the following concrete changes:

### 1. Version + Overview

- Change version references from `v1.4.1` to `v1.4.2`.
- Make the overview match the current codebase.

### 2. DB Access Patterns

Document and exemplify:

- `db.GetDB(name)` for simple SQL and `Rebind()`-based positional placeholder queries
- `db.GetEngine(name)` for named-parameter dynamic SQL
- `db.WithTransaction(name, fn)` for transaction handling

### 3. Engine Signature Contract

Explicitly state that in this repo's `sqlx v1.4.2` usage:

- `Engine` methods do **not** take an explicit `context.Context`
- Correct examples:
  - `eng.Exec(sql, params)`
  - `eng.Get(&dest, sql, params)`
  - `eng.Select(&dest, sql, params)`

Include at least one Wrong vs Correct pair.

### 4. Project DBManager Contract

Document the local wrapper behavior:

- `Open(name, driver, path)`
- `GetDB(name)`
- `GetEngine(name)`
- `WithTransaction(name, fn)`
- `AutoMigrate(name)`
- `Reopen(name)`
- `Close()`

Clarify that the project keeps its own `DBManager` because it adds migration and reopen behavior beyond raw sqlx handles.

### 5. Error / Validation Notes

Capture at least these behavioral constraints:

- unknown DB name returns an error
- unsupported driver returns an error
- `Reopen(name)` reopens an already-registered DB only
- dynamic SQL through `Engine` should use named params and `#[ ... ]`
- direct DB queries using `?` must still use `Rebind()`

### 6. Common Mistakes

Update the mistakes list so it no longer recommends passing `context.Background()` into `Engine` calls.

Add or retain concrete warnings for:

- using `?` placeholders with `Engine`
- forgetting `Rebind()` with direct `DB` SQL
- using `db.GetDB()` / `db.GetEngine()` inside a transaction callback instead of `tx`

## Acceptance Criteria

1. `database-guidelines.md` references `sqlx v1.4.2`, not `v1.4.1`.
2. All `Engine` examples in the updated spec match the current code usage and omit explicit `context.Background()`.
3. The spec contains executable examples for `GetDB`, `GetEngine`, and `WithTransaction`.
4. The spec includes code-spec depth sections appropriate for this infra/API contract update:
   - Scope / Trigger
   - Signatures
   - Contracts
   - Validation & Error Matrix
   - Good/Base/Bad Cases
   - Tests Required
   - Wrong vs Correct
5. The resulting spec is aligned with the actual code in:
   - `api/db/db.go`
   - `api/models/user.go`
   - `api/models/auth.go`
   - `api/models/order.go`

## Validation Plan

1. Read the updated spec and compare it against the current code.
2. Verify no example still shows `eng.Exec(context.Background(), ...)` or similar legacy calls.
3. Verify the `DBManager` description matches the simplified implementation in `api/db/db.go`.
4. Run a quick text search for stale `v1.4.1` references in relevant spec files after updating.

## Notes

- This is a documentation/spec capture task, but it is still high-value because incorrect spec guidance here would directly cause future bad code edits.
- The most important lesson to preserve is the `Engine` API shape change in `v1.4.2` and the project's corresponding wrapper usage.
