# Bitbucket Build Analysis

## Summary

The project can still be packaged after moving the repository to Bitbucket if the migration only changes the Git remote and keeps the existing Go module path (`github.com/tfnick/go-svelte-starter`). The production build flow is local-source based and does not depend on the repository host.

If the migration also changes `go.mod` to a Bitbucket module path, packaging will not work until all internal imports and architecture guard constants are updated consistently.

Separate from Bitbucket, the current working tree now has an unrelated Go compilation failure in `marketing.go`. That must be fixed before this exact worktree can package on any repository host.

## Current Build Flow

The production build has three steps:

1. Build frontend assets with `npm run build` in `frontend/`.
2. Ensure `frontend/dist/index.html` exists.
3. Run `go build` from the repository root so `static.go` embeds `frontend/dist`.

Relevant files:

* `static.go` embeds `frontend/dist` with `//go:embed frontend/dist`.
* `build.bat` runs `npm run build`, checks `frontend/dist/index.html`, then runs `go build -a`.
* `Makefile` runs `npm ci`, `npm run build`, then `go build -a`.
* `Dockerfile` builds frontend assets in a Node stage, downloads Go modules in a Go stage, copies `frontend/dist`, then runs `go build`.
* `.dockerignore` excludes local `frontend/dist`, but the Dockerfile rebuilds and copies it from the frontend build stage, so this is expected.

## Verification Results

Environment observed locally:

* Go: `go1.25.1`
* Node: `v22.19.0`
* npm: `11.6.2`
* Docker CLI: `28.3.3`
* Go proxy: `https://proxy.golang.org,direct`
* `GOPRIVATE` and `GONOSUMDB` are unset locally.

Commands run:

```text
npm run build
```

Result: passed. Vite produced `frontend/dist/index.html`, CSS, and JS assets.

```text
go test ./...
```

Result: passed.

```text
go build -a -o tmp\bitbucket-analysis\svelte-go-starter.exe .
```

Result: passed. Produced `tmp\bitbucket-analysis\svelte-go-starter.exe`.

Additional runtime verification:

```text
embedded binary verification passed
```

Result: the generated binary could start from `tmp\bitbucket-analysis` and serve the embedded Svelte application without relying on `frontend/dist` at runtime.

Docker image build before the unrelated marketing-site working-tree change:

```text
docker build -t go-svelte-starter-bitbucket-analysis .
```

Result: passed. The Dockerfile successfully rebuilt frontend assets, copied them into the Go build stage, and produced a runtime image.

## Current Worktree Build Blocker

After the initial Bitbucket migration analysis, the worktree contained unrelated marketing-site changes:

```text
 M index.go
 M static.go
?? marketing.go
?? marketing/
```

With those changes present, the frontend still builds:

```text
npm run build
```

Result: passed.

But Go compilation fails:

```text
go test ./...
go build -a -o tmp\bitbucket-analysis\svelte-go-starter-current.exe .
docker build -t go-svelte-starter-bitbucket-analysis-current .
```

Result: failed in the root Go package:

```text
marketing.go:110:29: field and method with the same name templates
marketing.go:33:2: other declaration of templates
marketing.go:115:3: cannot assign to r.templates
marketing.go:117:9: cannot use r.templates as *template.Template
```

This blocker is not caused by Bitbucket migration. The same source tree will fail on GitHub, Bitbucket, local Windows builds, and Docker builds until the `marketingRenderer` field/method naming conflict is fixed.

## Repository Host Dependencies

No build script inspected here references GitHub or Bitbucket directly. The current Git remote is:

```text
origin git@github.com:tfnick/go-svelte-starter.git
```

Changing that remote to a Bitbucket SSH/HTTPS URL does not by itself change Go compilation, frontend compilation, or Docker build behavior.

## Go Module Path Risk

The module path in `go.mod` is:

```text
module github.com/tfnick/go-svelte-starter
```

Internal imports use that module path across the codebase. A search found 139 Go files containing `github.com/tfnick/go-svelte-starter`.

The architecture guard test also hard-codes the module path:

```text
api/framework/archguard/layer_boundary_test.go
const modulePath = "github.com/tfnick/go-svelte-starter"
```

This means:

* Safe path: move the repository to Bitbucket but keep `go.mod` as `github.com/tfnick/go-svelte-starter`. The project can package.
* Rename path: change `go.mod` to something like `bitbucket.org/<workspace>/<repo>`. This requires a coordinated rewrite of internal imports and the arch guard constant before tests and builds are expected to pass.

Go module paths do not have to match the Git remote for a main application that is built from checked-out source. They do matter when other modules import this project by module path, or when CI checks out the project and then runs tests that expect internal imports to match `go.mod`.

## Bitbucket-Specific CI Notes

If using Bitbucket Pipelines, the build should install or use:

* Go 1.25.x or a compatible Go image.
* Node 22.x and npm.
* Network access to `proxy.golang.org`, npm registry, and any direct module hosts.
* Docker-in-Docker only if building the production Docker image inside Pipelines.

For private Bitbucket modules, configure `GOPRIVATE` and Git credentials in CI. This project currently depends on public modules, so the main observed private-module risk is future dependencies rather than the present build.

## Conclusion

Yes, the repository can still package after migration to Bitbucket if only the Git remote/repository host changes and the source tree itself is buildable.

Do not change `go.mod` to a Bitbucket module path unless you also plan a full module rename. That rename is mechanical but broad, because internal imports and architecture tests are coupled to the current GitHub-style module path.

For the current working tree specifically, packaging is blocked by the local `marketing.go` compilation error, independent of Bitbucket.

## Recommended Migration Checklist

1. Create/import the Bitbucket repository.
2. Change Git remote to the Bitbucket URL.
3. Keep `go.mod` unchanged for the first migration.
4. In Bitbucket CI, run:
   * `cd frontend && npm ci && npm run build`
   * `go test ./...`
   * `go build -a -o tmp/svelte-go-starter .`
5. If Docker deployment is needed, run `docker build -t go-svelte-starter .`.
6. Only after CI is green, decide separately whether the module path should be renamed.
