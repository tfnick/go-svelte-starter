# Current ID-to-Name State

## Existing Implementation

- `api/helpers/idname` provides low-level utilities:
  - `UniqueNonEmpty`
  - `RowsToMap`
  - `Load(ctx, ids, loader)`
- `api/helpers/orderdisplay` provides an order-specific `NameMaps` struct and `LoadNameMaps` function.
- `api/usecase/order.go` calls `orderdisplay.LoadNameMaps` in:
  - `CreateOrder`
  - `GetUserOrders`
  - `GetOrderDetail`
- `api/models/user.go` and `api/models/product.go` already expose batch loaders:
  - `GetUserNamesByIDs(ctx, ids)`
  - `GetProductNamesByIDs(ctx, ids)`

## Pain Points

- `orderdisplay.LoadNameMaps` hard-codes order models, user names, and product names.
- Adding another scene would likely create another scene-specific name map helper, repeating collection, dedupe, loader invocation, and result access patterns.
- The low-level `idname` helper is business-agnostic, so after the recent architecture cleanup it belongs under `api/framework`, not under a generic top-level helper package.
- `.trellis/spec/backend/api-contracts.md` still describes "small explicit helpers" under `api/helpers`; this should be updated after the framework design is accepted.

## Useful Constraints

- ID-to-name assembly belongs on the backend; frontend displays `xxx_name` directly.
- DTO/response fields must keep `xxx_id` and `xxx_name` paired when the ID is used for display.
- Name loading must avoid N+1 by batching IDs with `IN` queries and assembling with in-memory maps.
- Mapping into CO/DTO fields should stay explicit and readable; previous PRD explicitly avoided reflection or magic DTO mappers.
- Framework code must not import business models or usecases. Business layers can register model batch loaders with the framework.
