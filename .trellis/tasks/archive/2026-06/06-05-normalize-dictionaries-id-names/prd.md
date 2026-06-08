# 规范前后端字典和 ID 名称转换

## Goal

建立一套前后端统一遵守的字典、枚举、ID 到名称展示规则，避免同一含义在前端、后端、页面组件中重复翻译，减少展示不一致和 N+1 查询风险。最终让静态枚举、动态字典、关联 ID 名称字段各自有明确归属，并能通过代码结构和测试防止回退。

## What I Already Know

* 用户明确要求：静态枚举由前端 TS 维护，Label 和 Options 从同一个定义生成。
* 用户明确要求：动态字典由后端接口提供，前端使用全局 Store 缓存，并支持批量拉取。
* 用户明确要求：ID 到 Name 的转换必须在后端 DTO 层组装，前端直接展示，永远不做翻译。
* 用户明确要求：DTO 命名约定是有 `xxxId` 就必须附带 `xxxName`。
* 用户明确要求：禁止 N+1，关联查询一律使用批量 `IN` 查询加内存 `Map` 组装。
* 用户补充要求：ID 到 Name 除了不能出现 N+1 性能问题，还应尽量抽象成可复用模式，避免每个接口重复手写完整组装流程。
* 当前项目后端内部 `/api/*` 资源响应已有显式 DTO 约束，DTO 主要位于 `api/routes/*_responses.go`。
* 当前项目订单响应已有 `user_id`、`product_id`、`status` 等字段，可作为本需求的落地示例。
* 当前前端是 `frontend/` 下的 Svelte + Vite SPA，当前 API 边界在 `frontend/src/api.js`。
* 当前数据库访问使用 `sqlx`，固定 SQL 使用 `db.GetDB(name)` + `Rebind()`，动态 SQL可使用 `db.GetEngine(name)`。

## Requirements

### Static Enums

* 静态枚举只在前端维护，使用 TypeScript 定义。
* 每个静态枚举必须由单一数据源生成：
  * `label(value)` 或等价 label resolver。
  * `options` 或等价表单/筛选选项列表。
* Svelte 组件不得重复手写同一枚举的 label 文案或 option 列表。
* 静态枚举适用于发布时固定、无需后端配置或数据库驱动的取值，例如订单状态显示文案等。

### Dynamic Dictionaries

* 动态字典由后端提供接口，前端不得在本地维护动态字典内容。
* 后端需支持批量拉取多个字典类型，避免页面初始化时为每个字典单独请求。
* 前端需提供全局 Store 缓存动态字典结果。
* Store 应复用已缓存字典，避免组件重复请求同一字典。
* 动态字典适用于数据库、后台配置、租户配置或运行期可能变化的取值。

### ID To Name Display

* ID 到名称展示一律由后端 DTO/Response 层组装。
* 前端拿到 `xxxName` 后直接展示，禁止根据 `xxxId` 再查表、翻译、映射或调用字典 Store。
* 只要 DTO/Response 中出现 `xxxId`，就必须同时提供对应的 `xxxName`，除非该字段明确不用于展示且在 PRD/技术说明中说明原因。
* 命名约定：
  * Go DTO 字段使用 `XXXID` / `XXXName`，JSON 使用 `xxx_id` / `xxx_name`。
  * 如果未来前端 DTO 层转 camelCase，则必须保持 `xxxId` / `xxxName` 成对出现。
* 示例：订单项有 `product_id` 时，响应中应附带 `product_name`；订单有 `user_id` 且页面展示用户时，响应中应附带 `user_name`。

### N+1 Prevention

* 禁止在列表组装中对每一行调用一次关联查询。
* 所有关联名称组装必须使用批量 `IN` 查询一次性取回关联数据，再用内存 `map[id]name` 组装 DTO。
* 后端应为常见关联提供批量查询函数，例如按一组用户 ID 查询用户名称、按一组产品 ID 查询产品名称。
* 批量查询函数应去重输入 ID，并能处理空输入。
* 对缺失关联名称的处理必须明确：推荐返回空字符串并保留 ID，同时由后端日志或测试覆盖异常场景。

### Reusable ID Name Assembly Pattern

* ID 到 Name 组装应沉淀成一个可复用后端模式，而不是每个 handler 零散手写完整流程。
* 推荐抽象的复用点包括：
  * 从一组模型/DTO 输入中收集并去重关联 ID。
  * 根据关联类型调用批量加载函数。
  * 将批量查询结果转换为 `map[id]name`。
  * 在显式 DTO mapper 中按字段写入 `xxx_name`。
* 复用模式必须保持字段映射显式可读，禁止使用反射或通用魔法 mapper 自动复制/翻译 DTO 字段。
* 可接受的抽象形式包括小型 helper、resolver、assembler 或 loader，命名需体现业务关联，例如 user name resolver、product name resolver。
* 新增关联名称时应优先复用该模式，只为具体资源补充收集函数、批量查询函数和显式 DTO mapper。

## Acceptance Criteria

* [ ] 前端新增静态枚举模块，至少包含一个真实枚举示例，并从同一份定义导出 label resolver 和 options。
* [ ] 前端相关 Svelte 页面或 API 消费代码不再重复手写该枚举的 label/options。
* [ ] 后端新增动态字典批量接口，能一次请求多个字典类型并返回统一结构。
* [ ] 前端新增动态字典全局 Store，支持批量加载、缓存复用和避免重复请求。
* [ ] 至少一个现有响应 DTO 按 `xxx_id` + `xxx_name` 规则改造，并由后端组装名称。
* [ ] DTO 层测试覆盖：有展示用途的 `xxx_id` 字段必须附带对应 `xxx_name` 字段。
* [ ] 关联名称组装测试覆盖批量 `IN` 查询路径，禁止通过每行单独查询实现。
* [ ] ID 到 Name 组装有可复用后端模式，并至少被一个真实 DTO 组装场景使用。
* [ ] 新的复用模式保持字段映射显式，不使用反射或通用魔法 mapper。
* [ ] 前端不根据 ID 自行翻译名称，相关展示直接使用后端返回的 `xxx_name`。
* [ ] `go test ./...` 通过。
* [ ] `cd frontend && npm run build` 通过。

## Definition Of Done

* 后端 API、DTO、模型查询 helper、测试已更新。
* 前端 Svelte/TypeScript 枚举模块、字典 Store、API helper、相关页面消费已更新。
* PRD 中的静态枚举、动态字典、ID 到 Name 三类边界在代码中都有示例落地。
* 新规则沉淀到 `.trellis/spec/`，便于后续任务复用。
* 构建和测试通过。

## Out Of Scope

* 不要求一次性迁移所有历史 API，只要求先落地基础设施和至少一个真实示例。
* 不要求引入复杂 i18n 系统；本任务只规范枚举 label、字典 label、关联名称展示归属。
* 不要求前端用 ID 反查名称的历史写法在所有未来页面中自动检测，除非实现中选择增加轻量测试或 lint guard。
* 不要求实现一个跨所有资源的通用 ORM/反射式 DTO 组装框架；本任务只要求提炼可读、可复用的 ID 到 Name 组装模式。
* 不要求动态字典后台管理页面。

## Technical Approach

推荐按三层落地：

1. 前端静态枚举：新增 `frontend/src/enums/`，用 TypeScript 定义枚举元数据，再从同一元数据生成 `label` 和 `options`。
2. 动态字典：新增后端批量字典接口，例如 `GET /api/dictionaries?types=a,b` 或 `POST /api/dictionaries/batch`；前端新增 `frontend/src/stores/dictionaries.ts` 缓存结果。
3. ID 名称组装：后端在 route response DTO 组装层使用可复用模式批量收集关联 ID，调用批量查询 helper，使用内存 map 生成 `xxx_name` 字段。

## Decision (ADR-lite)

**Context**: 字典和 ID 名称展示如果散落在前端组件、后端模型和页面逻辑中，会造成展示不一致、重复请求和 N+1 查询。

**Decision**: 按数据性质划分归属：静态枚举归前端 TS，动态字典归后端接口和前端全局 Store，ID 到 Name 归后端 DTO/Response 层。

**Consequences**: 前端展示更简单，但后端 DTO 组装需要承担关联名称查询；因此必须同时引入可复用的 ID 到 Name 组装模式、批量查询 helper 和测试，防止 N+1。

## Open Questions

* MVP 示例优先落在哪个业务对象上：订单 `user_id/product_id/status`，还是新增一个更小的示例资源？

## Technical Notes

* Existing backend DTO guidance: `.trellis/spec/backend/api-contracts.md`
* Existing route handler guidance: `.trellis/spec/backend/route-handler-guidelines.md`
* Existing database guidance: `.trellis/spec/backend/database-guidelines.md`
* Existing frontend guidance: `.trellis/spec/frontend/svelte-vite-embed.md`
* Candidate backend files: `api/routes/order_responses.go`, `api/routes/order.go`, `api/models/order.go`, `api/models/product.go`, `api/models/user.go`
* Candidate frontend files: `frontend/src/api.js`, future `frontend/src/enums/`, future `frontend/src/stores/`
