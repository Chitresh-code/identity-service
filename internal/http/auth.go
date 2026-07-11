// Package http holds identity-service's transport layer: route handlers and
// middleware. Handlers stay thin -- decode/validate, call a domain store,
// encode a response.
package http

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/sales-intelligence1/identity-service/internal/auth0"
	"github.com/sales-intelligence1/identity-service/internal/session"
	"github.com/sales-intelligence1/identity-service/internal/user"
)

const stateCookieName = "auth0_state"

// userContextKey is where RequireSession stashes the resolved user for
// downstream handlers.
const userContextKey = "user"

// AuthHandler wires the Auth0 login/callback/logout routes and session lookup.
type AuthHandler struct {
	auth0      *auth0.Client
	users      user.Store
	sessions   session.Store
	appBaseURL string
}

// NewAuthHandler builds an AuthHandler.
func NewAuthHandler(auth0Client *auth0.Client, users user.Store, sessions session.Store, appBaseURL string) *AuthHandler {
	return &AuthHandler{auth0: auth0Client, users: users, sessions: sessions, appBaseURL: appBaseURL}
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
func (h *AuthHandler) login(c echo.Context) error {
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
	return c.Redirect(http.StatusFound, h.auth0.LoginURL(state))
}

// callback completes the Authorization Code flow: verify the CSRF state,
// exchange the code for a verified identity, sync the user, and start a
// session.
func (h *AuthHandler) callback(c echo.Context) error {
	stateCookie, err := c.Cookie(stateCookieName)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != c.QueryParam("state") {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid login state")
	}
	clearCookie(c, stateCookieName)

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
