// Package config loads and validates service configuration from the environment.
package config

import (
	"errors"
	"os"
)

// Config holds identity-service's runtime settings, sourced from environment variables.
type Config struct {
	Port        string
	DatabaseURL string
}

// Load reads configuration from the environment and validates required values.
func Load() (Config, error) {
	cfg := Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
