# Fix daisyUI 5 Styles Across Remaining Pages

## Goal

Use the repaired Login and Products pages as the daisyUI 5 style baseline, then fix the remaining frontend pages/components that still use daisyUI 4-era classes or visually broken layouts after the daisyUI 5 upgrade.

## What I Already Know

* The previous task upgraded the frontend to `daisyui@5.5.23` and Tailwind CSS 4.
* `Login.svelte`, `AuthCard.svelte`, and `Products.svelte` have already been adjusted to daisyUI 5 patterns.
* The new style baseline is:
  * forms use `fieldset`, `fieldset-legend`, `fieldset-label`, and `input`/`select`/`textarea w-full`
  * avoid daisyUI 4 classes such as `form-control`, `label-text`, `input-bordered`, `select-bordered`, and `textarea-bordered`
  * cards use `border-base-200`, `rounded-box` where appropriate, and compact `card-body` spacing
  * list/table pages avoid page-level horizontal overflow
  * dense admin pages should feel operational and scannable, not like landing pages
* Static scan found many remaining v4-era patterns under `frontend/src/pages` and `frontend/src/components`.

## Requirements

* Update all remaining auth pages to match the repaired Login page:
  * `Register.svelte`
  * `ForgotPassword.svelte`
  * `ResetPassword.svelte`
* Update admin/operational pages that still use daisyUI 4 form/table classes:
  * `Dashboard.svelte`
  * `Scheduler.svelte`
  * `Dictionary.svelte`
  * `Parameters.svelte`
  * `Variables.svelte`
  * `Users.svelte`
  * `Events.svelte`
  * `Notifications.svelte`
  * `Settings.svelte`
  * `Experiments.svelte`
* Review supporting components for visual consistency:
  * `AppSidebar.svelte`
  * `Header.svelte`
  * `NotificationCenter.svelte`
  * `TaskCenter.svelte`
* Replace daisyUI 4 form classes with daisyUI 5-compatible structures:
  * `form-control` -> `fieldset` or compact flex/grid wrapper
  * `label-text` -> `fieldset-legend` / `fieldset-label` / plain scoped text
  * `input-bordered`, `select-bordered`, `textarea-bordered`, `file-input-bordered` -> default daisyUI 5 control classes with explicit `w-full` or stable sizing
* Normalize cards, empty states, tables, selected rows, pagination, and filter bars to the Login/Product visual baseline.
* Preserve all existing API calls, route paths, auth behavior, form state, validation behavior, and business logic.

## Acceptance Criteria

* [x] No tracked Svelte page/component under `frontend/src/pages` or `frontend/src/components` uses `form-control`, `label-text`, `input-bordered`, `select-bordered`, or `textarea-bordered`.
* [x] Remaining `border-base-300` usage is intentionally reviewed and either replaced with `border-base-200` or kept only where visual contrast is specifically needed.
* [x] Tables/lists use daisyUI 5-friendly wrappers and do not create page-level horizontal overflow on narrow screens.
* [x] Auth pages visually match the repaired Login page.
* [x] Admin pages visually align with the repaired Products page: compact, scannable, consistent spacing, and stable widths.
* [x] `cd frontend && npm test` passes.
* [x] `cd frontend && npm run build` passes and prints `daisyUI 5.5.23`.
* [x] `go test ./...` passes unless the change is confirmed frontend-only and backend tests are explicitly deferred by the user.
* [x] `git diff --check` passes.
* [ ] Browser smoke check covers at least:
  * `/app/login`
  * `/app/register`
  * `/app/products`
  * one table-heavy admin page
  * one form-heavy admin page

## Definition of Done

* All MVP pages/components listed above are migrated to daisyUI 5-compatible classes.
* Tests/build/diff checks pass.
* Any newly discovered frontend style convention is captured in `.trellis/spec/frontend/`.
* User has had a chance to visually inspect the fixed pages before commit, unless they explicitly ask to commit immediately.

## Out of Scope

* No API path, request/response DTO, route guard, or backend behavior changes.
* No redesign of information architecture or navigation.
* No new UI component library.
* No broad copywriting/content rewrite.
* No migration of marketing/static public pages unless a visible daisyUI 5 regression is found inside the Svelte app shell.

## Technical Approach

1. Use `Login.svelte`, `AuthCard.svelte`, and `Products.svelte` as concrete style samples.
2. Run static scans for daisyUI 4-era classes and prioritize files with the highest visible impact.
3. Convert forms first, then table/list containers, then card/border/empty-state consistency.
4. Keep edits local to Svelte view markup unless a tiny helper removes meaningful duplication inside the same component.
5. Verify with frontend tests, production build, diff check, and browser smoke checks.

## Technical Notes

Initial static scan command:

```sh
rg -n "form-control|label-text|input-bordered|select-bordered|textarea-bordered|border-base-300|class:selected|table table-sm|overflow-x-auto" frontend/src/pages frontend/src/components
```

Key baseline files:

* `frontend/src/components/AuthCard.svelte`
* `frontend/src/pages/Login.svelte`
* `frontend/src/pages/Products.svelte`
* `.trellis/spec/frontend/svelte-vite-embed.md`

Current scan highlighted these high-priority files:

* `frontend/src/pages/Register.svelte`
* `frontend/src/pages/ForgotPassword.svelte`
* `frontend/src/pages/ResetPassword.svelte`
* `frontend/src/pages/Dashboard.svelte`
* `frontend/src/pages/Scheduler.svelte`
* `frontend/src/pages/Dictionary.svelte`
* `frontend/src/pages/Parameters.svelte`
* `frontend/src/pages/Variables.svelte`
* `frontend/src/pages/Users.svelte`
* `frontend/src/pages/Events.svelte`
* `frontend/src/pages/Notifications.svelte`
* `frontend/src/pages/Settings.svelte`
* `frontend/src/pages/Experiments.svelte`
* `frontend/src/components/AppSidebar.svelte`
* `frontend/src/components/Header.svelte`
* `frontend/src/components/NotificationCenter.svelte`
* `frontend/src/components/TaskCenter.svelte`

## Implementation Notes

* Converted remaining Svelte pages/components from daisyUI 4-era form classes to daisyUI 5-compatible `fieldset`, `fieldset-legend`, `fieldset-label`, and default control classes.
* Replaced `border-base-300` with `border-base-200` across migrated Svelte UI surfaces.
* Normalized list/table containers to `max-w-full overflow-x-auto rounded-box border border-base-200` and `table table-zebra table-sm` with stable `min-w-*`.
* Replaced old selected-row `class:selected` styling with `bg-primary/5`.
* Normalized cards, empty states, toggle rows, and compact center/popover components to match the repaired Login/Product baseline.

## Verification

* Static scan found no remaining `form-control`, `label-text`, `input-bordered`, `select-bordered`, `textarea-bordered`, `file-input-bordered`, `class:selected`, or `border-base-300` under `frontend/src/pages` and `frontend/src/components`.
* `cd frontend && npm test` passed.
* `cd frontend && npm run build` passed and printed `daisyUI 5.5.23`.
* `go test ./...` passed.
* `git diff --check` passed.
* Browser smoke:
  * `/app/register` checked successfully: no legacy form classes, 3 fieldsets, no horizontal overflow.
  * `/app/products` checked in logged-out state and correctly rendered Login auth flow.
  * HTTP smoke for `/app/login`, `/app/register`, `/app/products`, `/app/dictionary`, `/app/parameters`, `/app/events`, and `/app/settings` returned 200 from Vite.
  * Authenticated admin browser smoke is pending manual/user visual inspection because the Browser tool could not type credentials in this environment (`virtual clipboard is not installed`) and the read-only evaluate path cannot write `localStorage`.
