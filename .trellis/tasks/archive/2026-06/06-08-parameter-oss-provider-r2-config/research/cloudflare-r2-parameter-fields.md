# Cloudflare R2 Parameter Fields

Date: 2026-06-08

## Question

What fields should the Parameters page capture for Cloudflare R2 as the first OSS provider?

## Official Findings

Cloudflare R2 supports an S3-compatible API. The official docs state that existing S3 SDK/library code can work with R2 by changing the endpoint URL.

Minimum connection fields for the S3-compatible API:

- bucket name
- S3 API endpoint URL, usually `https://<ACCOUNT_ID>.r2.cloudflarestorage.com`
- Access Key ID
- Secret Access Key
- SDK region value, commonly `auto` in Cloudflare examples

Cloudflare R2 also has related but separate concepts:

- public bucket access / custom domains for serving objects over HTTP
- presigned URLs for private object access
- temporary credentials
- data location / jurisdiction-specific endpoints

Those are useful future storage-runtime concerns, but the Parameter MVP only needs to store enough config for a backend OSS adapter to instantiate an S3-compatible client later.

Sources:

- https://developers.cloudflare.com/r2/get-started/s3/
- https://developers.cloudflare.com/r2/api/
- https://developers.cloudflare.com/r2/api/s3/api/
- https://developers.cloudflare.com/r2/objects/upload-objects/
- https://developers.cloudflare.com/r2/api/s3/presigned-urls/

## Recommended Parameter Schema

Scenario:

```text
oss
```

Adapter key:

```text
oss.cloudflare_r2.s3_compatible
```

Provider code:

```text
cloudflare_r2
```

Credential type:

```text
s3_access_key
```

Credential format:

```text
json_object
```

Config fields:

- `endpoint_url` (url, required), default/placeholder `https://<account_id>.r2.cloudflarestorage.com`
- `bucket` (text, required)
- `region` (text, optional), default `auto`
- `public_base_url` (url, optional), for future public object URL composition
- `key_prefix` (text, optional), for future logical namespace separation

Credential fields:

- `access_key_id` (secret/text, required)
- `secret_access_key` (secret, required)

## Repo Constraints

- Parameter scenario validation currently only allows `payment`, `llm`, `sms`, and `email`.
- The frontend Parameters page has a hard-coded `scenarios` list and per-scenario state maps.
- Credential type validation reads the `integration_credential_type` dictionary, so adding schema alone is not enough. A seed/migration needs to add `s3_access_key`.
- The existing dynamic schema renderer already supports text, URL, number, boolean, secret, options, and JSON object credentials.
- The requested task is Parameter configuration only. Upload/download runtime, artifact storage, SDK client code, and connection testing should remain out of scope unless explicitly added later.

