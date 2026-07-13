// Package http holds identity-service's transport layer: route handlers and
// middleware. Handlers stay thin -- decode/validate, call a domain store,
// encode a response.
package http

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/sales-intelligence1/identity-service/pkg/auth0"
	"github.com/sales-intelligence1/identity-service/pkg/handoff"
	"github.com/sales-intelligence1/identity-service/pkg/session"
	"github.com/sales-intelligence1/identity-service/pkg/user"
)

const stateCookieName = "auth0_state"

// redirectURICookieName carries a requested relying-party redirect_uri from
// login through callback -- same short-lived-cookie pattern as the state
// cookie, since Auth0's callback only echoes back the "state" param, not
// arbitrary extra data.
const redirectURICookieName = "auth0_redirect_uri"

// userContextKey is where RequireSession stashes the resolved user for
// downstream handlers.
const userContextKey = "user"

// AuthHandler wires the Auth0 login/callback/logout routes and session lookup.
type AuthHandler struct {
	auth0               *auth0.Client
	users               user.Store
	sessions            session.Store
	handoffs            handoff.Store
	appBaseURL          string
	allowedRedirectURIs []string
}

// NewAuthHandler builds an AuthHandler. allowedRedirectURIs is the
// exact-match allowlist relying-party frontends must appear in to use the
// login handoff (see login's redirect_uri param); nil disables it.
func NewAuthHandler(auth0Client *auth0.Client, users user.Store, sessions session.Store, handoffs handoff.Store, appBaseURL string, allowedRedirectURIs []string) *AuthHandler {
	return &AuthHandler{
		auth0:               auth0Client,
		users:               users,
		sessions:            sessions,
		handoffs:            handoffs,
		appBaseURL:          appBaseURL,
		allowedRedirectURIs: allowedRedirectURIs,
	}
}

// Register wires the auth routes onto e.
func (h *AuthHandler) Register(e *echo.Echo) {
	e.GET("/auth/login", h.login)
	e.GET("/auth/callback", h.callback)
	e.GET("/auth/logout", h.logout)
	e.GET("/me", h.me, h.RequireSession)
}

// login starts the Authorization Code flow: stash a CSRF state value in a
// short-lived cookie, then redirect to Auth0's Universal Login page.
//
// An optional redirect_uri query param names a relying-party callback (e.g.
// sales-intel-web's own /auth/callback) to hand a one-time login code off to
// once Auth0 completes -- see callback. It must exactly match an entry in
// allowedRedirectURIs; unrecognized values are rejected rather than silently
// dropped, since a wrong redirect would resemble an open-redirect bug.
func (h *AuthHandler) login(c echo.Context) error {
	redirectURI := c.QueryParam("redirect_uri")
	if redirectURI != "" && !slices.Contains(h.allowedRedirectURIs, redirectURI) {
		return echo.NewHTTPError(http.StatusBadRequest, "redirect_uri not allowed")
	}

	state, err := randomToken(16)
	if err != nil {
		c.Logger().Errorf("generate login state: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to start login")
	}

	c.SetCookie(&http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	if redirectURI != "" {
		c.SetCookie(&http.Cookie{
			Name:     redirectURICookieName,
			Value:    redirectURI,
			Path:     "/",
			MaxAge:   300,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}
	return c.Redirect(http.StatusFound, h.auth0.LoginURL(state))
}

// callback completes the Authorization Code flow: verify the CSRF state,
// exchange the code for a verified identity, sync the user, and start a
// session -- identity-service's own, used by its admin UI/API. If login was
// started with a redirect_uri, also mint a one-time handoff code and send
// the browser there instead of returning JSON directly; the relying party
// exchanges that code server-side for its own token pair (see ExchangeHandler).
func (h *AuthHandler) callback(c echo.Context) error {
	stateCookie, err := c.Cookie(stateCookieName)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != c.QueryParam("state") {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid login state")
	}
	clearCookie(c, stateCookieName)

	redirectURI := ""
	if cookie, err := c.Cookie(redirectURICookieName); err == nil {
		redirectURI = cookie.Value
		clearCookie(c, redirectURICookieName)
	}

	code := c.QueryParam("code")
	if code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing authorization code")
	}

	ctx := c.Request().Context()

	claims, err := h.auth0.Exchange(ctx, code)
	if err != nil {
		c.Logger().Errorf("auth0 exchange: %v", err)
		return echo.NewHTTPError(http.StatusUnauthorized, "login failed")
	}

	u, err := h.users.UpsertByAuth0Sub(ctx, claims.Sub, claims.Email, claims.Name)
	if err != nil {
		c.Logger().Errorf("upsert user: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "login failed")
	}

	rawToken, tokenHash, err := session.NewToken()
	if err != nil {
		c.Logger().Errorf("new session token: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "login failed")
	}

	expiresAt := time.Now().Add(session.TTL)
	if _, err := h.sessions.Create(ctx, u.ID, tokenHash, expiresAt); err != nil {
		c.Logger().Errorf("create session: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "login failed")
	}

	c.SetCookie(&http.Cookie{
		Name:     session.CookieName,
		Value:    rawToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	if redirectURI != "" {
		rawCode, codeHash, err := session.NewToken()
		if err != nil {
			c.Logger().Errorf("new handoff code: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "login failed")
		}
		if _, err := h.handoffs.Create(ctx, u.ID, codeHash, time.Now().Add(handoff.TTL)); err != nil {
			c.Logger().Errorf("create handoff code: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "login failed")
		}

		dest, err := url.Parse(redirectURI)
		if err != nil {
			c.Logger().Errorf("parse redirect_uri %q: %v", redirectURI, err)
			return echo.NewHTTPError(http.StatusInternalServerError, "login failed")
		}
		q := dest.Query()
		q.Set("code", rawCode)
		dest.RawQuery = q.Encode()
		return c.Redirect(http.StatusFound, dest.String())
	}

	return c.JSON(http.StatusOK, u)
}

// logout clears the local session (if any), then sends the browser to Auth0's
// logout endpoint so its SSO session is cleared too, before returning here.
func (h *AuthHandler) logout(c echo.Context) error {
	cookie, err := c.Cookie(session.CookieName)
	if err == nil && cookie.Value != "" {
		if err := h.sessions.DeleteByTokenHash(c.Request().Context(), session.HashToken(cookie.Value)); err != nil {
			c.Logger().Errorf("delete session: %v", err)
		}
	}
	clearCookie(c, session.CookieName)
	return c.Redirect(http.StatusFound, h.auth0.LogoutURL(h.appBaseURL))
}

// me returns the current session's user. Requires RequireSession.
func (h *AuthHandler) me(c echo.Context) error {
	u, _ := c.Get(userContextKey).(user.User)
	return c.JSON(http.StatusOK, u)
}

// RequireSession is Echo middleware that resolves the session cookie into the
// current user, storing it on the context for handlers to read, and rejects
// the request with 401 if the cookie is missing, unknown, or expired.
func (h *AuthHandler) RequireSession(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie(session.CookieName)
		if err != nil || cookie.Value == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "not logged in")
		}

		ctx := c.Request().Context()
		sess, err := h.sessions.FindByTokenHash(ctx, session.HashToken(cookie.Value))
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "session expired or invalid")
		}

		u, err := h.users.ByID(ctx, sess.UserID)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "session expired or invalid")
		}

		c.Set(userContextKey, u)
		return next(c)
	}
}

// RequireAdmin is Echo middleware that requires a valid session (see
// RequireSession) whose user is an admin, rejecting non-admins with 403.
func (h *AuthHandler) RequireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return h.RequireSession(func(c echo.Context) error {
		u, _ := c.Get(userContextKey).(user.User)
		if !u.IsAdmin {
			return echo.NewHTTPError(http.StatusForbidden, "admin access required")
		}
		return next(c)
	})
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func clearCookie(c echo.Context, name string) {
	c.SetCookie(&http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}
