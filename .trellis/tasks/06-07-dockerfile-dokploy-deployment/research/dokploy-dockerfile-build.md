# Dokploy Dockerfile Build Research

## Sources

* [Dokploy Build Type documentation](https://docs.dokploy.com/docs/core/applications/build-type)
* [Dokploy Going Production documentation](https://docs.dokploy.com/docs/core/applications/going-production)
* [Docker build context documentation](https://docs.docker.com/build/building/context/)

## Findings

Dokploy supports `Dockerfile` as an application build type. The relevant UI fields are:

* `Dockerfile Path`: required path to the Dockerfile, for root-level Dockerfile use `Dockerfile`.
* `Docker Context Path`: build context path, for this repo use `.`.
* `Docker Build Stage`: optional target stage. For a multi-stage Dockerfile whose final stage is the runtime image, leave this empty unless a specific stage is required.

Dokploy separates build-time configuration from runtime configuration:

* Build Time Arguments are passed as Docker `ARG`; they should not contain secrets.
* Build-time Secrets should be used for sensitive build-only values.
* Runtime environment variables are configured separately in Dokploy's Environment tab.

Docker build context controls what files can be copied by `COPY`. Because this project needs both `frontend/` and Go backend files, the context should remain repo root (`.`), and `.dockerignore` should exclude runtime/local artifacts instead of moving the context to a subdirectory.

## Project Mapping

This repo production build is already two-stage at the application level:

1. `frontend/package-lock.json` and `frontend/package.json` support deterministic `npm ci`.
2. `npm run build` writes static assets to `frontend/dist`.
3. `go build .` embeds `frontend/dist` into the Go executable.

Runtime requirements discovered from code:

* App listens on `-port`, default `3000`.
* SQLite paths default to `data/app.db` and `data/shared.db`.
* File logs default to `logs/app.log`.
* Persistent Dokploy mounts should cover `/app/data` and optionally `/app/logs`.
* Important runtime env vars are `APP_JWT_SECRET` and `APP_INTEGRATION_MASTER_KEY`.

## Recommended Direction

Use one root-level multi-stage Dockerfile:

* `frontend-build` stage: Node LTS image, `npm ci`, `npm run build`.
* `go-build` stage: Go image matching `go.mod`, copy frontend dist and Go sources, build executable.
* `runtime` stage: slim Debian runtime, create `/app/data` and `/app/logs`, expose `3000`, run executable with explicit DB paths.

Add `.dockerignore` to keep local DB files, logs, `frontend/node_modules`, `frontend/dist`, `tmp`, and VCS/editor artifacts out of the Docker context.

Add README/Dokploy instructions:

* Build Type: `Dockerfile`
* Dockerfile Path: `Dockerfile`
* Docker Context Path: `.`
* Docker Build Stage: empty
* Port: `3000`
* Persistent storage: mount to `/app/data`; optionally `/app/logs`
* Runtime env vars: `APP_JWT_SECRET`, `APP_INTEGRATION_MASTER_KEY`
