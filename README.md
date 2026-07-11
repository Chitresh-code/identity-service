# identity-service

Auth/identity microservice for the `sales-intelligence` platform. Handles human/admin
login via Auth0, issues and manages application API keys, and exposes a JWKS endpoint
so other services (e.g. MarketPulse) can verify tokens without a shared secret.

Stack: Go, Echo, GORM, golang-migrate, Postgres, Auth0.

See `AGENTS.md` for coding conventions and `CONTRIBUTING.md` for the work item →
branch → PR workflow this project follows.

## Status

Scaffold complete (#2): Echo boots with a `/health` route, config loads from the
environment (`.env` locally via `godotenv`), and GORM connects to Postgres with schema
managed by `golang-migrate`. No business logic yet — that starts with #3 (Auth0
integration) and #4 (application model + API key issuance).

## Running locally

```
cp .env.example .env   # then fill in DATABASE_URL for your local Postgres
make migrate-up        # apply migrations
make run                # starts on :8080
```

See `Makefile` for all available commands (`build`, `test`, `vet`, `fmt`, `bruno`,
`migrate-up`/`migrate-down`).

## Manual API testing

A Bruno collection lives in `bruno/` — open it in the Bruno app, or run headlessly:
```
make bruno
```
