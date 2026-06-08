# Implement Creem Webhook Exposure And Signature Hardening

## Goal

Expose a production-ready Creem webhook ingress for payment success events and make the security boundary explicit: Creem calls a public HTTPS URL, while our system authenticates the request with Creem's `creem-signature` header and the channel's configured `webhook_secret`.

## What I Already Know

* The current backend route is `POST /api/integrations/payment/:channel_code/webhooks/creem`.
* The user expects Creem to call a URL like `https://mydomain.com/api/integrations/payment/creem-test/webhooks/creem`.
* `creem-test` is a `channel_code` used to locate DB-managed channel configuration; it is not a secret.
* Current code reads the raw body, reads `creem-signature`, loads the payment channel, checks `webhook_enabled`, and verifies the signature through the Creem adapter.
* The Creem credential schema already includes `api_key` and `webhook_secret`.
* Current route ACK returns `204 No Content`; Creem docs expect successful webhook handling to return `200 OK`.
* Provider webhook routes are external ingress and intentionally do not use internal frontend JSON envelope.

## Requirements

* Keep the public webhook route outside `RequireAuth()` and outside `/open-api/v1`.
* Document and enforce that Creem webhook authentication is based on `creem-signature` + `webhook_secret`, not JWT, cookie auth, Open API key, source IP, or URL secrecy.
* Keep route-level payload handling based on raw request body; signature verification must happen before trusting parsed JSON.
* Return `200 OK` for a successfully accepted Creem webhook, matching Creem's documented retry expectations.
* Continue rejecting missing channel, disabled webhook channel, invalid payload, missing signature, and invalid signature with safe errors.
* Keep receipt persistence and queue behavior unchanged for accepted webhooks.
* Update backend specs to make the public HTTPS address shape and security boundary explicit.

## Acceptance Criteria

* [ ] `POST /api/integrations/payment/:channel_code/webhooks/creem` succeeds without `RequireAuth()` when the Creem signature is valid.
* [ ] Successful Creem webhook ACK returns HTTP `200 OK`.
* [ ] Route/usecase tests assert `creem-signature` is required and invalid signatures do not enqueue webhook work.
* [ ] Tests continue to verify accepted webhooks are persisted and queued exactly once.
* [ ] Spec documents the production URL shape `https://<domain>/api/integrations/payment/<channel_code>/webhooks/creem`.
* [ ] Spec documents that `channel_code` is only a routing/config lookup key and is not an authentication secret.
* [ ] Spec documents that provider webhook ingress should not require user JWT/OpenAPI key and should not rely on source IP allowlisting.

## Definition Of Done

* `go test ./...` passes.
* Relevant route/usecase/adapter tests are updated.
* `.trellis/spec/backend/*` is updated if the public provider webhook contract changes.
* No database runtime files are committed.

## Technical Approach

* Keep the existing route path and channel configuration model.
* Change the Creem webhook route ACK from `204 No Content` to `200 OK` using a provider-ACK response that does not use the internal `{success,data}` envelope.
* Keep HMAC verification inside the Creem adapter so provider-specific details remain behind the payment port.
* Expand route/usecase tests around status code and invalid signature behavior if coverage is not already explicit.
* Update `api-contracts.md` and/or `route-handler-guidelines.md` to clarify webhook exposure and authentication.

## Decision (ADR-lite)

**Context**: Creem's dashboard stores our webhook URL and provides a signing secret. Creem cannot send our user JWT or Open API key.

**Decision**: The webhook route remains publicly reachable over HTTPS and authenticates provider requests with `creem-signature` plus the channel's configured `webhook_secret`.

**Consequences**: The route must not be hidden behind user auth middleware. Operational security depends on HTTPS, raw-body HMAC validation, secret rotation/config hygiene, idempotency, and receipt auditing. `channel_code` helps select the correct config but must never be treated as a credential.

## Out Of Scope

* Adding a full webhook URL generation UI.
* Adding IP allowlist enforcement.
* Rotating webhook secrets automatically.
* Changing the payment fulfillment queue architecture.
* Introducing mTLS or gateway-specific WAF rules.

## Technical Notes

* Relevant files inspected:
  * `index.go`
  * `api/routes/payment.go`
  * `api/usecase/payment.go`
  * `api/integrations/payment/creem/creem.go`
  * `.trellis/spec/backend/route-handler-guidelines.md`
  * `.trellis/spec/backend/api-contracts.md`
* Creem docs state webhook payloads are signed with HMAC-SHA256 and provided via the `creem-signature` header. They also note successful receipt should return `200 OK`, and Creem does not provide static source IPs for allowlisting.
