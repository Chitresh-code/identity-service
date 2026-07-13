package http

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/sales-intelligence1/identity-service/pkg/handoff"
	"github.com/sales-intelligence1/identity-service/pkg/session"
	"github.com/sales-intelligence1/identity-service/pkg/signingkey"
	"github.com/sales-intelligence1/identity-service/pkg/user"
	"github.com/sales-intelligence1/identity-service/pkg/userrefresh"
)

// accessTokenTTL is how long an issued end-user access token stays valid.
// Short-lived by the same reasoning as the app-credential token in
// token.go, but here a refresh token exists (unlike the app case) because a
// browser session can't just "call /token again" the way a service holding
// its own API key can -- it would mean re-running the full Auth0 login every
// hour.
const accessTokenTTL = time.Hour

// ExchangeHandler trades a post-login handoff code (see AuthHandler.callback)
// for a token pair, and later trades a refresh token for a fresh access
// token, for relying-party frontends.
type ExchangeHandler struct {
	handoffs      handoff.Store
	users         user.Store
	refreshTokens userrefresh.Store
	signingKeys   signingkey.Store
	issuer        string
}

// NewExchangeHandler builds an ExchangeHandler. issuer is used as the JWT
// "iss" claim and should be this service's base URL.
func NewExchangeHandler(handoffs handoff.Store, users user.Store, refreshTokens userrefresh.Store, signingKeys signingkey.Store, issuer string) *ExchangeHandler {
	return &ExchangeHandler{handoffs: handoffs, users: users, refreshTokens: refreshTokens, signingKeys: signingKeys, issuer: issuer}
}

// Register wires the exchange/refresh routes onto e, gated by rateLimiter --
// same tighter credential-verification limit as /token, since both routes
// turn a bearer credential (handoff code, refresh token) into a fresh JWT.
func (h *ExchangeHandler) Register(e *echo.Echo, rateLimiter echo.MiddlewareFunc) {
	e.POST("/auth/exchange", h.exchange, rateLimiter)
	e.POST("/auth/refresh", h.refresh, rateLimiter)
}

type exchangeRequest struct {
	Code string `json:"code"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenPairResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
}

func (h *ExchangeHandler) exchange(c echo.Context) error {
	var req exchangeRequest
	if err := c.Bind(&req); err != nil || req.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing code")
	}

	ctx := c.Request().Context()
	ho, err := h.handoffs.Consume(ctx, session.HashToken(req.Code))
	if err != nil {
		c.Logger().Warnf("token exchange denied: reason=invalid_code")
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired code")
	}

	u, err := h.users.ByID(ctx, ho.UserID)
	if err != nil {
		c.Logger().Errorf("load user %q after handoff: %v", ho.UserID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to issue token")
	}

	return h.issueTokenPair(c, u)
}

// refresh trades a valid refresh token for a new token pair, rotating the
// refresh token (the old one is consumed, a new one issued) so a leaked
// refresh token has a shrinking window of usefulness rather than being
// replayable indefinitely until its 24h TTL.
//
// ponytail: rotation here is two sequential store calls (delete old, create
// new), not one DB transaction like APIKeyStore.Rotate -- a browser refreshing
// twice at once just ends up with two valid new tokens instead of one, no
// data loss. Worth a transaction if concurrent refresh from the same client
// ever becomes a real attack surface.
func (h *ExchangeHandler) refresh(c echo.Context) error {
	var req refreshRequest
	if err := c.Bind(&req); err != nil || req.RefreshToken == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing refresh_token")
	}

	ctx := c.Request().Context()
	tokenHash := session.HashToken(req.RefreshToken)
	tok, err := h.refreshTokens.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		c.Logger().Warnf("refresh denied: reason=invalid_or_expired_token")
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired refresh token")
	}
	if err := h.refreshTokens.DeleteByTokenHash(ctx, tokenHash); err != nil {
		c.Logger().Errorf("delete rotated refresh token: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to refresh token")
	}

	u, err := h.users.ByID(ctx, tok.UserID)
	if err != nil {
		c.Logger().Errorf("load user %q for refresh: %v", tok.UserID, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to refresh token")
	}

	return h.issueTokenPair(c, u)
}

func (h *ExchangeHandler) issueTokenPair(c echo.Context, u user.User) error {
	ctx := c.Request().Context()

	signingK, err := h.signingKeys.Latest(ctx)
	if err != nil {
		c.Logger().Errorf("load signing key: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to issue token")
	}

	role := ""
	if u.Role != nil {
		role = string(*u.Role)
	}
	accessToken, accessExpiresAt, err := signingkey.IssueUserToken(signingK, h.issuer, u.ID, role, accessTokenTTL)
	if err != nil {
		c.Logger().Errorf("issue user token: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to issue token")
	}

	rawRefresh, refreshHash, err := session.NewToken()
	if err != nil {
		c.Logger().Errorf("new refresh token: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to issue token")
	}
	if _, err := h.refreshTokens.Create(ctx, u.ID, refreshHash, time.Now().Add(userrefresh.TTL)); err != nil {
		c.Logger().Errorf("create refresh token: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to issue token")
	}

	c.Logger().Infof("user token issued: user_id=%s role=%s kid=%s", u.ID, role, signingK.ID)
	return c.JSON(http.StatusOK, tokenPairResponse{
		AccessToken:      accessToken,
		TokenType:        "Bearer",
		ExpiresIn:        int(time.Until(accessExpiresAt).Seconds()),
		RefreshToken:     rawRefresh,
		RefreshExpiresIn: int(userrefresh.TTL.Seconds()),
	})
}
