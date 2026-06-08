# Creem Payment API Research

## Sources

* https://docs.creem.io/api-reference/introduction
* https://docs.creem.io/api-reference/endpoint/create-checkout
* https://docs.creem.io/code/webhooks
* https://docs.creem.io/features/checkout/checkout-api

## Findings

* Creem supports a hosted checkout flow. The server creates a checkout session, then the browser redirects the customer to the returned checkout URL.
* Checkout creation uses `POST /v1/checkouts`.
* Authentication uses the `x-api-key` request header.
* Creem has separate environments:
  * Test: `https://test-api.creem.io/v1`, usually with `creem_test_` keys.
  * Production: `https://api.creem.io/v1`, usually with `creem_` keys.
* Checkout creation requires `product_id`. Useful optional fields for this project include `request_id`, `success_url`, `customer`, `metadata`, and `units`.
* The checkout response includes `id`, `checkout_url`, `product_id`, and `status`. The browser should be redirected to `checkout_url`.
* Creem webhooks send signed event payloads. The signature is in the `creem-signature` header.
* Signature verification uses HMAC-SHA256 with the webhook secret as the key and the raw request payload as the message.
* The event most relevant for one-time order payment completion is `checkout.completed`.
* Creem success redirects can include query parameters such as `checkout_id`, `order_id`, `customer_id`, `product_id`, `request_id`, and `signature`, but business fulfillment should still rely on verified webhooks instead of only trusting browser redirects.

## Mapping to This Repo

* The existing `POST /api/orders/:id/pay` route directly marks the order as paid and triggers the `order.paid` domain event.
* A real hosted payment flow should split this:
  * User action creates an external checkout and leaves the local order `pending`.
  * Verified `checkout.completed` webhook calls the internal payment completion use case.
* The existing `usecase.PayOrder` behavior is still valuable as the internal fulfillment boundary because it is already idempotent for paid orders and publishes the `order.paid` domain event used by points and other listeners.
* The existing integration tables can manage payment channel configuration, credentials, and invocation records, but payment callbacks need a receipt table so raw callback payloads are persisted as part of the business/integration record instead of only logs.

## Implementation Notes

* Recommended first slice uses one configured Creem `product_id` or a small backend mapping strategy, then passes the internal `order_id` as `request_id` and metadata.
* The provider adapter should live under `api/integrations/payment/creem`.
* Business-facing ports should live under `api/usecase/integrations/payment`.
* Usecase code should depend on the payment port and registry, not the concrete Creem adapter.
* Seed/config data should not include real API keys. Credentials should remain DB-managed and encrypted by the existing integration credential helper.
