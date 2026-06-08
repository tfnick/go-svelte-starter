# P0-P2: DB driver abstraction + SQL dialect normalization

## Background

From the PostgreSQL migration analysis (archived task), implement the top 3 priorities to prepare the framework for future migration.

## Implementation Steps

- [x] P0: Add `driver` parameter to `Open()`, switch PRAGMAs and connection pool by driver
- [x] P1: 11 `?` placeholders → `Rebind()` in auth.go, order.go, product.go
- [x] P2: 6 `datetime('now')` → Go-side time in auth.go

## Verification

1. ✅ `go build` zero errors
2. ✅ DB init log shows "(sqlite)" driver tag
3. ✅ Seed user login — HTTP 200 (P2 `datetime`→Go time working)
4. ✅ Auth status — True, name 张三
5. ✅ Order creation — Rebind queries work, stock decremented (100→98)
6. ✅ Restart — migrations skipped correctly

## Reference

See archived PRD: `.trellis/tasks/archive/2026-05/05-25-analyze-framework-optimizations-for-future-sqlite-to-postgresql-migration/prd.md`
