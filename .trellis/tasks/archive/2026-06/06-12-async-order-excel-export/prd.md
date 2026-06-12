# Export Orders to Excel Asynchronously

## Goal

Add an asynchronous Excel export flow for order lists. Users should be able to start an export from the order list UI, see the background job in the bottom-left Tasks panel, and receive a bottom-left notification when the job finishes. The backend must reject overly large exports and generate the Excel file in a streaming/batched way so the process does not load all matching orders into memory.

## What I Already Know

- The user requires:
  - Export order lists to Excel through an async task.
  - Reject exports when the matched order count is greater than 100000.
  - Avoid reading all rows into memory at once.
  - Show the async task in the bottom-left Tasks list.
  - Send a bottom-left notification when the task completes.
- Existing backend already has:
  - `async_tasks` table and model in `api/models/async_task.go`.
  - Heavy task queue name `heavy-tasks` in `api/framework/queue/queue.go`.
  - Heavy task runner registration in `index.go`.
  - `POST /api/user/tasks` and `GET /api/user/tasks` in `api/routes/heavy_task.go`.
  - Order list usecases:
    - `ListMyOrders` for current user scope.
    - `ListAdminOrders` for admin cross-user scope.
  - Order list model methods use dynamic SQL optional filters with `#[ ... ]`.
  - Realtime WebSocket messages and frontend notification/task refresh handling.
  - Persisted notification support through `notifications` and `CreateNotification`.
- Existing frontend already has:
  - Bottom-left `TaskCenter` using `GET /api/user/tasks`.
  - Bottom-left `NotificationCenter` receiving realtime toasts.
  - `Dashboard.svelte` order list for the current user.
  - API helpers for `getMyOrders` and `listAdminOrders`.

## Requirements

- Add first-class async order export entrypoints instead of exposing arbitrary task creation to the UI.
- MVP must support both order list personas:
  - Current-user order export.
  - Admin order export.
- Export entrypoints must reuse the same permission semantics as the corresponding order list:
  - Current-user export uses the authenticated actor user ID as the owner filter.
  - Admin export requires admin access and may include optional `user_id` and `status` filters.
- Before enqueueing or before writing the file, count matching orders.
- If matching order count is greater than 100000, reject the export with a validation/business error and do not enqueue a background job.
- The worker must not fetch all matching orders into memory.
- The worker should stream or batch rows from the database and stream rows into the Excel writer.
- The async task row must be visible through the existing `GET /api/user/tasks` API and bottom-left Tasks panel.
- When the task finishes successfully or fails permanently, notify the requesting user through the existing bottom-left notification flow.
- The task result must contain enough information for the user to download the file after completion.
- Completed export files must be stored through the configured primary OSS provider, not local-only storage.
- Export output should include at least the columns currently shown by the order list:
  - Order ID
  - Product ID / product name
  - User ID / user name
  - Status
  - Subscription status
  - Amount
  - Created at
- The export must preserve data permission boundaries when the worker executes, not only when the HTTP request is received.

## Acceptance Criteria

- [x] `POST /api/user/orders/export` starts an async export for the current user's order list filters.
- [x] `POST /api/admin/orders/export` starts an async export for admin order list filters and rejects non-admin users.
- [x] Exports with more than 100000 matching rows are rejected before creating a queued job.
- [x] The export worker writes order rows in batches/streaming form and does not construct a full in-memory slice of all matching orders.
- [x] A queued export appears in `GET /api/user/tasks` and therefore in the bottom-left Tasks panel.
- [x] Successful completion updates the async task to `completed` with a result JSON containing the file/download reference.
- [x] Permanent failure updates the async task to `failed` with an error message.
- [x] Successful and failed terminal states send a realtime notification to the requesting user's bottom-left NotificationCenter.
- [x] The frontend order list has an export action that starts the async task and provides clear feedback.
- [x] Tests cover permission scope, 100000 row limit behavior, streaming/batched export behavior, task status updates, and notification emission.

## Technical Approach

Recommended approach:

- Add an order-export-specific usecase API, for example:
  - `EnqueueMyOrdersExcelExport(ctx, qry)`
  - `EnqueueAdminOrdersExcelExport(ctx, qry)`
- Use the existing heavy task queue and async task table.
- Store the export request in `async_tasks.payload_json`, including:
  - export kind: `orders_excel`
  - requester user ID
  - scope: `user` or `admin`
  - filters: `user_id`, `status`
  - created timestamp if needed for audit/debugging
- Extend heavy task execution to dispatch `orders_excel`.
- Count matching rows using the same scoped `OrderQuery`.
- If count is greater than 100000, return validation error before enqueueing.
- Add a model-level streaming or batch iterator for orders. Prefer keyset pagination using `(created_at, id)` descending if feasible; otherwise use fixed-size batches and keep memory bounded.
- Generate XLSX using `github.com/xuri/excelize/v2` `StreamWriter`.
- Upload the generated XLSX file through the existing primary OSS provider abstraction.
- Store OSS metadata in task `result_json`, including object key, content type, size, original filename, provider/channel metadata, and download expiry metadata when available.
- Add a download route scoped to the task owner, for example `GET /api/user/tasks/:id/download`, that verifies the task belongs to the current user and returns a presigned GET URL or redirects to it.
- Do not expose arbitrary object keys for unauthenticated or cross-user download.
- Use a 1 hour presigned download URL expiry for MVP, matching the existing OSS adapter default behavior.
- On terminal state, call existing notification/realtime mechanisms so:
  - TaskCenter refreshes.
  - NotificationCenter receives a completion/failure toast.

## Decision (ADR-lite)

Context:

- Order exports may be large, and Excel generation can be slow.
- The project already has async task, queue, realtime, and notification primitives.
- The system previously aligned order routes by persona (`/api/user/...` and `/api/admin/...`).

Decision:

- Build order export as an async heavy task using the existing `async_tasks` and `heavy-tasks` infrastructure.
- Add persona-specific export endpoints that map to a shared usecase implementation.
- Use streaming/batched database reads and Excel streaming writer output.

Consequences:

- The feature fits the existing Tasks and Notifications UI.
- Export files depend on a configured primary OSS provider; missing OSS configuration should fail clearly before or during export depending on when it is detected.
- Tests must prove both permission scope and memory-safe iteration behavior.

## Research References

- `research/excel-streaming.md` - Excelize StreamWriter supports streaming XLSX generation and can use temporary files for large streamed data.

## Open Questions

- None for MVP.

## Out of Scope

- Third-party `/open-api` order export.
- Importing orders from Excel.
- Emailing export files.
- Exporting more than 100000 rows by splitting files.
- Long-term export retention cleanup policy beyond storing files in primary OSS.

## Definition of Done

- Go tests pass with coverage for export permission, limit, worker, task state, and notification paths.
- Frontend tests pass where relevant.
- `go test ./...`, `cd frontend && npm test`, `cd frontend && npm run build`, and `git diff --check` pass.
- The implementation keeps memory bounded during export.
