// Package store holds persistence implementations.
package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewPostgres opens a GORM connection to Postgres and verifies it's reachable.
func NewPostgres(ctx context.Context, dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	// ponytail: kept low deliberately -- on serverless (Vercel), each warm
	// instance holds its own pool, and instances scale out independently, so a
	// generous per-instance limit multiplies into Postgres's global connection
	// cap fast. Revisit with a connection pooler (e.g. PgBouncer/Supabase
	// pooler) if concurrent instances start exhausting it.
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(2)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}
