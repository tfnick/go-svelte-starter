# Dictionary Management

## Goal

Add a `Dictionary` menu and management surface for dictionary types and dictionary values. The existing `/api/dictionaries` endpoint can keep serving batch lookup options, while authenticated users can create, edit, enable, disable, and inspect dictionaries from the app shell.

## Requirements

* Add authenticated internal API endpoints to manage dictionary types.
* Add authenticated internal API endpoints to manage dictionary values belonging to a dictionary type.
* Preserve the existing public/internal read helper `GET /api/dictionaries?types=...` for frontend option lookups.
* Add a `Dictionary` menu item in the logged-in app shell.
* Add a `Dictionary` page with dictionary type list, dictionary value list, create/edit forms, and enable/disable controls.
* Store dictionaries and values in the app database with migrations and validation.

## Acceptance Criteria

* [ ] `GET /api/dictionaries?types=...` returns enabled values from the database and remains compatible with current response shape.
* [ ] Authenticated management routes support list/create/update/toggle for dictionary types and values.
* [ ] Duplicate dictionary type keys and duplicate value codes within a dictionary are rejected with conflict responses.
* [ ] `/dictionary` and `/dictionary.html` resolve to the new app page and show `Dictionary` in the sidebar.
* [ ] Frontend API helper and router tests cover the new routes.
* [ ] Backend model/usecase/route tests cover CRUD, validation, conflict, and enabled filtering.

## Definition of Done

* `go test ./...`
* `cd frontend && npm test`
* `cd frontend && npm run build`
* Specs updated if new API/UI contracts become reusable project knowledge.

## Technical Approach

Follow the existing `Variable` management pattern: migration-backed model functions, usecase validation and DTO conversion, route-local response DTOs, frontend helper functions, router entry, and a Svelte page under `frontend/src/pages/`.

## Out of Scope

* Import/export of dictionary data.
* Bulk reordering workflows beyond editing a numeric sort order.
* Per-value localization.
* Secret or credential storage.

## Technical Notes

* Existing read path: `api/usecase/dictionary.go`, `api/routes/dictionaries.go`, `frontend/src/api.js#getDictionaries`.
* Similar management pattern: Variable menu/API/page.
* Existing dirty worktree includes Parameter task changes; this task should avoid committing unrelated Parameter or database WAL/SHM files.
