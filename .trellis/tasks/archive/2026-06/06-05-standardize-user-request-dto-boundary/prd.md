# Standardize User Request DTO Boundary

## Goal

Complete the user API DTO boundary by replacing direct request binding into `models.User` with explicit route-level request DTOs for internal user create/update handlers.

## What I Already Know

* The previous DTO task standardized user/auth response DTOs.
* `CreateUser` and `UpdateUser` in `api/routes/user.go` still bind request bodies directly into `models.User`.
* `models.User` contains database/internal fields such as `PasswordHash`, `EmailVerified`, `IsActive`, `CreatedAt`, and `UpdatedAt`.
* Route handler guidelines already say request DTOs should be defined near the handler and validation should happen before model calls.
* This task should not change frontend behavior or Open API contracts.

## Requirements

* Add explicit user request DTOs in `api/routes/user.go`.
* Replace `c.Bind(&user)` for create/update with binding into request DTOs.
* Map request DTOs into `models.User` only after validation.
* Keep current endpoint request contract compatible for supported fields:
  * create accepts `name` and `email`.
  * update accepts optional `name` and `email`.
* Keep validation messages and response shapes as compatible as practical.
* Prevent client-provided internal fields from flowing through route binding into model writes.
* Add focused tests for request DTO mapping and internal field exclusion.
* Update backend API contract docs with request DTO boundary rules.

## Acceptance Criteria

* [x] `CreateUser` no longer binds directly into `models.User`.
* [x] `UpdateUser` no longer binds directly into `models.User`.
* [x] User request DTOs include only client-supported input fields.
* [x] Request DTO mapping does not carry client-provided `id`, `password_hash`, `email_verified`, `is_active`, `created_at`, or `updated_at`.
* [x] Existing user response DTO behavior remains unchanged.
* [x] Focused user request DTO tests pass.
* [x] `go test ./...` passes.
* [x] Backend API contract/spec docs describe request DTO boundary rules.

## Out of Scope

* Changing `models.User` signatures.
* Adding password handling to internal user CRUD.
* Refactoring auth request DTOs.
* Refactoring order/product/admin request DTOs.
* Changing frontend API calls or UI behavior.
* Changing Open API DTOs.
* Fixing unrelated database runtime files.

## Technical Approach

* Add `CreateUserRequest` and `UpdateUserRequest` near user route handlers.
* Add small mapping helpers:
  * `toCreateUserModel(req CreateUserRequest) models.User`
  * `toUpdateUserModel(id string, req UpdateUserRequest) models.User`
* Bind and validate request DTOs in `CreateUser` and `UpdateUser`.
* Keep model functions unchanged.
* Add route-package tests for request-to-model mapping and JSON binding behavior.
* Extend `.trellis/spec/backend/api-contracts.md` with internal user request DTO rules.

## Technical Notes

* Relevant files:
  * `api/routes/user.go`
  * `api/routes/user_responses.go`
  * `api/routes/user_responses_test.go`
  * `.trellis/spec/backend/api-contracts.md`
  * `.trellis/spec/backend/route-handler-guidelines.md`
* Repo inspection:
  * `api/routes/user.go` is the only route file with `var user models.User` plus `c.Bind(&user)`.
  * Auth/order routes already define request DTOs near handlers.
