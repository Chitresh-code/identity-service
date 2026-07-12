// Package auth0 wraps Auth0's OIDC discovery, the OAuth2 Authorization Code
// flow, and ID token verification behind a small client.
package auth0

import (
	"context"
	"fmt"
	"net/url"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Client issues Auth0 login URLs and exchanges authorization codes for a
// verified caller identity.
type Client struct {
	oauth2Config oauth2.Config
	verifier     *oidc.IDTokenVerifier
	domain       string
}

// Claims holds the identity fields read out of a verified Auth0 ID token.
type Claims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// New discovers Auth0's OIDC configuration at domain and builds a ready-to-use
// Client. redirectURL must match one of the application's Allowed Callback URLs
// in the Auth0 dashboard.
func New(ctx context.Context, domain, clientID, clientSecret, redirectURL string) (*Client, error) {
	issuer := "https://" + domain + "/"
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discover auth0 oidc config at %q: %w", issuer, err)
	}

	return &Client{
		oauth2Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
		domain:   domain,
	}, nil
}

// LoginURL returns the Auth0 Universal Login URL for the given CSRF state.
// Callers must generate a random state, stash it (e.g. in a short-lived
// cookie), and check it matches on the callback before trusting the response.
func (c *Client) LoginURL(state string) string {
	return c.oauth2Config.AuthCodeURL(state)
}

// Exchange trades an authorization code from the callback redirect for tokens,
// verifies the returned ID token against Auth0's JWKS, and returns its claims.
func (c *Client) Exchange(ctx context.Context, code string) (Claims, error) {
	token, err := c.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return Claims{}, fmt.Errorf("exchange authorization code: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return Claims{}, fmt.Errorf("token response missing id_token")
	}

	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return Claims{}, fmt.Errorf("verify id token: %w", err)
	}

	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		return Claims{}, fmt.Errorf("parse id token claims: %w", err)
	}
	return claims, nil
}

// LogoutURL returns Auth0's logout endpoint, which clears the Auth0-side SSO
// session and redirects the browser back to returnTo afterward.
func (c *Client) LogoutURL(returnTo string) string {
	v := url.Values{}
	v.Set("client_id", c.oauth2Config.ClientID)
	v.Set("returnTo", returnTo)
	return fmt.Sprintf("https://%s/v2/logout?%s", c.domain, v.Encode())
}
