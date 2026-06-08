# Integrate Creem Checkout in Order Module

## Goal

Make the order module actually create a local payable order, start a Creem test checkout, and let the Creem webhook mark the order paid and trigger fulfillment side effects.

## Current Decision

Creem is the source of truth for the sellable product and price in this MVP.

The local `orders` table is a payment and fulfillment ledger. A newly created order is a local pending shell that exists so checkout, webhook reconciliation, event publishing, and points fulfillment have a stable local `order_id`.

The local `products` table and `/api/products` endpoint are retained for legacy/demo/admin compatibility, but they are not part of the Creem checkout flow for this task. Creating an order for Creem checkout must not require local `product_id`, `quantity`, product stock, or local price agreement.

## What I Already Know

* The user has already configured a Creem test product and required keys outside this repo.
* The target scope is the `order` module order-to-payment loop.
* The repo already has:
  * `POST /api/orders` to create local orders.
  * `POST /api/orders/:id/payment-checkout` to create Creem checkout.
  * `POST /api/integrations/payment/:channel_code/webhooks/creem` to receive Creem webhooks.
  * `payment.creem.hosted_checkout` registered in `index.go`.
  * `PayOrder` and the `order.paid` durable event side effect for points.
* Creem test checkout uses `https://test-api.creem.io/v1`, `POST /checkouts`, and `x-api-key`.
* Creem payment success is represented by `checkout.completed`.
* Creem test checkout is backed by Stripe test-mode card rules. Real/non-test cards are rejected in test mode with a card-declined message. Use a Stripe test card such as `4242 4242 4242 4242`, any future expiry, any 3-digit CVC, and any postal code.
* A successful Creem browser return to `/orders` can include query parameters such as `request_id`, `checkout_id`, provider `order_id`, `customer_id`, `subscription_id`, `product_id`, and `signature`.
* The local `data/app.db` check on 2026-06-08 found one payment channel:
  * `channel_code`: `123`
  * `adapter_key`: `payment.creem.hosted_checkout`
  * `base_url`: `https://test-api.creem.io/v1`
  * `product_id`: `111`
  * credential exists; secret values were not printed.
  * `webhook_enabled` was changed from `0` to `1`.
  * no operation config row exists; the single-channel fallback is acceptable for this MVP.
* There are unrelated dirty deletions under `.trellis/tasks/06-07-creem-payment-integration-slice/*`; this task must not include them.

## Requirements

* `POST /api/orders` creates a pending local order ledger using `user_id` only.
* Local order creation does not require local products, order items, inventory reservation, or local price matching.
* For compatibility, legacy `items` in the create-order payload may be accepted, but the current Creem checkout flow ignores them.
* `POST /api/orders/:id/payment-checkout` creates a Creem checkout against the configured payment channel product.
* The frontend order page offers a direct create-and-pay action instead of a local product selector.
* The frontend must not display local product price as the Creem charge. If local `orders.amount` is `0`, the amount display should make clear that Creem checkout determines the amount.
* Creem webhook/callback remains authoritative for marking orders paid.
* Duplicate webhooks/callbacks must not duplicate fulfillment or accounting side effects.
* Missing Creem config must return safe business errors without leaking credentials.
* `success_url` is only the browser return target. Final paid state still depends on webhook processing.
* Query parameters returned on `success_url` are useful for manual diagnosis, but must not be trusted as the payment settlement signal.

## Acceptance Criteria

* [ ] Calling the create-order API with only `user_id` returns a pending order.
* [ ] The created order has no local `order_items` for this Creem checkout MVP.
* [ ] The frontend can create the local order and immediately request a Creem checkout URL.
* [ ] A pending order can still request checkout from the order list.
* [ ] A successful Creem test payment updates the local order to paid through webhook processing.
* [ ] Duplicate webhook/callback delivery does not duplicate points or ledger effects.
* [ ] Backend tests cover order creation without products, checkout creation, webhook verification/idempotency, and error paths.
* [ ] Frontend tests/build pass after removing product selection from the checkout flow.

## Out of Scope

* Deleting the local `products` table, `/api/products`, or historical `order_items` support.
* Adding product-to-Creem mapping.
* Adding env bootstrap for Creem config; the user has configured Parameters manually.
* Live-mode rollout, refunds, disputes, coupons, invoices, tax, subscriptions, or multi-product routing.
* Provider amount capture into `orders.amount`; future work can normalize Creem amount from webhook payload if needed.

## Technical Notes

* Research reference: `research/creem-checkout-webhooks.md`.
* Existing payment config schema is in `api/usecase/parameter_schema.go`.
* Current checkout adapter sends the configured Creem `product_id`; provider product/price decides the actual charge.
* `orders.amount = 0` is acceptable for a newly created Creem ledger order in this MVP because local amount is not the payment authority.
* Historical order details may still return `order_items` and product display names when legacy rows exist.

## Manual Test Notes

* On 2026-06-08, attempting Creem test checkout with a non-test card produced a provider decline because the request was in test mode. This is expected; use Stripe test card data instead of a real card.
* Valid happy-path card data for Creem test checkout:
  * Card number: `4242 4242 4242 4242`
  * Expiry: any future date, for example `12/34`
  * CVC: any 3 digits, for example `123`
  * ZIP/postal: any value, for example `10001`
* Observed successful browser return shape:
  * `/orders?request_id=<local-order-id>&checkout_id=<creem-checkout-id>&order_id=<creem-order-id>&customer_id=<creem-customer-id>&subscription_id=<creem-subscription-id>&product_id=<creem-product-id>&signature=<signature>`
* The browser return confirms the user came back from checkout, but local fulfillment should still wait for the verified Creem webhook.

## Implementation Plan

* Add a model helper that inserts a pending order without local items.
* Change `CreateOrder` usecase to validate the user and insert a pending ledger order with amount `0`.
* Change the route to no longer map request `items` into the usecase command.
* Change the frontend order page to stop loading local products and to create checkout immediately after order creation.
* Update tests and specs to reflect Creem-as-product-source-of-truth.
* Run `go test ./...`, frontend tests, and frontend production build.
