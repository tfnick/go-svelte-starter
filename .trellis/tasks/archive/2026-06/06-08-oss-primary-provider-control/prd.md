# OSS Primary Provider Control

## Goal

Add an OSS-specific control under Parameter -> OSS so administrators can mark one OSS provider/channel as the primary provider. At any time, only one OSS provider may be primary; enabling one provider as primary must ensure every other OSS provider is no longer primary.

## What I Already Know

* The user wants this as a new Trellis task.
* The feature belongs under the existing Parameter -> OSS provider/channel configuration UI.
* Current Parameter integration channels are stored in `integration_channels` and managed through shared create/update/list APIs.
* Parameter currently supports scenarios `payment`, `llm`, `sms`, `email`, and `oss`.
* The OSS tab already exists in `frontend/src/pages/Parameters.svelte`.
* The user requirement is explicit: if one OSS provider is enabled as primary, all other OSS providers must not be primary; there can only be one primary provider.

## Confirmed Decisions

* The system allows zero primary OSS providers. This keeps setup flexible when no OSS provider is ready yet, and enforces "at most one" rather than "exactly one".
* Recommended: when an admin saves or updates an OSS channel with `is_primary = true`, the backend should atomically unset `is_primary` on all other OSS channels.
* Recommended: disabling a primary OSS channel should also unset its primary flag so the UI never shows a disabled provider as the active primary.
* Recommended: the `is_primary` field should exist on integration channels but only be editable/meaningful for `scenario = oss`; non-OSS scenarios always return false or ignore the field.

## Requirements

* Add a primary provider control for OSS channels in Parameter -> OSS.
* Persist whether an OSS integration channel is the primary OSS provider.
* API list/create/update responses include the primary flag.
* API create/update requests can set the primary flag.
* When an OSS channel is saved as primary, all other OSS channels are unset as primary in the same transaction.
* Non-OSS scenarios must not accidentally gain multiple primary behavior or visible controls.
* The OSS channel list should clearly show which provider is primary.
* The OSS channel form should allow setting/unsetting primary status.

## Acceptance Criteria

* [x] Parameter -> OSS form includes a "Primary provider" toggle.
* [x] OSS channel list displays a primary indicator for the current primary provider.
* [x] Creating an OSS channel with primary enabled saves it as primary.
* [x] Updating an OSS channel to primary unsets every other OSS channel's primary flag.
* [x] Updating an OSS channel to not primary leaves other OSS channels unchanged.
* [x] Disabling a primary OSS channel unsets primary status for that channel.
* [x] At most one OSS channel can be primary after create/update/enable-disable operations.
* [x] Zero OSS primary providers is a valid state.
* [x] Payment/LLM/SMS/Email channels do not show the primary provider control.
* [x] Backend tests cover create, update, and disable behavior for the OSS uniqueness rule.
* [x] Frontend tests cover API helper payload/response shape and route/UI behavior where practical.

## Definition Of Done

* Backend migration and model/usecase/route updates are implemented.
* Frontend Parameter OSS tab supports the new field.
* Relevant backend and frontend tests pass.
* Existing Parameter scenarios remain compatible.
* Trellis check/spec review is completed before commit.

## Out Of Scope

* Implementing actual OSS upload/download routing through the selected primary provider.
* Health checks or connection tests for OSS providers.
* Provider-specific primary selection rules per environment.
* Enforcing that there must always be one primary provider.
* Applying the primary-provider concept to payment, LLM, SMS, or email channels.

## Technical Notes

Likely backend areas:

* `api/db/migrations/app/*`
* `api/models/integration.go`
* `api/usecase/parameter.go`
* `api/routes/parameter.go`
* `api/models/integration_test.go`
* `api/usecase/parameter_test.go`
* `api/routes/parameter_test.go`

Likely frontend areas:

* `frontend/src/pages/Parameters.svelte`
* `frontend/src/api.js`
* `frontend/src/api.test.js`

Recommended implementation shape:

* Add `is_primary INTEGER NOT NULL DEFAULT 0` to `integration_channels`.
* Add a partial unique index for SQLite: one primary OSS row only, e.g. on `scenario` where `scenario = 'oss' AND is_primary = 1`.
* In usecase transaction, when OSS `IsPrimary = true`, unset other OSS channels before saving the selected one as primary.
* On disable, clear `is_primary` for the disabled channel.
* Keep the backend as the final authority even if the frontend prevents obvious conflicts.

## Implementation Summary

* Added `integration_channels.is_primary` plus a SQLite partial unique index that permits at most one primary OSS channel.
* Normalized `is_primary` to OSS-only, enabled-only behavior in the Parameter usecase.
* Clearing of existing OSS primary rows runs in the same transaction before saving a new primary OSS channel.
* Disabling a channel clears its own primary flag.
* Parameter route DTOs and the OSS tab UI now read/write and display `is_primary`.
* Updated backend and frontend Trellis specs with the OSS primary-provider contract.

## Verification

* `go test ./api/models ./api/usecase ./api/routes -run "Test(OSSPrimary|CreateParameterIntegrationChannel|UpdateParameterIntegrationChannel|SetParameterIntegrationChannelEnabled|ListParameterIntegrationChannels|NonOSSParameterChannelIgnoresPrimaryFlag|ParameterIntegrationChannel)"`
* `go test ./api/...`
* `cd frontend; npm test`
* `cd frontend; npm run build`
