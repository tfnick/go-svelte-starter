# Svelte Go Starter

A Go + Echo backend with a Svelte/Vite frontend. The development workflow uses Vite hot reload, while production builds embed the compiled Svelte static assets into the Go executable.

## Development

### Windows

Double-click `dev.bat`, or run:

```bat
dev.bat
```

Optional backend port:

```bat
dev.bat 3000
```

The script prepares the frontend, runs an initial Svelte static build, starts the Go backend in one command window, and starts the Vite dev server in another. Open `http://127.0.0.1:5173` in the browser. Svelte source changes hot reload through Vite without restarting the Go backend. Close the two spawned command windows to stop the development stack.

### Unix-like Shell

```sh
./dev.sh
```

## Production Build

Production builds are intentionally two-stage:

1. Install/check frontend dependencies.
2. Build Svelte static assets into `frontend/dist/`.
3. Compile Go so `frontend/dist/` is embedded into the executable.

Windows:

```bat
build.bat
```

Make:

```sh
make build
```

The final executable is written under `tmp/`. It should run without `frontend/`, `frontend/dist/`, or `public/` present on disk because the frontend assets are compiled into the binary.

To verify that property on Windows:

```bat
verify-build.bat
```

## Important Runtime Difference

Development and production serve the frontend differently:

- Development: `dev.bat` starts Vite; browser updates come from Vite HMR.
- Production: running the executable directly uses embedded static files captured during `go build`.

If you change Svelte source code and want those changes in the production executable, rerun the production build.

## Docker / Dokploy Deployment

The root `Dockerfile` builds the production image in three stages: install and build the Svelte frontend, compile the Go executable with embedded `frontend/dist`, then copy only the runtime binary into a slim image.

Local image build:

```sh
docker build -t go-svelte-starter .
```

Local run example:

```sh
docker run --rm -p 3000:3000 \
  -v "$(pwd)/data:/app/data" \
  -v "$(pwd)/logs:/app/logs" \
  -e APP_JWT_SECRET="<set-a-stable-secret>" \
  -e APP_INTEGRATION_MASTER_KEY="<set-a-stable-secret>" \
  go-svelte-starter
```

Dokploy application settings:

| Setting | Value |
| --- | --- |
| Build Type | `Dockerfile` |
| Dockerfile Path | `Dockerfile` |
| Docker Context Path | `.` |
| Docker Build Stage | leave empty |
| App Port | `3000` |

Persistent storage:

- Mount persistent storage to `/app/data` for SQLite files.
- Optionally mount `/app/logs` if you want file logs to survive redeploys.

Runtime environment:

- Set `APP_JWT_SECRET` to a stable secret so existing JWTs are not invalidated on every restart.
- Set `APP_INTEGRATION_MASTER_KEY` to a stable secret so encrypted integration values remain decryptable across restarts.
- Do not put runtime secrets into Docker build args.

## Frontend UI

The Svelte frontend uses Tailwind CSS and daisyUI. Prefer daisyUI component classes for common controls such as buttons, inputs, cards, alerts, modals, navbars, and loading states.

## API

The Go backend owns `/api/*` routes. The Vite development server proxies `/api` to the Go backend so frontend code can call the same relative URLs in development and production.
