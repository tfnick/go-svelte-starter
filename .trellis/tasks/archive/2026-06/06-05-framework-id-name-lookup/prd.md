# 框架级 ID 转名称能力

## Goal

将目前按场景手写的 ID 到名称组装逻辑，升级为 `api/framework` 下的通用能力。开发者在订单、用户、商品或未来业务场景中，只需要声明需要解析的资源类型、收集关联 ID、注册批量加载函数，即可一次性解析出名称映射，并在 CO/DTO 组装时显式写入 `xxxName`。

本任务的直接动因是订单用例中仍存在较 case-by-case 的代码：

```go
nameMaps, err := orderdisplay.LoadNameMaps(ctx.Std(), []models.Order{*order}, persistedItems)
if err != nil {
    return OrderCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load order display names", err)
}

return orderCoFromModel(order, nameMaps), nil
```

期望后续不再为每个业务场景复制一套 `xxxdisplay.LoadNameMaps`。

## What We Already Know

- 用户要求：ID 到 Name 由后端组装，前端直接展示，永远不做翻译。
- 用户要求：DTO 命名约定是有 `xxxId` 就必须附带 `xxxName`。
- 用户要求：禁止 N+1，关联查询一律批量 `IN` 查询加内存 `map` 组装。
- 用户补充：ID 转 Name 除了不能出现 N+1，还应抽象成可复用模式。
- 当前已有 `api/helpers/idname`，只覆盖去重、空值跳过、行转 map、调用 loader。
- 当前已有 `api/helpers/orderdisplay`，但它绑定了订单、用户、商品，不能作为跨场景框架能力。
- 当前架构已升级为 `routes -> usecase -> models`，并新增 `api/framework` 存放业务无关能力。
- usecase 层返回 client object，例如 `OrderCo`，routes 层再转换为 response DTO。
- 上下文从 routes 传到 usecase，再传到 models；新能力必须保留这条 context 传递路径。

## Requirements

### Framework Ownership

- 新能力应放在 `api/framework/namelookup` 或同等 framework 子包中。
- `namelookup` 不得 import `api/models`、`api/usecase`、`api/routes` 或任何具体业务包。
- 现有 `api/helpers/idname` 中业务无关的能力应迁移或整合进 framework。
- 订单专用 `api/helpers/orderdisplay` 不应继续作为主要使用方式；订单用例应迁移为 framework 使用示例。
- 如确需保留业务小 helper，只允许保留薄封装，不得重复实现框架已有的去重、批量解析和结果读取逻辑。

### Resolver Capability

- 框架应提供资源类型概念，例如 `Resource` / `Kind` / `Key`，用于区分 `user`、`product` 等名称来源。
- 框架应提供统一 loader 签名：

```go
type Loader func(context.Context, []string) (map[string]string, error)
```

- 框架应支持在一次解析过程中收集多个资源类型的多个 ID。
- 框架必须对同一资源类型下的 ID 去重并跳过空字符串。
- 每个资源类型在一次解析中最多调用一次批量 loader。
- loader 未注册但存在待解析 ID 时，应返回清晰错误，不能静默吞掉。
- loader 返回错误时，应包装资源类型信息，便于定位失败来源。
- 缺失名称的处理规则保持明确：返回空字符串，并保留原始 ID。
- 解析结果应提供显式读取方法，例如 `Name(resource, id)` 或 `Map(resource)`。

### Layering

- routes 层不做 ID 到 Name 翻译，只消费 usecase 返回的 `xxxName`。
- usecase 层负责在组装 CO 时调用 framework resolver。
- models 层继续提供按资源批量查询能力，例如 `GetUserNamesByIDs`、`GetProductNamesByIDs`。
- framework resolver 调用 loader 时必须传递调用方给出的 `context.Context`，确保 routes -> usecase -> models 的 context 链路不丢失。
- framework 不直接管理事务，但如果 usecase 在事务上下文内调用 resolver，loader 应自然复用该 context 中携带的事务信息。

### Developer Experience

- 新增场景时，开发者应能按固定模式完成：
  - 注册资源类型和批量 loader。
  - 从模型或中间结果中收集 ID。
  - 调用 resolver 一次性解析。
  - 在显式 mapper 中写入 `xxxName`。
- 框架 API 应避免反射、struct tag 自动 mapper 或隐式字段名推断。
- 使用代码应比 `orderdisplay.LoadNameMaps` 更通用，同时保持阅读时能看出每个 `xxxName` 来自哪里。
- 错误信息应能说明是哪个资源类型加载失败。

## Confirmed Design

用户已确认采用 A 方案：framework 只负责“收集 ID -> 批量加载 -> 返回名称结果”，CO/DTO 字段仍由显式 mapper 写入。

名称来源采用可复用展示语义，不绑定具体页面场景。例如推荐使用 `user.display_name` 表示“给一批 user ID 返回用于展示的用户名称”。当前实现可以读取 `users.nickname` 或 `users.name`，未来展示规则变化时，调用方不需要知道底层字段变化。

## Recommended Design

推荐采用“显式 resolver + 显式 CO/DTO mapper”的 MVP：

```go
resolver := namelookup.New(
    namelookup.Resource("user.display_name", models.GetUserDisplayNamesByIDs),
    namelookup.Resource("product.display_name", models.GetProductDisplayNamesByIDs),
)

resolver.Add("user.display_name", order.UserID)
for _, item := range items {
    resolver.Add("product.display_name", item.ProductID)
}

names, err := resolver.Resolve(ctx.Std())
if err != nil {
    return OrderDetailCo{}, fwusecase.E(fwusecase.CodeInternal, "failed to load display names", err)
}

return orderCoFromModel(order, names), nil
```

CO mapper 仍然显式写字段：

```go
UserName: names.Name("user.display_name", order.UserID)
ProductName: names.Name("product.display_name", item.ProductID)
```

这样 framework 负责通用流程，业务代码保留可读的字段装配。

## Alternatives

### A. 显式 Resolver，只返回名称结果（推荐）

- 优点：低魔法、易测试、无反射、无业务 import、符合当前代码风格。
- 优点：framework 复用批量解析流程，业务 mapper 仍清楚展示字段来源。
- 代价：每个 CO/DTO mapper 仍要显式写 `UserName: names.Name(...)`。

### B. Resolver 加 collect/apply 回调

- 优点：可以进一步减少 usecase 中的循环和字段赋值样板。
- 代价：回调结构更重，调试时需要跳转更多层，容易把简单 mapper 变复杂。

### C. 反射或 struct tag 自动写入 `xxxName`

- 优点：业务代码最少。
- 代价：字段来源不直观，容易违反现有“显式 DTO mapper”约束；暂不建议。

## Acceptance Criteria

- [x] 新增 `api/framework/namelookup` 或同等 framework 子包。
- [x] framework resolver 支持多资源类型、多 ID 收集、去重、空值跳过、批量 loader 调用。
- [x] framework resolver 对未注册 loader、loader 错误、缺失名称有明确行为和测试。
- [x] `api/helpers/idname` 的业务无关能力迁移或整合到 framework，不再作为散落 helper 使用。
- [x] 订单 usecase 不再直接调用 `orderdisplay.LoadNameMaps`。
- [x] 订单场景使用 framework resolver 解析 `user_name` 和 `product_name`。
- [x] 订单 CO/response 中已有的 `user_id/user_name`、`product_id/product_name` 行为保持不变。
- [x] 不引入反射式 DTO mapper。
- [x] 不引入 N+1 查询；用户名、商品名仍通过批量模型方法加载。
- [x] framework 代码不依赖业务包，业务包只向 framework 注册 loader。
- [x] 更新 `.trellis/spec/backend/api-contracts.md` 中 ID/name 组装规范，从 `api/helpers` 改为 framework resolver 模式。
- [x] `go test ./...` 通过。

## Out Of Scope

- 不要求一次性迁移所有未来资源；本任务以订单场景作为真实落地示例。
- 不做前端字典 Store 或静态枚举调整；那些已由前序字典任务覆盖。
- 不做进程级 ID/name 缓存，避免数据陈旧和失效策略复杂化。
- 不做反射、struct tag、ORM hook 或全自动 DTO 字段写入。
- 不改变现有 `/api/*` 响应字段语义。

## Decision Notes

- MVP 不做 collect/apply 回调式自动组装。
- MVP 不做反射或 struct tag 自动写入。
- lookup key 按展示语义命名，例如 `user.display_name`，而不是按页面场景命名，例如 `product.created_by_name`。
- 同一个 lookup key 可以在多个场景复用，例如产品创建人、产品修改人、评论作者、审计操作人都可以使用 `user.display_name`。
