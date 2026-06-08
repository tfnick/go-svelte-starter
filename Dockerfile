# syntax=docker/dockerfile:1

FROM node:22-bookworm-slim AS frontend-build

WORKDIR /src/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

FROM golang:1.25-bookworm AS go-build

WORKDIR /src

ENV CGO_ENABLED=0 \
    GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
COPY --from=frontend-build /src/frontend/dist ./frontend/dist

RUN go build -trimpath -ldflags="-s -w" -o /out/app .

FROM debian:bookworm-slim AS runtime

WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 app \
    && useradd --uid 10001 --gid app --home-dir /app --shell /usr/sbin/nologin app \
    && mkdir -p /app/data /app/logs \
    && chown -R app:app /app

COPY --from=go-build --chown=app:app /out/app /app/app

USER app

EXPOSE 3000

ENTRYPOINT ["/app/app"]
CMD ["-port", "3000", "-db", "/app/data/app.db", "-shared-db", "/app/data/shared.db"]
