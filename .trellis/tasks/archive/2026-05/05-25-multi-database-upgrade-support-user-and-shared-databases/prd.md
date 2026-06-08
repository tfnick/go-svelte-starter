# Multi-Database Upgrade

Upgrade from single SQLite database to dual-database support (user DB + shared DB).

## Background

- Local server produces metrics data → stored in shared SQLite DB → file uploaded to remote → replaces shared.db
- Remote server needs separate user DB for user-level data (profiles, sessions, orders, payments)
- shared DB is read-only from app perspective

## Implementation Steps

- [x] Step 1: Restructure migration directories (user/ and shared/ subdirectories)
- [x] Step 2: Rewrite api/db/db.go — DBManager replacing globals
- [x] Step 3: Update index.go startup (dual DB init, new CLI flags)
- [x] Step 4: Update api/models/user.go (7 sites → "user" DB)
- [x] Step 5: Update api/models/auth.go (12 sites → "user" DB)
- [x] Step 6: Update api/models/product.go (3 sites → "shared" DB)
- [x] Step 7: Rewrite CreateOrder saga + add GetOrderByID (api/models/order.go)
- [x] Step 8: Fix GetOrderDetail (api/routes/order.go)
- [x] Step 9: Create api/routes/admin.go (reload endpoint)
- [x] Step 10: Delete old migration files
- [x] Step 11: Build and verify

## Verification Results

1. ✅ `go build` and `go vet` — zero errors
2. ✅ Fresh start — both user.db and shared.db created, all 5 migrations applied
3. ✅ Restart — all migrations skipped ("已是最新版本"), no duplicates
4. ✅ Register + login → session cookie → `/api/auth/me` returns user
5. ✅ Products accessible from shared DB seed data (p001-p004)
6. ✅ Cross-DB saga: order in user.db, stock decremented in shared.db (p001: 100→98)
7. ✅ Stock exhaustion: fully rejected, stock unchanged (no orphan decrement)
8. ✅ Shared DB reload: `POST /api/admin/reload-shared-db` works
