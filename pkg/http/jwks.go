package http

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/sales-intelligence1/identity-service/pkg/signingkey"
)

// JWKSHandler exposes this service's public signing keys so resource servers
// can verify issued tokens without a shared secret.
type JWKSHandler struct {
	signingKeys signingkey.Store
}

// NewJWKSHandler builds a JWKSHandler.
func NewJWKSHandler(signingKeys signingkey.Store) *JWKSHandler {
	return &JWKSHandler{signingKeys: signingKeys}
}

// Register wires the JWKS route onto e. It's public -- no auth -- per the
// well-known JWKS convention.
func (h *JWKSHandler) Register(e *echo.Echo) {
	e.GET("/.well-known/jwks.json", h.jwks)
}

func (h *JWKSHandler) jwks(c echo.Context) error {
	keys, err := h.signingKeys.All(c.Request().Context())
	if err != nil {
		c.Logger().Errorf("list signing keys: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load signing keys")
	}
	return c.JSON(http.StatusOK, signingkey.JWKS(keys))
}
