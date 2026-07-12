package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"

	"github.com/sales-intelligence1/identity-service/pkg/config"
	"github.com/sales-intelligence1/identity-service/pkg/server"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on real environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	e, err := server.New(context.Background(), cfg)
	if err != nil {
		log.Fatalf("build server: %v", err)
	}

	if err := e.Start(":" + cfg.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
