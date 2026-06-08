# OpenAPI Client Guide

This document explains how to call the current external OpenAPI surface exposed by this service.

Current base path:

- `/open-api/v1`

Current endpoints:

- `GET /open-api/v1/health`
- `GET /open-api/v1/account/me`

## Base URL

Local development example:

```text
http://127.0.0.1:3000
```

If the backend is running on another host or port, replace the base URL accordingly.

---

## Authentication

The OpenAPI surface supports machine-to-machine authentication with API keys.

Current supported header styles:

Primary:

```http
Authorization: Bearer <api-key>
```

Compatibility fallback:

```http
X-API-Key: <api-key>
```

Recommended:

- use `Authorization: Bearer <api-key>`

Demo key currently seeded for local testing:

```text
demo-open-api-key
```

Important:

- pass the **raw API key**
- do **not** pass the SHA-256 hash stored in the database

---

## 1. Health Check

Use this endpoint to verify that the OpenAPI surface is reachable.

- Method: `GET`
- Path: `/open-api/v1/health`
- Auth required: `No`

### Example Request

```bash
curl http://127.0.0.1:3000/open-api/v1/health
```

### Example Response

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "surface": "open-api",
    "version": "v1"
  }
}
```

### JavaScript Example

```js
const response = await fetch("http://127.0.0.1:3000/open-api/v1/health");
const data = await response.json();
console.log(data);
```

### Python Example

```python
import requests

resp = requests.get("http://127.0.0.1:3000/open-api/v1/health", timeout=10)
resp.raise_for_status()
print(resp.json())
```

---

## 2. Get Current OpenAPI Account

Use this endpoint to retrieve the account/profile associated with the authenticated API key.

- Method: `GET`
- Path: `/open-api/v1/account/me`
- Auth required: `Yes`

### Example Request

```bash
curl \
  -H "Authorization: Bearer demo-open-api-key" \
  http://127.0.0.1:3000/open-api/v1/account/me
```

Alternative header style:

```bash
curl \
  -H "X-API-Key: demo-open-api-key" \
  http://127.0.0.1:3000/open-api/v1/account/me
```

### Example Success Response

```json
{
  "success": true,
  "data": {
    "id": "019ea0c1-0001-7000-8000-000000000001",
    "name": "张三",
    "email": "zhangsan@example.com",
    "status": "active",
    "email_verified": false,
    "created_at": "2026-06-04 10:00:00"
  }
}
```

Notes:

- `external_ref` may be omitted when empty
- `status` is a normalized external field, not the raw internal integer flag

### JavaScript Example

```js
const response = await fetch("http://127.0.0.1:3000/open-api/v1/account/me", {
  headers: {
    Authorization: "Bearer demo-open-api-key",
  },
});

if (!response.ok) {
  throw new Error(`OpenAPI request failed: ${response.status}`);
}

const data = await response.json();
console.log(data);
```

### Python Example

```python
import requests

resp = requests.get(
    "http://127.0.0.1:3000/open-api/v1/account/me",
    headers={"Authorization": "Bearer demo-open-api-key"},
    timeout=10,
)
resp.raise_for_status()
print(resp.json())
```

---

## Error Format

Authenticated OpenAPI endpoints use a partner-safe error envelope:

```json
{
  "success": false,
  "error": {
    "code": "unauthorized",
    "message": "invalid api key"
  }
}
```

Common error codes:

- `unauthorized`
- `forbidden`
- `not_found`
- `internal_error`

### Example Unauthorized Response

```json
{
  "success": false,
  "error": {
    "code": "unauthorized",
    "message": "missing api key"
  }
}
```

---

## Implementation Notes for Clients

- `GET /open-api/v1/health` is intended for monitoring and connectivity checks.
- `GET /open-api/v1/account/me` is intended for authenticated account/profile reads.
- Health check does not require an API key.
- Business endpoints under `/open-api/v1` may require an API key even if the health endpoint does not.
- All open-api responses now use a unified envelope:
  - success: `{"success": true, "data": ...}`
  - failure: `{"success": false, "error": ...}`

---

## Quick Test Flow

1. Check that the service is reachable:

```bash
curl http://127.0.0.1:3000/open-api/v1/health
```

2. Call an authenticated endpoint:

```bash
curl \
  -H "Authorization: Bearer demo-open-api-key" \
  http://127.0.0.1:3000/open-api/v1/account/me
```

If step 1 succeeds and step 2 fails with `401`, the service is up but the API key is missing or invalid.
