# Svelte + Vite Static Build And Go Embed Research

## Question

What frontend setup best supports Svelte hot reload during development while producing static files that can be embedded into a Go executable?

## Sources

* Vite official docs: development server and build behavior, including dev-time server options and proxy configuration.
* Svelte official docs: Svelte apps commonly use Vite-based tooling for local development and bundling.
* Go standard library docs: `embed` supports compiling static files into the Go binary as an `embed.FS`.

## Findings

### Common Pattern

Plain Svelte with Vite is a good fit when the frontend is a browser-rendered SPA/static app:

* Vite handles dev serving and hot module replacement.
* Vite builds production assets into a static output directory such as `frontend/dist`.
* The Go server can treat that output as static files in production.
* Go can embed the production output at compile time with `//go:embed`.
* daisyUI can be installed as a Tailwind CSS plugin and used inside Svelte components through daisyUI/Tailwind classes.

### Mapping To This Repo

Current repo state:

* Go/Echo is already the backend and executable entrypoint.
* Static files are currently served from `public/`.
* There is no SSR requirement.
* Existing auth endpoints already accept JSON, so the Svelte app can call the current API directly.
* The frontend should use daisyUI for common controls and layout pieces to reduce custom CSS and speed up the migration.

Recommended local development shape:

* Go backend: `go run . --dev --port 3000`
* Svelte/Vite frontend: run from `frontend/`, likely on `5173`
* Vite config: proxy `/api` to `http://localhost:3000`
* Tailwind CSS + daisyUI configured in the frontend stylesheet/build pipeline.
* Browser opens the Vite URL during frontend development.
* Windows one-click entrypoint: `dev.bat` starts both the Go backend and Vite frontend so double-click startup remains available.
* `dev.bat` can run an initial `npm install`/`npm ci` check and `npm run build` before starting the Go process, but live Svelte edits during the session should be surfaced by Vite HMR rather than by repeatedly rebuilding the Go binary.

Recommended production shape:

* Run frontend dependency installation/checks first, such as `npm ci` when a lockfile is available.
* Run `npm run build` in `frontend/` to generate static assets.
* Build Go only after frontend assets exist.
* Embed the static output into Go.
* Serve embedded files for non-API requests.
* Return `index.html` for SPA routes that do not map to a concrete embedded asset.
* Provide production scripts such as `build.bat` and Makefile `build` that enforce this order.
* Add a clear guard for missing `frontend/dist/`; avoid silently producing a binary that lacks the frontend.
* Verify the artifact by copying the executable to a temporary empty directory and confirming the Svelte app plus `/api/*` endpoints work without `frontend/`, `frontend/dist/`, or `public/`.
* Treat `public/` as retired for production frontend serving unless remaining files are explicitly documented as compatibility or unrelated static assets.

## Feasible Approaches

### Approach A: Plain Svelte + Vite SPA (Recommended)

How it works:

* Add `frontend/` with Svelte and Vite.
* Add Tailwind CSS and daisyUI for component styling.
* Use Vite HMR during development.
* Build static files into a known directory.
* Embed those files into the Go binary.

Pros:

* Smallest conceptual change from static HTML to static frontend bundle.
* Directly matches the single-executable production goal.
* Avoids SSR/server adapter complexity.
* daisyUI keeps common UI implementation compact while staying framework-agnostic.

Cons:

* Requires client-side routing and API state handling in Svelte.
* Development uses two processes: Go backend plus Vite frontend, so `dev.bat` needs process orchestration.

### Approach B: SvelteKit With Static Adapter

How it works:

* Use SvelteKit and configure static adapter output for Go embedding.

Pros:

* Better routing conventions and future growth path.
* More batteries included.

Cons:

* More framework surface than this repo currently needs.
* Static adapter and SPA fallback behavior need extra care.
* Adds complexity before there is a clear SSR or filesystem routing requirement.

## Recommendation

Use Approach A for the first migration. It directly satisfies hot reload in development and static embedding in production with the least framework overhead.
