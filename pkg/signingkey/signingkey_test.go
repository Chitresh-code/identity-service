package signingkey

import (
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func TestIssueTokenVerifiesAgainstJWKS(t *testing.T) {
	key, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	token, expiresAt, err := IssueToken(key, "https://identity.example", "app-123", time.Minute)
	if err != nil {
		t.Fatalf("IssueToken() error = %v", err)
	}

	set := JWKS([]Key{key})
	matches := set.Key(key.ID)
	if len(matches) != 1 {
		t.Fatalf("JWKS lookup for kid %q returned %d keys, want 1", key.ID, len(matches))
	}

	parsed, err := jwt.ParseSigned(token, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		t.Fatalf("ParseSigned() error = %v", err)
	}

	var claims jwt.Claims
	if err := parsed.Claims(matches[0].Key, &claims); err != nil {
		t.Fatalf("Claims() error = %v", err)
	}

	if claims.Subject != "app-123" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "app-123")
	}
	if claims.Issuer != "https://identity.example" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "https://identity.example")
	}
	if !claims.Expiry.Time().Equal(expiresAt.Truncate(time.Second)) {
		t.Errorf("Expiry = %v, want ~%v", claims.Expiry.Time(), expiresAt)
	}
}

func TestGenerateProducesUniqueKeys(t *testing.T) {
	a, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	b, err := Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if a.ID == b.ID {
		t.Errorf("two calls to Generate() produced the same ID %q", a.ID)
	}
}
