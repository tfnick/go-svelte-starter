# Dockerfile Dokploy Deployment

## Goal

为当前 Go + Svelte/Vite 项目新增 Dockerfile 构建能力，使 Dokploy 可以直接选择 Dockerfile build type，从仓库根目录构建生产镜像并部署运行。

## What I Already Know

* 用户希望创建任务：编写 `Dockerfile`，实现在 Dokploy 中基于 Dockerfile 构建并部署。
* 项目生产构建流程是先构建 `frontend/dist`，再由 Go `embed` 把前端产物打进可执行文件。
* `README.md` 现有生产构建说明使用 `make build`，最终产物在 `tmp/`。
* 后端入口 `index.go` 使用 flag：
  * `-port` 默认 `3000`
  * `-db` 默认 `data/app.db`
  * `-shared-db` 默认 `data/shared.db`
* 运行时会写入 SQLite 数据文件到 `data/`，日志文件到 `logs/app.log`。
* 关键运行时环境变量包括：
  * `APP_JWT_SECRET`
  * `APP_INTEGRATION_MASTER_KEY`
* Dokploy 官方 Dockerfile build type 支持 `Dockerfile Path`、`Docker Context Path`、可选 `Docker Build Stage`，并区分 build args、build secrets 和 runtime env。

## Requirements

* 新增根目录 `Dockerfile`，可在 Dokploy 中以 Dockerfile build type 直接构建。
* Dockerfile 使用 multi-stage build：
  * 前端 stage 使用 `npm ci` 和 `npm run build` 生成 `frontend/dist`。
  * Go build stage 编译生产二进制，并确保 `frontend/dist` 已被 embed。
  * runtime stage 只包含运行所需的二进制和目录，不携带源码、`node_modules` 或本地 DB 文件。
* 新增 `.dockerignore`，排除本地运行时文件和构建噪音：
  * `data/`
  * `logs/`
  * `tmp/`
  * `frontend/node_modules/`
  * `frontend/dist/`
  * VCS/editor/cache 文件
* runtime 镜像默认：
  * 工作目录为 `/app`
  * 暴露端口 `3000`
  * 创建 `/app/data` 和 `/app/logs`
  * 使用 `/app/data/app.db` 和 `/app/data/shared.db`
* 更新文档，明确 Dokploy 配置：
  * Build Type: `Dockerfile`
  * Dockerfile Path: `Dockerfile`
  * Docker Context Path: `.`
  * Docker Build Stage: 留空
  * App Port: `3000`
  * Persistent mount: `/app/data`
  * Optional persistent mount: `/app/logs`
  * Runtime env: `APP_JWT_SECRET`、`APP_INTEGRATION_MASTER_KEY`
* 不把任何密钥写进 Dockerfile、README 示例或 build args。

## Acceptance Criteria

* [ ] 根目录存在 `Dockerfile`，本地可执行 `docker build -t go-svelte-starter .`。
* [ ] 构建过程会运行前端 `npm ci && npm run build`，然后运行 Go build。
* [ ] 镜像运行后监听 `3000`，并使用 `/app/data` 存储 SQLite 文件。
* [ ] `.dockerignore` 不会把本地 DB、日志、`node_modules`、`frontend/dist` 打进 build context。
* [ ] README 包含 Dokploy Dockerfile 部署配置说明。
* [ ] `go test ./...` 通过。
* [ ] `cd frontend && npm test` 和 `cd frontend && npm run build` 通过。

## Definition Of Done

* Dockerfile 与 `.dockerignore` 已提交。
* README 或部署文档已更新。
* 本地质量检查通过。
* 若本地环境支持 Docker，则验证至少一次 `docker build`；如果无法运行 Docker，需在最终说明中明确未验证的原因。

## Technical Approach

推荐使用根目录 multi-stage Dockerfile：

```text
frontend-build -> go-build -> runtime
```

Dockerfile 需要优先复制 `frontend/package*.json` 以提高依赖层缓存命中，再复制前端源码构建 `frontend/dist`。Go build stage 再复制 Go module 文件下载依赖，复制源码，并从前端 stage 拿到 `frontend/dist` 后编译。

runtime stage 使用 slim Linux 镜像，设置 `WORKDIR /app`，复制二进制，创建 `data` 与 `logs` 目录，`EXPOSE 3000`，并用 exec-form entrypoint/cmd 启动应用。

## Decision (ADR-lite)

**Context**: Dokploy 支持 Nixpacks/Railpack/Dockerfile 等 build type，但用户明确希望基于 Dockerfile 构建部署。项目又要求生产二进制 embed 前端产物，因此需要可控的 multi-stage build。

**Decision**: 使用根目录 Dockerfile + 根目录 build context (`.`)，由 Dockerfile 显式完成前端构建和 Go 编译，不引入 Docker Compose 或外部镜像仓库流程。

**Consequences**: Dokploy 配置简单，构建过程可复现；镜像构建会消耗一定 CPU/RAM，后续如生产构建压力较大，可再扩展为 CI 预构建镜像并让 Dokploy 拉取镜像。

## Out of Scope

* 不引入 Docker Compose。
* 不配置远程镜像仓库、CI/CD push image 流程。
* 不迁移数据库到外部 PostgreSQL。
* 不修改业务代码或 API 行为。
* 不修改日志模块实现，只在部署文档说明 `/app/logs` 是否挂载。

## Research References

* [`research/dokploy-dockerfile-build.md`](research/dokploy-dockerfile-build.md) - Dokploy Dockerfile build 字段、Docker context、runtime persistence 的项目映射。

## Technical Notes

* `go.mod` 当前声明 `go 1.25.0`，Dockerfile 的 Go builder 镜像应匹配 Go 1.25 系列。
* `frontend/package-lock.json` 存在，应使用 `npm ci` 而不是 `npm install`。
* `frontend/dist` 是构建产物，Docker build 应重新生成，不应依赖本地已有文件。
* `.dockerignore` 排除 `data/` 之后，运行时 SQLite 数据必须依靠容器内新建目录或 Dokploy persistent mount。
