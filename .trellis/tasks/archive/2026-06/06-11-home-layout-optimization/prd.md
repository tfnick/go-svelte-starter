# Home Layout Optimization

## Goal

Optimize the logged-in app shell from a top-header plus workspace-style sidebar into a cleaner left navigation plus right work area. The result should give the main pages more operating space, remove sidebar noise, and feel simpler on desktop and mobile.

## Requirements

* Desktop logged-in app uses a stable left/right shell:
  * Left side is a fixed navigation and user-action rail.
  * Right side is the full work area for Dashboard, Orders, Products, Users, Settings, and other app pages.
* Left side is split into clear zones:
  * Top logo area only.
  * Middle menu area with concise labels and semantic icons.
  * Bottom profile/action area with current user, notifications, tasks, and sign out.
* Remove the old `Workspace` sidebar block.
* Move notification and task triggers from global floating buttons into the left-side bottom action area.
* Mobile uses a compact layout:
  * Default view shows a small top bar with menu trigger, logo, and current route title.
  * Navigation opens through a drawer/overlay instead of occupying permanent space.
  * Content stays full width with no page-level horizontal overflow.
* Keep existing routes, auth behavior, permissions, API calls, and business page data logic unchanged.
* Use daisyUI 5 and Tailwind 4 patterns already used in the project.

## Acceptance Criteria

* [x] Logged-in desktop app no longer shows the global top `Header`.
* [x] Desktop shell uses a fixed left sidebar and a right content area.
* [x] Sidebar top shows logo/brand only and no `Workspace` text.
* [x] Sidebar menu removes route descriptions.
* [x] Every visible menu item has an icon.
* [x] Current route is visually highlighted across the full menu row.
* [x] User profile, notification button, task button, and sign-out button are visible in the sidebar bottom area.
* [x] Notification and task panels are still usable from their new docked triggers.
* [x] Mobile default layout is compact, with content prioritized and drawer navigation hidden until opened.
* [x] Desktop and mobile smoke checks show no page-level horizontal overflow.
* [x] Auth pages still render with the existing header and are not wrapped by the logged-in app shell.
* [x] `cd frontend && npm test` passes.
* [x] `cd frontend && npm run build` passes.
* [x] `go test ./...` passes.
* [x] `git diff --check` passes, with only Windows CRLF warnings.
* [x] Browser smoke check covers desktop `1440x900` and mobile `390x844`.

## Implementation Summary

* Added `lucide-svelte` as the icon source for app navigation and shell actions.
* Added stable `icon` keys to `appRoutes` in `frontend/src/router.js`.
* Updated `frontend/src/router.test.js` so visible menu route tests assert icon keys.
* Changed `frontend/src/App.svelte` so the global `Header` only renders for auth/loading/logged-out states.
* Reworked `frontend/src/components/AppSidebar.svelte` into the logged-in app shell:
  * desktop permanent sidebar,
  * mobile compact top bar and drawer,
  * logo-only sidebar top,
  * icon menu,
  * bottom profile/action area,
  * shell-owned sign out.
* Updated `NotificationCenter.svelte` and `TaskCenter.svelte` to support `docked` mode while preserving their floating fallback behavior.

## Verification

* `cd frontend && npm test`: passed, 43 tests.
* `cd frontend && npm run build`: passed, daisyUI 5.5.23 build completed.
* `go test ./...`: passed.
* `git diff --check`: passed with CRLF warnings only.
* Browser smoke:
  * Started temporary backend on port `3000` with SQLite DBs under the system temp directory.
  * Registered a temporary smoke user through `/app/register`.
  * Desktop `1440x900`: sidebar width `288px`, content starts at `x=288`, menu row width `263px`, menu has SVG icons, no `Workspace` text, no horizontal overflow.
  * Mobile `390x844`: default sidebar is off-canvas, top menu trigger is visible, main content is full width, no horizontal overflow.
  * Mobile drawer opened successfully, sidebar is visible, menu has icons, bottom actions are visible, no horizontal overflow.

## Notes

* `npx prettier --write` formatted `router.js` and `router.test.js`.
* The current project does not include a Svelte Prettier parser, so Prettier cannot infer a parser for `.svelte` files. Svelte formatting was kept manually consistent.
* This task intentionally avoids redesigning individual business page internals.

## Out of Scope

* No route permission changes.
* No API contract changes.
* No redesign of page-level tables, forms, or cards.
* No theme switching or dark mode work.
