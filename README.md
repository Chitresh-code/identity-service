# identity-service

Auth/identity microservice for the `sales-intelligence` platform. Handles human/admin
login via Auth0, issues and manages application API keys, and exposes a JWKS endpoint
so other services (e.g. MarketPulse) can verify tokens without a shared secret.

Stack: Go, Echo, GORM, golang-migrate, Postgres, Auth0.

See `AGENTS.md` for coding conventions and `CONTRIBUTING.md` for the work item →
branch → PR workflow this project follows.

## Status

Scaffold (#2), Auth0 login/session (#3), application/API key issuance (#4), key
rotation/revocation (#5), and JWKS + token issuance (#6) complete:
- Echo boots with a `/health` route; config loads from the environment (`.env` locally
  via `godotenv`); GORM connects to Postgres with schema managed by `golang-migrate`.
- `GET /auth/login` redirects to Auth0's Universal Login (Authorization Code flow).
  `GET /auth/callback` verifies the returned ID token against Auth0's JWKS, upserts the
  matching row in our own `users` table (keyed on the Auth0 `sub`), and starts a
  server-side session: a random token is set as an HTTP-only cookie, and only its SHA-256
  hash is stored in the `sessions` table (same pattern this project uses for hashed
  credential storage generally) -- a stolen DB row can't be replayed as a cookie, and
  logout/expiry are enforced server-side, not just by trusting an unrevocable JWT.
  `GET /auth/logout` clears the local session and Auth0's SSO session.
- `GET /me` (behind the `RequireSession` middleware) returns the current session's user.
- `POST /admin/applications`, `GET /admin/applications` let an admin register a service
  client. `POST /admin/applications/:id/api-keys` issues it an API key: Stripe/GitHub PAT
  pattern (`<prefix>.<secret>`), plaintext shown exactly once, only the secret's SHA-256
  hash stored server-side. The very first user ever to log in is automatically granted
  admin (see the "Auth0 Login" and "Applications & API Keys" wiki pages).
- `GET /admin/applications/:id/api-keys` lists an application's keys (metadata only).
  `POST .../api-keys/:keyId/rotate` atomically issues a replacement key and revokes the
  old one -- a partial failure can't leave zero or two live keys. `POST
  .../api-keys/:keyId/revoke` revokes a key outright, idempotently. All admin-gated.

- On boot the service ensures it has an RSA signing key, generating and persisting one
  (`signing_keys` table) the first time. `GET /.well-known/jwks.json` publishes the
  public half -- no auth, standard JWKS convention. `POST /token` exchanges an API key
  (`Authorization: Bearer <prefix>.<secret>`) for a short-lived (15 min) RS256 JWT
  (`sub` = application id, `iss` = this service's base URL), rejecting unknown, wrong, or
  revoked keys with 401. Resource servers (e.g. MarketPulse) can now verify tokens purely
  against the JWKS, with no shared secret and no call back to this service per request
  (see the "JWKS & Token Issuance" wiki page).
- (#9) `GET /health` is now a readiness check -- it verifies the signing key is loadable
  (which also proves the database is reachable), returning `503` rather than a static
  `200`. `POST /token` has an HTTP-level test suite (`internal/http/token_test.go`)
  covering valid/wrong/unknown/revoked/malformed credentials. The revocation-vs-TTL and
  no-refresh-token tradeoffs are documented in code and on the wiki rather than left
  implicit.

Next up: #7 (rate limiting + structured logging, including a tighter limit and audit
logging specifically for `/token`), then #8 (retrofit MarketPulse to verify tokens via
this service's JWKS instead of its shared-secret HS256 check).

## Running locally

```
cp .env.example .env   # fill in DATABASE_URL and the AUTH0_* values (see below)
make migrate-up        # apply migrations
make run                # starts on :8080
```

Auth0 setup: create a **Regular Web Application** in the Auth0 dashboard, set
**Allowed Callback URLs** to `http://localhost:8080/auth/callback` and **Allowed Logout
URLs** to `http://localhost:8080`, then copy the Domain/Client ID/Client Secret into
`.env`.

See `Makefile` for all available commands (`build`, `test`, `vet`, `fmt`, `bruno`,
`migrate-up`/`migrate-down`).

## Manual API testing

A Bruno collection lives in `bruno/` — open it in the Bruno app, or run headlessly:
```
make bruno
```
