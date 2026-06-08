# Marketing Landing + SaaS SEO Architecture

## Goal

设计一个技术方案并进入实现准备：让当前 Svelte + Go 项目在保持单个可执行文件部署的前提下，支持 SEO 友好的营销网站，并与登录后的 SaaS 应用和 checkout 流程一体化。公开营销页使用 Go 原生 `html/template` 渲染，可选 htmx 做渐进增强；SaaS 后台继续使用现有 Svelte SPA。

## Requirements

* 公开营销站必须由服务端返回完整语义 HTML，不能继续让 `/` 返回空 SPA shell。
* SaaS 应用入口采用 `/app`，营销站和 SaaS 应用形成清晰边界。
* `/api/*` 是后端 API 的保留命名空间；不得因页面路由调整迁移、改名或被 marketing / SPA fallback 捕获。
* MVP 公开 SEO 页面范围包括 `/` landing page、`/pricing`、`/features`、`robots.txt`、`sitemap.xml`。
* 生产仍构建为一个 Go 可执行文件，Docker / Dokploy 仍只需运行该二进制。
* Landing 内容采用代码模板固定结构，并读取少量 site settings，例如站点名、logo、public base URL。后续换模板时以模板目录为替换单元，不引入 CMS。
* Marketing handler 使用 `html/template` 输出完整 HTML，包括 page title、meta description、canonical、Open Graph、JSON-LD 等基础 SEO 信息。
* 营销页必须与 checkout 打通。产品/套餐 CTA 至少携带本地 `product_id`，进入 `/app/checkout?product_id=...`。
* Checkout MVP 采用登录后继续购买：未登录用户先登录/注册，登录后保留所选 `product_id`，继续现有 `CreateOrder` + `CreateOrderPaymentCheckout` 流程并跳转 Creem checkout。
* htmx 仅作为渐进增强；无 JS 时营销内容和 CTA 仍可访问。

## Acceptance Criteria

* [ ] 访问 `/` 返回 server-rendered marketing HTML，响应体包含主标题、meta description、canonical 或 Open Graph 信息。
* [ ] 访问 `/pricing` 和 `/features` 返回 server-rendered marketing HTML，并拥有页面级 title / description / canonical。
* [ ] `robots.txt` 和 `sitemap.xml` 返回正确 content type，包含公开营销 URL，且不被 SPA fallback 捕获。
* [ ] 访问 `/app` 及其子路径返回嵌入的 Svelte SPA，并能进入登录/后台流程。
* [ ] `/api/*` 未匹配路径仍返回 API 404，不返回 marketing 或 Svelte HTML。
* [ ] 现有后端 API path 不因营销页或 SaaS base path 改造而迁移。
* [ ] 营销首页和 `/pricing` 的产品/套餐 CTA 能进入 `/app/checkout?product_id=...`。
* [ ] 未登录用户会先进入登录/注册，登录后保留所选 `product_id` 并继续创建订单、创建 Creem checkout、跳转 provider checkout URL。
* [ ] Marketing 模板可通过替换模板文件进行整体改版，不需要新增 DB/CMS 能力。
* [ ] 生产构建仍然通过 `build.bat` / `make build` / Dockerfile 生成可单独运行的可执行文件。
* [ ] `verify-build.bat` 或等价测试覆盖 marketing root、SPA route、API route 三类路径。

## Technical Approach

采用 Go-rendered marketing pages + `/app` Svelte SaaS boundary。

* 新增 Go 内嵌的 marketing templates，例如 `marketing/templates/*.html` 和 `marketing/assets/*`。
* 在 frontend SPA fallback 之前注册 marketing routes：`/`、`/pricing`、`/features`、`/robots.txt`、`/sitemap.xml`。
* 将 Svelte SaaS 应用放到 `/app` 边界下，由 `/app/*` 继续回退到 embedded Svelte `index.html`。
* 页面路由边界只影响浏览器访问的 HTML 页面；`/api/*` 继续由现有 Go API route group 处理。
* Marketing view model 由 Go handler 组装：站点设置、SEO metadata、启用的本地产品/套餐、checkout CTA URL。
* Marketing 模板文件只消费稳定 view model，避免把营销文案深埋到 Go handler，便于后续整套模板替换。
* SPA 新增 `/app/checkout` continuation：读取 `product_id`，若未登录则通过登录/注册保留 redirect；登录后调用现有受保护订单 API 并跳转 Creem。

## Decision (ADR-lite)

Context: 当前项目需要公开营销页获得 SEO 能力，同时保持 SaaS 应用和 Creem checkout 的现有登录态订单归属模型。现有 checkout API 位于受保护路由中，依赖登录用户、订单台账、会员权益发放和 webhook 回写。

Decision: 采用 Go-rendered marketing pages + `/app` Svelte SaaS boundary。营销页使用可替换的代码模板和少量 site settings，公开页面范围为 `/`、`/pricing`、`/features`、`robots.txt`、`sitemap.xml`。Checkout 首版采用登录后继续购买：CTA 携带本地 `product_id` 进入 `/app/checkout?product_id=...`，未登录则登录/注册后恢复该购买 intent。

Consequences: SEO 页面能返回完整 HTML，部署仍是单个 Go 可执行文件；checkout 复用现有安全边界，避免匿名订单和账号绑定复杂度。代价是需要迁移 Svelte 页面到 `/app` 边界，并在 SPA 中新增 checkout intent continuation。

## Implementation Plan

1. Route boundary: add marketing route registration before SPA fallback; make `/app/*` serve Svelte SPA; keep `/api/*` as hard API boundary.
2. Marketing templates: embed replaceable template/assets directory; render `/`, `/pricing`, `/features` with page metadata and checkout CTAs.
3. SEO endpoints: implement `robots.txt` and `sitemap.xml` with public marketing URLs and correct content types.
4. Checkout continuation: add `/app/checkout?product_id=...` SPA route/state handling; redirect unauthenticated users through login/register and resume checkout.
5. Build and docs: update tests, `verify-build.bat`, and README for marketing + SaaS one-binary deployment.

## Definition of Done

* Backend route/static tests cover public marketing pages, `/app` SPA fallback, and `/api/*` boundary.
* Frontend router/API tests cover `/app` aliases and checkout continuation.
* `go test ./...` passes.
* `cd frontend && npm test` passes.
* `cd frontend && npm run build` passes.
* Packaging verification is updated and passes.
* README or deployment docs explain marketing + SaaS one-binary deployment.

## Out of Scope

* 不迁移到 SvelteKit 或 Node SSR。
* 不实现完整 CMS、后台可视化页面编辑器或 Markdown 博客系统。
* 不实现多语言 SEO、复杂 A/B test。
* 不替换现有 Svelte SaaS UI。
* 首版不做匿名公开 checkout endpoint；guest order、邮箱归属、支付后账号绑定、防滥用控制留作后续增强。

## Technical Notes

Inspected files:

* `static.go` - 当前 embed `frontend/dist` 并对所有非 `/api/*` path 做 SPA fallback。
* `index.go` - API routes 和 frontend routes 的注册顺序；已有 `APP_PUBLIC_BASE_URL` 相关配置。
* `frontend/src/router.js` - 当前 `/` 是 Dashboard，多个 SaaS 页面为顶级路径。
* `frontend/src/pages/Dashboard.svelte` - 当前登录后 checkout flow：选择 product、创建 order、创建 Creem checkout、跳转 provider URL。
* `api/routes/order.go`、`api/usecase/order.go`、`api/usecase/payment.go` - checkout 依赖登录用户、本地 order、product、payment channel。
* `Makefile`、`build.bat`、`Dockerfile`、`verify-build.bat` - 单二进制生产构建与验证流程。
* `static_test.go` - 当前测试断言 SPA fallback 和 API boundary。

Research:

* [`research/marketing-landing-architecture.md`](research/marketing-landing-architecture.md)
