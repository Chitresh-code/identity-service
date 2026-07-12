package http

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/sales-intelligence1/identity-service/pkg/session"
)

// CredentialIdentifier extracts a rate-limiting identity from the request's
// credential -- the session cookie or API key -- rather than its IP, so
// distinct callers behind the same NAT/proxy don't share a limit and one
// caller can't dodge its limit by rotating source IPs. Falls back to IP for
// requests presenting no (or an unusable) credential, e.g. anonymous /auth
// and /token attempts, which is exactly the traffic IP-based limiting is
// meant to catch.
//
// This only reads the credential, it never validates it -- a forged cookie or
// key still gets its own bucket, which is fine: the goal is fair sharing and
// abuse containment, not authorization.
func CredentialIdentifier(c echo.Context) (string, error) {
	if cookie, err := c.Cookie(session.CookieName); err == nil && cookie.Value != "" {
		return "session:" + session.HashToken(cookie.Value), nil
	}
	if prefix, ok := apiKeyPrefix(c.Request().Header.Get("Authorization")); ok {
		return "apikey:" + prefix, nil
	}
	return "ip:" + c.RealIP(), nil
}

func apiKeyPrefix(authHeader string) (string, bool) {
	raw := bearerToken(authHeader)
	if raw == "" {
		return "", false
	}
	prefix, _, ok := strings.Cut(raw, ".")
	if !ok || prefix == "" {
		return "", false
	}
	return prefix, true
}
