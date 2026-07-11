// Package apikey holds identity-service's API key domain: credentials issued
// to applications for service-to-service calls.
package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// ErrKeyNotFound is returned when a Store method can't find the requested key
// (for the given application, where applicable).
var ErrKeyNotFound = errors.New("api key not found")

// ErrKeyRevoked is returned by Rotate when the key being rotated is already revoked.
var ErrKeyRevoked = errors.New("api key is already revoked")

// APIKey is an issued credential's metadata -- never the secret itself, which
// is only ever available as plaintext once, at creation.
type APIKey struct {
	ID            string     `json:"id"`
	ApplicationID string     `json:"application_id"`
	Prefix        string     `json:"prefix"`
	CreatedAt     time.Time  `json:"created_at"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
}

// Store persists and retrieves API keys.
type Store interface {
	Create(ctx context.Context, applicationID, prefix, secretHash string) (APIKey, error)
	// List returns every key issued to applicationID, newest first.
	List(ctx context.Context, applicationID string) ([]APIKey, error)
	// Rotate atomically issues a new key for the same application and revokes
	// oldKeyID. Returns ErrKeyNotFound if oldKeyID doesn't belong to
	// applicationID, or ErrKeyRevoked if it's already revoked.
	Rotate(ctx context.Context, oldKeyID, applicationID, newPrefix, newSecretHash string) (APIKey, error)
	// Revoke marks id (belonging to applicationID) revoked. Revoking an
	// already-revoked key is a no-op, not an error. Returns ErrKeyNotFound if
	// id doesn't belong to applicationID.
	Revoke(ctx context.Context, id, applicationID string) error
}

// NewKey generates a fresh API key: a plaintext prefix for fast lookup and a
// secret whose hash alone is ever stored. The full key handed to the caller is
// "<prefix>.<secret>" -- the Stripe/GitHub PAT pattern -- shown once at
// creation and unrecoverable afterward.
func NewKey() (plaintext, prefix, secretHash string, err error) {
	prefixBytes := make([]byte, 6)
	if _, err := rand.Read(prefixBytes); err != nil {
		return "", "", "", fmt.Errorf("generate key prefix: %w", err)
	}
	prefix = hex.EncodeToString(prefixBytes)

	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", "", "", fmt.Errorf("generate key secret: %w", err)
	}
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)

	return prefix + "." + secret, prefix, HashSecret(secret), nil
}

// HashSecret returns the lookup hash for a key's secret half.
func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
