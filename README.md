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

Runtime environment files:

- The executable automatically loads dotenv-style files at startup and only fills variables that are not already set by the operating system.
- Checked paths are `.env`, `data/.env`, `.env` and `data/.env` next to the executable, and the same two paths in the executable directory's parent. The parent lookup is useful when the Windows build output runs from `tmp/`.
- The format is `KEY=value`; blank lines and `#` comments are ignored, `export KEY=value` is accepted, and single or double quoted values are supported.
- For Docker/Dokploy, you can still set real environment variables or mount a persistent env file such as `/app/data/.env`.

Example `.env`:

```env
APP_PUBLIC_BASE_URL=http://127.0.0.1:5173
GOOGLE_OAUTH_CLIENT_ID=<google-client-id>
GOOGLE_OAUTH_CLIENT_SECRET=<google-client-secret>
GITHUB_OAUTH_CLIENT_ID=<github-client-id>
GITHUB_OAUTH_CLIENT_SECRET=<github-client-secret>
APP_JWT_SECRET=<set-a-stable-secret>
APP_INTEGRATION_MASTER_KEY=<set-a-stable-secret>
```

Runtime environment:

- Set `APP_JWT_SECRET` to a stable secret so existing JWTs are not invalidated on every restart.
- Set `APP_INTEGRATION_MASTER_KEY` to a stable secret so encrypted integration values remain decryptable across restarts.
- Set `APP_PUBLIC_BASE_URL` to the browser-facing origin used by OAuth callbacks. In local development this is usually `http://127.0.0.1:5173`; in production it is your public site URL.
- To enable Google OAuth, set `GOOGLE_OAUTH_CLIENT_ID` and `GOOGLE_OAUTH_CLIENT_SECRET`, then configure the provider callback URL as `<APP_PUBLIC_BASE_URL>/api/auth/oauth/google/callback`.
- To enable GitHub OAuth, set `GITHUB_OAUTH_CLIENT_ID` and `GITHUB_OAUTH_CLIENT_SECRET`, then configure the provider callback URL as `<APP_PUBLIC_BASE_URL>/api/auth/oauth/github/callback`.
- Do not put runtime secrets into Docker build args.

## Frontend UI

The Svelte frontend uses Tailwind CSS and daisyUI. Prefer daisyUI component classes for common controls such as buttons, inputs, cards, alerts, modals, navbars, and loading states.

## API

The Go backend owns `/api/*` routes. The Vite development server proxies `/api` to the Go backend so frontend code can call the same relative URLs in development and production.
