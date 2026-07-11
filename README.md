# identity-service

Auth/identity microservice for the `sales-intelligence` platform. Handles human/admin
login via Auth0, issues and manages application API keys, and exposes a JWKS endpoint
so other services (e.g. MarketPulse) can verify tokens without a shared secret.

Stack: Go, Echo, GORM, golang-migrate, Postgres, Auth0.

See `AGENTS.md` for coding conventions and `CONTRIBUTING.md` for the work item →
branch → PR workflow this project follows.

## Status

Scaffold (#2) and Auth0 login/session (#3) complete:
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
- `GET /me` (behind the new `RequireSession` middleware) returns the current session's
  user -- the first protected endpoint, and how other endpoints will require login going
  forward.

Next up: #4 (application model + API key issuance).

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
