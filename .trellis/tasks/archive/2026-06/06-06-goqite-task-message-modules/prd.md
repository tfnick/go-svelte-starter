# Goqite Task and Message Modules

## Goal

Design and implement two admin-facing modules based on `goqite`: a task management module for defining and observing scheduled tasks, and a message management module that provides a persistent message mechanism plus DDD event delivery backed by goqite. The result should fit the current `routes -> usecase -> models` architecture, keep transaction control in usecase, and expose simple Svelte pages for operations.

## What I Already Know

* The backend architecture is `routes -> usecase -> models -> db`.
* Business-free reusable infrastructure belongs under `api/framework`.
* Business transaction control belongs in `api/usecase` through `fwusecase.WithAppTx(...)`.
* Transactions only apply to the `app` DB, not `shared`.
* Current DDD-style events live under `api/framework/events` and support `async_best_effort` and `sync_tx`.
* Existing async events are in-memory best-effort. This task should introduce goqite-backed durable async delivery.
* Goqite-backed messages must use persistent delivery, not in-memory best-effort.
* One message or DDD event must support multiple domain listeners. Each listener needs independent durable processing state.
* After durable DDD event support is implemented, the current in-memory EventBus implementation is expected to be replaced.
* The logged-in Svelte shell already has a `Scheduler` menu and placeholder page.
* `goqite` is a persistent queue backed by a `goqite` SQL table and supports transaction-aware send APIs.
* `goqite/jobs` provides a runner, named jobs, message timeout extension, retry-by-redelivery, and graceful stop on context cancellation.
* Goqite does not define recurring schedules by itself; the project must store schedule definitions, parse cron expressions, and enqueue due executions.
* User decision: MVP should support cron expressions from the beginning.

## Research References

* [`research/goqite-capabilities.md`](research/goqite-capabilities.md) - goqite queue, jobs runner, schema, transaction APIs, and fit/risk notes for this repo.

## Requirements

### Task Management Module

* Add a Scheduler page implementation behind the existing `/scheduler` menu.
* Support listing scheduled task definitions.
* Support creating scheduled task definitions.
* Support modifying scheduled task definitions.
* Support enabling/disabling scheduled task definitions.
* Support viewing scheduled task execution history.
* Each scheduled task definition should include at least:
  * `id`
  * `name`
  * `job_name`
  * `schedule_type`
  * `schedule_value`
  * `payload_json`
  * `enabled`
  * `next_run_at`
  * `last_run_at`
  * `created_at`
  * `updated_at`
* MVP schedule types:
  * `cron`
  * `once_at`
* `cron` should use a mature parser such as `github.com/robfig/cron/v3`; do not hand-roll cron parsing.
* `schedule_value` for `cron` stores the cron expression.
* `schedule_value` for `once_at` stores an RFC3339 timestamp.
* The scheduler should compute and persist `next_run_at` after create/update and after each successful enqueue.
* Execution history should include at least:
  * `id`
  * `task_id`
  * `job_name`
  * `message_id`
  * `status`
  * `scheduled_at`
  * `started_at`
  * `finished_at`
  * `error_message`
* The scheduler loop should enqueue due task executions into a goqite queue and record execution history.
* Job execution should update history status on success/failure where possible.
* Job handlers should be registered at application startup, not inside request paths.

### Message Management Module

* Add a message management UI route or tab that can list goqite messages.
* The UI should show queue name, message id, created/updated times, timeout/availability, receive count, priority, and a short body preview.
* The message list should support filtering by queue name.
* The message list is read-only in MVP.
* Add backend APIs for listing goqite queue messages.
* Messages should live in the `app` DB only.
* Goqite messages are the durable operational source for pending/in-flight/retryable work.
* The implementation should not make `routes`, `usecase`, or `models` import raw goqite directly.

### DDD Event Integration

* Introduce framework-level goqite integration under `api/framework` so raw `maragu.dev/goqite` imports are isolated.
* Preserve the public DDD event facade concept in `api/framework/events`.
* Add a durable async event path backed by goqite persistence.
* Publishing a durable async event from inside `fwusecase.WithAppTx(...)` must enqueue the goqite message in the same app DB transaction.
* Publishing a durable async event outside an active transaction may enqueue immediately.
* `sync_tx` events should continue to run inside the active app transaction.
* Existing usecase code should keep publishing events through framework events, not through raw goqite queues.
* A single DDD event must support multiple domain subscribers.
* Each durable subscriber must receive an independent processing message or equivalent independent durable execution state.
* One subscriber failure must not block other subscribers for the same event.
* One subscriber retry state must not overwrite another subscriber retry state.
* Durable subscriber identity must be stable and explicit, for example `points.award_on_order_paid`.
* Durable event message body should be a stable JSON envelope with at least:
  * `event_id`
  * `subscriber`
  * `topic`
  * `aggregate_type`
  * `aggregate_id`
  * `payload_json`
  * `metadata_json`
  * `occurred_at`
* Durable event handlers should be registered at startup and dispatched by a goqite-backed runner.
* Handler failures should leave the message retryable according to goqite timeout/max receive behavior.
* The new durable DDD event implementation should be designed as the future replacement for the current in-memory EventBus implementation.

### Architecture and Layering

* `api/framework/queue` or equivalent should own goqite queue construction, schema assumptions, runner lifecycle, and low-level queue operations.
* `api/framework/events` should own domain event registration and publishing semantics.
* `api/usecase` should expose `XxxQry`, `XxxCmd`, and `XxxCo` APIs for task and message management.
* `api/models` should own SQL for scheduled task definitions, execution history, and read-only goqite message listing.
* `api/routes` should own internal `/api/*` DTOs and handler mapping.
* New frontend API helpers should live in `frontend/src/api.js`.
* New Svelte UI should use existing Tailwind/daisyUI conventions.
* Open API routes are out of scope unless explicitly requested later.

## Proposed API Shape

Internal authenticated APIs:

```text
GET    /api/scheduler/tasks
POST   /api/scheduler/tasks
PUT    /api/scheduler/tasks/:id
PATCH  /api/scheduler/tasks/:id/enabled
GET    /api/scheduler/tasks/:id/history
GET    /api/messages
```

Draft request/response concepts:

```go
type ListScheduledTasksQry struct {
    Enabled *bool
}

type CreateScheduledTaskCmd struct {
    Name          string
    JobName       string
    ScheduleType  string
    ScheduleValue string
    PayloadJSON   string
    Enabled       bool
}

type UpdateScheduledTaskCmd struct {
    ID            string
    Name          string
    JobName       string
    ScheduleType  string
    ScheduleValue string
    PayloadJSON   string
    Enabled       bool
}

type ListMessagesQry struct {
    Queue string
}
```

Return values should follow current usecase convention:

```go
ScheduledTaskCo
ScheduledTaskExecutionCo
QueueMessageCo
```

## Proposed Storage

All tables live in the `app` DB, but ownership must be explicit.

### Goqite-Owned Component Table

`goqite` is the upstream component table required by `maragu.dev/goqite`.

Rules:

* The table is owned by the goqite component, not by project business modules.
* Project migrations may create the table from the upstream goqite schema because this repo owns DB migration execution.
* Do not add project-specific columns to `goqite`.
* Do not encode business state only in `goqite.body`; successful messages are deleted.
* Application code may read `goqite` for operational message listing, but business history and event delivery status must live in project-owned extension tables.

```sql
-- goqite-owned component schema, copied from upstream goqite v0.4.0.
CREATE TABLE IF NOT EXISTS goqite (
  id TEXT PRIMARY KEY DEFAULT ('m_' || lower(hex(randomblob(16)))),
  created TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ')),
  updated TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ')),
  queue TEXT NOT NULL,
  body BLOB NOT NULL,
  timeout TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ')),
  received INTEGER NOT NULL DEFAULT 0,
  priority INTEGER NOT NULL DEFAULT 0
) STRICT;

CREATE TRIGGER IF NOT EXISTS goqite_updated_timestamp
AFTER UPDATE ON goqite
BEGIN
  UPDATE goqite SET updated = strftime('%Y-%m-%dT%H:%M:%fZ') WHERE id = old.id;
END;

CREATE INDEX IF NOT EXISTS goqite_queue_priority_created_idx
ON goqite (queue, priority DESC, created);
```

### Project-Owned Extension Tables

The following tables are owned by this project. They are not part of goqite upstream.

* `scheduled_tasks`: project task definition table.
* `scheduled_task_executions`: project task execution history table.
* `domain_events`: project durable DDD event table.
* `domain_event_deliveries`: project per-subscriber durable delivery table.

### Migration Split

```text
api/db/migrations/app/007_add_goqite.sql                 # goqite-owned component table only
api/db/migrations/app/008_add_scheduled_tasks.sql        # project-owned scheduler tables
api/db/migrations/app/009_add_domain_event_delivery.sql  # project-owned DDD event extension tables
```

Draft project-owned extension tables:

```sql
CREATE TABLE IF NOT EXISTS scheduled_tasks (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  job_name TEXT NOT NULL,
  schedule_type TEXT NOT NULL,
  schedule_value TEXT NOT NULL,
  payload_json TEXT NOT NULL DEFAULT '{}',
  enabled INTEGER NOT NULL DEFAULT 1,
  next_run_at DATETIME,
  last_run_at DATETIME,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS scheduled_task_executions (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL,
  job_name TEXT NOT NULL,
  message_id TEXT,
  status TEXT NOT NULL,
  scheduled_at DATETIME,
  started_at DATETIME,
  finished_at DATETIME,
  error_message TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (task_id) REFERENCES scheduled_tasks(id)
);

CREATE TABLE IF NOT EXISTS domain_events (
  id TEXT PRIMARY KEY,
  topic TEXT NOT NULL,
  aggregate_type TEXT NOT NULL,
  aggregate_id TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  occurred_at DATETIME NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS domain_event_deliveries (
  id TEXT PRIMARY KEY,
  event_id TEXT NOT NULL,
  subscriber TEXT NOT NULL,
  message_id TEXT,
  status TEXT NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(event_id, subscriber),
  FOREIGN KEY (event_id) REFERENCES domain_events(id)
);
```

## Decision (ADR-lite)

**Context**: The project currently has in-memory async events and no real scheduler implementation. The user wants goqite to become the base for task management, message management, and DDD event messaging.

**Decision**: Use goqite as a framework-level persistent queue in the `app` DB. Build a scheduler module that stores schedule definitions in project-owned tables and enqueues due jobs into goqite. Extend the DDD event facade so durable async events persist the event and enqueue one independent goqite delivery per durable subscriber atomically with app transactions. Keep raw goqite isolated under `api/framework`.

**Consequences**:

* Async events become persistent, observable, and retryable instead of purely best-effort.
* Fan-out is explicit: one event can produce multiple subscriber deliveries, and each delivery has independent retry state.
* Successful goqite messages are deleted, so business execution history must be stored separately.
* Scheduler definitions are project-owned because goqite only provides queue/job execution, not recurring schedules.
* A small app lifecycle/shutdown abstraction may be needed so goqite runners stop cleanly.
* Implementation should update eventing specs because the current spec says no outbox/event persistence.

## Expansion Sweep

### Future Evolution

* Manual run, pause/resume, retry-now, dead-letter views, and per-job concurrency limits can be added later.
* Open API access for task/message operations can be added later if third-party management is needed.

### Related Scenarios

* DDD event publishing should remain consistent with existing `PublishAsync` / `PublishInTx` semantics.
* Scheduler UI should share the logged-in app shell and route/menu conventions.
* Durable events should be able to replace the current EventBus implementation once subscriber migration is complete.

### Failure and Edge Cases

* A job can fail and be retried by goqite until `MaxReceive` prevents further receipt.
* A node restart must not lose queued messages.
* Unknown job names should be visible in message management.
* Transaction rollback must not leave a durable event or scheduled execution message behind.
* One failed subscriber must remain retryable without hiding successful deliveries for other subscribers.
* Invalid cron expressions must be rejected at create/update time with a safe validation message.

## Acceptance Criteria

* [ ] `goqite` dependency is added and its schema is migrated into the app DB.
* [ ] PRD-defined storage ownership is honored: goqite-owned `goqite` table is separate from project-owned extension tables.
* [ ] No project-specific columns are added to the goqite-owned `goqite` table.
* [ ] Raw `maragu.dev/goqite` imports are confined to framework packages.
* [ ] Scheduler APIs support list/create/update/enable-disable/history.
* [ ] Scheduler supports `cron` and `once_at` schedule types in MVP.
* [ ] Invalid cron expressions are rejected by backend validation.
* [ ] Scheduler Svelte page replaces the placeholder and can list/edit scheduled tasks plus view history.
* [ ] Message APIs can list goqite messages with queue filtering.
* [ ] Message UI can show goqite queue messages read-only.
* [ ] Durable DDD event publishing enqueues goqite messages in the same app transaction when called inside `WithAppTx`.
* [ ] One DDD event can fan out to multiple durable subscribers.
* [ ] Durable subscriber deliveries have independent processing/retry state.
* [ ] A failure in one durable subscriber does not block successful processing by other subscribers.
* [ ] Rollback tests prove no durable message is left after a failed app transaction.
* [ ] Existing `sync_tx` event behavior remains compatible.
* [ ] Job runner starts at application startup and stops cleanly with application shutdown.
* [ ] `go test ./...` passes.
* [ ] `cd frontend && npm test` passes.
* [ ] `cd frontend && npm run build` passes.

## Definition of Done

* Tests added or updated for queue integration, transaction rollback, scheduler calculations, route DTO mapping, and frontend API helpers.
* Tests added for durable event fan-out and per-subscriber failure isolation.
* Relevant specs updated, especially backend eventing/database/directory guidelines and frontend scheduler UI notes.
* No business code imports raw goqite directly.
* No route directly calls model/db.
* Production build still embeds the Svelte frontend.

## Out of Scope

* Hand-written cron parser.
* Distributed locking across multiple app processes.
* Dead-letter queue mutation or retry-now actions in MVP UI.
* Open API endpoints for scheduler/message management.
* Role/permission-based menu visibility.
* External queue systems such as Redis, SQS, RabbitMQ, or Kafka.

## Technical Notes

* Relevant existing files inspected:
  * `index.go`
  * `api/db/db.go`
  * `api/db/tx.go`
  * `api/framework/events/events.go`
  * `api/framework/usecase/transaction.go`
  * `frontend/src/pages/Scheduler.svelte`
  * `frontend/src/router.js`
* Relevant specs:
  * `.trellis/spec/backend/directory-structure.md`
  * `.trellis/spec/backend/database-guidelines.md`
  * `.trellis/spec/backend/eventing-guidelines.md`
  * `.trellis/spec/backend/api-contracts.md`
  * `.trellis/spec/backend/route-handler-guidelines.md`
  * `.trellis/spec/frontend/svelte-vite-embed.md`
  * `.trellis/spec/guides/cross-layer-thinking-guide.md`
  * `.trellis/spec/guides/code-reuse-thinking-guide.md`

## Open Questions

* None.
