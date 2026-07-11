// Package user holds identity-service's account domain: one row per Auth0 subject.
package user

import (
	"context"
	"time"
)

// User is an identity-service account, synced from an Auth0 profile on login.
type User struct {
	ID        string    `json:"id"`
	Auth0Sub  string    `json:"-"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store persists and retrieves users.
type Store interface {
	// UpsertByAuth0Sub creates the user for sub if it doesn't exist, or refreshes
	// its email/name from the latest Auth0 profile if it does.
	UpsertByAuth0Sub(ctx context.Context, sub, email, name string) (User, error)
	// ByID looks up a user by their identity-service id.
	ByID(ctx context.Context, id string) (User, error)
}
