package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"wikilivee/config"
	"wikilivee/internal/database"
	"wikilivee/internal/handlers"
	"wikilivee/internal/server"
	"wikilivee/internal/ws"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.NewDatabase(cfg.DBURL, cfg.MIGRAURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	hub := ws.NewHub(db)
	handler := handlers.NewHandler(db, cfg)
	srv := server.NewServer(cfg, handler, hub)

	defer srv.Shutdown(ctx)

	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
