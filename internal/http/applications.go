package http

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/sales-intelligence1/identity-service/internal/apikey"
	"github.com/sales-intelligence1/identity-service/internal/application"
)

// ApplicationsHandler wires the admin-gated application and API key routes.
type ApplicationsHandler struct {
	applications application.Store
	apiKeys      apikey.Store
}

// NewApplicationsHandler builds an ApplicationsHandler.
func NewApplicationsHandler(applications application.Store, apiKeys apikey.Store) *ApplicationsHandler {
	return &ApplicationsHandler{applications: applications, apiKeys: apiKeys}
}

// Register wires the admin routes onto e, gated by requireAdmin.
func (h *ApplicationsHandler) Register(e *echo.Echo, requireAdmin echo.MiddlewareFunc) {
	g := e.Group("/admin/applications", requireAdmin)
	g.POST("", h.create)
	g.GET("", h.list)
	g.POST("/:id/api-keys", h.mintKey)
	g.GET("/:id/api-keys", h.listKeys)
	g.POST("/:id/api-keys/:keyId/rotate", h.rotateKey)
	g.POST("/:id/api-keys/:keyId/revoke", h.revokeKey)
}

type createApplicationRequest struct {
	Name string `json:"name"`
}

func (h *ApplicationsHandler) create(c echo.Context) error {
	var req createApplicationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	app, err := h.applications.Create(c.Request().Context(), req.Name)
	if err != nil {
		c.Logger().Errorf("create application: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create application")
	}
	return c.JSON(http.StatusCreated, app)
}

func (h *ApplicationsHandler) list(c echo.Context) error {
	apps, err := h.applications.List(c.Request().Context())
	if err != nil {
		c.Logger().Errorf("list applications: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list applications")
	}
	return c.JSON(http.StatusOK, apps)
}

// mintKeyResponse includes the plaintext key -- the only time it's ever
// returned. Only the hash of its secret half is stored server-side.
type mintKeyResponse struct {
	ID     string `json:"id"`
	Prefix string `json:"prefix"`
	Key    string `json:"key"`
}

func (h *ApplicationsHandler) mintKey(c echo.Context) error {
	appID := c.Param("id")
	ctx := c.Request().Context()

	if _, err := h.applications.ByID(ctx, appID); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "application not found")
	}

	plaintext, prefix, secretHash, err := apikey.NewKey()
	if err != nil {
		c.Logger().Errorf("generate api key: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to mint api key")
	}

	key, err := h.apiKeys.Create(ctx, appID, prefix, secretHash)
	if err != nil {
		c.Logger().Errorf("store api key: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to mint api key")
	}

	return c.JSON(http.StatusCreated, mintKeyResponse{ID: key.ID, Prefix: key.Prefix, Key: plaintext})
}

func (h *ApplicationsHandler) listKeys(c echo.Context) error {
	appID := c.Param("id")
	ctx := c.Request().Context()

	if _, err := h.applications.ByID(ctx, appID); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "application not found")
	}

	keys, err := h.apiKeys.List(ctx, appID)
	if err != nil {
		c.Logger().Errorf("list api keys: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list api keys")
	}
	return c.JSON(http.StatusOK, keys)
}

// rotateKey issues a new key for the same application and revokes the one at
// :keyId, atomically. Returns the new plaintext key, same shape as mintKey.
func (h *ApplicationsHandler) rotateKey(c echo.Context) error {
	appID := c.Param("id")
	keyID := c.Param("keyId")
	ctx := c.Request().Context()

	plaintext, prefix, secretHash, err := apikey.NewKey()
	if err != nil {
		c.Logger().Errorf("generate api key: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to rotate api key")
	}

	newKey, err := h.apiKeys.Rotate(ctx, keyID, appID, prefix, secretHash)
	switch {
	case errors.Is(err, apikey.ErrKeyNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "api key not found")
	case errors.Is(err, apikey.ErrKeyRevoked):
		return echo.NewHTTPError(http.StatusConflict, "api key is already revoked")
	case err != nil:
		c.Logger().Errorf("rotate api key: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to rotate api key")
	}

	return c.JSON(http.StatusCreated, mintKeyResponse{ID: newKey.ID, Prefix: newKey.Prefix, Key: plaintext})
}

// revokeKey revokes the key at :keyId. Revoking an already-revoked key
// succeeds as a no-op.
func (h *ApplicationsHandler) revokeKey(c echo.Context) error {
	appID := c.Param("id")
	keyID := c.Param("keyId")

	err := h.apiKeys.Revoke(c.Request().Context(), keyID, appID)
	switch {
	case errors.Is(err, apikey.ErrKeyNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "api key not found")
	case err != nil:
		c.Logger().Errorf("revoke api key: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to revoke api key")
	}
	return c.NoContent(http.StatusNoContent)
}
