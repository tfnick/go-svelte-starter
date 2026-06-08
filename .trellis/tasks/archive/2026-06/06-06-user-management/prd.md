# User management

## Goal

Add a User management screen so authenticated operators can browse users with standard pagination and enable or disable a user account from the list. Disabling uses the existing `users.is_active` security semantics, so disabled users cannot log in or continue using protected APIs.

## What I already know

* User asked for a new left-side `User` menu.
* The page must support paginated User querying.
* The page must support disabling and enabling a specific User.
* Existing routes already expose `GET /api/users`, `GET /api/users/:id`, `POST /api/users`, `PUT /api/users/:id`, and `DELETE /api/users/:id`, but the list route currently returns all users.
* Existing `users` table already has `is_active INTEGER DEFAULT 1`.
* Existing auth and auth middleware already reject `is_active=0` users with `account is disabled`.
* Existing pagination conventions use `page`, `page_size`, `items`, and `pagination`.
* Existing frontend routes are centralized in `frontend/src/router.js`, and pages are switched in `frontend/src/App.svelte`.
* Existing management pages use daisyUI tables, badges, buttons, and responsive overflow containers.

## Assumptions

* This task is an admin/operator style management page, not a public profile page.
* Existing protected-route authentication is enough for access control for this task.
* The old `GET /api/users` endpoint can evolve from an array response to a paginated envelope because this frontend is the primary client in this starter project.
* Create, edit, delete, search, and role-based authorization are out of scope unless explicitly added later.

## Open Questions

* Resolved: the current logged-in user must not be allowed to disable their own account.

## Requirements

* Add `/users` and `/users.html` frontend route aliases.
* Add a `User` entry to the logged-in left-side menu.
* Add a `User` page that lists users in a paginated table.
* Show at least name, email, email verification, active status, created time, and actions in the list.
* Use the project-standard pagination contract: `page`, `page_size`, `items`, `pagination`.
* Add backend pagination support for `GET /api/users?page=1&page_size=10`.
* Add backend support for toggling user active state.
* Keep disable/enable as a soft state update of `users.is_active`; do not delete rows.
* Prevent the current logged-in user from disabling their own account.
* Reflect enable/disable changes in the list after the action completes.
* Use responsive layout so the table works on narrower screens.

## Acceptance Criteria

* [x] A logged-in user can open the left-side `User` menu.
* [x] The `User` page loads users from a paginated backend endpoint.
* [x] Pagination controls let the operator navigate user pages.
* [x] Each row shows whether the user is active or disabled.
* [x] Clicking Disable changes `is_active` to false and refreshes the visible row/list.
* [x] Clicking Enable changes `is_active` to true and refreshes the visible row/list.
* [x] Disabling the current logged-in user is rejected with a validation error.
* [x] Backend validates invalid pagination input consistently with existing pagination helpers.
* [x] Backend returns not found for toggling a missing user.
* [x] Existing authentication behavior continues to reject disabled users.
* [x] Tests cover backend pagination, enable/disable usecase/route behavior, API helper paths, and frontend route registration.

## Definition of Done

* Tests added or updated for changed backend and frontend behavior.
* `go test ./...` passes.
* `cd frontend && npm test` passes.
* `cd frontend && npm run build` passes.
* Specs updated if the User management API/UI establishes reusable conventions.
* Work is committed before Trellis finish-work archival.

## Out of Scope

* Creating users from the management page.
* Editing user profile fields from the management page.
* Deleting users.
* Searching or filtering users.
* Role/permission management.
* Audit log for enable/disable actions.

## Technical Notes

* Existing user model file: `api/models/user.go`.
* Existing user usecase file: `api/usecase/user.go`.
* Existing user route file: `api/routes/user.go`.
* Existing protected route registration: `index.go`.
* Existing frontend API helpers: `frontend/src/api.js`.
* Existing frontend route/menu registration: `frontend/src/router.js`.
* Existing frontend page switch: `frontend/src/App.svelte`.
* Existing Event page is the closest pagination UI precedent: `frontend/src/pages/Events.svelte`.
* Existing Scheduler page is the closest enable/disable action precedent: `frontend/src/pages/Scheduler.svelte`.
* Existing backend pagination helper: `api/framework/usecase/pagination.go`.
* Implemented backend pagination and active-state toggle in `api/models/user.go`, `api/usecase/user.go`, and `api/routes/user.go`.
* Registered `PATCH /api/users/:id/active` in `index.go`.
* Implemented frontend helpers in `frontend/src/api.js`, route/menu wiring in `frontend/src/router.js` and `frontend/src/App.svelte`, and page UI in `frontend/src/pages/Users.svelte`.
* Added backend tests in `api/usecase/user_management_test.go` and `api/routes/user_management_test.go`.
* Added frontend helper/router test coverage in `frontend/src/api.test.js` and `frontend/src/router.test.js`.
* Updated specs: `.trellis/spec/backend/api-contracts.md` and `.trellis/spec/frontend/svelte-vite-embed.md`.
* Verification passed: `go test ./...`, `cd frontend && npm test`, `cd frontend && npm run build`.
