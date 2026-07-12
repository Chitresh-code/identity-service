// Package handler is the Vercel Go serverless entrypoint. Vercel builds every
// .go file under api/ as its own function and calls the exported Handler; this
// is the only one, and vercel.json rewrites every path to it so Echo's own
// router still does the real routing -- identical behavior to the traditional
// binary in cmd/identity-service, just a different process lifecycle.
package handler

import (
	"context"
	"log"
	"net/http"

	"github.com/sales-intelligence1/identity-service/pkg/config"
	"github.com/sales-intelligence1/identity-service/pkg/server"
)

// app and initErr are set once per cold start and reused across every
// invocation the same warm instance serves afterward -- Vercel keeps the
// process alive between requests, so building the Echo app (and its Postgres
// connection pool) in init() rather than per-request is both correct and
// required: opening a fresh pool on every invocation would blow through
// Postgres's connection cap immediately.
var (
	app     http.Handler
	initErr error
)

func init() {
	cfg, err := config.Load()
	if err != nil {
		initErr = err
		return
	}

	e, err := server.New(context.Background(), cfg)
	if err != nil {
		initErr = err
		return
	}
	app = e
}

// Handler is the exported entrypoint Vercel's Go runtime calls per request.
func Handler(w http.ResponseWriter, r *http.Request) {
	if initErr != nil {
		log.Printf("identity-service failed to initialize: %v", initErr)
		http.Error(w, "service unavailable", http.StatusInternalServerError)
		return
	}
	app.ServeHTTP(w, r)
}
