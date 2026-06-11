# Optimize Marketing Website

## Goal

优化 `marketing` 营销官网，让首页、功能页、价格页不仅更现代，而且更明确地推动目标行为：访客理解产品价值，选择一个计划，并进入 `/app/checkout` 或 `/app` 继续转化。底层行为设计遵从 Fogg Behavior Model：Behavior = Motivation + Ability + Prompt。

## What I Already Know

* 用户明确要求创建新任务，优化 `marketing` 营销官网。
* 底层理论必须遵从福格行为模型 `B=MAP`。
* 设计样式可以参考网上前沿布局和样式。
* 当前 marketing 页面由 Go 服务端动态渲染 HTML，不是 Vite 构建期生成静态页面。
* 当前相关文件主要是 `marketing.go`、`marketing/templates/*.html`、`marketing/assets/marketing.css`、`static_test.go`。
* 当前页面已有 `/`、`/pricing`、`/features`、`/robots.txt`、`/sitemap.xml`，并且价格卡片来自启用且配置了 Creem Product ID 的产品目录。

## Assumptions

* 本任务主要优化公开营销页，不重构 Svelte `/app` 内部应用。
* 继续使用 Go 标准库 `html/template`，保留服务端渲染和自动 HTML 转义。
* 页面需要保持单二进制部署模式：模板和 marketing CSS 通过 Go `embed` 打包。

## Open Questions

* 当前开放问题已收敛，等待最终确认后进入实现。

## Requirements

* MVP 明确偏向高转化落地页：优先推动访客选计划并进入 checkout，品牌叙事服务于转化，不做大篇幅形象官网。
* 首页首屏主 CTA 指向 `/pricing`，让访客先选择计划，再从具体产品卡进入 `/app/checkout?product_id=...`。
* 信任/证明模块使用真实“产品内证明”：SSR SEO、embedded Svelte app、catalog-backed pricing、checkout continuation、auth、events、one binary deploy 等，不编造客户 logo、案例或指标。
* 页面信息架构要显式覆盖 B=MAP：
  * Motivation：更清楚表达痛点、收益、可信度、结果。
  * Ability：降低行动门槛，展示清晰步骤、低风险路径、价格/产品选择。
  * Prompt：提供明确、及时、上下文相关的 CTA。
* 首页应优先服务单一主要行为，避免多个同级 CTA 互相稀释。
* 价格页应继续使用真实产品目录生成 public offer 卡片。
* 功能页应从“功能清单”升级为“结果/场景导向”的解释结构。
* SEO 元信息、canonical、Open Graph、JSON-LD、robots、sitemap 继续可用。
* 移动端布局必须清晰、无文字溢出、CTA 可点击。

## Acceptance Criteria

* [ ] `/` 返回服务端渲染 HTML，且不是 Svelte SPA `index.html`。
* [ ] 首页首屏有一个明确主 CTA 指向 `/pricing`，且文案服务于“选择计划”。
* [ ] `/pricing` 能展示真实产品卡片或空状态。
* [ ] `/features` 有清晰的信息层级和 CTA。
* [ ] 页面包含基于真实产品能力的 proof/trust 模块，不出现虚构客户 logo、案例或数据。
* [ ] 页面内容结构能映射到 Motivation / Ability / Prompt。
* [ ] `go test ./...` 通过。
* [ ] 如 CSS/模板变动影响视觉，使用本地浏览器或截图验证桌面和移动端布局。

## Definition of Done

* Tests added/updated where behavior is covered today.
* Lint / typecheck / build checks relevant to changed layers are green.
* Docs/notes updated if a durable convention is discovered.
* Rollback risk is low because changes remain isolated to marketing templates/CSS unless explicitly expanded.

## Out of Scope

* 不改造 `/app` 内部 Svelte dashboard 的业务功能。
* 不引入重型前端框架或客户端 hydration。
* 不切换到 `fasttemplate`，除非后续明确决定放弃 `html/template` 的结构化模板和自动转义能力。
* 不实现 A/B 测试、埋点分析或外部营销工具集成，除非后续纳入范围。
* 不做以品牌展示为主的大型官网重写；本次内容和视觉优先服务转化。

## Technical Notes

* `marketing.go` 使用 `//go:embed marketing/templates/*.html marketing/assets/*` 嵌入模板与 CSS。
* `marketingRenderer.loadTemplates()` 使用 `html/template.ParseFS` 解析所有 marketing 模板。
* `renderPage()` 在请求时执行模板并返回 `text/html; charset=utf-8`。
* `registerMarketingRoutes()` 注册 `/`、`/pricing`、`/features`、`/robots.txt`、`/sitemap.xml`、`/marketing/assets/*`。
* `registerFrontendRoutes()` 只服务 `/app` 和 `/assets` 等 SPA 相关路径。
* `static_test.go` 已覆盖 marketing root 不应返回 Svelte app、SEO endpoints、frontend app route 等行为。

## Research References

* `research/fogg-behavior-model.md` - B=MAP 在营销官网中的页面策略映射。
* `research/saas-landing-design-trends.md` - 2026 SaaS 落地页布局和视觉趋势摘要。

## Research Notes

### What Similar Sites And Design References Suggest

* SaaS landing pages increasingly use product-led hero sections: clear outcome, visible product/system preview, and a focused CTA.
* CTA should be treated as part of the behavior chain, not just a button: repeat one main action at the right moments instead of scattering unrelated actions.
* Bento grids remain a strong fit for feature/value sections because they force visual hierarchy and collapse cleanly on mobile.
* For this repo, "real product preview" should mean an honest system/product panel about SSR marketing, embedded Svelte, checkout, auth, products, and events, not a fake screenshot.

### Feasible Approaches

**Approach A: Product-Led Bento SaaS** (Recommended)

* How: rebuild the home/features/pricing content around one conversion path, with a product/system hero preview, B=MAP bento sections, clearer pricing CTAs, and final CTA.
* Pros: modern SaaS look, strong conversion fit, works with static server-rendered templates and CSS.
* Cons: needs disciplined hierarchy so modules do not become decorative clutter.
* Decision: selected for MVP, with high-conversion landing page emphasis.

## Decision (ADR-lite)

**Context**: 首页首屏需要一个主行为提示，避免 `/pricing`、`/app`、`/features` 多个目标并列导致转化稀释。

**Decision**: 首页首屏主 CTA 指向 `/pricing`。价格页承接产品目录，并由具体产品卡片触发 `/app/checkout?product_id=...`。

**Consequences**: 转化路径多一步，但降低了用户过早进入应用或 checkout 的认知成本；也更适合动态产品目录和多计划场景。

**Context**: 高转化页面需要提升 Motivation，但当前没有真实客户 logo、案例或量化数据。

**Decision**: 使用产品内证明作为 trust/proof：展示项目已经具备的 SSR SEO、embedded app、checkout、auth、events、single binary deploy 等能力。

**Consequences**: 可信度来自真实功能，不会制造虚假社会证明；如果未来有真实客户或指标，可以把 proof 模块扩展为案例/数据模块。

## Technical Approach

* 保持 Go `html/template` 服务端渲染和现有路由，不引入客户端 JS。
* 主要修改 `marketing/templates/layout.html`、`home.html`、`features.html`、`pricing.html` 和 `marketing/assets/marketing.css`。
* 首页结构建议：
  * Hero：高转化 headline、主 CTA 到 `/pricing`、次 CTA 降级为文本链接或轻量按钮。
  * Ability path：用 3 步说明从选择计划到 checkout/app 的路径。
  * Proof bento：展示真实产品内证明，提升 Motivation。
  * Pricing preview：复用 `product-list`，把 prompt 放在具体产品卡。
  * Final CTA：再次指向 `/pricing`。
* Pricing 页结构建议：
  * 更短的行动型 hero。
  * 产品卡更强调 plan selection 和 checkout prompt。
  * 空状态继续引导到 `/app/products`。
* Features 页结构建议：
  * 从功能列表改为结果/场景导向，末尾回到 `/pricing`。
* CSS 采用 Product-Led Bento SaaS 方向：清晰层级、克制现代视觉、多色中性 palette、卡片半径不超过 8px、移动端优先验证。

## Implementation Plan

* PR1: 更新模板内容和结构，确保 B=MAP 和 CTA 路径清晰。
* PR2: 重写 marketing CSS 视觉系统，验证桌面和移动端布局。
* PR3: 更新/补充 tests，运行 `go test ./...`，必要时做浏览器截图检查。

**Approach B: Editorial Founder-Tool Page**

* How: use larger narrative blocks, restrained typography, and fewer cards to explain "one binary SaaS starter" as a strong product story.
* Pros: distinctive and readable, good for trust.
* Cons: less immediately product-like; CTAs need extra support.

**Approach C: Techno-Futurist Dashboard**

* How: dark/high-contrast interface-inspired sections with dense system panels and stronger technical mood.
* Pros: developer-oriented and visually punchy.
* Cons: higher risk of generic dark SaaS look and one-note palette.
