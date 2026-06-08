# Creem Product and Webhook Notes

Research date: 2026-06-08

## Sources

* https://docs.creem.io/llms-full.txt
* https://creem.io/SKILL.md
* https://docs.creem.io/api-reference/introduction
* https://docs.creem.io/code/webhooks

## Findings

* Creem products support two billing modes: `onetime` and `recurring`.
* Creem recurring products use billing periods that map directly to the requested local intervals:
  * `every-month`
  * `every-three-months`
  * `every-six-months`
  * `every-year`
* Creem webhook events relevant to this task include:
  * `checkout.completed`
  * `subscription.active`
  * `subscription.paid`
  * `subscription.canceled`
  * `subscription.scheduled_cancel`
  * `subscription.past_due`
  * `subscription.expired`
  * `subscription.paused`
* Creem signs webhook payloads with the `creem-signature` header using HMAC-SHA256 over the raw request body. The current adapter already implements this verification.
* Creem recommends server-side webhook fulfillment for production entitlements. Browser success redirects are useful for UX only.
* Creem distinguishes `canceled` from `scheduled_cancel`. This task only explicitly requires local handling for actual cancellation, but the order subscription-status field should allow scheduled cancellation so future handling does not need another schema redesign.

## Implications For This Repo

* The local product should become the app-owned catalog record and should carry the Creem product id used at checkout time.
* The integration channel can keep shared payment configuration such as base URL, API key, webhook secret, success URL, and units, but the Creem product id used for checkout should come from the selected local product.
* The payment webhook normalizer needs to expose enough data for subscription lifecycle handling: provider checkout id, provider order id, provider customer id, provider subscription id when present, provider product id, and event type.
* The fulfillment usecase should continue to treat webhooks as authoritative and must be idempotent because Creem can retry failed webhook deliveries.
