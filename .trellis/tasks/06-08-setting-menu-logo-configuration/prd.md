# Setting Menu Logo Configuration

## Goal

Add an admin Setting menu with tabbed layout. The first two tabs are `General` and `Retain`.

The `General` tab lets an admin configure the app logo shown in the top-left `navbar-start` area. The header should replace the current text `Svelte Go Starter` with an image. If no custom logo is configured, it should show the default public asset `/logo.png`.

The rendered logo size is fixed at `110x25`.

## Scope

- Add a Settings app route/menu item.
- Keep Settings admin-only, consistent with Parameter and Notification.
- Add a `Settings` Svelte page with `General` and `Retain` tabs.
- Add frontend API helpers for reading site settings and uploading the logo.
- Add backend internal API:
  - `GET /api/settings/site`
  - `POST /api/settings/site/logo`
- Add a public logo asset endpoint usable by an unauthenticated `<img>` tag:
  - `GET /api/settings/public/logo`
- Add persistence for site settings.
- Use the existing OSS usecase port for custom logo storage.
- Add a default `public/logo.png` asset.

## Architecture Decision

The page, route, and usecase should not know any concrete cloud provider SDK.

For the first implementation, register a small local OSS adapter at startup for site logo storage. It implements `api/usecase/integrations/oss.Adapter` and writes objects under the application `data/` directory. This keeps the General logo flow based on the OSS port today, while allowing future R2/Aliyun-backed storage to replace the adapter/config resolution without changing the Settings UI or route contract.

Layering remains:

```text
routes -> usecase -> models
                 -> usecase/integrations/oss port
index.go registers concrete adapter
```

## UX Requirements

- Navbar top-left shows an image button that navigates to `/`.
- Image display is `110x25`, using object fit containment.
- If the configured image fails to load, fall back to `/logo.png`.
- Settings page uses tabs:
  - `General`: logo preview and upload.
  - `Retain`: placeholder tab surface for future settings.
- Settings page calls `frontend/src/api.js` helpers, not direct `fetch`.
- Upload accepts common browser image files and shows API safe errors.

## Backend Contract

`GET /api/settings/site` response payload:

```json
{
  "logo_url": "/logo.png",
  "logo_configured": false,
  "logo_updated_at": ""
}
```

When a logo is configured, `logo_url` should point to `/api/settings/public/logo?v=<updated_at-or-version>` so browsers refresh the image after upload.

`POST /api/settings/site/logo` accepts multipart form data with a `logo` file field and returns the same DTO as `GET /api/settings/site`.

Validation:

- Empty upload: `CodeValidation`, `logo file is required`.
- Unsupported content type: `CodeValidation`, `logo image type is not supported`.
- Oversized file: `CodeValidation`, `logo file is too large`.
- Missing OSS adapter: `CodeInternal`, `logo storage is not configured`.

## Data Contract

Persist a keyed setting for the site logo metadata. Store only safe metadata:

- object key
- content type
- size
- updated timestamp/version

Do not store raw image bytes in the DB.

## Tests

- Backend model/usecase tests for default logo fallback and saved logo metadata.
- Backend route test for site settings envelope and upload response.
- Frontend API helper tests for settings endpoints and multipart behavior.
- Router tests for `/settings`, `/settings.html`, admin-only visibility, and route title.
- Run:
  - `go test ./...`
  - `cd frontend && npm test`
  - `cd frontend && npm run build`
