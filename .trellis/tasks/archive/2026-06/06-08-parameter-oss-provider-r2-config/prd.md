# Parameter OSS Provider R2 Configuration

## Goal

Add an Object Storage Service (OSS) integration scenario to the Parameter page, with Cloudflare R2 as the first configurable provider.

The MVP should let an admin create/edit/list an R2 integration channel and persist the correct provider config/credentials. It should not implement file upload/download yet.

## What I Already Know

- User requested a new Trellis task for adding OSS provider configuration under Parameter.
- First provider is Cloudflare R2.
- Current Parameter integration scenarios are hard-coded as `payment`, `llm`, `sms`, and `email`.
- Backend scenario validation currently rejects anything outside those four scenarios.
- Parameter adapter schemas are code-owned in `api/usecase/parameter_schema.go`.
- Frontend `frontend/src/pages/Parameters.svelte` dynamically renders config and credential fields from the backend schema, but its tab/scenario list is hard-coded.
- Credential type validation depends on dictionary type `integration_credential_type`.
- Existing integration tables can store the OSS channel through `integration_channels`, `integration_credentials`, and related config JSON without new DB tables.

## Research References

- [`research/cloudflare-r2-parameter-fields.md`](research/cloudflare-r2-parameter-fields.md) - Cloudflare R2 S3-compatible fields and repo constraints.

## Requirements

- Add `oss` as a valid integration scenario.
- Add a Parameter adapter schema for Cloudflare R2:
  - scenario: `oss`
  - provider code: `cloudflare_r2`
  - adapter key: `oss.cloudflare_r2.s3_compatible`
  - credential type: `s3_access_key`
  - credential format: `json_object`
- R2 config fields:
  - `endpoint_url` (required URL), default/placeholder `https://<account_id>.r2.cloudflarestorage.com`
  - `bucket` (required text)
  - `region` (optional text), default `auto`
  - `public_base_url` (optional URL)
  - `key_prefix` (optional text)
- R2 credential fields:
  - `access_key_id` (required secret/text)
  - `secret_access_key` (required secret)
- Add `s3_access_key` to the integration credential type dictionary seed/migration so backend validation allows saving R2 credentials.
- Update the Parameters frontend to show an `OSS` tab and load/list/save OSS channels like the other scenarios.
- Update tests for backend schema filtering, scenario validation, route DTO behavior where needed, and frontend API/helper or page expectations where covered.

## Technical Approach

This is a Parameter configuration task, not an OSS runtime task.

Use the existing integration configuration storage:

```text
Parameter UI
  -> /api/parameters/integration-schemas?scenario=oss
  -> /api/parameters/integration-channels?scenario=oss
  -> integration_channels scenario=oss
  -> integration_credentials credential_type=s3_access_key
```

No new runtime package is required for MVP.

Do not add upload/download code yet:

```text
api/usecase/integrations/oss
api/integrations/oss/cloudflare_r2
api/framework/storage
artifacts table
```

Those are follow-up work for actual file operations.

## Expected Files

Likely modified:

- `api/models/integration.go` - add `IntegrationScenarioOSS`.
- `api/usecase/parameter.go` - allow `oss` in scenario normalization.
- `api/usecase/parameter_schema.go` - add Cloudflare R2 schema.
- `api/db/migrations/app/*_add_oss_integration_seed.sql` - add `s3_access_key` dictionary value.
- `frontend/src/pages/Parameters.svelte` - add OSS scenario metadata and state maps.
- `frontend/src/api.test.js` - cover OSS schema/channel helper paths if current tests enumerate scenarios.
- `api/usecase/parameter_test.go` - cover OSS schema filtering and R2 validation.
- `api/routes/parameter_test.go` - cover schema DTO if the route test enumerates schema scenarios.
- `.trellis/spec/backend/api-contracts.md` / `.trellis/spec/frontend/svelte-vite-embed.md` - update Parameter scenario docs if implementation changes the contract.

## Acceptance Criteria

- [x] `GET /api/parameters/integration-schemas?scenario=oss` returns exactly the Cloudflare R2 schema for MVP.
- [x] Creating an OSS channel with valid R2 config and credential JSON succeeds.
- [x] Creating an OSS channel with missing required R2 config/credential fields returns validation error.
- [x] The Parameters page shows an `OSS` tab.
- [x] Selecting the R2 adapter pre-fills config defaults where defined.
- [x] The credential type dropdown allows/saves `s3_access_key`.
- [x] Existing Payment, LLM, SMS, and Email Parameter behavior remains unchanged.
- [x] Relevant Go and frontend tests pass.

## Definition of Done

- Backend schema, scenario validation, and credential type seed are updated.
- Frontend Parameter page can create/edit/list OSS channels.
- Tests are updated and passing for touched behavior.
- No upload/download runtime is implemented in this task.

## Out of Scope

- Uploading files to R2.
- Downloading files from R2.
- Presigned URL generation.
- Connection testing / "Test credentials" button.
- Artifact tables or generated-file lifecycle.
- A runtime OSS adapter under `api/integrations/oss`.
- Multiple OSS providers beyond Cloudflare R2.

## Decision

Per user confirmation, this task follows the recommended MVP: Parameter configuration only, no backend connection test. This keeps the current anti-corruption boundary intact and avoids pulling OSS SDK/runtime behavior into the Parameter surface.
