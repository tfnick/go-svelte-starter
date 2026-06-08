# Migrate Frontend From HTMX To Svelte

## Goal

Replace the current HTMX/Hyperscript static frontend with a Svelte frontend under `frontend/`, while preserving the Go backend as the executable entrypoint. During development, Svelte changes must hot reload without restarting the Go service. For production, the Svelte build output must be pure static files that are embedded into the Go binary so the final app still ships as a single executable.

## What I Already Know

* The current app serves `public/` directly via Echo: `router.Static("/", "public/")`.
* The current browser live reload watches `public/` with `github.com/aarol/reload` and exposes `/reload_ws`.
* Current pages and partials live under `public/*.html` and `public/components/**/*.html`.
* Current HTMX-specific backend surface includes `/api/components/*` and `/api/auth/status-component`.
* Auth endpoints already support JSON and form submissions. This makes a Svelte migration practical without requiring immediate backend API redesign.
* The final app currently does not embed frontend static files into the Go executable; static files are read from disk.
* There is no existing `frontend/`, Vite config, Svelte package, or Node toolchain in the repo.
* Current Windows development entrypoint is `dev.bat`; the migrated workflow must keep double-click startup as a first-class path.
* Production embedding requires a two-stage build: Svelte static assets must exist before `go build`, because Go `embed` captures files at compile time.

## Requirements

* Create a `frontend/` Svelte app that replaces the user-facing HTMX/Hyperscript pages.
* Use a development workflow where frontend code changes hot reload through the Svelte/Vite dev server without restarting the Go backend.
* Configure Tailwind CSS and daisyUI in the Svelte/Vite frontend, and prefer daisyUI components/classes for common UI such as buttons, inputs, cards, alerts, modals, navbars, loading states, and forms.
* Keep the Go backend responsible for `/api/*` endpoints.
* In development, make frontend-to-backend API calls work without manual URL rewrites for each call. Recommended approach: Vite dev server proxies `/api` to the Go backend.
* In production, compile `frontend/` into pure static files.
* Embed the compiled static files into the Go executable, using Go `embed.FS` or an equivalent standard-library static file embedding approach.
* Serve the embedded Svelte app from Go in production.
* Preserve SPA routing behavior if Svelte uses client-side routes: unknown non-API paths should fall back to the built `index.html`.
* Preserve existing auth flows: login, register, logout, forgot password, reset password, current-user/status display.
* Remove user-facing dependency on HTMX/Hyperscript markup and CDN scripts once Svelte replaces the relevant pages.
* Keep the final production command capable of building a single executable that does not require `public/` or `frontend/dist/` at runtime.
* Update development scripts/docs so developers know how to run Go and Svelte hot reload together.
* Preserve one-click Windows development startup: double-clicking `dev.bat` must start the development service stack as before.
* `dev.bat` must automatically perform the frontend preparation needed for startup, including installing dependencies when needed and compiling Svelte static files at least once before launching the Go service.
* After `dev.bat` starts the stack, editing Svelte source files must be reflected in the browser without manually restarting the Go backend or rerunning `dev.bat`.
* Development hot updates should come from the Svelte/Vite dev server rather than from Go `embed`, because embedded files are fixed at Go compile time.
* Add or update production build entrypoints, such as `build.bat` and Makefile `build`, so they run frontend dependency installation/checks, then `npm run build`, then `go build`.
* Production `go build` must fail with a clear error if `frontend/dist/` or required built assets are missing; it must not silently produce an executable with missing frontend resources.
* Clearly separate development and production frontend behavior: `dev.bat` may run an initial Svelte build, but live source edits after startup are reflected through Vite HMR; embedded production assets update only after rerunning the production build.
* Running the production executable directly with no flags must default to production serving and must not redirect browser page requests to the Vite dev server.
* Decide and document the old `public/` lifecycle. Production must not continue reading user-facing frontend files from `public/` after the Svelte embed migration; if any `public/` files remain, they must be explicitly classified as temporary compatibility or non-frontend static assets.

## Acceptance Criteria

* [ ] `frontend/` contains a Svelte app and build configuration.
* [ ] `frontend/` includes Tailwind CSS and daisyUI configuration.
* [ ] Migrated Svelte UI uses daisyUI components/classes for common controls instead of custom-building basic button/input/card/alert/modal styling.
* [ ] `npm run dev` or an equivalent frontend dev command starts hot module reload for Svelte.
* [ ] A documented dev workflow runs the Go backend and Svelte dev server together.
* [ ] Double-clicking `dev.bat` starts the full development stack without requiring extra manual commands.
* [ ] `dev.bat` performs frontend dependency/setup checks and runs an initial Svelte static build before starting the Go backend.
* [ ] Editing a Svelte component in development updates the browser without restarting the Go service.
* [ ] After `dev.bat` has started, subsequent Svelte source edits appear in the browser through hot reload or automatic refresh.
* [ ] Frontend calls to `/api/auth/login`, `/api/auth/register`, `/api/auth/logout`, `/api/auth/status`, `/api/auth/forgot-password`, and `/api/auth/reset-password` work from the Svelte app.
* [ ] Production build compiles Svelte into static assets.
* [ ] Production build is available through documented scripts, including Windows `build.bat` and Makefile `build` or an equivalent repo-standard command.
* [ ] Production build order is enforced: install/check frontend dependencies, build Svelte static assets, then compile Go.
* [ ] Running Go compilation without required `frontend/dist/` assets fails with a clear message instead of producing a broken executable.
* [ ] Go production build embeds the compiled frontend assets into the executable.
* [ ] Running the production executable serves the Svelte app and API without requiring frontend files on disk.
* [ ] Running the production executable with no `--dev` flag serves embedded Svelte pages on `http://127.0.0.1:<port>/` without redirecting to port `5173`.
* [ ] Final binary verification copies the executable to a temporary empty directory, runs it there, and confirms both the Svelte app and `/api/*` endpoints are reachable.
* [ ] Direct navigation to SPA paths returns the Svelte `index.html`, while `/api/*` keeps normal API behavior.
* [ ] Production serving no longer depends on `public/` for the migrated frontend.
* [ ] HTMX component routes are removed or explicitly left as temporary compatibility only if needed by a documented migration step.
* [ ] Lint/type-check/build commands pass for the touched Go and frontend code.

## Definition Of Done

* Tests added or updated where practical for static serving and auth-facing behavior.
* Go build passes.
* Frontend build passes.
* Development and production commands are documented.
* No runtime dependency on HTMX/Hyperscript remains in the migrated frontend.
* Rollback is straightforward: old `public/` files can be restored from git if the migration needs to be backed out.

## Technical Approach

Recommended implementation path:

1. Scaffold a plain Svelte + Vite app under `frontend/`.
2. Configure Tailwind CSS and daisyUI in the Svelte app.
3. Configure Vite to proxy `/api` to the Go backend during development.
4. Keep Go running on port `3000`; run the Vite dev server on a separate port such as `5173`.
5. Implement Svelte pages/components for the existing flows, using daisyUI component classes for the shared UI surface.
6. Update Go static serving so development can keep serving API only, while production serves embedded files from the Svelte build output.
7. Add SPA fallback for non-API routes.
8. Update `dev.bat` so double-click startup runs the full dev stack: ensure/install frontend dependencies, run an initial frontend build, start the Go backend, start Vite dev/HMR, and provide the browser URL.
9. Add production build scripts, including `build.bat` and Makefile `build`, that run the required two-stage production build.
10. Add a pre-Go-build guard or generated/embed placeholder strategy so missing `frontend/dist/` fails loudly with an actionable message before or during Go compilation.
11. Update build verification so the resulting executable is copied to a temporary empty directory and run there to prove no frontend disk files are required.
12. Retire `public/` from production frontend serving, or document any remaining files as non-frontend/temporary compatibility assets.

## Decision (ADR-lite)

**Context**: The project currently uses HTMX/Hyperscript HTML files served from `public/`, but the target is a Svelte frontend that still ships inside one Go executable.

**Decision**: Use plain Svelte with Vite, not SvelteKit, for the initial migration. Use Vite's dev server and HMR in development, then embed Vite's static build output into Go for production.

**Consequences**: Development needs a Node/Vite process alongside the Go process, but `dev.bat` must hide that complexity behind one-click startup. Svelte edits hot reload cleanly through Vite. Production assets are fixed at Go compile time, so any Svelte production change requires rerunning the production build before a new executable includes it. Client-side routing requires an explicit Go fallback to `index.html`.

## Out Of Scope

* Rewriting the backend data model.
* Changing the database schema for this migration alone.
* Introducing SSR.
* Migrating to SvelteKit unless a later decision requires it.
* Redesigning the whole visual language beyond what is needed to replace the existing pages.
* Adding new product features unrelated to the frontend migration.

## Research References

* [`research/svelte-vite-static-embed.md`](research/svelte-vite-static-embed.md) - Vite/Svelte development and static-build notes mapped to this repo.

## Technical Notes

* Existing entrypoint: `index.go`.
* Existing static root: `public/`.
* Existing frontend pages: `public/index.html`, `public/login.html`, `public/register.html`, `public/forgot-password.html`, `public/reset-password.html`.
* Existing HTMX partials: `public/components/**/*.html`.
* Existing JSON-capable auth API: `api/routes/auth.go`.
* Current CORS allows `http://localhost:3000` and `http://localhost:4000`; if Vite uses `5173`, CORS or proxy behavior must account for it.
* `dev.bat` currently accepts an optional port and runs `go run . --dev --port %PORT%`; the migrated script should preserve this convenience while also managing the frontend dev process.
* Existing Makefile `build` currently runs only `go build -o ./tmp/ .`; it must be updated to include the frontend production build first.
* Current `public/` directory should not remain an implicit production frontend source after migration.
* daisyUI should be treated as the frontend component styling system. Use official daisyUI/Tailwind setup and avoid adding another Svelte UI component framework unless a specific need is documented.
* The project has backend Trellis specs only. Frontend conventions should be introduced conservatively and documented if they become durable project rules.

## Open Questions

* Should the implementation preserve old `.html` URLs such as `/login.html`, or is it acceptable to migrate to SPA routes such as `/login` with redirects for compatibility?
