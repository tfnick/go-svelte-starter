# Logged-In Menu Shell

## Goal

Modify the logged-in default homepage into a simple application shell with menu navigation. The logged-in default page should become a lightweight Dashboard welcome page, the existing order management UI should move behind an Order Manage menu item, and Scheduler should be present as a reserved menu page for future scheduled task management.

## What I Already Know

* The frontend is a Svelte + Vite SPA under `frontend/`.
* `frontend/src/App.svelte` currently renders auth pages by path, and renders the existing `Dashboard.svelte` for all other paths.
* `frontend/src/pages/Dashboard.svelte` currently contains the existing order management feature.
* `frontend/src/components/Header.svelte` contains the top bar, logged-in user actions, logout, and the export-toast trigger button.
* `frontend/src/router.js` owns `normalizePath`, `navigate`, and `routeTitle`.
* Existing UI uses Tailwind CSS and daisyUI classes.
* The requested menus are:
  * `dashboard`: simple welcome page.
  * `order manage`: existing order management feature.
  * `scheduler`: scheduled task management placeholder.

## Requirements

* After login, the default `/` route should show a simple Dashboard welcome page, not the order management UI.
* Add a simple logged-in menu/navigation capability with three items:
  * `Dashboard`
  * `Order Manage`
  * `Scheduler`
* `Dashboard` should route to the new simple welcome page.
* `Order Manage` should route to the existing order management capability.
* `Scheduler` should route to a reserved placeholder page for scheduled task management.
* Existing login, register, forgot password, reset password routes must keep working.
* Existing order management behavior must keep working after being moved under the Order Manage route.
* Existing realtime toast / points WebSocket behavior should keep working for the order management page.
* Logged-in application navigation should use a left sidebar layout instead of a top-only menu.
* The menu and app layout must be responsive on mobile:
  * desktop/tablet width: persistent left sidebar.
  * mobile width: collapsed menu with a clear trigger, for example a daisyUI drawer or equivalent responsive pattern.
  * content must remain readable without horizontal overflow.
* Menu UI should use existing Tailwind/daisyUI style conventions and avoid introducing a new component library.
* Unauthenticated users should still see auth flow pages and should not get a confusing logged-in application menu.

## Acceptance Criteria

* [x] Visiting `/` after login displays a simple Dashboard welcome page.
* [x] The menu contains Dashboard, Order Manage, and Scheduler.
* [x] The menu is shown as a left sidebar for logged-in application pages.
* [x] On mobile viewport widths, the menu is accessible through a collapsed/drawer-style interaction and does not force horizontal scrolling.
* [x] Clicking Dashboard navigates to the welcome page.
* [x] Clicking Order Manage navigates to the existing order management feature.
* [x] Clicking Scheduler navigates to a placeholder page that clearly indicates the feature is reserved.
* [x] Existing auth pages still render for `/login`, `/register`, `/forgot-password`, and `/reset-password`.
* [x] `routeTitle(...)` returns appropriate titles for the new app routes.
* [x] Frontend tests cover route title / route normalization behavior for the new menu routes.
* [x] `cd frontend && npm test` passes.
* [x] `cd frontend && npm run build` passes.

## Technical Approach

Recommended structure:

```text
frontend/src/pages/
  DashboardHome.svelte       # new welcome dashboard
  Dashboard.svelte           # either renamed later or kept as current order management page
  Scheduler.svelte           # scheduler placeholder
frontend/src/components/
  Header.svelte              # keeps top user actions such as logout / export toast trigger
  AppSidebar.svelte          # left sidebar menu for logged-in application pages
frontend/src/router.js       # route titles and aliases for new routes
frontend/src/App.svelte      # route-to-page mapping
```

Suggested routes:

```text
/             -> Dashboard welcome page
/orders       -> existing order management feature
/scheduler    -> Scheduler placeholder
```

Use `navigate(path)` for menu clicks and keep route matching in `App.svelte` simple. The logged-in app shell should render a responsive sidebar plus main content area for `/`, `/orders`, and `/scheduler`; auth pages should keep the current centered/simple layout without the sidebar. Prefer daisyUI/Tailwind responsive utilities such as `drawer`, `lg:drawer-open`, grid/flex breakpoints, and overflow-safe content containers.

## Decision (ADR-lite)

**Context**: The existing default page mixes the application landing page and order management feature. Adding more capabilities needs a small app shell to avoid turning the default route into a pile of feature panels.

**Decision**: Add a lightweight logged-in responsive sidebar and route the existing order management UI to `/orders`. Keep the default `/` route as a simple Dashboard welcome page. Add `/scheduler` as a placeholder route only. Desktop should keep the sidebar visible; mobile should collapse it behind an explicit menu trigger.

**Consequences**:

* Future feature pages can be added behind menu entries without bloating the default page.
* The existing order management feature remains intact but moves to a clearer route.
* This task intentionally avoids a full permission-based menu system or scheduler backend.

## Out of Scope

* Building real scheduler APIs, models, or backend job execution.
* Role/permission-based dynamic menus.
* Deep route nesting or a full router library.
* Reworking auth, JWT, WebSocket, or order management business behavior.
* Major visual redesign beyond a simple menu shell.

## Technical Notes

* Relevant files inspected:
  * `frontend/src/App.svelte`
  * `frontend/src/components/Header.svelte`
  * `frontend/src/pages/Dashboard.svelte`
  * `frontend/src/router.js`
  * `.trellis/spec/frontend/svelte-vite-embed.md`
* Relevant specs:
  * `.trellis/spec/frontend/svelte-vite-embed.md`
  * `.trellis/spec/guides/code-reuse-thinking-guide.md`

## Open Questions

* None. Decision: use a left sidebar for logged-in application navigation.
