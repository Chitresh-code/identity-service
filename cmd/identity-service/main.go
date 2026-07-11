package main

import (
	"context"
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/sales-intelligence1/identity-service/internal/auth0"
	"github.com/sales-intelligence1/identity-service/internal/config"
	identityhttp "github.com/sales-intelligence1/identity-service/internal/http"
	"github.com/sales-intelligence1/identity-service/internal/store"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on real environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	db, err := store.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("get sql.DB: %v", err)
	}
	defer sqlDB.Close()

	auth0Client, err := auth0.New(ctx, cfg.Auth0Domain, cfg.Auth0ClientID, cfg.Auth0ClientSecret, cfg.AppBaseURL+"/auth/callback")
	if err != nil {
		log.Fatalf("init auth0 client: %v", err)
	}
	authHandler := identityhttp.NewAuthHandler(
		auth0Client,
		store.NewUserStore(db),
		store.NewSessionStore(db),
		cfg.AppBaseURL,
	)

	e := echo.New()
	e.Use(middleware.Logger())
	e.GET("/health", healthHandler)
	authHandler.Register(e)

	if err := e.Start(":" + cfg.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}
