# Analyze: Framework optimizations for future SQLite → PostgreSQL migration

## Goal

分析当前框架在代码层面与 SQLite 的耦合程度，识别提前做哪些优化设计可以降低未来迁移到 PostgreSQL 的成本。

**本次产出**：完整分析报告 + 优先级排序。**不做代码改动**，具体优化作为后续独立任务。

---

## Decision Record (ADR-lite)

### D1: PostgreSQL 驱动选择 — `lib/pq`

**Context**：PostgreSQL Go 驱动有两大主流选择，占位符处理方式截然不同。
**Decision**：选择 `github.com/lib/pq`。
**Consequences**：
- `lib/pq` 自动将 `:name` 翻译为 `$N`——Engine 命名参数几乎无需改动
- `lib/pq` 处于维护模式（稳定但不再活跃开发），长期可评估 pgx
- `?` 占位符通过 `Rebind()` 处理，不需要手写 `$1, $2...`

### D2: DBManager 抽象 — DriverType 参数

**Context**：`Open()` 目前硬编码 SQLite PRAGMA 和连接池参数。
**Decision**：`Open(name, path, driver)` 增加 `driver` 参数（如 `"sqlite"`/`"postgres"`），根据类型自动选择 PRAGMA 和连接池参数。当前只实现 sqlite 分支，PG 分支留空占位。
**Consequences**：不引入完整的 Dialect 接口层，改动最小。未来换 PG 时只需补充 PG 分支。

### D3: 迁移文件 — 双目录并存

**Context**：SQLite 和 PostgreSQL 的 DDL 语法不兼容。
**Decision**：`migrations/{db}/sqlite/` + `migrations/{db}/postgres/`，`AutoMigrate` 根据 DriverType 选目录。现在只创建 postgres 空目录，不填充内容。实际迁移时用各自目录的 SQL 文件。
**Consequences**：两套 DDL 独立维护，清晰无耦合。新增迁移时需要写两份。

### D4: SQL 方言 — Rebind + Go 端时间

**Context**：代码中有 11 处 `?` 占位符和 6 处 `datetime('now')` 函数，PostgreSQL 不兼容。
**Decision**：
- `?` 占位符：每处加一行 `d.Rebind(query)`，sqlx 根据驱动自动翻译为 `$N`
- `datetime('now')`：改为 `time.Now().Format("2006-01-02 15:04:05")` 在 Go 端生成，作为 `:now` 参数传入
**Consequences**：每处改动量 1~2 行，不引入不必要的 Engine 依赖，保持多参数调用风格。

---

## Exploration Findings

代码库中发现了 **37 处 SQLite 特定代码**，分布在 9 个文件中：

### 分类统计

| 类别 | 数量 | 说明 |
|------|------|------|
| 驱动/PRAGMA | 11 处 | `sqlx.Open("sqlite", path)`、5 个 PRAGMA、`SetMaxOpenConns(1)` |
| SQL DDL 类型 | 7 处 | `DATETIME` 需改为 `TIMESTAMP` |
| SQL 函数 | 6 处 | `datetime('now')` 需改为 `NOW()` |
| 占位符 `?` | 11 处 | PostgreSQL 使用 `$1, $2...` |
| 方言语法 | 2 处 | `INSERT OR IGNORE` → `ON CONFLICT DO NOTHING` |

### 关键差异点

| SQLite | PostgreSQL |
|--------|------------|
| `sqlx.Open("sqlite", filepath)` | `sqlx.Open("postgres", dsn)` |
| 5 个 PRAGMA | 不需要（通过 GUC/默认行为） |
| `SetMaxOpenConns(1)` | 典型值 `25` |
| `?` 占位符 | `$1`, `$2`, `$N` |
| `:name` 命名参数（Engine 用） | `lib/pq` 自动翻译为 `$N`，无需改 |
| `datetime('now')` | `time.Now().Format(...)` Go 端生成 |
| `DATETIME` 列类型 | `TIMESTAMP` |
| `INSERT OR IGNORE` | `INSERT ... ON CONFLICT DO NOTHING` |
| `TEXT PRIMARY KEY`（UUID）| `UUID PRIMARY KEY`（可选优化） |

---

## 详细差异清单（37 处逐项对照）

### 类别 1: 驱动/PRAGMA — `api/db/db.go`

| # | 位置 | SQLite 代码 | PostgreSQL 等价 | 预优化策略 |
|---|------|------------|----------------|-----------|
| 1 | L12 | `import _ "modernc.org/sqlite"` | `import _ "github.com/lib/pq"` | Open 增加 `driver` 参数，内部 switch 驱动 |
| 2 | L46 | `sqlx.Open("sqlite", path)` | `sqlx.Open("postgres", dsn)` | 同上 |
| 3 | L52 | `SetMaxOpenConns(1)` | `SetMaxOpenConns(25)` | DriverType 分支自动设值 |
| 4 | L53 | `SetMaxIdleConns(1)` | `SetMaxIdleConns(5)` | 同上 |
| 5 | L60 | `PRAGMA foreign_keys = ON` | 不需要（PG 默认） | sqlite 分支保留，pg 分支跳过 |
| 6 | L61 | `PRAGMA journal_mode = WAL` | 不需要 | 同上 |
| 7 | L62 | `PRAGMA synchronous = NORMAL` | 不需要 | 同上 |
| 8 | L63 | `PRAGMA cache_size = -64000` | 不需要 | 同上 |
| 9 | L64 | `PRAGMA temp_store = MEMORY` | 不需要 | 同上 |
| 10-11 | L88/L309 | 日志信息 `(WAL)` | 移除或条件化 | 日志中 `(WAL)` 字样条件化 |

### 类别 2: SQL DDL 类型 — 迁移文件

| # | 文件 | 行 | SQLite | PostgreSQL |
|---|------|-----|--------|------------|
| 12 | `app/001_init.sql` | 11-12 | `DATETIME DEFAULT CURRENT_TIMESTAMP` | `TIMESTAMP DEFAULT CURRENT_TIMESTAMP` |
| 13 | `app/001_init.sql` | 21 | 同上 | 同上 |
| 14 | `app/001_init.sql` | 27-28 | 同上 | 同上 |
| 15 | `app/003_add_auth.sql` | 7-8 | `DATETIME NOT NULL` | `TIMESTAMP NOT NULL` |
| 16 | `app/003_add_auth.sql` | 8 | `DATETIME DEFAULT CURRENT_TIMESTAMP` | `TIMESTAMP DEFAULT CURRENT_TIMESTAMP` |
| 17 | `app/003_add_auth.sql` | 17 | `DATETIME` | `TIMESTAMP` |
| 18 | `app/003_add_auth.sql` | 20 | `DATETIME DEFAULT CURRENT_TIMESTAMP` | `TIMESTAMP DEFAULT CURRENT_TIMESTAMP` |
| 19 | `shared/001_init.sql` | 9 | `DATETIME DEFAULT CURRENT_TIMESTAMP` | `TIMESTAMP DEFAULT CURRENT_TIMESTAMP` |

**预优化策略**：按 D3 双目录方案，sqlite 目录保持原样，postgres 目录写入 PG 等价 DDL。迁移时 AutoMigrate 根据 DriverType 选择目录。

### 类别 3: SQL 方言语法 — 迁移文件

| # | 文件 | 行 | SQLite | PostgreSQL |
|---|------|-----|--------|------------|
| 20 | `app/002_seed.sql` | 3 | `INSERT OR IGNORE INTO` | `INSERT INTO ... ON CONFLICT DO NOTHING` |
| 21 | `shared/002_seed.sql` | 3 | 同上 | 同上 |

**预优化策略**：同上，归入 postgres 迁移目录。

### 类别 4: `datetime('now')` → Go 端时间 — `api/models/auth.go`

| # | 位置 | SQLite 代码 | PostgreSQL 等价写法 |
|---|------|------------|-------------------|
| 22 | L105 | `WHERE expires_at > datetime('now')` | Go: `now := time.Now().Format(...)`; SQL: `WHERE expires_at > :now` |
| 23 | L151 | `WHERE expires_at < datetime('now')` | 同上 |
| 24 | L206 | `AND expires_at > datetime('now')` | 同上 |
| 25 | L226 | `SET used_at = datetime('now')` | SQL: `SET used_at = :now` |
| 26 | L245 | `SET updated_at = datetime('now')` | SQL: `SET updated_at = :now` |
| 27 | L281 | `SET updated_at = datetime('now')` | 同上 |

**预优化策略**：在 Go 端生成 `now := time.Now().Format("2006-01-02 15:04:05")`，SQL 中改为 `:now` 命名参数。现在改好即对 SQLite 完全兼容，未来换 PG 零改动。

### 类别 5: `?` 占位符 → `Rebind()` — `api/models/auth.go`

| # | 位置 | SQLite 代码 | 改后 |
|---|------|------------|------|
| 28 | L262 | `d.Get(&user, query, email)` where `query = "... WHERE email = ?"` | `d.Get(&user, d.Rebind(query), email)` |

### 类别 6: `?` 占位符 → `Rebind()` — `api/models/order.go`

| # | 位置 | SQLite 代码 | 改后（加一行 Rebind） |
|---|------|------------|---------------------|
| 29 | L50-51 | `sharedDB.Exec("UPDATE products ... WHERE id = ? AND stock >= ?", qty, pid, qty)` | `sharedDB.Exec(sharedDB.Rebind("UPDATE products ... WHERE id = ? AND stock >= ?"), qty, pid, qty)` |
| 30 | L60 | `sharedDB.Exec("UPDATE products ... WHERE id = ?", qty, pid)` | 同上模式 |
| 31 | L101 | 同上（补偿回滚） | 同上模式 |
| 32 | L117 | `d.Get(&order, query, orderID)` where `query = "... WHERE id = ?"` | `d.Get(&order, d.Rebind(query), orderID)` |
| 33 | L133 | `d.Select(&orders, query, userID)` where `query = "... WHERE user_id = ?"` | `d.Select(&orders, d.Rebind(query), userID)` |
| 34 | L149 | `d.Select(&items, query, orderID)` where `query = "... WHERE order_id = ?"` | 同上模式 |
| 35 | L164 | `d.Exec(query, status, orderID)` where `query = "... SET status = ? WHERE id = ?"` | `d.Exec(d.Rebind(query), status, orderID)` |

### 类别 7: `?` 占位符 → `Rebind()` — `api/models/product.go`

| # | 位置 | SQLite 代码 | 改后 |
|---|------|------------|------|
| 36 | L47 | `d.Get(&product, query, id)` where `query = "... WHERE id = ?"` | `d.Get(&product, d.Rebind(query), id)` |
| 37 | L61 | `d.Exec(query, newStock, productID)` where `query = "... SET stock = ? WHERE id = ?"` | `d.Exec(d.Rebind(query), newStock, productID)` |

---

## 优先级排序

### P0 — 驱动/配置抽象（改动 1 个文件，收益最大）

**文件**：`api/db/db.go`
**改动**：`Open(name, path string)` → `Open(name, path, driver string)`
- 内部 switch `driver`：sqlite 保持现状，postgres 分支用 `lib/pq`，连接池参数不同
- PRAGMA 块移到 sqlite 分支内
- 连接池 `MaxOpenConns`/`MaxIdleConns` 按驱动设值
- 日志 `(WAL)` 条件化

### P1 — SQL 占位符归一化（改动 4 个文件，11 处，每处 1 行）

**文件**：`api/models/auth.go`、`order.go`、`product.go`
**改动**：每个 `d.Exec(query, args...)` / `d.Get(dest, query, args...)` 前加 `query = d.Rebind(query)`
**收益**：换 PG 后 `?` 自动变为 `$N`，零改动

### P2 — SQL 函数提取（改动 1 个文件，6 处）

**文件**：`api/models/auth.go`
**改动**：6 处 `datetime('now')` 改为 Go 端 `time.Now().Format(...)` + `:now` 参数
**收益**：跨数据库兼容，且 Go 端时间更可控

### P3 — 迁移文件双目录（新增 3 个空目录）

**改动**：
- 创建 `api/db/migrations/app/postgres/`（空）
- 创建 `api/db/migrations/shared/postgres/`（空）
- `AutoMigrate(name)` 增加 `driver` 参数，选择对应目录
**收益**：扩展点已就绪，实际迁移时只需填充 PG 版本 SQL

### P4 — 废弃依赖清理

**改动**：`go.mod` 移除 `github.com/mattn/go-sqlite3`（未使用）
**收益**：减少依赖噪音

---

## 不改无需改的内容

| 项目 | 原因 |
|------|------|
| Engine `:name` 命名参数（user.go 全部 + auth.go 大部分） | `lib/pq` 自动翻译，无需改动 |
| `#[ ]` 动态 SQL 片段 | `tfnick/sqlx` 预处理语法，不涉及数据库方言 |
| `schema_migrations` 表的 `DATETIME` | 运行时只执行迁移 SQL 文件，不在代码中操作该表类型 |
| UUID `TEXT PRIMARY KEY` | SQLite 和 PostgreSQL 均兼容 |

---

## Acceptance Criteria

- [x] 列出所有 SQLite 特定代码点（37 处）
- [x] 分类每个点的迁移成本和预优化策略
- [x] 给出推荐的优化优先级排序（P0-P4）
- [x] 产出决策记录（D1-D4）
- [x] 不改无需改的内容标记

## Out of Scope

- 不实施代码改动（P0-P4 作为后续独立任务）
- 不填充 PostgreSQL 迁移文件（只建空目录）
- 不引入完整 Dialect 接口层（保持简单 DriverType 分支）

## Technical Notes

- `github.com/tfnick/sqlx` 是 `jmoiron/sqlx` 的定制分支，内置 `Rebind()` 方法，根据驱动自动翻译占位符
- `lib/pq` 通过 `database/sql` 的 `NamedExec` 标准接口自动将 `:name` 翻译为 `$N`
- PostgreSQL 连接字符串格式：`host=localhost port=5432 dbname=mydb user=myuser password=mypass sslmode=disable`
- 文件级数据库路径改为连接字符串后，Reopen 逻辑不变（关闭旧连接 + 重新 `sqlx.Open`）
