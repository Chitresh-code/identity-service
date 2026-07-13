// Package userrefresh holds refresh tokens for end-user login sessions:
// long-lived-but-revocable opaque credentials a relying-party frontend
// trades for a fresh short-lived access JWT, so it isn't stuck re-running
// the full Auth0 login every hour. Same hashed-opaque-token shape as
// pkg/session and pkg/apikey, applied to a third case rather than a new one.
package userrefresh

import (
	"context"
	"time"
)

// TTL is how long a refresh token stays valid after issuance.
const TTL = 24 * time.Hour

// Token is an issued refresh token's metadata.
type Token struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

// Store persists and retrieves refresh tokens.
type Store interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (Token, error)
	// FindByTokenHash returns the token for tokenHash, if it exists and
	// hasn't expired.
	FindByTokenHash(ctx context.Context, tokenHash string) (Token, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
}
