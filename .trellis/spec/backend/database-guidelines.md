# Database Guidelines

> 本文是 DB manager、SQL、migration、事务 executor 的权威说明。业务事务由 usecase 发起，但底层 transaction-aware executor 规则在这里定义。

---

## Overview

当前数据库约定：

* Database: SQLite via `modernc.org/sqlite`
* SQL helper: `github.com/tfnick/sqlx v1.4.2`
* Dynamic SQL: `sqlx.Engine` with `#[ ... ]`
* Migrations: embedded SQL under `api/db/migrations/`
* Named DB: `app` and `shared`
* SQLite connection pool: `SetMaxOpenConns(1)` and `SetMaxIdleConns(1)`

`api/db/db.go` 管理连接、migration、reopen。
`api/db/tx.go` 管理 transaction-aware executor。
`api/framework/usecase/transaction.go` 提供业务层使用的 `fwusecase.WithAppTx(...)`。

---

## Named Databases

| Name | Purpose |
| --- | --- |
| `app` | 应用数据，例如 users、auth、products、orders、open_api partners/keys |
| `shared` | 预留给可替换的共享数据文件；当前不承载 project-owned tables |

`index.go` 启动时打开并 migrate 两个 DB：

```go
mgr.Open("app", "sqlite", *appDBPath)
mgr.AutoMigrate("app")
mgr.Open("shared", "sqlite", *sharedDBPath)
mgr.AutoMigrate("shared")
```

---

## Model Access Rules

model 函数应接收标准 `context.Context`：

```go
func GetUserByID(ctx context.Context, id string) (*User, error)
```

固定 SQL 使用：

```go
d, err := db.ExecutorFor(ctx, "app")
query := d.Rebind("SELECT * FROM users WHERE id = ?")
err = d.Get(&user, query, id)
```

动态 SQL 使用：

```go
eng, err := db.DynamicExecutorFor(ctx, "app")
err = eng.Select(&users, `
    SELECT * FROM users
    WHERE 1=1
        #[ AND id = :id ]
        #[ AND email LIKE :email ]
`, query)
```

不要在 usecase 调用的 model 中直接使用 `db.GetDB()` 或 `db.GetEngine()`，否则事务上下文无法复用。

---

## Transaction Rules

业务事务边界在 `api/usecase`：

```go
err := fwusecase.WithAppTx(ctx, func(txCtx fwusecase.Context) error {
    order, items, err := models.InsertOrderWithItems(txCtx.Std(), userID, orderItems)
    if err != nil {
        return err
    }
    _ = order
    _ = items
    return nil
})
```

规则：

* `WithAppTx` 只作用于 `app` DB。
* `shared` 不进入当前事务模型。
* model 通过 `txCtx.Std()` 接收 active transaction。
* model 不调用 `Begin`、`Commit`、`Rollback`、`db.WithTransaction`。
* nested `WithAppTx` / `db.WithTx(ctx, "app", ...)` 复用 active transaction。
* callback 返回 error 会 rollback；返回 nil 会 commit。
* commit 后才执行 `RegisterAfterCommit` 注册的 callback。

`products` 属于 `app` DB；`CreateOrder` 必须在同一个 `WithAppTx` 中扣减 `products.stock` 并写入 `orders/order_items`，失败时依靠 app transaction rollback 恢复库存。

---

## Dynamic SQL Rules

`sqlx.Engine` 使用 named parameters 和 `#[ ... ]` 条件片段：

```go
sql := `
UPDATE users SET
    updated_at = :updated_at
    #[ , name = :name ]
    #[ , email = :email ]
WHERE id = :id
`

result, err := eng.Exec(sql, user)
```

禁止手写字符串拼接 WHERE 条件。
禁止在同一条 SQL 中混用 `?` 和 `:named`。

---

## Timezone Contract

时间处理遵守同一条跨层规范：DB 层使用 UTC，应用层业务时间也使用 UTC，展示层再按用户 local timezone 展示。

后端规则：

* DB 中的 `created_at`、`updated_at`、`occurred_at`、`next_run_at`、`expires_at` 等时间字段统一表示 UTC。
* SQLite `CURRENT_TIMESTAMP` 可以用于 DB default 或 trigger；它输出 UTC 的 `YYYY-MM-DD HH:MM:SS`。
* Go 代码生成要持久化或进入业务事件的时间时，使用 `api/framework/timefmt`，例如 `timefmt.NowSQLiteDateTime()`、`timefmt.RFC3339(value)`、`timefmt.NowUTC()`。
* 禁止直接用 `time.Now().Format("2006-01-02 15:04:05")` 写 DB 或 DTO，因为它会受运行机器 local timezone 影响。
* 新增需要跨 API 展示的时间字段时，优先返回带 timezone 的 UTC RFC3339 字符串，例如 `2026-06-07T12:30:00Z`。
* 兼容既有 SQLite `YYYY-MM-DD HH:MM:SS` 字符串时，前端必须把它解释为 UTC，而不是浏览器 local time。
* `time.Now()` 可继续用于纯耗时统计，例如 `time.Since(startedAt)`，这类值不进入 DB、事件、DTO 时间字段。

示例：

```go
now := timefmt.NowSQLiteDateTime()
event.OccurredAt = timefmt.NowUTC()
nextRunAt := timefmt.RFC3339(next)
```

---

## Migrations

路径：

```text
api/db/migrations/app/*.sql
api/db/migrations/shared/*.sql
```

规则：

* 文件名前缀使用 `001_`、`002_`、`003_`。
* `AutoMigrate(name)` 按文件名排序执行。
* 已执行 migration 记录在 `schema_migrations`。
* 使用 `IF NOT EXISTS` 提升幂等性。
* 表和字段使用 `snake_case`。
* 主键当前使用 `TEXT` UUID；新 UUID 生成遵循 [UUID Generation](./uuid-generation.md)，默认使用 UUID v7，保持 ID 大致按时间有序。

`goqite` 队列表是 framework component-owned 表；项目业务扩展表必须拆到独立 migration。当前 durable event 设计允许 `domain_events` 和 `domain_event_deliveries`，但不使用已退役的 `domain_event_executions`。

### Scenario: Pre-Launch SQL Baseline Rewrite

#### 1. Scope / Trigger

Use this scenario only when the project/database has not gone live and the task explicitly allows deleting old SQL migrations. For launched databases, write forward-only migrations instead.

#### 2. Signatures

Baseline layout:

```text
api/db/migrations/app/001_schema.sql
api/db/migrations/app/002_seed.sql
api/db/migrations/app/007_add_goqite.sql
api/db/migrations/shared/001_schema.sql
api/db/migrations/shared/002_seed.sql
```

Seed IDs:

```sql
'019ea0c1-0001-7000-8000-000000000001'
```

#### 3. Contracts

* The rewrite targets fresh DB creation. Existing local DB files with old `schema_migrations` rows may need to be recreated.
* `001_schema.sql` should represent the current final table shape, not the historical path that produced it.
* `002_seed.sql` should contain deterministic seed rows. Seed row primary IDs must follow UUID v7 format when the ID is project-owned.
* Retired tables/columns must not be reintroduced during consolidation. Current retired examples include `sessions`, `variables.purpose`, and `domain_event_executions`.
* `goqite` remains component-owned and must stay isolated in `007_add_goqite.sql`, even during a baseline rewrite.

#### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| Existing deployed DB must be preserved | Do not use baseline rewrite; add a forward migration |
| Seed ID uses short aliases such as `u001` / `p001` | Replace with UUID v7-style IDs and update tests/docs that intentionally reference seeds |
| `goqite` is created outside `007_add_goqite.sql` | Archguard fails |
| Retired tables/columns appear in the new baseline | Remove them before merge |
| Tests rely on seed records | Update tests to use the new stable UUIDv7 seed constants |

#### 5. Good/Base/Bad Cases

Good: App project tables are consolidated into `001_schema.sql`, seed dictionaries/users/API keys into `002_seed.sql`, and `goqite` remains in `007_add_goqite.sql`.

Base: Product table and product seed rows live in the app baseline; shared baseline files may be comment-only placeholders while shared has no project-owned tables.

Bad: Keep a long chain of obsolete `003_...018_` migrations after deciding the project is pre-launch, or move project-owned scheduler/event columns into `goqite`.

#### 6. Tests Required

* Run `go test ./...` to exercise fresh app/shared DB migrations and archguard.
* Run frontend API tests when seed IDs appear in frontend helper examples.
* Search migrations for retired artifacts: `sessions`, `variables_new`, `domain_event_executions`, and old short seed IDs.

#### 7. Wrong vs Correct

#### Wrong

```sql
INSERT OR IGNORE INTO users (id, name) VALUES ('u001', 'Demo');
```

#### Correct

```sql
INSERT OR IGNORE INTO users (id, name)
VALUES ('019ea0c1-0001-7000-8000-000000000001', 'Demo');
```

#### Wrong

```sql
CREATE TABLE IF NOT EXISTS goqite (...);
-- inside 001_schema.sql
```

#### Correct

```text
api/db/migrations/app/007_add_goqite.sql
```

### Scenario: Goqite-Owned Queue and Project-Owned Extensions

#### 1. Scope / Trigger

新增或修改 goqite、scheduler、durable DDD event 存储时必须遵守本节。触发点包括 `api/db/migrations/app/*.sql`、`api/framework/queue`、`api/models/*scheduler*`、`api/models/*domain_event*`。

#### 2. Signatures

Migration split:

```text
api/db/migrations/app/007_add_goqite.sql
api/db/migrations/app/008_add_scheduled_tasks.sql
api/db/migrations/app/009_add_domain_event_delivery.sql
```

Project tables:

```sql
scheduled_tasks(id, name, job_name, schedule_type, schedule_value, payload_json, enabled, next_run_at, last_run_at)
scheduled_task_executions(id, task_id, job_name, message_id, status, scheduled_at, started_at, finished_at, error_message)
domain_events(id, topic, aggregate_type, aggregate_id, payload_json, metadata_json, occurred_at)
domain_event_deliveries(id, event_id, subscriber, message_id, status, attempts, last_error)
```

#### 3. Contracts

* `007_add_goqite.sql` 只创建 upstream `goqite` component 表、trigger、index；不能加入项目字段。
* `scheduled_tasks` 和 `scheduled_task_executions` 是项目 scheduler 状态，不能只靠 `goqite.body` 推断历史。
* `domain_events` 保存一次领域事件事实；`domain_event_deliveries` 保存每个 durable subscriber 的独立状态。
* 业务读取 SQL 仍放在 `api/models`，通过 `db.ExecutorFor(ctx, "app")` 复用事务。
* goqite 需要 `*sql.DB` 和 `*sql.Tx` 时，只能通过 `api/db/tx.go` 暴露的 framework helper 间接取得。

#### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| `007_add_goqite.sql` contains project table | archguard fails |
| non-framework package imports raw `maragu.dev/goqite` | archguard fails |
| scheduler update/delete affects zero rows | model returns `modelerror.ErrNotFound` wrapped cause |
| nullable operational columns scanned to DTO | model SELECT uses `COALESCE(..., '')` before string scan |
| app transaction rolls back after queue send | goqite message and project extension row roll back together |

#### 5. Good/Base/Bad Cases

Good: `fwusecase.WithAppTx` calls a usecase that writes project rows and queues goqite messages through `api/framework/queue`, so rollback removes both.

Base: Message management reads `goqite` rows for operational inspection only and returns a body preview, not a mutable business object.

Bad: Adding `task_id` or `subscriber` columns directly to `goqite`; this couples project state to an upstream component table.

#### 6. Tests Required

* `go test ./...`
* `api/framework/archguard/layer_boundary_test.go` validates raw goqite imports and migration ownership.
* Durable rollback/fan-out tests should assert independent `domain_event_deliveries` rows and queue messages.
* Scheduler tests should assert invalid cron rejection, one-shot disabling, and history status updates.

#### 7. Wrong vs Correct

#### Wrong

```sql
ALTER TABLE goqite ADD COLUMN subscriber TEXT;
```

#### Correct

```sql
CREATE TABLE IF NOT EXISTS domain_event_deliveries (
    id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL,
    subscriber TEXT NOT NULL,
    message_id TEXT
);
```

---

## SQLite Configuration

打开或 reopen SQLite 时应用：

| PRAGMA | Value |
| --- | --- |
| `foreign_keys` | `ON` |
| `journal_mode` | `WAL` |
| `synchronous` | `NORMAL` |
| `cache_size` | `-64000` |
| `temp_store` | `MEMORY` |

SQLite 连接池保持单连接，避免并发写入和 WAL 行为复杂化。

---

## Reopen Shared DB

`DBManager.Reopen(name)` 用于重新打开已注册 DB，当前主要服务 admin reload shared DB 场景。规则：

* 只能 reopen 已经 `Open` 的 DB。
* reopen 失败时保留旧连接。
* reopen 成功后关闭旧连接。
* 相关操作日志使用 `component:"db"`。

---

## Tests Required

* `go test ./...`
* 事务改动要覆盖 commit、rollback、nested transaction。
* `WithTx(ctx, "shared", fn)` 必须返回 app-only transaction error 且不运行 callback。
* migration 改动至少验证 app/shared DB 能启动并 migrate。

---

## Common Mistakes

* model 中直接 `db.GetDB()` 导致事务不生效。
* usecase 直接 `db.WithTx(ctx.Std(), "app", ...)` 后手写 `ctx.WithStd(...)`。
* 对 `shared` 使用事务。
* `Engine` SQL 使用 `?` placeholder。
* 直接 SQL 忘记 `Rebind()`。
* UPDATE/DELETE 不检查 `RowsAffected()`。
