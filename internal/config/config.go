// Package config loads and validates service configuration from the environment.
package config

import (
	"errors"
	"os"
)

// Config holds identity-service's runtime settings, sourced from environment variables.
type Config struct {
	Port              string
	DatabaseURL       string
	Auth0Domain       string
	Auth0ClientID     string
	Auth0ClientSecret string
	AppBaseURL        string
}

// Load reads configuration from the environment and validates required values.
func Load() (Config, error) {
	cfg := Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		Auth0Domain:       os.Getenv("AUTH0_DOMAIN"),
		Auth0ClientID:     os.Getenv("AUTH0_CLIENT_ID"),
		Auth0ClientSecret: os.Getenv("AUTH0_CLIENT_SECRET"),
		AppBaseURL:        getEnv("APP_BASE_URL", "http://localhost:8080"),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.Auth0Domain == "" || cfg.Auth0ClientID == "" || cfg.Auth0ClientSecret == "" {
		return Config{}, errors.New("AUTH0_DOMAIN, AUTH0_CLIENT_ID, and AUTH0_CLIENT_SECRET are required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
