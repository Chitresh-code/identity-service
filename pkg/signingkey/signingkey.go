// Package signingkey holds this service's JWT signing key material: the RSA
// keypair used to issue tokens and publish a JWKS so resource servers can
// verify them without a shared secret.
package signingkey

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// ErrNoKeys is returned by Store.Latest when no signing key has been
// generated yet.
var ErrNoKeys = errors.New("no signing keys available")

// Key is an RSA signing keypair identified by ID (used as the JWT "kid").
type Key struct {
	ID         string
	PrivateKey *rsa.PrivateKey
	CreatedAt  time.Time
}

// Store persists signing keys.
type Store interface {
	// Latest returns the most recently created key, used to sign new tokens.
	// Returns ErrNoKeys if none exists.
	Latest(ctx context.Context) (Key, error)
	// All returns every key, newest first, so a JWKS can keep publishing an
	// old key's public half during rotation.
	All(ctx context.Context) ([]Key, error)
	Save(ctx context.Context, k Key) error
}

// Generate creates a fresh 2048-bit RSA signing key with a random ID.
func Generate() (Key, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return Key{}, fmt.Errorf("generate rsa key: %w", err)
	}

	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		return Key{}, fmt.Errorf("generate key id: %w", err)
	}

	return Key{ID: hex.EncodeToString(idBytes), PrivateKey: priv, CreatedAt: time.Now()}, nil
}

// EnsureLatest returns the current signing key, generating and persisting one
// if the store is empty.
//
// ponytail: two instances booting against an empty store for the first time
// can both generate a key; the JWKS endpoint publishes all keys so tokens
// signed with either still verify. Fix with a DB-level advisory lock if that
// race ever matters in practice.
func EnsureLatest(ctx context.Context, store Store) (Key, error) {
	k, err := store.Latest(ctx)
	if err == nil {
		return k, nil
	}
	if !errors.Is(err, ErrNoKeys) {
		return Key{}, err
	}

	k, err = Generate()
	if err != nil {
		return Key{}, err
	}
	if err := store.Save(ctx, k); err != nil {
		return Key{}, err
	}
	return k, nil
}

// JWKS builds the public JSON Web Key Set for keys.
func JWKS(keys []Key) jose.JSONWebKeySet {
	set := jose.JSONWebKeySet{}
	for _, k := range keys {
		set.Keys = append(set.Keys, jose.JSONWebKey{
			Key:       &k.PrivateKey.PublicKey,
			KeyID:     k.ID,
			Algorithm: string(jose.RS256),
			Use:       "sig",
		})
	}
	return set
}

// IssueToken signs a short-lived JWT asserting subject (an application ID),
// using key. Returns the serialized token and its expiry.
func IssueToken(key Key, issuer, subject string, ttl time.Duration) (token string, expiresAt time.Time, err error) {
	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: key.PrivateKey},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", key.ID),
	)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("build signer: %w", err)
	}

	now := time.Now()
	expiresAt = now.Add(ttl)
	claims := jwt.Claims{
		Issuer:   issuer,
		Subject:  subject,
		IssuedAt: jwt.NewNumericDate(now),
		Expiry:   jwt.NewNumericDate(expiresAt),
	}

	token, err = jwt.Signed(signer).Claims(claims).Serialize()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return token, expiresAt, nil
}
