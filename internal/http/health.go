package http

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/sales-intelligence1/identity-service/internal/signingkey"
)

// HealthHandler reports service readiness: not just that the process is up,
// but that it can actually do its job (e.g. a signing key is loaded, which
// also proves the database is reachable).
type HealthHandler struct {
	signingKeys signingkey.Store
}

// NewHealthHandler builds a HealthHandler.
func NewHealthHandler(signingKeys signingkey.Store) *HealthHandler {
	return &HealthHandler{signingKeys: signingKeys}
}

// Register wires the health route onto e.
func (h *HealthHandler) Register(e *echo.Echo) {
	e.GET("/health", h.health)
}

func (h *HealthHandler) health(c echo.Context) error {
	if _, err := h.signingKeys.Latest(c.Request().Context()); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
