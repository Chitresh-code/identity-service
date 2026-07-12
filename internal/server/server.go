// Package server builds the fully-wired Echo application shared by both
// deployment targets: cmd/identity-service (traditional long-running binary,
// calls e.Start) and api/index.go (Vercel serverless, wraps the same *echo.Echo
// as an http.Handler). Keeping construction in one place means the two targets
// can't drift on routing, middleware, or rate limits.
package server

import (
	"context"
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	gommonlog "github.com/labstack/gommon/log"
	"golang.org/x/time/rate"

	"github.com/sales-intelligence1/identity-service/internal/auth0"
	"github.com/sales-intelligence1/identity-service/internal/config"
	identityhttp "github.com/sales-intelligence1/identity-service/internal/http"
	"github.com/sales-intelligence1/identity-service/internal/signingkey"
	"github.com/sales-intelligence1/identity-service/internal/store"
)

// Rate limits, matching market-data-service's pattern of a tighter limit on
// its token/login route than everywhere else: 60/min default, 10/min for
// /token specifically, since that's the route that actually verifies a
// credential and is what an attacker guessing API keys would hammer.
const (
	defaultRateLimit = rate.Limit(1) // ~60/minute
	defaultBurst     = 20
	tokenRateLimit   = rate.Limit(10.0 / 60.0) // 10/minute
	tokenBurst       = 3
)

// New connects to Postgres, wires every handler, and returns the ready-to-serve
// Echo application. Callers own the process lifecycle (e.Start for a long-running
// binary, ServeHTTP per-invocation for serverless).
func New(ctx context.Context, cfg config.Config) (*echo.Echo, error) {
	db, err := store.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	auth0Client, err := auth0.New(ctx, cfg.Auth0Domain, cfg.Auth0ClientID, cfg.Auth0ClientSecret, cfg.AppBaseURL+"/auth/callback")
	if err != nil {
		return nil, fmt.Errorf("init auth0 client: %w", err)
	}
	authHandler := identityhttp.NewAuthHandler(
		auth0Client,
		store.NewUserStore(db),
		store.NewSessionStore(db),
		cfg.AppBaseURL,
	)
	apiKeyStore := store.NewAPIKeyStore(db)
	applicationsHandler := identityhttp.NewApplicationsHandler(
		store.NewApplicationStore(db),
		apiKeyStore,
	)

	signingKeyStore := store.NewSigningKeyStore(db)
	if _, err := signingkey.EnsureLatest(ctx, signingKeyStore); err != nil {
		return nil, fmt.Errorf("ensure signing key: %w", err)
	}
	jwksHandler := identityhttp.NewJWKSHandler(signingKeyStore)
	tokenHandler := identityhttp.NewTokenHandler(apiKeyStore, signingKeyStore, cfg.AppBaseURL)
	healthHandler := identityhttp.NewHealthHandler(signingKeyStore)

	e := echo.New()
	// Echo defaults its logger to ERROR, which would silently drop the
	// Info/Warn-level audit logging in the token handler.
	e.Logger.SetLevel(gommonlog.INFO)
	e.Use(middleware.RequestID())
	e.Use(requestLogger(cfg.Environment))
	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper:             func(c echo.Context) bool { return c.Path() == "/health" },
		IdentifierExtractor: identityhttp.CredentialIdentifier,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate: defaultRateLimit, Burst: defaultBurst,
		}),
	}))

	tokenRateLimiter := middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		IdentifierExtractor: identityhttp.CredentialIdentifier,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate: tokenRateLimit, Burst: tokenBurst,
		}),
	})

	healthHandler.Register(e)
	authHandler.Register(e)
	applicationsHandler.Register(e, authHandler.RequireAdmin)
	jwksHandler.Register(e)
	tokenHandler.Register(e, tokenRateLimiter)

	return e, nil
}

// requestLogger returns the access-log middleware: human-readable text in
// "local" (nicer for a developer's terminal), JSON everywhere else (the
// default echo.middleware.Logger() format already includes the request id,
// method, path, status, and latency) -- environment defaults to "local" if
// unset, so it fails toward the readable option rather than silently
// emitting JSON a developer didn't ask for.
func requestLogger(environment string) echo.MiddlewareFunc {
	if environment != "local" {
		return middleware.Logger()
	}
	return middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${time_rfc3339} ${status} ${method} ${uri} ${latency_human} id=${id}\n",
	})
}
