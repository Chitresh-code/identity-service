# identity-service

Auth/identity microservice for the `sales-intelligence` platform. Handles human/admin
login via Auth0, issues and manages application API keys, and exposes a JWKS endpoint
so other services (e.g. MarketPulse) can verify tokens without a shared secret.

Stack: Go, Echo, GORM, golang-migrate, Postgres, Auth0.

See `AGENTS.md` for coding conventions and `CONTRIBUTING.md` for the work item →
branch → PR workflow this project follows.

## Status

Scaffolding in progress — no code yet.
