# Deployment Guidelines

> 本文定义后端生产部署、Dockerfile 构建、Dokploy 配置、运行时目录与环境变量的约定。前端生产产物如何被 Go embed 见 [Svelte Vite Embed](../frontend/svelte-vite-embed.md)。

---

## Scenario: Dockerfile Dokploy Deployment

### 1. Scope / Trigger

当修改以下内容时，必须对齐本节：

* 根目录 `Dockerfile`
* 根目录 `.dockerignore`
* Dokploy Dockerfile build 配置说明
* 生产镜像 runtime command、端口、数据目录、日志目录
* 运行时 env key，例如 `APP_JWT_SECRET`、`APP_INTEGRATION_MASTER_KEY`

### 2. Signatures

Dokploy application settings:

```text
Build Type: Dockerfile
Dockerfile Path: Dockerfile
Docker Context Path: .
Docker Build Stage: <empty>
App Port: 3000
```

Runtime command:

```text
/app/app -port 3000 -db /app/data/app.db -shared-db /app/data/shared.db
```

Runtime directories:

```text
/app/data
/app/logs
```

Important runtime env:

```text
APP_JWT_SECRET
APP_INTEGRATION_MASTER_KEY
```

Runtime env files:

```text
.env
data/.env
<executable-dir>/.env
<executable-dir>/data/.env
<executable-parent>/.env
<executable-parent>/data/.env
```

### 3. Contracts

* 根目录 `Dockerfile` 必须可以用 repo root 作为 build context 构建：`docker build -t go-svelte-starter .`。
* Dockerfile 必须使用 multi-stage build：先执行 frontend `npm ci` / `npm run build`，再执行 Go build，最终 runtime stage 只复制运行所需 binary。
* Go build 必须发生在 `frontend/dist` 已生成之后，因为 `static.go` 使用 `//go:embed frontend/dist`。
* runtime image 必须创建 `/app/data` 和 `/app/logs`，并使用 `/app/data/app.db`、`/app/data/shared.db` 作为 SQLite 路径。
* Dokploy 持久化至少挂载 `/app/data`；如果希望文件日志跨部署保留，可额外挂载 `/app/logs`。
* runtime image 应包含 CA certificates，保证 payment、LLM、SMS、email 等 provider HTTPS 调用可以验证证书。
* runtime image 使用固定非 root 用户 `10001:10001`；如果外部 bind mount 导致写入失败，应调整挂载目录权限或使用 Docker managed volume。
* 生产部署必须设置稳定的 `APP_JWT_SECRET` 和 `APP_INTEGRATION_MASTER_KEY`；不要把这些 secret 写进 Dockerfile、README 明文示例或 Docker build args。
* 可执行文件启动时会读取 dotenv-style env file，并且只填充进程里尚未存在的变量；系统/容器环境变量优先级高于 env file。
* env file 支持 `KEY=value`、`export KEY=value`、单引号/双引号值、空行和 `#` 注释。日志只允许记录 env file 路径和填充数量，不得记录具体值。
* 本地 Windows build 输出通常位于 `tmp/`，因此启动时会额外检查 executable parent 下的 `.env` 和 `data/.env`；Linux/Docker 部署可把 env file 放在二进制旁边或持久化的 `/app/data/.env`。
* `.env` 和 `data/.env` 必须被 git 忽略；只允许提交 `.env.example` 这类无真实 secret 的模板文件。
* `.dockerignore` 必须排除本地 runtime/build artifact，例如 `data/`、`logs/`、`tmp/`、`frontend/node_modules/`、`frontend/dist/`。

### 4. Validation & Error Matrix

| Condition | Expected behavior |
| --- | --- |
| `frontend/dist` 没有在 Docker build 中重新生成 | 构建应失败或修正 Dockerfile，不允许依赖本地 ignored dist |
| `.dockerignore` 把 `frontend/` 整体排除 | Docker build 失败；改为只排除 `frontend/node_modules/` 和 `frontend/dist/` |
| 未挂载 `/app/data` | 容器可启动，但 SQLite 数据会随容器生命周期丢失 |
| `/app/data` 对 UID `10001` 不可写 | 启动时数据库或日志初始化失败；修正 volume 权限 |
| 未设置 `APP_JWT_SECRET` | 容器可启动，但重启后旧 JWT 失效 |
| 未设置 `APP_INTEGRATION_MASTER_KEY` | 容器可启动，但重启后历史加密集成值可能无法解密 |
| runtime image 缺少 CA certificates | 外部 HTTPS provider 调用可能失败 |
| Dokploy `Docker Context Path` 不是 `.` | Dockerfile 可能无法复制 backend/frontend 所需文件 |

### 5. Good/Base/Bad Cases

Good: Dokploy 使用 `Dockerfile` build type，context 为 `.`，挂载 `/app/data`，运行时配置稳定的 `APP_JWT_SECRET` 和 `APP_INTEGRATION_MASTER_KEY`。

Base: 本地执行 `docker build -t go-svelte-starter .` 可以成功，并且 `docker run --rm go-svelte-starter -h` 能显示应用 flag。

Bad: 在 Dockerfile 中写入真实 secret、把 `data/*.db` 打进镜像、或让 runtime 从源码目录读取 `frontend/dist`。

### 6. Tests Required

修改 Dockerfile、`.dockerignore` 或 Dokploy 部署说明后，至少运行：

```sh
go test ./...
cd frontend && npm test
cd frontend && npm run build
docker build -t go-svelte-starter .
```

如果本地 Docker daemon 不可用，最终说明必须明确未运行 `docker build` 的原因。

### 7. Wrong vs Correct

#### Wrong

```dockerfile
COPY data/ /app/data/
COPY frontend/dist /app/frontend/dist
```

#### Correct

```dockerfile
RUN npm run build
COPY --from=frontend-build /src/frontend/dist ./frontend/dist
RUN go build -trimpath -ldflags="-s -w" -o /out/app .
CMD ["-port", "3000", "-db", "/app/data/app.db", "-shared-db", "/app/data/shared.db"]
```
