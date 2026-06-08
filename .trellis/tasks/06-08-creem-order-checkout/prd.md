# Integrate Creem Checkout in Order Module

## Goal

在 `order` 模块真正打通 Creem 测试支付：用户可以从系统内创建订单、发起 Creem 支付，支付成功后系统能可靠更新订单状态并发放对应权益。

## What I Already Know

* 用户已经在 Creem 开通测试所需产品和相关 key。
* 目标范围是 `order` 模块的下单和支付闭环。
* 仓库已有 `api/usecase/order.go`、`api/routes/order.go`、`api/usecase/payment.go`、`api/routes/payment.go`、`api/integrations/payment/creem/creem.go` 等支付相关代码。
* `index.go` 已注册 `payment.creem.hosted_checkout` adapter，并暴露：
  * `POST /api/orders` 创建订单。
  * `POST /api/orders/:id/payment-checkout` 创建 Creem checkout。
  * `POST /api/integrations/payment/:channel_code/webhooks/creem` 接收 Creem webhook。
* `frontend/src/pages/Dashboard.svelte` 已有订单页面、商品下单、待支付订单跳转 checkout、积分 SSE 刷新入口。
* `api/usecase/payment.go` 已有 payment config 读取、integration invocation、webhook receipt、队列、幂等和 webhook job 处理。
* `api/usecase/order.go` 已有 `PayOrder`，会在事务中标记订单 paid 并发布 `order.paid` 事件。
* `api/usecase/events/order_paid_points.go` 已有订单支付后发放积分的 durable event handler。
* Creem 官方文档确认 test checkout 使用 `https://test-api.creem.io/v1` + `POST /checkouts` + `x-api-key`，支付成功事件为 `checkout.completed`。
* 当前会话中存在与本任务无关的旧 Trellis 删除痕迹：`.trellis/tasks/06-07-creem-payment-integration-slice/*`，本任务不处理它们。

## Assumptions (Temporary)

* 本任务优先打通 Creem test mode，不要求同时上线 live mode。
* Creem 产品与系统内产品/积分套餐之间需要建立映射，MVP 可优先复用现有 payment channel 的 `product_id` 配置。
* 支付成功后的订单状态更新应以 Creem webhook/callback 为准，而不是只信任前端跳转返回。

## Open Questions

* None for MVP implementation.

## Requirements (Evolving)

* 支持用户在 order 模块创建待支付订单。
* 支持对待支付订单发起 Creem checkout/payment。
* 支持 Creem 支付成功回调后更新订单支付状态。
* 支持通过现有 Parameters / Integration settings 手动配置测试环境所需的 Creem API key、webhook secret、product ID、base URL、success URL。
* 支付回调必须具备基础安全校验和幂等处理。
* 保持 `PayOrder` 的人工/测试入口不直接替代 Creem 支付流程；真实支付以 webhook 触发订单 paid 为准。
* `success_url` 只作为支付完成后的浏览器返回地址；订单最终 paid 状态仍以 webhook 为准。
* 用户已完成 Creem test channel 的手动配置；实现阶段不新增 env bootstrap。

## Acceptance Criteria (Evolving)

* [ ] 调用下单接口后可以得到一笔待支付订单。
* [ ] 对待支付订单发起支付后可以得到 Creem checkout/payment URL。
* [ ] Creem 测试支付成功后，系统订单状态更新为 paid/success。
* [ ] 重复 webhook/callback 不会重复发放权益或重复记账。
* [ ] Creem 配置缺失时返回清晰的业务错误，不泄漏密钥。
* [ ] 后端测试覆盖订单创建、支付发起、回调验签/幂等、异常路径。
* [ ] 如需用户手动配置，PRD/最终说明给出明确的 Creem channel 配置字段和 webhook URL。
* [ ] `success_url` 支持留空或配置为当前应用的 `/orders` 页面；配置为临时 URL 时不影响 webhook 入账，但用户不会自动回到订单页。

## Definition of Done

* Tests added/updated where appropriate.
* `go test ./...` passes.
* Frontend tests/build run if frontend behavior changes.
* Docs/notes updated if runtime env/config changes.
* Rollout/rollback considered for payment risk.

## Out of Scope (Explicit)

* Live mode 正式上线切换，除非实现时仅需配置开关。
* 复杂退款、订阅、争议处理、优惠码、发票税务。
* 支付方式以外的大规模订单 UI 改版。
* 多支付渠道路由策略，除非现有 configuration lookup 已自然支持。

## Technical Notes

* Research reference: `research/creem-checkout-webhooks.md`.
* Existing adapter may need validation against exact Creem live test event payload/header format.
* Existing payment schema is in `api/usecase/parameter_schema.go`.
* Existing parameter UI can create payment integration channels; likely no admin UI rewrite needed for MVP.
* Existing seed data does not appear to include a default payment channel; user/manual setup is likely required unless this task adds a safer env/seed mechanism.
* Local `data/app.db` check on 2026-06-08 found one payment channel:
  * `channel_code`: `123`
  * `adapter_key`: `payment.creem.hosted_checkout`
  * `base_url`: `https://test-api.creem.io/v1`
  * credential exists; secret values were not printed.
  * `webhook_enabled` was changed from `0` to `1` so Creem webhook ingress will be accepted.
  * no `create_payment` operation row exists, which is acceptable for the single-channel MVP because `GetEnabledPaymentConfig` falls back to the first enabled payment channel when the operation lookup is absent.
* Relevant specs likely include backend route, API contract, database, error handling, eventing, logging, quality, plus frontend API/Svelte guide if UI changes.

## Research Notes

### Feasible Approaches

**Approach A: Single Creem Product Channel (Recommended MVP)**

* Use the existing payment integration channel config (`product_id`, `base_url`, `success_url`, `units`) for all order checkout calls.
* Existing code path already supports this shape.
* Fastest path to verify real test payment end to end.
* Trade-off: local order amount/product list may not exactly mirror multiple Creem products unless the configured Creem test product matches the intended sellable item.
* Decision: selected for this MVP.

**Approach B: Product-Level Creem Mapping**

* Add a mapping from each local product to a Creem `product_id` and use that during checkout.
* Better if multiple system products must map to separate Creem products/prices.
* Trade-off: needs DB/config model changes and more tests; larger scope.

**Approach C: Environment Variable Bootstrap**

* Add env vars for Creem key/product/webhook secret and bootstrap/update the payment integration config at startup or migration.
* Easier for deployment if the user prefers Dokploy/env-driven setup.
* Trade-off: must avoid leaking secrets and must define precedence against admin-edited Parameter settings.
* Decision: not included in this MVP because the user has already configured via Parameters page.

## Expansion Sweep

### Future Evolution

* Later tasks may add multiple Creem products, subscriptions, refunds, or live/test channel switching.
* Preserve adapter/config boundaries so product mapping can evolve without rewriting webhook fulfillment.

### Related Scenarios

* Admin Parameters page should remain the primary integration settings surface unless we choose env bootstrap.
* Order list should remain consistent after returning from Creem checkout; webhook is authoritative.

### Failure and Edge Cases

* Missing config, invalid API key, invalid webhook signature, duplicate webhook event, non-success webhook event, checkout created but user never pays.
* Payment side effects must remain idempotent; points award already has `UNIQUE(order_id, type)`.

## Decision (ADR-lite)

**Context**: Creem test mode is already prepared externally, and the immediate goal is to prove the real order-to-payment-to-fulfillment loop.

**Decision**: Use a single Creem test product for the MVP. All local pending orders create checkout sessions against the configured payment channel `product_id`.

**Consequences**: This avoids DB/product mapping changes and lets the task focus on real checkout + webhook reliability. Multi-product Creem mapping remains future work.

**Configuration Decision**: Use the existing Parameters / Integration settings page for this MVP. No env bootstrap is required in this task.

## Implementation Plan

* Verify current Creem checkout request/response contract against tests and official docs.
* Verify webhook signature and payload normalization against Creem test events.
* Ensure order checkout and webhook processing produce clear errors, do not leak credentials, and remain idempotent.
* Add or adjust tests for real-MVP assumptions: single configured product, `success_url` behavior, webhook-driven payment, duplicate webhook handling.
* Keep frontend changes minimal unless a broken order checkout/return UX is discovered.
