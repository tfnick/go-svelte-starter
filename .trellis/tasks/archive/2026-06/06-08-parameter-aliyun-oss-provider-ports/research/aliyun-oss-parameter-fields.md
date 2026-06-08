# Aliyun OSS Parameter Fields

Date: 2026-06-08

## Question

What fields should the Parameters page capture for Aliyun OSS, and what usecase-level OSS port should be reserved for future runtime adapters?

## Official Findings

Aliyun OSS supports S3-compatible access, which lets S3-compatible SDK/client code talk to OSS through a compatible endpoint and AccessKey credentials. OSS endpoints are region-specific, for example `oss-cn-hangzhou.aliyuncs.com`, and applications still need a bucket name to address objects.

The minimum configuration for a future S3-compatible runtime adapter is:

* OSS endpoint URL
* bucket name
* optional region, useful for SDK configuration and documentation parity
* AccessKey ID
* AccessKey secret

Separate runtime concerns include object ACLs, custom domains, presigned URLs, STS temporary credentials, storage class, and lifecycle rules. Those should not be mixed into the Parameter MVP unless a runtime usecase needs them.

Sources:

* https://help.aliyun.com/zh/oss/developer-reference/compatibility-with-amazon-s3
* https://www.alibabacloud.com/help/en/oss/developer-reference/compatibility-with-amazon-s3
* https://help.aliyun.com/zh/oss/user-guide/regions-and-endpoints

## Recommended Parameter Schema

Scenario:

```text
oss
```

Adapter key:

```text
oss.aliyun_oss.s3_compatible
```

Provider code:

```text
aliyun
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

* `endpoint_url` (url, required), placeholder `https://oss-cn-hangzhou.aliyuncs.com`
* `bucket` (text, required)
* `region` (text, optional), placeholder `cn-hangzhou`
* `public_base_url` (url, optional), for future public/custom-domain object URL composition
* `key_prefix` (text, optional), for future logical namespace separation

Credential fields:

* `access_key_id` (secret/text, required)
* `secret_access_key` (secret, required)

## Recommended Usecase Port

Create `api/usecase/integrations/oss/ports.go` and keep it provider-agnostic:

* `ProviderConfig` for channel/provider/adapter plus endpoint, bucket, region, public base URL, key prefix, and credentials.
* `PutObjectRequest` / `PutObjectResult` for upload.
* `GetObjectRequest` / `GetObjectResult` for download.
* `DeleteObjectRequest` / `DeleteObjectResult` for deletion.
* `PresignObjectRequest` / `PresignObjectResult` for future private object access.
* `Adapter` interface containing these operations.

Do not implement an Aliyun adapter in this task. Provider adapter code should later live under `api/integrations/oss/aliyun` and be registered through a usecase registry.

## Repo Constraints

* `api/usecase` must not import `api/integrations`.
* Existing registries use `Register<Scenario>Adapter(adapterKey string, adapter <scenario>.Adapter)` and an unexported lookup helper.
* Parameter schema validation already supports required URL/text fields and JSON object credential fields.
* `s3_access_key` is already seeded, so Aliyun OSS does not need a new credential dictionary value.
