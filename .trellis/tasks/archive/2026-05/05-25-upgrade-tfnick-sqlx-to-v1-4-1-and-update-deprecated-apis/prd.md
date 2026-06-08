# Upgrade tfnick/sqlx to v1.4.1

## Goal

Upgrade sqlx dependency and adopt new v1.4.1 APIs where beneficial.

## Research Findings

| New API | Old way | New way |
|---------|---------|---------|
| `DB.LazyEngine()` | `sqlx.NewEngine(db)` | `db.LazyEngine()` — lazy init, thread-safe singleton |
| `DB.WithTransaction(fn)` | `db.WithTransaction("app", fn)` via DBManager | `d.WithTransaction(fn)` — available on *DB directly |
| `sqlx.Manager` | Our custom `DBManager` | Keep ours (has AutoMigrate, Reopen, driver config) |

## Implementation Steps

- [x] Step 1: Upgrade dependency to v1.4.1 (`go get`, `go mod tidy`)
- [x] Step 2: Replace `sqlx.NewEngine(db)` → `db.LazyEngine()` in Open/Reopen
- [x] Step 3: Remove stored `engine` field from `namedDB` (use LazyEngine on demand)
- [x] Step 4: Simplify `GetEngine` to call `ndb.db.LazyEngine()` directly
- [x] Step 5: Simplify `WithTransaction` to delegate to `db.WithTransaction(fn)`
- [x] Step 6: Build and verify — all tests pass
- [x] Step 7: Update backend spec (database-guidelines.md)

## Changes Summary

| File | Change |
|------|--------|
| `go.mod` | `tfnick/sqlx` v1.4.0 → v1.4.1 |
| `api/db/db.go` | Remove `engine` from namedDB; use `LazyEngine()`; delegate WithTransaction to sqlx native |
| `.trellis/spec/backend/database-guidelines.md` | Add v1.4.1 API table, Rebind pattern, update transaction docs |
