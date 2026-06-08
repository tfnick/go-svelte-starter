# Backend Development Guidelines

> 后端开发规范索引。标题和章节使用英文，正文尽量使用中文；代码、路径、类型名、函数名、字段名保持英文。

---

## Overview

后端代码位于 `api/`，当前采用轻量分层：

```text
routes -> usecase -> models -> db
```

`api/framework` 放可复用、业务无关的基础能力。业务逻辑不要放进 `framework`。新增功能时，先按本索引选择相关 spec；不要把同一条规则重复写进多个文件。

---

## Guidelines Index

| Guide | Scope |
| --- | --- |
| [Directory Structure](./directory-structure.md) | 后端目录、分层职责、导入边界、命名约定 |
| [Route Handler Guidelines](./route-handler-guidelines.md) | 内部 `/api/*` route handler 的标准流程 |
| [API Contracts](./api-contracts.md) | 内部 API envelope、DTO 边界、`Co -> DTO` 映射规则 |
| [Open API Guidelines](./open-api-guidelines.md) | 外部 `/open-api/v1/*` 契约、鉴权、公开 DTO、错误 envelope |
| [Database Guidelines](./database-guidelines.md) | `sqlx`、DB manager、migrations、transaction-aware executor |
| [UUID Generation](./uuid-generation.md) | UUID 生成约定，项目内默认使用 UUID v7 |
| [Error Handling](./error-handling.md) | usecase typed error、model sentinel error、安全错误响应 |
| [Logging Guidelines](./logging-guidelines.md) | Zerolog 初始化、文件日志、request logging、敏感字段控制 |
| [Eventing Guidelines](./eventing-guidelines.md) | Durable DDD event、goqite-backed delivery、subscriber idempotency |
| [Deployment Guidelines](./deployment-guidelines.md) | Dockerfile、Dokploy、生产运行时目录、环境变量 |
| [Quality Guidelines](./quality-guidelines.md) | 代码质量总则、禁止模式、测试和评审清单 |

---

## Pre-Development Checklist

开始后端开发前：

* 修改目录结构或跨层依赖时，读 [Directory Structure](./directory-structure.md)。
* 修改内部 `/api/*` handler 时，读 [Route Handler Guidelines](./route-handler-guidelines.md) 和 [API Contracts](./api-contracts.md)。
* 修改 `/open-api/v1/*` 时，读 [Open API Guidelines](./open-api-guidelines.md)。
* 修改 SQL、migration、事务或模型查询时，读 [Database Guidelines](./database-guidelines.md)。
* 新增或修改 UUID 生成点时，读 [UUID Generation](./uuid-generation.md)。
* 修改错误码、错误映射或异常处理时，读 [Error Handling](./error-handling.md)。
* 修改日志、request logging、API surface 字段时，读 [Logging Guidelines](./logging-guidelines.md)。
* 修改事件发布或订阅时，读 [Eventing Guidelines](./eventing-guidelines.md)。
* 修改 Dockerfile、dockerignore、Dokploy 部署或生产运行时 env/volume 时，读 [Deployment Guidelines](./deployment-guidelines.md)。
* 任意后端改动都应最后对照 [Quality Guidelines](./quality-guidelines.md)。

---

## Quality Check

常规后端检查：

```sh
go test ./...
```

涉及前端 API 客户端、嵌入构建或 envelope 变化时，同时运行：

```sh
cd frontend && npm test
cd frontend && npm run build
```

涉及 Dockerfile、dockerignore 或 Dokploy 部署说明时，同时运行：

```sh
docker build -t go-svelte-starter .
```

---

## Spec Writing Rule

后续更新 `.trellis/spec/` 时遵循：

* 文件名、文档标题、章节标题使用英文，便于检索和工具识别。
* 正文说明尽量使用中文，降低团队理解成本。
* 代码示例、路径、类型名、函数名、字段名、JSON key、SQL 保持英文。
* 同一规则只在一个主 spec 中展开；其他 spec 只做引用或一句话提醒。
