# Creem Checkout and Webhook Research

## Sources

* Creem API Reference: `https://docs.creem.io/api-reference/endpoint/create-checkout`
* Creem Webhooks: `https://docs.creem.io/skills/creem-api/WEBHOOKS`

## Checkout API Notes

* Creem test API base URL is `https://test-api.creem.io/v1`.
* Hosted checkout creation uses `POST /checkouts`.
* Authentication uses `x-api-key`.
* Request fields relevant to this repo:
  * `product_id`: Creem product identifier.
  * `request_id`: can carry the local order ID for reconciliation/idempotency.
  * `success_url`: optional browser redirect target after successful payment.
  * `customer.email`: optional customer identity.
  * `metadata`: can carry `order_id`, `user_id`, and amount for recovery.
  * `units`: optional quantity.
* Response fields relevant to this repo:
  * `id`: provider checkout/payment identifier.
  * `checkout_url`: URL where the frontend should redirect the user.
  * `status`: provider checkout status.

## Webhook Notes

* Payment success is represented by `checkout.completed`.
* Webhook endpoint must read the raw body for signature verification.
* Creem signs payloads with HMAC-SHA256 using the webhook secret.
* The project currently reads the `creem-signature` header and verifies the raw body with the configured webhook secret.
* The webhook object can carry `request_id` and/or metadata, which the adapter should normalize to the local `order_id`.

## Mapping to Current Repo

* Current adapter path: `api/integrations/payment/creem/creem.go`.
* Current checkout route: `POST /api/orders/:id/payment-checkout`.
* Current webhook route: `POST /api/integrations/payment/:channel_code/webhooks/creem`.
* Current payment config schema already asks for:
  * `base_url`
  * `product_id`
  * `success_url`
  * `units`
  * `api_key`
  * `webhook_secret`
* Current payment flow records integration invocations and webhook receipts, queues webhook jobs, and marks orders paid through `PayOrder`.

## Risks / Checks

* Confirm exact Creem event payload casing against live test events; code currently expects `eventType`.
* Confirm exact signature header name and format against live test delivery; code currently expects raw hex digest in `creem-signature`.
* Product mapping is currently channel-level, so all orders use one Creem product unless we add product-level mapping.
* Current frontend already redirects to `checkout_url`, but post-checkout return UX may need refresh/polling if `success_url` returns to `/orders`.
