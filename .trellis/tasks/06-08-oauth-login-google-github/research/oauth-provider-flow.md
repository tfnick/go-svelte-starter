# OAuth Provider Flow Notes

## Sources

* Google Identity OAuth 2.0 web server flow: https://developers.google.com/identity/protocols/oauth2/web-server
* GitHub OAuth app web application flow: https://docs.github.com/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps

## Common Flow

* Browser starts at an application endpoint such as `/api/auth/oauth/:provider/start`.
* Backend generates `state`, stores short-lived state server-side, then redirects to provider authorization URL.
* Provider redirects back to backend callback with `code` and `state`.
* Backend verifies state, exchanges code for provider access token, fetches trusted profile data, finds or creates a local user, then issues this app's existing JWT.
* Frontend receives the app JWT and stores it in `localStorage` using the existing `app_access_token` contract.

## Provider Data Needed

* Google: request OpenID Connect profile/email scopes and use the verified email claim/profile endpoint.
* GitHub: request user identity and email scope, fetch primary verified email when the public profile email is absent.

## Repo Fit

* The repo currently uses stateless JWT bearer tokens, no server session or auth cookie.
* OAuth callback should preserve that contract by redirecting to a frontend route with a short exchange token or an app JWT.
* Preferred safer shape: callback creates a short-lived one-time OAuth login result token, frontend calls an internal API to exchange it for the normal JWT.
* MVP can store OAuth state and login result records in the app SQLite DB.

