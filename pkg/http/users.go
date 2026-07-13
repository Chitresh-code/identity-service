package http

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/sales-intelligence1/identity-service/pkg/user"
)

// UsersHandler wires the admin-gated user-management routes.
type UsersHandler struct {
	users user.Store
}

// NewUsersHandler builds a UsersHandler.
func NewUsersHandler(users user.Store) *UsersHandler {
	return &UsersHandler{users: users}
}

// Register wires the admin user routes onto e, gated by requireAdmin.
func (h *UsersHandler) Register(e *echo.Echo, requireAdmin echo.MiddlewareFunc) {
	e.POST("/admin/users/:id/role", h.setRole, requireAdmin)
}

type setRoleRequest struct {
	Role string `json:"role"`
}

// setRole assigns a product Role (member/lead) to a user. No self-service
// role-assignment UI exists in v1 -- this is the only path, an authenticated
// admin API call.
func (h *UsersHandler) setRole(c echo.Context) error {
	id := c.Param("id")

	var req setRoleRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	role := user.Role(req.Role)
	if role != user.RoleMember && role != user.RoleLead {
		return echo.NewHTTPError(http.StatusBadRequest, "role must be \"member\" or \"lead\"")
	}

	ctx := c.Request().Context()
	if err := h.users.SetRole(ctx, id, role); err != nil {
		c.Logger().Errorf("set role for user %q: %v", id, err)
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	u, err := h.users.ByID(ctx, id)
	if err != nil {
		c.Logger().Errorf("load user %q after role change: %v", id, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load updated user")
	}
	return c.JSON(http.StatusOK, u)
}
