# Implement Creem Payment Integration Slice

## Goal

Implement a concrete payment vertical slice based on the previously designed anti-corruption layer: integrate Creem hosted checkout, keep payment provider details outside business use cases, and replace the current direct "mark paid" user flow with a real external-payment confirmation path.

## What I Already Know

* The user wants the next implementation slice to be payment, based on Creem official API documentation.
* Creem checkout creation is `POST /v1/checkouts`, authenticated by `x-api-key`.
* Creem has separate test and production API bases: `https://test-api.creem.io/v1` and `https://api.creem.io/v1`.
* Creem checkout creation requires `product_id` and returns `checkout_url`.
* Creem webhooks use `creem-signature`, HMAC-SHA256, and raw payload verification.
* `checkout.completed` is the relevant success event for one-time order payment.
* The current backend route `POST /api/orders/:id/pay` directly marks an order paid.
* The current `usecase.PayOrder` is idempotent for already-paid orders and publishes `order.paid`, which downstream DDD event listeners use for points and related side effects.
* The existing integration framework already has DB-backed credentials, channels, operation configs, invocations, and the LLM slice pattern.

## Assumptions (Temporary)

* Payment provider configuration should be managed from DB/admin-side configuration, consistent with the LLM design.
* Real API keys and webhook secrets will not be committed or seeded in source code.
* Browser success redirects are useful for UX, but local fulfillment should rely on verified webhooks.
* The first payment slice should keep the existing order domain model and reuse `PayOrder` as the internal fulfillment boundary.
* The first payment slice will use a backend-configured Creem `product_id` for checkout creation instead of mapping every local product.

## Research References

* [`research/creem-payment-api.md`](research/creem-payment-api.md) - Creem hosted checkout, API auth, webhook signature, and repo mapping notes.

## Requirements (Evolving)

* Implement Approach A: hosted checkout plus webhook fulfillment.
* Use a backend/admin-configured Creem `product_id` from payment channel/operation config.
* Add a provider-neutral payment port under `api/usecase/integrations/payment`.
* Add a Creem adapter under `api/integrations/payment/creem`.
* Add usecase orchestration to create a payment checkout for an existing pending order.
* Store payment provider calls in `integration_invocations`.
* Add a callback receipt model/table for payment webhooks, including raw payload, safe metadata, signature header hash or safe signature metadata, provider event ID, event type, and processing status.
* Add a public webhook route for Creem callbacks that reads the raw body, verifies the signature, persists the encrypted callback receipt, deduplicates by provider event ID, enqueues an `integration-callbacks` technical job, and returns quickly.
* Add a callback worker that loads the receipt, asks the Creem adapter to normalize the payload, updates receipt processing status, and calls internal order payment completion for `checkout.completed`.
* On normalized payment success, call the internal order payment completion behavior so `order.paid` and its listeners still work.
* Update the frontend order payment action to launch hosted checkout if the chosen MVP includes end-user payment flow replacement.

## Acceptance Criteria (Evolving)

* [ ] A pending order can request a Creem checkout and receive a redirectable `checkout_url`.
* [ ] Creating a checkout does not mark the local order as paid.
* [ ] A valid `checkout.completed` webhook marks the matching order as paid.
* [ ] Duplicate webhook delivery is safe and does not duplicate points/event side effects.
* [ ] Invalid webhook signatures are rejected and recorded safely.
* [ ] Payment channel credentials/configuration are DB-managed and provider details do not leak into order business logic.
* [ ] Tests cover adapter request construction, signature verification, usecase orchestration, route behavior, and idempotent webhook processing.

## Feasible Approaches

### Approach A: Hosted Checkout + Webhook Fulfillment (Recommended)

The order payment button creates a Creem checkout, opens or redirects to `checkout_url`, and leaves the order pending until a verified `checkout.completed` webhook confirms payment.

Pros:
* Matches Creem's intended hosted checkout flow.
* Preserves eventual consistency and provider-independent domain events.
* Avoids trusting browser redirect parameters for fulfillment.

Cons:
* Requires webhook route, receipt persistence, and local test strategy for callback simulation.
* The frontend payment UX changes from instant "paid" to "checkout started / waiting for confirmation".

### Approach B: Adapter and Checkout API Only

Implement the Creem adapter, DB config lookup, and backend checkout creation endpoint, but keep webhook fulfillment and frontend replacement out of this task.

Pros:
* Smaller first slice and easier to test without external callback setup.
* Useful if we want to validate Creem API shape first.

Cons:
* Does not fully replace the current direct event flow.
* No real payment confirmation path yet.

### Approach C: Full Payment Management Slice

Implement checkout creation, webhook fulfillment, callback receipt management, payment status UI, retry/admin tools, and channel management screens.

Pros:
* More complete operationally.
* Better visibility for payment support/admin workflows.

Cons:
* Much larger scope and likely too broad for one vertical slice.
* Risks mixing core payment boundary work with admin UX decisions.

## Decision (ADR-lite)

**Context**: The current order payment route marks orders paid immediately. Creem uses hosted checkout and asynchronous webhook confirmation, so directly marking the order paid at checkout creation would break the external-payment boundary.

**Decision**: Use hosted checkout plus webhook fulfillment for the MVP. Creating a checkout returns `checkout_url` and leaves the order `pending`; a verified `checkout.completed` callback is first persisted as an encrypted receipt and queued for asynchronous processing, and the callback worker calls the internal `PayOrder` fulfillment behavior after provider payload normalization.

**Consequences**: Payment becomes eventually consistent, duplicate webhook delivery must be safe, and the frontend needs a "checkout started / waiting for confirmation" UX instead of instant success. Existing `order.paid` domain event listeners continue to run from the internal fulfillment boundary. Provider callbacks remain integration pipeline records/jobs, not DDD domain events.

## Product Mapping Decision

**Context**: Creem checkout creation requires `product_id`, while the current local order amount is calculated from local products. Supporting per-product mapping or dynamic price creation would broaden the first slice.

**Decision**: For the MVP, use a backend/admin-configured Creem `product_id` in payment integration configuration. The checkout request includes the internal order ID as `request_id` and metadata so the webhook can map the completed checkout back to the local order.

**Consequences**: The first slice validates the payment anti-corruption boundary and webhook fulfillment path quickly. Exact local order item-to-Creem product reconciliation is out of scope and can be added later through a product mapping table or richer payment pricing strategy.

## Open Questions

* None for the MVP.

## Definition of Done (Team Quality Bar)

* Tests added/updated for backend usecase/model/route/adapter behavior.
* Frontend tests updated if the payment button behavior changes.
* Lint, typecheck, and test commands pass or any failures are explicitly documented.
* No real Creem secrets are committed.
* Rollout/rollback notes are captured if route behavior changes.

## Out of Scope (Explicit)

* Subscription lifecycle management.
* License-key flows.
* Real production credential provisioning.
* Admin UI for editing integration channel credentials unless explicitly added later.
* Automatic provider failover.
* Per-local-product to Creem product mapping.
* Dynamic checkout price creation from local order line items.

## Technical Notes

* Existing direct payment route: `api/routes/order.go`.
* Existing internal payment fulfillment: `api/usecase/order.go`.
* Existing integration DB tables: `api/db/migrations/app/010_add_integrations.sql`.
* Existing integration model helpers: `api/models/integration.go`.
* Existing LLM adapter pattern: `api/usecase/integrations/llm/ports.go` and `api/integrations/llm/deepseek/deepseek.go`.
* Existing frontend payment action: `frontend/src/api.js` and `frontend/src/pages/Dashboard.svelte`.
* Must follow recovered design task: `.trellis/tasks/06-06-external-integration-anticorruption-layer/info.md`.
* Payment design conventions from the recovered design task:
  * business-scenario-first port under `api/usecase/integrations/payment`;
  * provider adapter under `api/integrations/payment/creem`;
  * callback route owns HTTP ingress and provider ACK shape, not business mutation;
  * callback receipt persists encrypted raw payload, hash, safe snapshot, idempotency key, and processing status;
  * `integration-callbacks` is a technical queue; it is not a DDD domain event;
  * domain events remain usecase-owned and are emitted only after provider payloads are normalized into business facts.

## Divergence Considerations

* Future evolution: multiple payment providers, subscriptions, refunds, chargebacks, payment reconciliation, and admin channel switching.
* Related scenarios: SMS manual resend/channel switching and HRM callback receipts follow the same invocation/receipt pattern.
* Failure and edge cases: checkout creation timeout, pending checkout retry, duplicate webhook delivery, invalid signature, unknown order ID in metadata, and provider event payload schema drift.
