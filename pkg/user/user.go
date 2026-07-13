// Package user holds identity-service's account domain: one row per Auth0 subject.
package user

import (
	"context"
	"time"
)

// Role is a product-level permission scope for end users of relying-party
// apps (sales-intel-web and friends) -- a separate axis from IsAdmin, which
// only gates this service's own /admin/* routes. A user can be an admin,
// hold a product Role, both, or neither.
type Role string

const (
	RoleMember Role = "member" // sees/works only their own assigned accounts
	RoleLead   Role = "lead"   // sees the whole pipeline
)

// User is an identity-service account, synced from an Auth0 profile on login.
type User struct {
	ID        string    `json:"id"`
	Auth0Sub  string    `json:"-"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	IsAdmin   bool      `json:"is_admin"`
	Role      *Role     `json:"role,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store persists and retrieves users.
type Store interface {
	// UpsertByAuth0Sub creates the user for sub if it doesn't exist, or refreshes
	// its email/name from the latest Auth0 profile if it does. The very first
	// user ever created is granted admin; nothing about later logins changes an
	// existing user's admin status.
	UpsertByAuth0Sub(ctx context.Context, sub, email, name string) (User, error)
	// ByID looks up a user by their identity-service id.
	ByID(ctx context.Context, id string) (User, error)
	// SetRole assigns id's product Role. No self-service UI for this in v1 --
	// only reachable via the admin-gated /admin/users/:id/role route.
	SetRole(ctx context.Context, id string, role Role) error
}
