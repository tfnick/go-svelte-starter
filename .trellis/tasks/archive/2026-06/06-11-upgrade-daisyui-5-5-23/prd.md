# Upgrade DaisyUI To 5.5.23

## Goal

将前端 daisyUI 升级到 `5.5.23`，并按 daisyUI 5 / Tailwind CSS 4 的接入方式调整前端样式构建配置，确保现有 Svelte + Vite 页面可以继续通过测试和生产构建。

## What I Already Know

* 用户明确要求：将当前前端 daisyUI 版本升级到 `5.5.23`。
* 当前 `frontend/package.json` 使用 `daisyui: ^4.12.24` 和 `tailwindcss: ^3.4.17`。
* 当前 lockfile 解析到 `daisyui 4.12.24` 和 `tailwindcss 3.4.19`。
* 当前 daisyUI 接入在 `frontend/tailwind.config.cjs`：`plugins: [require('daisyui')]`，并设置 `themes: ['light']`。
* 当前 PostCSS 接入在 `frontend/postcss.config.cjs`：`tailwindcss` + `autoprefixer`。
* 当前 CSS 使用 Tailwind 3 指令：`@tailwind base/components/utilities`。
* `npm view daisyui@5.5.23` 确认目标版本存在，并且是当前 `latest`。
* `npm view tailwindcss@latest` 和 `npm view @tailwindcss/vite@latest` 当前均为 `4.3.0`。

## Research References

* [`research/daisyui-5-5-23-upgrade.md`](research/daisyui-5-5-23-upgrade.md) — 记录当前依赖、目标版本、Tailwind/daisyUI 5 接入影响和验证要求。

## Assumptions

* 本任务不只是改 `daisyui` 版本号；需要同步迁移到 Tailwind CSS 4 兼容接入方式。
* 保持现有 visual language，不主动重做页面样式。
* 继续只启用 `light` theme。
* 如果 Tailwind 4 / daisyUI 5 构建需要移除旧的 `tailwind.config.cjs` 或 `postcss.config.cjs`，应优先减少过时配置，而不是保留无效配置。
* 任何必要的依赖调整都限定在 `frontend/package.json` 与 lockfile。

## Open Questions

* None.

## Requirements

* 将 `frontend` 的 `daisyui` 依赖升级到 `5.5.23`。
* 按 daisyUI 5 / Tailwind CSS 4 要求同步调整 Tailwind CSS 相关依赖与 Vite/CSS 配置。
* 保持现有 Svelte 页面源码尽量不变，除非构建或 daisyUI 5 class 行为明确要求修改。
* 保持 `light` theme。
* 更新 lockfile。
* 不引入第二套 UI component library。

## Acceptance Criteria

* [x] `frontend/package.json` 中 `daisyui` 指向 `5.5.23`。
* [x] `frontend/package-lock.json` 解析到 `daisyui 5.5.23`。
* [x] Tailwind/daisyUI 构建配置符合 Tailwind CSS 4 + daisyUI 5 的接入方式。
* [x] `cd frontend && npm test` 通过。
* [x] `cd frontend && npm run build` 通过。
* [x] `go test ./...` 通过。
* [x] `git diff --check` 通过。

## Definition of Done

* 前端依赖升级和配置迁移完成。
* 相关测试和构建通过。
* 如发现新的前端样式系统约定，更新 `.trellis/spec/frontend/`。
* 任务提交并归档。

## Out of Scope

* 不重设计 UI。
* 不批量重构 Svelte 页面布局。
* 不新增其他 UI 库。
* 不改变后端 API。
* 不升级与本迁移无关的前端依赖，除非 npm 解析或 Tailwind 4 接入必须。

## Technical Approach

Recommended approach:

1. 使用 npm 更新 `daisyui@5.5.23`、Tailwind CSS 4 及必要的 Vite integration package。
2. 将 Tailwind/daisyUI plugin wiring 从 Tailwind 3 config/PostCSS 方式迁移到 Tailwind CSS 4 / Vite + CSS plugin 方式。
3. 保持 `light` theme 设置。
4. 运行前端测试、前端 build、后端测试和 diff check。

## Technical Notes

* Inspected files:
  * `frontend/package.json`
  * `frontend/package-lock.json`
  * `frontend/tailwind.config.cjs`
  * `frontend/postcss.config.cjs`
  * `frontend/vite.config.js`
  * `frontend/src/styles.css`
  * `.trellis/spec/frontend/index.md`
  * `.trellis/spec/frontend/svelte-vite-embed.md`
* The repo is clean at task creation time.

## Implementation Notes

* Upgraded frontend dependencies to:
  * `daisyui: ^5.5.23`
  * `tailwindcss: ^4.3.0`
  * `@tailwindcss/vite: ^4.3.0`
* Removed direct frontend devDependencies on `autoprefixer` and `postcss`.
* Removed obsolete Tailwind 3 config files:
  * `frontend/tailwind.config.cjs`
  * `frontend/postcss.config.cjs`
* Added `tailwindcss()` Vite plugin before `svelte()`.
* Replaced Tailwind 3 CSS directives with Tailwind 4 CSS import and daisyUI 5 CSS plugin block:

```css
@import "tailwindcss";
@plugin "daisyui" {
  themes: light --default;
}
```

* After the first migration pass, Login/Product pages still used daisyUI 4 form classes (`form-control`, `label-text`, `input-bordered`, `select-bordered`, `textarea-bordered`) whose v5 behavior left layouts visually broken.
* Updated `AuthCard.svelte`, `Login.svelte`, and `Products.svelte` to daisyUI 5 form structure: `fieldset`, `fieldset-legend`, default `input`/`select`/`textarea` with explicit `w-full`, and current v5 card/border tokens.
* Updated the Products list to use a daisyUI 5 table on desktop and compact product cards on narrow screens, with truncated Creem IDs, status badges, and selected-row highlighting.
* User requested commit after inspecting the Login/Product/Product list fixes.

## Verification

* `cd frontend && npm test` passed.
* `cd frontend && npm run build` passed and printed `daisyUI 5.5.23`.
* `go test ./...` passed.
* `git diff --check` passed.
* Static scan confirmed no tracked `frontend/tailwind.config.cjs` or `frontend/postcss.config.cjs` usage remains outside task research/PRD history.
* After Product list styling updates, `cd frontend && npm test`, `cd frontend && npm run build`, and `git diff --check` passed again.
