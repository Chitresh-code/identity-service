// Package apikey holds identity-service's API key domain: credentials issued
// to applications for service-to-service calls.
package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

// APIKey is an issued credential's metadata -- never the secret itself, which
// is only ever available as plaintext once, at creation.
type APIKey struct {
	ID            string
	ApplicationID string
	Prefix        string
	CreatedAt     time.Time
}

// Store persists and retrieves API keys.
type Store interface {
	Create(ctx context.Context, applicationID, prefix, secretHash string) (APIKey, error)
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
