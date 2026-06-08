# Frontend Development Guidelines

> 前端开发规范索引。标题和章节使用英文，正文尽量使用中文；代码、路径、函数名、字段名保持英文。

---

## Overview

前端位于 `frontend/`，是 Svelte + Vite SPA，使用 Tailwind CSS 和 daisyUI。开发模式下由 Vite 提供 HMR，并把 `/api` 代理到 Go 后端；生产模式下 `frontend/dist` 被 Go `embed` 进可执行文件。

---

## Guidelines Index

| Guide | Scope |
| --- | --- |
| [Svelte Vite Embed](./svelte-vite-embed.md) | Svelte/Vite 开发、Go embed、API client、dictionary/enum 约定 |

---

## Pre-Development Checklist

修改前端或前端服务行为前：

* 读 [Svelte Vite Embed](./svelte-vite-embed.md)。
* 判断变更影响开发模式、生产 embed 模式，还是两者都影响。
* 如果 Svelte 源码变更需要进入 Go 可执行文件，重新运行 production build。
* 如果内部 API envelope 或 DTO 变化，检查 `frontend/src/api.js` 和相关 Svelte 调用。

---

## Quality Check

```sh
cd frontend && npm test
cd frontend && npm run build
go test ./...
```

涉及打包脚本或可执行文件验证时：

```bat
build.bat
verify-build.bat
```
