# Google And GitHub OAuth Login

## Goal

Add OAuth login so users can sign in with Google OAuth accounts and GitHub accounts while preserving the existing local JWT auth contract and password login flow.

## What I Already Know

* User requested a new Trellis task for OAuth login supporting Google OAuth and GitHub.
* Existing auth routes support register/login/logout/forgot/reset/status/me.
* Existing frontend stores `app_access_token` in `localStorage` and sends `Authorization: Bearer <token>`.
* Existing backend issues stateless JWT through `api/framework/http/auth`.
* Existing user table has unique `email`, optional `password_hash`, `email_verified`, `is_active`, `is_admin`, membership fields, and timestamps.
* Current app does not use cookie session auth; an archguard test explicitly rejects retired cookie/session auth symbols.
* MVP OAuth provider credentials will use runtime environment variables loaded from OS env or dotenv-style env files, while preserving a future path to configure OAuth providers through Parameter integration channels.

## Research References

* [`research/oauth-provider-flow.md`](research/oauth-provider-flow.md) - Google and GitHub both fit a backend authorization-code callback that ultimately issues the app's existing JWT.

## Assumptions (Temporary)

* "Google OAuth" means Google account login with email/profile identity, not Gmail API mailbox access.
* MVP supports login and first-time account creation, not connecting/disconnecting providers from an account settings page.
* OAuth-created users are active by default and have `email_verified=true` only when the provider returns a verified email.
* Password login remains available and unchanged.

## Open Questions

* None blocking.

## Decisions

* MVP uses runtime environment variables for provider credentials:
  * `GOOGLE_OAUTH_CLIENT_ID`
  * `GOOGLE_OAUTH_CLIENT_SECRET`
  * `GITHUB_OAUTH_CLIENT_ID`
  * `GITHUB_OAUTH_CLIENT_SECRET`
  * `APP_PUBLIC_BASE_URL` or equivalent public base URL config for callback URLs
* The executable auto-loads dotenv-style `.env` files at startup from the working directory, `data/`, the executable directory, and the executable directory's parent. System/container env values take precedence over file values.
* Parameter channel configuration for OAuth providers is intentionally reserved for a future task.
* Only provider identities with a verified email can create or auto-link local users. Missing or unverified email is rejected.
* Existing linked OAuth identity wins first: if `(provider, provider_user_id)` exists, login that mapped local user.
* If no linked identity exists but the provider returns a verified email matching an existing local user, automatically link the provider identity to that user and log in.
* If no linked identity exists and the verified email is new, create a local user with empty `password_hash`, `email_verified=true`, `is_active=1`, then link the provider identity.
* Disabled local users cannot log in via OAuth even when provider authentication succeeds.
* OAuth callback must not put the app JWT in the URL. It creates a short-lived one-time `oauth_login_results` token and redirects to the frontend callback route; the frontend exchanges that one-time token for the normal `AuthTokenResponse`.

## Requirements (Evolving)

* Add Google OAuth and GitHub OAuth login entry points.
* Add backend OAuth callback handling with state validation.
* Add persistence for external auth identities so a local user can be linked to a provider account.
* On successful OAuth login, issue the same `AuthTokenResponse` shape used by password login/register.
* Add frontend login buttons for Google and GitHub.
* Preserve existing JWT/localStorage auth behavior.
* Reject disabled local users even if the OAuth provider authenticates successfully.
* Do not store provider access tokens unless they are needed for the login flow.
* Store OAuth `state` and one-time login result tokens as hashes, not plaintext.
* One-time login result tokens expire quickly and can be used only once.

## Acceptance Criteria (Evolving)

* [x] Login page shows Google and GitHub login options.
* [x] OAuth start endpoints redirect to the correct provider authorization URL.
* [x] OAuth callback verifies `state` before accepting provider `code`.
* [x] OAuth callback fetches provider identity and maps it to a local user.
* [x] Existing linked `(provider, provider_user_id)` identity logs in the mapped user.
* [x] Verified provider email matching an existing local user automatically links that provider identity.
* [x] Verified provider email not matching any local user creates an active local user with empty `password_hash`.
* [x] Missing or unverified provider email is rejected and does not create or link a user.
* [x] OAuth callback redirects with a one-time exchange token, not an app JWT.
* [x] OAuth exchange endpoint returns the same `AuthTokenResponse` shape as password login/register.
* [x] OAuth state and login result tokens reject expired or repeated use.
* [x] Missing provider env config returns a safe error and does not redirect to an invalid provider URL.
* [x] Existing JWT auth status/me/logout behavior continues to work after OAuth login.
* [x] Disabled users cannot log in through OAuth.
* [x] Backend tests cover provider identity mapping and account linking behavior.
* [x] Frontend tests cover OAuth helper URL/button behavior where practical.

## Definition Of Done

* [x] Backend migrations, models, usecase, routes, and tests are implemented.
* [x] Frontend login UI and API helpers are implemented.
* [x] Relevant specs are updated if auth API/frontend contracts change.
* [x] `go test ./api/...`, `cd frontend && npm test`, and `cd frontend && npm run build` pass.
* [x] OAuth provider environment setup is documented in the task notes or spec.

## Implementation Notes

Implemented on 2026-06-08:

* Added `oauth_identities`, `oauth_states`, and `oauth_login_results` migration.
* Added Google OAuth and GitHub OAuth adapters using backend authorization-code flow.
* Added public start/callback/exchange routes while preserving JWT issuance in the route layer.
* Added login buttons and `/login/oauth/callback` frontend route.
* Documented OAuth environment variables and provider callback URLs in `README.md`.
* Added runtime env file support for Windows exe and Linux executable deployments, plus `.env.example`.
* Verified with `go test ./api/...`, `cd frontend && npm test`, `cd frontend && npm run build`, and a local browser smoke check on `http://127.0.0.1:5173/login`.

## Technical Notes

Likely backend areas:

* `api/db/migrations/app/*`
* `api/models/auth.go`
* `api/models/user.go`
* `api/usecase/auth.go`
* `api/routes/auth.go`
* `index.go`

Likely frontend areas:

* `frontend/src/api.js`
* `frontend/src/pages/Login.svelte`
* `frontend/src/pages/Register.svelte` if parity is desired
* `frontend/src/App.svelte` and router if a callback route is added
* `frontend/src/api.test.js`

Potential tables:

* `oauth_identities(provider, provider_user_id, user_id, email, email_verified, display_name, created_at, updated_at)` with `UNIQUE(provider, provider_user_id)`.
* `oauth_states(state_hash, provider, redirect_path, expires_at, used_at, created_at)`; `state_hash` stores only a hash of the browser state value.
* `oauth_login_results(token_hash, user_id, expires_at, used_at, created_at)`; `token_hash` stores only a hash of the frontend exchange token.

Recommended endpoint shape:

* `GET /api/auth/oauth/:provider/start?redirect_path=/orders`
* `GET /api/auth/oauth/:provider/callback?code=...&state=...`
* `POST /api/auth/oauth/exchange` with `{ "token": "<one-time-result-token>" }`

Recommended frontend route shape:

* `/login/oauth/callback?token=<one-time-result-token>`
* The route calls `exchangeOAuthLoginResult(token)`, stores `access_token` through the existing API helper path, refreshes auth, and navigates to the original redirect path or `/`.

## Expansion Sweep

Future evolution:

* Account settings could later support linking/unlinking multiple OAuth providers.
* Admin settings could later configure OAuth providers through Parameter integration channels rather than environment variables.

Related scenarios:

* Existing password login/register/reset must keep working.
* Auth status and `/auth/me` should not distinguish password vs OAuth users for normal app use.

Failure and edge cases:

* OAuth state expiry/replay.
* Provider returns unverified or missing email.
* Provider account verified email already belongs to an existing local account.
* Disabled local account tries OAuth login.
* Provider credentials missing or disabled.

## Out Of Scope (Draft)

* Gmail API mailbox access.
* Account settings page for linking/unlinking providers.
* Refreshing or storing provider access tokens for provider API calls.
* Enterprise SSO/SAML.
* Replacing JWT/localStorage auth with cookie sessions.
* Admin UI for configuring OAuth providers through Parameter channels.

## Decision (ADR-lite)

Context: OAuth login needs provider redirects, account linking, and token handoff while this app already uses stateless JWT bearer auth and `localStorage`.

Decision: Implement Google OAuth and GitHub OAuth with env-configured provider credentials for MVP. Use backend authorization-code callbacks, hash-backed state validation, verified-email-only account creation/linking, and a short-lived one-time login result token that the frontend exchanges for the normal JWT response.

Consequences: This keeps implementation focused and avoids putting JWTs in URLs. Parameter-based OAuth provider configuration remains possible later, but the MVP does not require admin configuration UI or provider token storage.
