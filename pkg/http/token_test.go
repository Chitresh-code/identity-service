package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/sales-intelligence1/identity-service/pkg/apikey"
	"github.com/sales-intelligence1/identity-service/pkg/signingkey"
)

type fakeAPIKeyStore struct {
	prefix     string
	secretHash string
	key        apikey.APIKey
}

func (f *fakeAPIKeyStore) Create(context.Context, string, string, string) (apikey.APIKey, error) {
	return apikey.APIKey{}, errors.New("not implemented")
}
func (f *fakeAPIKeyStore) List(context.Context, string) ([]apikey.APIKey, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeAPIKeyStore) Rotate(context.Context, string, string, string, string) (apikey.APIKey, error) {
	return apikey.APIKey{}, errors.New("not implemented")
}
func (f *fakeAPIKeyStore) Revoke(context.Context, string, string) error {
	return errors.New("not implemented")
}
func (f *fakeAPIKeyStore) ByPrefix(_ context.Context, prefix string) (apikey.APIKey, string, error) {
	if prefix != f.prefix {
		return apikey.APIKey{}, "", apikey.ErrKeyNotFound
	}
	return f.key, f.secretHash, nil
}

type fakeSigningKeyStore struct {
	key signingkey.Key
}

func (f *fakeSigningKeyStore) Latest(context.Context) (signingkey.Key, error) {
	return f.key, nil
}
func (f *fakeSigningKeyStore) All(context.Context) ([]signingkey.Key, error) {
	return []signingkey.Key{f.key}, nil
}
func (f *fakeSigningKeyStore) Save(context.Context, signingkey.Key) error {
	return errors.New("not implemented")
}

func TestTokenHandlerIssue(t *testing.T) {
	signingK, err := signingkey.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	const secret = "supersecret"
	now := time.Now()
	apps := &fakeAPIKeyStore{
		prefix:     "abc123",
		secretHash: apikey.HashSecret(secret),
		key:        apikey.APIKey{ID: "key-1", ApplicationID: "app-1"},
	}
	revokedApps := &fakeAPIKeyStore{
		prefix:     "abc123",
		secretHash: apikey.HashSecret(secret),
		key:        apikey.APIKey{ID: "key-1", ApplicationID: "app-1", RevokedAt: &now},
	}
	keys := &fakeSigningKeyStore{key: signingK}

	for _, tt := range []struct {
		name       string
		apiKeys    apikey.Store
		authHeader string
		wantStatus int
	}{
		{"valid key", apps, "Bearer abc123." + secret, http.StatusOK},
		{"wrong secret", apps, "Bearer abc123.wrongsecret", http.StatusUnauthorized},
		{"unknown prefix", apps, "Bearer zzzzzz." + secret, http.StatusUnauthorized},
		{"missing header", apps, "", http.StatusUnauthorized},
		{"malformed header", apps, "Bearer not-a-key", http.StatusUnauthorized},
		{"revoked key", revokedApps, "Bearer abc123." + secret, http.StatusUnauthorized},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := NewTokenHandler(tt.apiKeys, keys, "https://identity.example")

			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/token", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			gotErr := h.issue(c)
			status := statusFromResult(t, gotErr, rec)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, want %d (err=%v)", status, tt.wantStatus, gotErr)
			}

			if tt.wantStatus == http.StatusOK {
				var body tokenResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("unmarshal response: %v", err)
				}
				if body.AccessToken == "" {
					t.Error("expected non-empty access_token")
				}
				if body.TokenType != "Bearer" {
					t.Errorf("token_type = %q, want %q", body.TokenType, "Bearer")
				}
			}
		})
	}
}

func statusFromResult(t *testing.T, err error, rec *httptest.ResponseRecorder) int {
	t.Helper()
	if err == nil {
		return rec.Code
	}
	var he *echo.HTTPError
	if errors.As(err, &he) {
		return he.Code
	}
	t.Fatalf("handler returned non-HTTPError: %v", err)
	return 0
}
