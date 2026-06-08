# Product Catalog Membership Checkout

## Goal

Upgrade the completed single-product Creem checkout flow into an app-owned product catalog and membership entitlement flow. Operators should be able to create local products that map to Creem products, users should be able to choose a product before checkout, and verified Creem webhook events should update user membership level and expiry.

## What I Already Know

* The prior single-product Creem checkout MVP is complete and uses webhook-confirmed payment as the authoritative fulfillment signal.
* The user wants a new Product menu with non-paginated list, create, and edit flows.
* Each local product can configure the associated Creem platform product ID.
* Membership levels are `basic`, `premium`, and `super`, displayed as Basic, Advanced, and Super in the UI.
* Every user should have a membership level and a membership expiration time.
* The order module should allow selecting a local product for checkout; that local product is equivalent to the Creem product used for payment.
* For subscription products, payment fulfillment extends membership expiry by the configured subscription interval: monthly, three-month, six-month, or yearly.
* For one-time service products, payment fulfillment sets membership expiry to year 2099.
* When a subscription is canceled in Creem, the corresponding order subscription status should become canceled without changing membership level or expiry.
* The user wants debugging to keep using backend port `3000` and the database under `data/`.

## Current Repo Context

* `products` currently has legacy/demo fields only: `id`, `name`, `description`, `price`, `stock`, `created_at`.
* `GET /api/products` exists, but no product create/edit API currently exists.
* `POST /api/orders` currently accepts only `user_id` for the Creem ledger flow and creates a `pending` order with `amount = 0`.
* The payment configuration still has a channel-level `product_id`; for this task, checkout should use the selected local product's Creem product id.
* `orders` currently stores `id`, `user_id`, `amount`, `status`, and `created_at`; it does not store local product id, provider checkout/order/customer/subscription ids, or subscription status.
* `users` currently has no membership level or expiry fields.
* `HandlePaymentWebhookJob` only fulfills `payment.succeeded` derived from Creem `checkout.completed`.
* The Creem adapter currently verifies signatures and normalizes checkout completion but does not expose subscription lifecycle events as business events.
* Frontend routing has `/orders` but no `/products` route/menu yet.

## Requirements

* Add a Product menu and `/products` app route.
* Product list is non-paginated and shows all configured local products.
* Product create/edit supports:
  * name
  * description
  * optional display price/currency for the local UI
  * active/enabled status
  * Creem product id
  * product billing type: subscription or one-time service
  * membership level granted by successful payment
  * subscription interval for subscription products only: month, three months, six months, year
* Local products are the app catalog. Creem remains the payment provider/catalog authority for actual checkout pricing.
* Add membership fields to users:
  * membership level: basic, premium, super
  * membership expires at
* Existing and new users default to permanent `basic` membership with `membership_expires_at = 2099-12-31 23:59:59`.
* Upgrade order creation so the user selects one local product before checkout.
* Checkout creation sends the selected product's Creem product id to Creem.
* Orders persist the selected local product id and useful provider identifiers when known.
* Payment fulfillment runs from verified webhook events, not from the browser success redirect.
* On successful payment:
  * mark order payment status paid if not already paid
  * update the user's membership level to the product's configured membership level
  * for subscription products, add the configured interval to the existing membership expiry
  * if the existing membership expiry is already in the past, use current time as the subscription extension base
  * for one-time service products, set membership expiry to `2099-12-31 23:59:59`
* If the Creem subscription cancellation event arrives:
  * find the corresponding local order by provider subscription id or order metadata
  * set order subscription status to canceled
  * do not change user membership level
  * do not change user membership expiry
* The fulfillment logic must be idempotent under webhook retry.

## Recommended Assumptions

* Subscription extension uses `max(now, current_membership_expires_at)` as the base, then adds the product interval. This avoids shortening active membership and avoids extending from a stale expired date.
* Existing and new users default to `basic` with `membership_expires_at = 2099-12-31 23:59:59`.
* Order subscription status should be separate from payment status. Suggested values: empty/not_applicable, active, scheduled_cancel, canceled, past_due, expired, paused.
* The legacy `stock` behavior should be ignored for the new Creem membership product flow.
* Product price and currency are optional display fields only. Actual billed amount remains whatever Creem has configured for the mapped product id.

## Open Questions

* None.

## Acceptance Criteria

* [x] Admin/user can navigate to Product menu.
* [x] Product list loads all local products without pagination.
* [x] Product create and edit can save Creem product id, billing type, interval, membership level, and enabled status.
* [x] Product create and edit allow optional local display price/currency without using those fields for actual checkout billing.
* [x] Existing and new users default to permanent `basic` membership expiring at `2099-12-31 23:59:59`.
* [x] Orders page requires selecting an enabled local product before creating checkout.
* [x] Checkout creation uses the selected local product's Creem product id.
* [x] Successful Creem payment webhook marks the order paid and updates the user's membership level and expiry.
* [x] Subscription products extend membership by month/three-month/six-month/year according to the product config.
* [x] Expired users who buy a subscription extend from current time rather than from the stale expired timestamp.
* [x] One-time service products set membership expiry to `2099-12-31 23:59:59`.
* [x] Creem subscription cancellation webhook marks the order subscription status canceled and leaves membership unchanged.
* [x] Webhook retries do not double-extend membership.
* [x] Relevant backend tests cover product CRUD, checkout product selection, membership fulfillment, and subscription cancellation.
* [x] Relevant frontend tests cover API helpers/routes and product/order UI changes.
* [x] Local smoke test uses port `3000` and the `data/` database.

## Implementation Summary

* Added app-owned product catalog fields, create/edit APIs, and a Product page with local Creem product mapping.
* Added membership level and expiry fields to users and surfaced them in auth/user/order UI responses.
* Upgraded order creation to require an enabled local product, use that product's Creem product id for checkout, and persist provider references.
* Added webhook-driven membership fulfillment with retry idempotency and subscription cancellation handling that only updates order subscription status.

## Verification

* `go test ./api/...`
* `npm test`
* `npm run build`
* Local smoke test against backend port `3000`, Vite `5173`, and `data/app.db`/`data/shared.db`.

## Out Of Scope

* Creating or editing Creem products through the local UI.
* Syncing Creem prices back into the local product catalog.
* Customer self-service billing portal.
* Changing membership when a subscription is merely scheduled to cancel.
* Revoking or downgrading membership on cancellation, expiration, pause, refund, or chargeback beyond the explicit cancellation order-status requirement.
* Pagination for the Product list.

## Research References

* [research/creem-product-webhook-notes.md](research/creem-product-webhook-notes.md) - Creem product billing modes, periods, webhook events, and local implications.

## Technical Notes

Likely backend areas:

* `api/db/migrations/app/*`
* `api/models/product.go`
* `api/usecase/product.go`
* `api/routes/product.go`
* `api/models/user.go`
* `api/usecase/user.go`
* `api/models/order.go`
* `api/usecase/order.go`
* `api/routes/order.go`
* `api/usecase/payment.go`
* `api/usecase/integrations/payment/ports.go`
* `api/integrations/payment/creem/creem.go`

Likely frontend areas:

* `frontend/src/api.js`
* `frontend/src/router.js`
* `frontend/src/App.svelte`
* `frontend/src/components/AppSidebar.svelte`
* `frontend/src/pages/Dashboard.svelte`
* new Product page under `frontend/src/pages/`

Useful implementation direction:

* Treat local product as the selected entitlement SKU.
* Keep Creem product id on the local product, not on the order creation UI as a free-form value.
* Keep payment status (`orders.status`) and subscription lifecycle status as separate columns.
* Add a fulfillment marker or membership-applied timestamp/order id guard so webhook retries cannot add time twice.
