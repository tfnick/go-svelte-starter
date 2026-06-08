# Usecase Transaction Management

## Goal

Move transaction ownership to the usecase layer so each business use case can define its atomic write boundary while routes stay focused on HTTP concerns and models stay focused on SQL access. The implementation should reuse the existing `routes -> usecase -> models` context propagation path and make transaction reuse work across horizontal usecase calls.

## Requirements

* Usecase methods are the only layer that may start an application transaction for business operations.
* Routes must not start transactions; routes continue to handle middleware-based authentication, authorization, binding, and response mapping.
* Models must not start business transactions. Models receive `context.Context` and obtain the current database executor from the db helper.
* If a usecase calls another usecase while a transaction is already active for the same database, the nested call must reuse the existing transaction instead of opening a second transaction.
* If no transaction is active, models must continue to work by using the normal database executor.
* Transaction helpers must support the existing named database model such as `app` and `shared`.
* Business transactions are only supported for the `app` database. `shared`
  writes remain outside usecase transactions and must use explicit
  compensation when needed.
* The order creation flow must be the first concrete migration target:
  * Usecase controls the `app` transaction boundary.
  * Models provide granular operations for inserting orders and order items.
  * Shared stock reservation/compensation remains explicit and must not pretend to be a distributed transaction.
* Transaction behavior must be covered by focused backend tests.

## Acceptance Criteria

* [x] `api/db` exposes a transaction helper that stores the active transaction in `context.Context`.
* [x] `api/db` exposes an executor helper that returns the active transaction when available and falls back to the normal DB/engine when not.
* [x] Usecase can wrap a business operation in a transaction and pass the transaction context to models.
* [x] Nested usecase calls under the same database transaction do not open another transaction.
* [x] `api/models/order.go` no longer owns the `app` transaction boundary for order creation.
* [x] `db.WithTx` rejects non-`app` database names with a clear error.
* [x] Existing backend tests pass with `go test ./...`.
* [x] Frontend checks still pass because this is an architecture change with no intended UI contract changes.

## Definition of Done

* Tests added or updated for transaction helper behavior and the migrated order usecase path.
* Backend package tests pass.
* Frontend test/build checks pass if the full project check remains practical.
* PRD and backend specs updated to document the transaction ownership convention.

## Technical Approach

Add a db-level transaction context helper, tentatively:

* `db.WithTx(ctx, dbName, fn)` starts a transaction if none is active for `dbName`, stores it in a derived context, and calls `fn(txCtx)`.
* If `WithTx` sees an active transaction for the same `dbName`, it calls `fn(ctx)` directly so horizontal usecase calls can compose.
* `db.Executor(ctx, dbName)` returns a small project-owned executor abstraction backed by either the active transaction or the normal database connection.
* Models call `db.Executor(ctx, "app")` or `db.Executor(ctx, "shared")` instead of starting transactions themselves.
* Usecase methods use `fwusecase.WithAppTx(ctx, fn)` when they need an app
  transaction. The helper injects the transaction into the standard context and
  passes a full `fwusecase.Context` into `fn`, so nested usecase calls can forward
  the same transaction-aware context without hand-rolling `ctx.WithStd(...)`.

## Decision (ADR-lite)

**Context**: Transaction scope is a business decision, and business decisions live in usecase. The current model-level transaction in order creation makes it hard for usecases to compose multiple model operations under one boundary.

**Decision**: Put transaction boundaries in usecase and propagate the active transaction through standard `context.Context`. Keep models unaware of transaction ownership by giving them an executor helper.

**Consequences**: This keeps route and model responsibilities clean and supports horizontal usecase reuse. The trade-off is that database helpers become a small infrastructure abstraction. Cross-database operations remain explicit compensation/Saga-style logic rather than distributed transactions.

## Out of Scope

* Distributed transactions across `app` and `shared`.
* Rewriting every read-only model query in the project if it is not required for the transaction helper migration.
* Introducing a repository interface layer beyond the existing `models` package.
* Changing route contracts or frontend behavior.

## Technical Notes

* Existing order creation currently starts an `app` transaction inside `api/models/order.go`.
* Existing `db.DBManager` already has `WithTransaction(name, fn)` based on `sqlx.DB.WithTransaction`.
* The previous context propagation task already changed usecase methods to receive `fwusecase.Context` and models to receive standard `context.Context`.
* Implemented `api/db/tx.go` with `WithTx`, `ExecutorFor`, and `DynamicExecutorFor`.
* Moved reusable transaction support to `api/framework/usecase/transaction.go`; `WithAppTx` is the preferred usecase
  layer transaction entry point.
* Migrated order creation so `api/usecase/order.go` owns the `app` transaction boundary and `api/models/order.go` exposes granular write/compensation functions.
* Verification commands passed:
  * `go test ./...`
  * `cd frontend && npm test`
  * `cd frontend && npm run build`
