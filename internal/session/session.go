// Package session holds identity-service's server-side login session domain.
package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

// CookieName is the cookie identity-service uses to carry the raw session token.
const CookieName = "identity_session"

// TTL is how long a session stays valid after login.
const TTL = 24 * time.Hour

// Session is a logged-in session for a user, looked up by the hash of its token.
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Store persists and retrieves sessions.
type Store interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (Session, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
}

// NewToken generates a fresh random session token and its lookup hash. The raw
// token goes in the cookie; only the hash is ever stored, so a stolen database
// row can't be replayed as a cookie -- same pattern as hashed API key storage.
func NewToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate session token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashToken(raw), nil
}

// HashToken returns the lookup hash for a raw session token.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
