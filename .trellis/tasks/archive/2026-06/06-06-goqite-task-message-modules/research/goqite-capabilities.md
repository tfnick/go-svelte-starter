# Goqite Capabilities Research

## Sources

* <https://goqite.maragu.dev/>
* <https://pkg.go.dev/maragu.dev/goqite>
* <https://pkg.go.dev/maragu.dev/goqite/jobs>
* <https://raw.githubusercontent.com/maragudk/goqite/v0.4.0/schema_sqlite.sql>

## Library Facts

* `goqite` is a persistent message queue library backed by a database table. The official page describes the primary use case as a SQLite-backed queue inspired by SQS.
* The current pkg.go.dev version inspected is `v0.4.0`.
* Core queue construction:

```go
q := goqite.New(goqite.NewOpts{
    DB:        db,
    Name:      "jobs",
    SQLFlavor: goqite.SQLFlavorSQLite,
})
```

* `NewOpts` accepts `DB *sql.DB`, `Name string`, `SQLFlavor`, `Timeout`, and `MaxReceive`.
* `Message` has `ID`, `Body []byte`, `Delay time.Duration`, and `Priority int`.
* Queue APIs include `Send`, `SendAndGetID`, `Receive`, `ReceiveAndWait`, `Extend`, and `Delete`.
* Transaction-aware APIs exist: `SendTx`, `SendAndGetIDTx`, `ReceiveTx`, `ExtendTx`, and `DeleteTx`.
* The SQLite schema uses one table named `goqite` with columns including `id`, `queue`, `body`, `timeout`, `received`, and `priority`. Multiple logical queues are separated by the `queue` column.
* Treat the upstream `goqite` table as component-owned. Project-specific scheduler state, event state, and subscriber delivery state should be stored in separate project-owned tables instead of extending the `goqite` table.
* `goqite/jobs` provides:

```go
r := jobs.NewRunner(jobs.NewRunnerOpts{
    Queue:        q,
    Limit:        1,
    PollInterval: 100 * time.Millisecond,
})
r.Register("job_name", func(ctx context.Context, m []byte) error { return nil })
r.Start(ctx)
```

* `jobs.Create` and `jobs.CreateTx` create job messages for registered job names.
* `Runner.Start(ctx)` blocks until context cancellation and waits for running jobs to finish.
* The jobs runner extends message timeout while a job is running and deletes the queue message on success.
* If a job returns an error, the message is not deleted and may be retried after the timeout until `MaxReceive` is reached.

## Fit With This Repo

* Current app DB uses `modernc.org/sqlite` with `database/sql` under `github.com/tfnick/sqlx`, and `goqite` works with `*sql.DB`, so a direct integration is feasible.
* Current `sqlx.Tx` embeds `*sql.Tx`, so a controlled framework helper can expose the active app transaction to goqite without forcing business code to manage transactions manually.
* Current business transaction boundary is `fwusecase.WithAppTx(...)`. Persistent event/job enqueue must happen inside that transaction when the source business change is transactional.
* `shared` DB must remain outside app transactions per existing specs. Goqite tables should live in the `app` DB.
* The existing `api/framework/events` facade already owns DDD event APIs. The persistent async path should extend or replace the async side there instead of letting usecase code import raw goqite.

## Design Implications

* Goqite does not provide recurring schedule definitions by itself. The task management module needs its own `scheduled_tasks` table and a scheduler loop that enqueues due jobs into a goqite queue.
* Cron expressions should be parsed with a mature library. Recommended candidate: `github.com/robfig/cron/v3`, using standard 5-field cron parsing via `cron.ParseStandard(...)` and `schedule.Next(now)` to compute `next_run_at`.
* Keep cron parsing behind a small framework or usecase helper so the rest of the task management module depends on project-level schedule semantics, not a raw third-party parser.
* Execution history should not be inferred only from `goqite` rows, because successful messages are deleted. Store task execution history in a dedicated app table.
* The message management module can inspect the raw `goqite` table for pending, in-flight, delayed, and exhausted messages.
* A persistent async DDD event flow can be modeled as a `domain-events` queue where the message body is a stable event envelope.
* For DDD event fan-out, do not model one event as one shared goqite message when multiple subscribers exist. A single queue message is consumed/deleted by one runner path, so each durable subscriber should get its own message or an equivalent independent durable delivery row and retry state.
* A good fit is: persist one `domain_events` row, then create one `domain_event_deliveries` row plus one goqite message per durable subscriber. The message body should carry `event_id` and `subscriber`, and the worker should load the event payload from durable storage before invoking the subscriber.
* Use `jobs.CreateTx` or `Queue.SendTx` from inside `fwusecase.WithAppTx(...)` for event/job creation that must commit atomically with business state.
* Worker lifecycle should be started at application startup and stopped through a root context on server shutdown. Current `index.go` does not yet have graceful server shutdown wiring, so this may need to be added as part of implementation.

## Risks

* The upstream examples use `github.com/mattn/go-sqlite3`; this repo uses `modernc.org/sqlite`. Integration tests must verify schema creation, insert, receive, retry, and delete on the current driver.
* The goqite schema table name is fixed as `goqite`; project migrations must avoid naming conflicts and should keep raw queue inspection read-only.
* A failed job stays in the queue until timeout and receive limit rules make it unavailable. The UI should make exhausted messages visible rather than pretending they are dead-lettered elsewhere.
* `jobs.Runner` panics on unregistered job names. The framework layer should register known jobs at startup and make unknown queued job messages operationally visible.
* DDD event dispatch should isolate subscriber failures. If one subscriber fails, its goqite message should retry independently; successful subscriber messages should be deleted and their delivery rows marked completed.
