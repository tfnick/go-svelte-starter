# Settings Logo Upload Uses Primary OSS Provider

## Goal

Modify `Setting -> General` logo upload so site logos are stored in the configured primary OSS provider instead of the local OSS provider. The feature should keep the existing anti-corruption boundary: settings usecase depends on OSS ports and integration-channel configuration, not provider SDK details.

## Requirements

* Do not use the local OSS provider for site-logo uploads or reads.
* Resolve exactly one enabled primary OSS channel (`scenario=oss`, `is_primary=true`) from Parameter integration configuration.
* Build `oss.ProviderConfig` from the primary OSS channel `config_json` and `credential_value`.
* Use the registered OSS adapter for the primary channel `adapter_key` when saving and loading the configured logo.
* If no enabled primary OSS provider is configured, logo upload is not allowed.
* `GET /api/settings/site` should expose whether logo upload is currently available so the Settings page can disable upload before submit.
* `GET /api/settings/public/logo` should stream the configured logo from the same primary provider used for upload; when no logo is configured it still redirects to `/logo.png`.
* Keep the default logo fallback as `frontend/public/logo.png`.
* Keep accepted logo formats as PNG, JPEG, and WebP with magic-byte validation and 2 MiB max size.

## Acceptance Criteria

* [ ] Fresh install with no primary OSS provider shows default logo and disables Settings logo upload.
* [ ] Upload endpoint returns a safe validation error when no enabled primary OSS provider is configured.
* [ ] With an enabled primary OSS provider and registered adapter, upload stores bytes through that adapter and persists logo metadata.
* [ ] Public logo read uses the persisted provider/channel metadata and registered adapter, not the local adapter.
* [ ] Settings page disables file selection/upload when `logo_upload_available=false`.
* [ ] `go test ./...`, `cd frontend && npm test`, and `cd frontend && npm run build` pass.

## Technical Approach

* Add model/usecase helpers to find the enabled primary OSS integration channel.
* Add a usecase mapper from integration-channel config JSON and credential JSON to `oss.ProviderConfig`.
* Update site-logo metadata to store the OSS channel/adapter/provider used for the uploaded object, so public read can use the same provider even if the current primary changes later.
* Register provider-backed OSS adapters for configured OSS adapter keys. Use a generic S3-compatible adapter for Cloudflare R2 and Aliyun OSS, keeping provider HTTP/signing details inside `api/integrations/oss`.
* Update Settings API DTO and frontend state to include upload availability and a short unavailable reason.

## Decision (ADR-lite)

**Context**: The previous implementation used a special local adapter key for site logos. This was useful for bootstrapping but conflicts with the Parameter-owned OSS provider model.

**Decision**: Settings resolves the primary OSS channel from integration configuration and invokes the provider adapter through `api/usecase/integrations/oss`. The local adapter is not registered for site logos in app startup.

**Consequences**: Logo upload now requires admin OSS configuration first. The settings UI can explain the disabled state instead of allowing an upload that will fail later. Provider-specific transport remains behind OSS adapter implementations.

## Out of Scope

* Object deletion or cleanup of old logo objects.
* Direct browser upload/presigned upload flow.
* Multiple logo variants or image resizing.
* A separate public asset CDN management UI.

## Technical Notes

* Current task directory: `.trellis/tasks/06-08-settings-logo-upload-primary-oss`
* Existing site settings usecase: `api/usecase/setting.go`
* Existing OSS port: `api/usecase/integrations/oss/ports.go`
* Existing primary OSS flag: `integration_channels.is_primary`
* Existing Settings page: `frontend/src/pages/Settings.svelte`
