# 优化会员等级与 Creem 付费订阅关系

## Goal

理顺产品(product)、会员等级(membership_level)、Creem 订阅付费三者之间的关系，确保：
- 购买时正确计算有效期（当前日期 + 订阅周期）
- 禁止降级购买
- Creem 自动续费时正确延长会员有效期

## What I already know

### 当前数据模型
- **User**: `membership_level` (basic/premium/super), `membership_expires_at` (SQLite datetime)
- **Product**: `billing_type` (one_time/subscription), `membership_level`, `subscription_interval`, `creem_product_id`
- **Order**: `membership_applied_at` (用于幂等), `provider_subscription_id`, `subscription_status`
- 常量：`PermanentMembershipExpiresAt = "2099-12-31 23:59:59"`

### 当前流程
1. `CreateOrder` → 创建订单
2. `CreateOrderPaymentCheckout` → 调用 Creem 创建 checkout，传 `request_id=orderID`
3. Creem webhook `checkout.completed` → `HandlePaymentWebhookJob` → `PayOrder` → 触发 `ApplyOrderMembership`
4. `membershipExpiresAtForProduct`: 当前逻辑是从**当前过期日延长**（如果未过期），用户要求改为始终从**当前日期**计算

### Creem Webhook 事件（已研究）
| Event | 触发时机 | 关键字段 |
|-------|---------|---------|
| `checkout.completed` | 首次支付成功 | `request_id`(我们的orderID), `order`, `subscription`, `customer`, `product`, `metadata` |
| `subscription.paid` | 续费扣款成功 | `subscription.id`, `customer.email`, `product.id`, `metadata`, billing周期 |
| `subscription.canceled` | 取消订阅 | `subscription.id`, `customer` |
| `subscription.active` | 新订阅创建 | `subscription.id`, `customer`, `product` |
| `subscription.past_due` | 扣款失败 | subscription info |
| `subscription.expired` | 过期 | subscription info |

**关键发现**：`subscription.paid` 不包含 `request_id` 和 `order` 对象。续费时匹配用户需要通过 `subscription_id`（我们存在 order 的 `provider_subscription_id`）或 `customer.email`。

### 当前代码未处理续费
- `NormalizePaymentWebhook` 只处理 `checkout.completed` 和 `subscription.canceled`
- `subscription.paid` 事件当前被忽略（`MarkIntegrationWebhookReceiptIgnored`）
- `ApplyOrderMembership` 通过 `membership_applied_at` 做幂等，续费时无法重复应用

## Requirements

1. **默认 basic 会员**：用户注册后默认 basic，有效期 2099-12-31（已实现，不变）
2. **首次购买**：
   - 会员有效期 = 当前日期 + product.subscription_interval
   - 修改 `membershipExpiresAtForProduct`：不再从当前过期日延长，始终从 now 计算
3. **禁止降级购买**（下单时验证）：
   - 当前会员未过期 + 要购买的产品 membership_level < 当前 membership_level → 拒绝
   - 当前会员未过期 + 要购买的产品 membership_level == 当前 membership_level → 也拒绝（不能重复购买同等级）
   - 当前会员已过期 → 允许任意购买
   - 购买更高级别（upgrade）→ 允许
4. **Creem 自动续费（subscription.paid webhook）**：
   - 收到 `subscription.paid` → 延长用户会员有效期 = 当前日期 + 订阅周期
   - 通过 `customer.email` 查找用户，通过 `product.id` 查找本地产品
5. **取消后重新购买**：
   - 如果会员已过期 → 正常购买，有效期 = now + interval
   - 如果会员未过期 → 不允许（规则3已覆盖）
6. **过期自动降级为 basic（Lazy Evaluation）**：
   - 用户会员过期后，DB 中 `membership_level` 保持原值（保留购买历史）
   - 新增 `EffectiveMembership()` helper，判权和降级检查统一使用
   - 过期时一律视为 basic 会员，有效期 2099-12-31

## Acceptance Criteria

- [ ] basic 用户注册后 membership_level='basic', expires_at='2099-12-31 23:59:59'
- [ ] basic 用户购买 premium 月度产品 → membership_level='premium', expires_at = now + 1 month
- [ ] premium 用户购买 super 产品 → membership_level='super', expires_at = now + interval
- [ ] super 未过期用户购买 premium 产品 → 返回错误（禁止降级）
- [ ] premium 未过期用户购买 premium 产品 → 返回错误（禁止重复购买同等级）
- [ ] premium 已过期用户重新购买 premium → 成功，expires_at = now + interval
- [ ] Creem `subscription.paid` webhook（续费）→ 会员有效期更新为 `current_period_end_date`
- [ ] 首次支付时 `subscription.paid` 与 `checkout.completed` 同时到达 → 不重复延长（`membership_applied_at` 判空跳过）
- [ ] Creem `subscription.canceled` webhook → 不影响会员有效期（订单 subscription_status 更新）
- [ ] 过期 premium 用户被视为 basic（`EffectiveMembership` 返回 basic）
- [ ] 过期 premium 用户可以重新购买 premium（不会被降级检查误拦）
- [ ] `checkout.completed` 首次购买流程不受影响（已有测试保持通过）

## Technical Approach

### 1. 修改有效期计算 (`membershipExpiresAtForProduct`)

```go
// Before: extends from current expiry if not expired
base := timefmt.NowUTC()
current, err := parseSQLiteTime(currentExpiresAt)
if err == nil && current.After(base) {
    base = current  // extend from current
}

// After: always from now
base := timefmt.NowUTC()
```

简化：移除 `currentExpiresAt` 参数，始终从当前时间计算。

### 2. EffectiveMembership（Lazy Evaluation）

新增 helper，过期会员视为 basic：

```go
func EffectiveMembership(user *models.User) (level string, expiresAt string) {
    if expires, err := parseSQLiteTime(user.MembershipExpiresAt); err == nil && expires.Before(time.Now()) {
        return MembershipLevelBasic, PermanentMembershipExpiresAt
    }
    return user.MembershipLevel, user.MembershipExpiresAt
}
```

降级检查、权限判断等统一使用此函数，而非直接读 `user.MembershipLevel`。

### 3. 下单时禁止降级 (`CreateOrderPaymentCheckout`)

使用 `EffectiveMembership()` 获取有效等级后再做降级检查：

```
membershipLevelRank: basic=0, premium=1, super=2
effective := EffectiveMembership(user)
if effective未过期 && productLevel <= effectiveLevel → reject
```

### 4. 处理 `subscription.paid` 续费（独立事件 `subscription.renewed`）

`subscription.paid` 不与 `checkout.completed` 共用 `payment.succeeded`，而是映射到新的业务事件 `subscription.renewed`，避免首次支付被重复处理。

1. 在 `NormalizePaymentWebhook` 中添加 `subscription.paid` → `subscription.renewed`
2. 在 `HandlePaymentWebhookJob` 中新增 `subscription.renewed` 处理分支：
   - 通过 `provider_subscription_id` 查找原始 order
   - 若 `membership_applied_at` 为空 → 首次支付尚未处理，跳过（`checkout.completed` 负责）
   - 若 `membership_applied_at` 已设置 → 续费，从 webhook 提取 `current_period_end_date`
   - 直接更新 `users.membership_expires_at = current_period_end_date`
3. 有效期以 Creem 返回的 `current_period_end_date` 为准，不再本地计算

### 5. 新增常量和 webhook 字段

在 `api/usecase/integrations/payment/ports.go` 添加：
```go
WebhookEventSubscriptionRenewed = "subscription.renewed"
```

在 `creem.go` 的 `webhookCheckoutObject` 中添加 billing 周期字段：
```go
CurrentPeriodStartDate string `json:"current_period_start_date"`
CurrentPeriodEndDate   string `json:"current_period_end_date"`
```

## Decision (ADR-lite)

### Decision 1: 续费处理

**Context**: Creem `subscription.paid` 续费 webhook 不含 `request_id`，且与首次支付的 `checkout.completed` 是不同的独立事件。

**Decision**: 
1. `subscription.paid` 映射为独立业务事件 `subscription.renewed`，不共用 `payment.succeeded`
2. 通过已有的 `provider_subscription_id` 查找订单，使用 `membership_applied_at` 区分首次支付与续费
3. 有效期直接使用 Creem 返回的 `current_period_end_date`，不本地计算

**Consequences**: 
- 首次支付和续费走不同代码路径，不会互相干扰
- 有效期与 Creem 结算周期严格一致，天然幂等（同一事件重复处理设置相同期限）

### Decision 2: 过期会员降级为 basic（Lazy Evaluation）

**Context**: 会员过期后 DB 中 `membership_level` 仍保留原值（如 "premium"），直接读取会误判会员等级。

**Decision**: 使用 `EffectiveMembership()` helper 做 Lazy Evaluation，过期 → basic，不改 DB 中的 `membership_level`。

**Consequences**: 
- DB 中 `membership_level` 保留购买历史，不会被覆写
- 所有判权/检查入口统一调用 `EffectiveMembership()`，避免直接读 `user.MembershipLevel`
- 不依赖 `subscription.expired` webhook

## Out of Scope

- one_time 产品的会员逻辑（保持现有：永久会员 2099-12-31）
- `subscription.past_due` / `subscription.expired` 事件（后续迭代）
- 前端 UI 改动

## Technical Notes

- 关键文件：
  - `api/usecase/order.go` — `membershipExpiresAtForProduct`, `ApplyOrderMembership`, `CreateOrderPaymentCheckout`（在 `api/usecase/payment.go`）
  - `api/integrations/payment/creem/creem.go` — `NormalizePaymentWebhook`
  - `api/usecase/payment.go` — `HandlePaymentWebhookJob`
  - `api/usecase/integrations/payment/ports.go` — 常量定义
  - `api/models/user.go` — `UpdateUserMembership`
- 测试文件：`api/usecase/product_membership_checkout_test.go`

## Open Questions

- (已确认) 有效期始终从当前日期计算
- (已确认) 续费由 Creem 自动处理
