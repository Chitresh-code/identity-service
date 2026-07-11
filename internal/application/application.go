// Package application holds identity-service's application (service client)
// domain: the machine callers that get issued API keys, e.g. client-data-service
// calling market-data-service.
package application

import (
	"context"
	"time"
)

// Application is a registered service client that can be issued API keys.
type Application struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Store persists and retrieves applications.
type Store interface {
	Create(ctx context.Context, name string) (Application, error)
	List(ctx context.Context) ([]Application, error)
	ByID(ctx context.Context, id string) (Application, error)
}
