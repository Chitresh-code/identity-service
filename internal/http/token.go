package http

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/sales-intelligence1/identity-service/internal/apikey"
	"github.com/sales-intelligence1/identity-service/internal/signingkey"
)

// tokenTTL is how long an issued access token stays valid.
//
// Revoking the underlying API key does not invalidate tokens already issued
// against it -- they simply keep working until they expire on their own.
// Accepted tradeoff: a short TTL bounds a revoked key's blast radius to at
// most tokenTTL without needing a token blocklist or introspection endpoint.
// Revisit if a consumer ever needs revocation to take effect immediately.
const tokenTTL = 15 * time.Minute

// TokenHandler exchanges an API key for a short-lived signed JWT that
// resource servers can verify against this service's JWKS.
//
// No refresh token: callers already hold their API key and can just call
// this endpoint again, so a refresh token would be one more credential to
// secure for no real benefit to a service-to-service caller.
type TokenHandler struct {
	apiKeys     apikey.Store
	signingKeys signingkey.Store
	issuer      string
}

// NewTokenHandler builds a TokenHandler. issuer is used as the JWT "iss"
// claim and should be this service's base URL.
func NewTokenHandler(apiKeys apikey.Store, signingKeys signingkey.Store, issuer string) *TokenHandler {
	return &TokenHandler{apiKeys: apiKeys, signingKeys: signingKeys, issuer: issuer}
}

// Register wires the token route onto e.
func (h *TokenHandler) Register(e *echo.Echo) {
	e.POST("/token", h.issue)
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (h *TokenHandler) issue(c echo.Context) error {
	prefix, secret, ok := strings.Cut(bearerToken(c.Request().Header.Get("Authorization")), ".")
	if !ok || prefix == "" || secret == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid api key")
	}

	ctx := c.Request().Context()
	key, secretHash, err := h.apiKeys.ByPrefix(ctx, prefix)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid api key")
	}
	if key.RevokedAt != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid api key")
	}
	if subtle.ConstantTimeCompare([]byte(apikey.HashSecret(secret)), []byte(secretHash)) != 1 {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid api key")
	}

	signingK, err := h.signingKeys.Latest(ctx)
	if err != nil {
		c.Logger().Errorf("load signing key: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to issue token")
	}

	token, expiresAt, err := signingkey.IssueToken(signingK, h.issuer, key.ApplicationID, tokenTTL)
	if err != nil {
		c.Logger().Errorf("issue token: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to issue token")
	}

	return c.JSON(http.StatusOK, tokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int(time.Until(expiresAt).Seconds()),
	})
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimPrefix(header, prefix)
	}
	return ""
}
