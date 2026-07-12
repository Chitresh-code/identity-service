# Identity Service

Auth and identity microservice for the sales-intelligence platform. Handles admin login via Auth0, issues and manages application API keys for service-to-service auth, and publishes a JWKS endpoint so other services can verify tokens without a shared secret.

**Stack:** Go, Echo, GORM, golang-migrate, Postgres, Auth0.

Auth is the one path every other service depends on, so it favors a small, fast, low-overhead runtime over ecosystem breadth — Go's static binaries and strong stdlib crypto/JWT support fit that well. Auth0 handles the actual login flow rather than rolling custom password/session handling, since correctly implementing auth from scratch is a well-known way to introduce security bugs.

## What it does

- Human login via Auth0's Universal Login, backed by a server-side session (HTTP-only cookie, hashed token stored in Postgres).
- Admin-gated endpoints to register applications and issue, rotate, and revoke their API keys (`<prefix>.<secret>`, shown once, only a hash stored server-side).
- `POST /token` exchanges an application API key for a short-lived signed JWT.
- `GET /.well-known/jwks.json` publishes the public key, so any other service can verify those JWTs independently, with no shared secret and no call back to this service per request.

## Running locally

Requires a local Postgres instance and an Auth0 application.

```bash
cp .env.example .env   # fill in DATABASE_URL and the AUTH0_* values
make migrate-up
make run                # starts on :8080
```

Auth0 setup: create a Regular Web Application in the Auth0 dashboard, set Allowed Callback URLs to `http://localhost:8080/auth/callback` and Allowed Logout URLs to `http://localhost:8080`, then copy the Domain/Client ID/Client Secret into `.env`.

See the `Makefile` for all available commands (`build`, `test`, `vet`, `fmt`, `bruno`, `migrate-up`/`migrate-down`).

## Manual API testing

A Bruno collection lives in `bruno/` — open it in the Bruno app, or run headlessly:

```bash
make bruno
```

## Contributing

See `AGENTS.md` for coding conventions and `CONTRIBUTING.md` for the contribution workflow.
