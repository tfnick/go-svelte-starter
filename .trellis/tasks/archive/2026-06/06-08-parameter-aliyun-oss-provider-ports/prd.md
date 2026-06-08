# Parameter Aliyun OSS Provider And OSS Ports

## Goal

Continue the OSS integration work by adding Aliyun OSS as another configurable Parameter provider, and introduce a usecase-layer OSS port so future runtime upload/download adapters can plug in without leaking provider-specific details into business usecases.

## What I Already Know

* The user asked to create a new task and continue Aliyun OSS provider integration.
* The user explicitly asked to add OSS ports in the usecase layer.
* The existing OSS/R2 task already added `scenario=oss`, `credential_type=s3_access_key`, the Cloudflare R2 schema, and the frontend OSS tab.
* Existing integration boundaries use `api/usecase/integrations/<scenario>/ports.go` plus a top-level usecase registry, for example LLM and payment.
* This task should preserve the anti-corruption design: Parameter stores config/credential; provider SDK/runtime implementations live outside usecase.

## Research References

* [`research/aliyun-oss-parameter-fields.md`](research/aliyun-oss-parameter-fields.md) - Aliyun OSS S3-compatible fields and MVP scope.

## Requirements

* Add a Parameter adapter schema for Aliyun OSS:
  * scenario: `oss`
  * provider code: `aliyun`
  * adapter key: `oss.aliyun_oss.s3_compatible`
  * credential type: `s3_access_key`
  * credential format: `json_object`
* Aliyun OSS config fields:
  * `endpoint_url` (required URL), placeholder `https://oss-cn-hangzhou.aliyuncs.com`
  * `bucket` (required text)
  * `region` (optional text), placeholder `cn-hangzhou`
  * `public_base_url` (optional URL)
  * `key_prefix` (optional text)
* Aliyun OSS credential fields:
  * `access_key_id` (required secret)
  * `secret_access_key` (required secret)
* Add `api/usecase/integrations/oss/ports.go` with usecase-owned request/result/config DTOs and an `Adapter` interface.
* Add a usecase registry for OSS adapters using the same pattern as LLM/payment registries.
* Update backend specs and tests so OSS schema listing returns both Cloudflare R2 and Aliyun OSS.

## Acceptance Criteria

* [x] `GET /api/parameters/integration-schemas?scenario=oss` returns Cloudflare R2 and Aliyun OSS schemas.
* [x] Creating an Aliyun OSS channel with valid endpoint/bucket/credentials succeeds.
* [x] Missing required Aliyun OSS config or credential fields returns a validation error.
* [x] The usecase layer exposes an OSS adapter port without importing provider implementations.
* [x] Existing R2 OSS behavior remains unchanged.
* [x] Relevant Go tests pass.

## Definition of Done

* Backend schema, ports, registry, and tests are updated.
* Specs document the new Aliyun OSS schema and the OSS port boundary.
* No upload/download runtime implementation or SDK dependency is added.
* `go test ./...` passes.

## Technical Approach

Use the existing Parameter storage and schema renderer. Add Aliyun OSS as a second schema under `scenario=oss`, sharing `s3_access_key` credentials with R2. Add a minimal usecase-owned OSS port with upload/download/delete/presign operations shaped around business objects and safe metadata, but leave provider adapters and runtime usecases out of scope for this task.

## Decision (ADR-lite)

**Context**: OSS providers need admin configuration now, and later generated artifacts/PPT workflows will need upload/download or presigned URL behavior.

**Decision**: Add provider config schema now and define usecase OSS ports now, but do not implement provider SDK clients in this task.

**Consequences**: Future Aliyun/R2 runtime adapters can implement a stable usecase port. The current Parameter UI remains configuration-only, so this does not introduce network calls, SDK dependencies, or artifact lifecycle tables yet.

## Out of Scope

* Actual upload/download/delete/presigned URL execution.
* Aliyun OSS SDK or S3 SDK dependency.
* `api/integrations/oss/aliyun` provider adapter implementation.
* Artifact tables, generated-file lifecycle, or PPT generation storage.
* Frontend upload/test-connection UI.

## Technical Notes

* Existing port examples: `api/usecase/integrations/llm/ports.go`, `api/usecase/integrations/payment/ports.go`.
* Existing registries: `api/usecase/llm_registry.go`, `api/usecase/payment_registry.go`.
* Existing OSS schema: `api/usecase/parameter_schema.go`.
* Main spec sections: `.trellis/spec/backend/directory-structure.md` external integration boundary and `.trellis/spec/backend/api-contracts.md` Parameter Integration Management.
