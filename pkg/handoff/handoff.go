// Package handoff holds short-lived, single-use login handoff codes: the
// bridge that lets a relying-party frontend on a different domain (e.g.
// sales-intel-web) obtain a token for the user who just finished logging in
// here, without sharing identity-service's own session cookie cross-domain.
package handoff

import (
	"context"
	"errors"
	"time"
)

// TTL is how long a handoff code stays valid before the relying party must
// exchange it. Short, since the exchange happens immediately after the
// browser redirect that carries the code.
const TTL = 60 * time.Second

// ErrNotFound is returned by Consume when the code doesn't exist, is
// expired, or was already used.
var ErrNotFound = errors.New("handoff code not found or expired")

// Handoff is a pending login handoff for a user.
type Handoff struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

// Store persists and retrieves handoff codes.
type Store interface {
	Create(ctx context.Context, userID, codeHash string, expiresAt time.Time) (Handoff, error)
	// Consume atomically finds and deletes an unexpired handoff by its code
	// hash -- a code is single-use, so finding it also spends it. Returns
	// ErrNotFound if no unexpired code matches.
	Consume(ctx context.Context, codeHash string) (Handoff, error)
}
