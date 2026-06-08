# Webhook Architecture Micro Diagnosis

## Goal

Clarify the long-term boundary for third-party webhook ingress so future providers and scenarios stay maintainable. The concrete question is whether webhook requests should continue to flow through `third request -> routes -> usecase -> integrations & models`, move toward `third request -> routes -> integrations -> usecase`, or be modeled primarily as durable DDD events.

## What I Already Know

* Current Creem payment webhook route is `POST /api/integrations/payment/:channel_code/webhooks/creem` in `index.go`.
* `api/routes/payment.go` handles HTTP concerns: raw body size limit, reading `creem-signature`, constructing `fwusecase.Context`, calling `usecase.ReceivePaymentWebhook`, and ACKing with `204/200`-style provider response behavior.
* `api/usecase/payment.go` currently owns most webhook orchestration: channel lookup, webhook enabled check, provider config resolution, adapter lookup, signature verification via adapter, normalized webhook receipt persistence, queue enqueue, worker handling, and order/membership updates.
* `api/integrations/payment/creem` implements provider-specific payload parsing, signature verification, HTTP calls, and maps raw Creem events into `payment.NormalizedWebhook`.
* Existing backend architecture spec states the default dependency direction is `routes -> usecase -> models -> db`, with external integration adapters implemented in `api/integrations/<scenario>/<provider>` and registered into usecase-visible ports.
* Existing eventing spec states routes must not publish domain events; usecase publishes durable DDD events after domain facts happen. DDD events are durable, queue-backed, subscriber-oriented, and intended for internal domain side effects.

## Diagnosis

The current shape is broadly correct for the repository's architecture, but the payment usecase currently carries too much integration-ingress detail. The clearest long-term architecture is not `routes -> integrations -> usecase`; it is:

```text
third request
  -> routes                         # HTTP/provider endpoint adapter only
  -> usecase webhook ingress         # application orchestration and durable receipt
  -> integration adapter port        # provider verification + normalization
  -> usecase business command        # PayOrder / CancelOrderSubscription / etc.
  -> DDD event                       # internal side effects after domain facts
```

In other words, keep usecase as the application boundary, but split the current "payment webhook" usecase into clearer sub-responsibilities over time.

## Recommended Direction

**Recommended: Usecase-owned ingress orchestration with provider normalization adapters, then publish DDD events only after business state changes.**

This keeps the dependency direction stable:

```text
routes -> usecase -> usecase/integrations ports -> provider adapters
routes -> usecase -> models
usecase -> framework/events
```

Provider adapters should understand provider wire formats, signatures, and raw event mapping. They should not decide business outcomes, write app models, enqueue app jobs directly, or call business usecases.

Usecase should own:

* channel/config/credential resolution
* webhook enabled checks
* safe receipt persistence and idempotency
* queueing or transaction boundaries
* mapping normalized provider events to business commands
* marking receipt status
* publishing domain events from business usecases after domain facts are committed

## Alternatives Considered

### A. Keep Current Flow: `routes -> usecase -> integrations & models`

Pros:

* Fits existing architecture and archguard boundaries.
* Keeps HTTP and provider details out of domain models.
* Allows usecase to manage transactions, idempotency, queueing, and safe errors.
* DDD events continue to represent internal business facts rather than external transport messages.

Cons:

* If left as-is, `usecase/payment.go` can become a large integration sink.
* Provider-specific names can leak upward, as seen in route reading `creem-signature`.

Verdict: Correct base direction, but needs responsibility refinement.

### B. Move to `routes -> integrations -> usecase`

Pros:

* Seems intuitive because webhook payloads are provider-shaped.
* Could isolate provider parsing near the route.

Cons:

* Makes `api/integrations` an application orchestrator, which conflicts with the current boundary where integrations are provider adapter implementations.
* Pushes business command selection into provider code, making providers aware of usecase workflows.
* Risks coupling provider packages to app models, queueing, receipts, and domain decisions.
* Harder to enforce consistent idempotency, observability, and safe error mapping across providers.

Verdict: Not recommended as the primary architecture.

### C. Treat Incoming Webhook As DDD Event First

Pros:

* Durable, retryable, observable pipeline already exists.
* Event subscribers can decouple side effects.

Cons:

* Third-party webhook payloads are external integration messages, not domain facts.
* Signature verification and idempotent receipt must happen before publishing any trusted domain event.
* DDD event payload rules forbid raw/sensitive/provider-shaped payloads.
* Existing spec says routes do not publish events; usecase publishes stable domain facts.

Verdict: Use DDD events after normalized webhook handling causes a real business fact, such as `order.paid`. Do not use DDD event as the first ingress abstraction.

## Proposed Target Shape

Short-term target:

```text
routes.ReceivePaymentWebhook
  -> usecase.ReceivePaymentWebhook
    -> payment.Adapter.Verify/NormalizePaymentWebhook
    -> integration_webhook_receipts
    -> integration-webhooks queue

integration-webhooks worker
  -> usecase.HandlePaymentWebhookJob
    -> payment.Adapter.NormalizePaymentWebhook
    -> usecase.PayOrder / CancelOrderSubscription / Apply renewal command
      -> events.Publish(order.paid) when applicable
```

Medium-term refinement:

* Extract generic webhook ingress helpers for shared receipt/idempotency/queue mechanics when a second webhook scenario appears.
* Keep scenario-specific orchestration in usecase packages, for example payment-specific mapping from normalized event to `PayOrder`.
* Avoid letting `api/integrations/<scenario>/<provider>` call usecase or models.
* Consider a provider-agnostic signature header map in `ReceivePaymentWebhookCmd` so routes do not need Creem-specific header names forever.

## Proposed Directory Shape

Phase 1 should avoid a broad package migration. Keep the existing import direction and split the current payment webhook logic into focused files:

```text
api/
  routes/
    payment.go                         # HTTP body limit, route params, headers, provider ACK

  usecase/
    payment.go                         # checkout/payment usecases plus shared payment helpers
    payment_webhook.go                 # payment webhook ingress, receipt/idempotency helpers, job dispatch
    payment_registry.go                # adapter registry, unchanged

    integrations/
      payment/
        ports.go                       # payment Adapter port, WebhookRequest, NormalizedWebhook

  integrations/
    payment/
      creem/
        creem.go                       # Creem HTTP/API calls, signature verification, payload normalization
        creem_test.go

  models/
    integration_webhook*.go            # receipt storage remains model-owned

  framework/
    queue/
    events/
```

If a second webhook scenario appears, then consider a shared usecase helper package:

```text
api/usecase/integrations/webhook/
  receipt.go                           # scenario-agnostic receipt/idempotency helper
  enqueue.go                           # scenario-agnostic integration webhook queue helper
```

Do not create a provider-first application package such as `api/integrations/payment/creem/usecase.go`; provider packages should remain adapters, not business orchestrators.

For the current codebase size, avoid splitting webhook logic into `payment_webhook_receipt.go` and `payment_webhook_dispatch.go` immediately. Those names are useful future extraction points, but today they would over-fragment one payment webhook flow before another scenario proves the shared shape.

## Refactor Scope Estimate

Recommended first implementation scope:

* Move existing payment webhook entry points and directly related helper functions out of `api/usecase/payment.go` into one `api/usecase/payment_webhook.go` file without behavior changes.
* Keep payment config parsing helpers in `payment.go` for now unless the implementation shows a very small, obvious move is needed.
* Change `ReceivePaymentWebhookCmd` / `payment.WebhookRequest` to carry provider headers in a provider-agnostic shape, such as `Headers map[string]string`, so the route no longer needs to know `creem-signature`.
* Update the Creem adapter to extract and verify its own signature header.
* Keep HTTP success ACK as `200 OK` with empty body unless a provider requires another response.
* Keep the existing `integration-webhooks` queue and `integration_webhook_receipts` persistence model.
* Keep DDD event publishing inside business usecases such as `PayOrder`, not in routes or provider adapters.

Expected touched areas:

```text
api/routes/payment.go
api/routes/payment_test.go
api/usecase/payment.go
api/usecase/payment_webhook.go
api/usecase/payment_test.go
api/usecase/integrations/payment/ports.go
api/integrations/payment/creem/creem.go
api/integrations/payment/creem/creem_test.go
```

The first pass should be mostly mechanical plus tests. A deeper generic `webhook` helper should wait until there is another real scenario, otherwise it risks abstracting around only Creem/payment.

## Requirements

* Record the recommended architecture direction for webhook ingress.
* Distinguish external integration messages from internal domain events.
* Identify what should remain in routes, usecase, integration adapters, models, queues, and DDD events.
* Avoid immediate code refactor unless a follow-up implementation task is explicitly created.
* Preserve existing working behavior while guiding future payment/SMS/email/LLM webhook additions.

## Acceptance Criteria

* [ ] The task documents a clear recommendation among current flow, integration-first flow, and DDD-event-first flow.
* [ ] The recommendation is grounded in current repository code and `.trellis/spec/backend` boundaries.
* [ ] The decision identifies a safe incremental migration path rather than requiring a disruptive rewrite.
* [ ] Follow-up implementation opportunities are listed separately from the architectural decision.

## Decision (ADR-lite)

**Context**: Webhooks are provider-originated HTTP requests with provider-specific payload, signature, retry, and ACK rules. The application must verify and persist them durably before applying internal business effects.

**Decision**: Keep `routes -> usecase` as the primary ingress boundary. Let provider integration adapters verify and normalize provider wire formats through usecase-owned ports. Use DDD events only after a domain fact has been established by a usecase.

**Consequences**:

* Provider code stays replaceable and does not own application workflow.
* Usecase remains the place for transactions, idempotency, safe errors, receipt state, and business command selection.
* DDD event payloads remain stable business facts and do not become raw provider message envelopes.
* Future cleanup should reduce the size of `api/usecase/payment.go` by extracting focused helpers, not by reversing dependency direction.

## Out of Scope

* No code changes in this task unless explicitly promoted to an implementation task.
* No migration of existing Creem webhook behavior.
* No change to public webhook URL contract.
* No change to queue/event framework behavior.

## Follow-up Implementation Ideas

* Add a provider-agnostic `Headers map[string]string` or `SignatureHeaders` to webhook cmd/port so routes do not know `creem-signature`.
* Split `HandlePaymentWebhookJob` into smaller scenario-level handlers: load receipt, normalize payload, dispatch business command, mark receipt.
* Introduce a usecase-level `webhook_ingress` helper only after at least two scenarios need shared receipt/idempotency logic.
* Add archguard coverage if provider adapter packages ever start importing `api/usecase` root, `api/models`, or route/http packages.
* Consider emitting a dedicated internal event only after a webhook receipt reaches a trusted normalized state if multiple internal workflows need to react to "trusted provider message received"; keep raw payload out of that event.

## Technical Notes

* Inspected `api/routes/payment.go`, `api/usecase/payment.go`, `api/usecase/integrations/payment/ports.go`, `api/integrations/payment/creem/creem.go`, `api/usecase/order.go`, and `api/usecase/events/*`.
* Relevant specs: `.trellis/spec/backend/directory-structure.md`, `.trellis/spec/backend/route-handler-guidelines.md`, `.trellis/spec/backend/eventing-guidelines.md`, `.trellis/spec/guides/cross-layer-thinking-guide.md`.
* Current eventing contract explicitly says routes must not publish domain events and payloads must be stable business DTOs rather than raw models or sensitive external payloads.
