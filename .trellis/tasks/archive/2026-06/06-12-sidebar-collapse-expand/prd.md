# Add Sidebar Collapse Expand

## Goal

实现首页登录态应用壳的侧边栏折叠/展开功能，让桌面端用户可以在需要更大操作区时收起菜单，同时仍保留可识别的 icon 导航入口。移动端保持当前简洁 drawer 交互，不引入额外复杂布局。

## What I Already Know

* 用户希望创建一个小任务，实现 “侧边栏折叠/展开”（Sidebar Collapse/Expand）。
* 当前登录态布局由 `frontend/src/components/AppSidebar.svelte` 承载。
* 桌面端当前使用 daisyUI drawer 的 `lg:drawer-open`，侧边栏固定 `w-72`。
* 移动端当前通过顶部菜单按钮打开 drawer，体验已经比较简洁。
* 菜单项已经有 lucide icon，折叠后可以继续显示 icon。
* 左下角已有 profile、notification、tasks、logout 区域；折叠状态下需要避免文字挤压和面板溢出。

## Requirements

* 桌面端侧边栏支持折叠和展开。
* 展开状态保持现有视觉结构：logo、菜单 label、profile 信息、通知/任务/退出操作。
* 折叠状态侧边栏变窄，只保留关键 icon 操作：logo/home 入口、菜单 icon、通知、任务、退出。
* 折叠/展开按钮使用 lucide icon，放在侧边栏顶部区域，用户可以随时切换。
* 折叠状态下菜单项仍可点击导航，并通过 `aria-label` 保持可访问名称。
* 当前激活菜单在折叠和展开状态下都应清晰可见。
* 移动端继续使用当前 drawer，不需要新增折叠按钮或持久化状态。
* 不改变路由、权限、通知、任务中心的业务行为。

## Acceptance Criteria

* [x] 桌面宽度下侧边栏有折叠/展开按钮。
* [x] 展开状态下布局与当前侧边栏基本一致。
* [x] 折叠后主内容区域获得更多横向空间，侧边栏只显示 icon 级内容。
* [x] 折叠后菜单、通知、任务、退出仍可操作。
* [x] 折叠后没有文字溢出、按钮挤压或面板被侧边栏裁切的问题。
* [x] 手机端仍显示顶部栏 + drawer 菜单，不出现桌面折叠按钮。
* [x] `cd frontend && npm test` 通过。
* [x] `cd frontend && npm run build` 通过。

## Definition Of Done

* Implementation is scoped to frontend shell/sidebar behavior.
* Existing API and backend behavior remain untouched.
* Frontend quality gate passes.
* Task changes committed before finish-work/archive.

## Technical Approach

Recommended MVP:

* Add local `collapsed` state inside `AppSidebar.svelte`.
* Use responsive classes so collapse only affects `lg` desktop layout.
* Change desktop aside width between expanded `w-72` and collapsed narrow width such as `w-20`.
* Hide text labels and profile details in collapsed mode; keep icon buttons centered.
* Keep `NotificationCenter` and `TaskCenter` docked buttons available in the lower action area.
* Add a top collapse/expand icon button with `PanelLeftClose` / `PanelLeftOpen` from `lucide-svelte`.
* Do not persist collapsed preference in localStorage in this task; that can be added later if users ask for it.

## Decision

**Context**: The current sidebar already has a good mobile drawer and a new desktop left/right layout. The most useful improvement is giving desktop users more content width without changing navigation semantics.

**Decision**: Implement a local, desktop-only collapsed state in `AppSidebar.svelte`.

**Consequences**: The feature remains small and reversible. It improves workspace width immediately, while avoiding cross-session preference complexity and mobile layout churn.

## Open Questions

* None for MVP.

## Out Of Scope

* Persisting sidebar collapsed state across reloads.
* User preference settings for layout density.
* Reworking mobile drawer behavior.
* Changing route definitions or menu permissions.
* Backend/API changes.

## Technical Notes

* Main file: `frontend/src/components/AppSidebar.svelte`.
* Existing layout uses daisyUI drawer, Tailwind CSS 4, daisyUI 5, and lucide-svelte icons.
* Relevant frontend spec: `.trellis/spec/frontend/svelte-vite-embed.md`.

## Verification

* `cd frontend && npm test`
* `cd frontend && npm run build`
* `go test ./...`
* `git diff --check`
* Browser smoke test with temporary Vite + mock API:
  * Desktop 1280px: sidebar width changed from 288px to 80px after collapse; menu labels hidden; expand button visible.
  * Desktop collapsed panels: Notifications panel rendered at x=96 with width 320px; Tasks panel rendered at x=96 with width 384px.
  * Mobile 390px: drawer width stayed 288px; menu label/profile stayed visible; desktop collapse button stayed hidden.
