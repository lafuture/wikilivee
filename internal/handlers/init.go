package handlers

import (
	"encoding/json"
	"net/http"
	"time"
	"wikilivee/config"
	"wikilivee/internal/database"
)

type Handler struct {
	db         *database.Postgres
	cfg        *config.Config
	mwsClient  *http.Client
}

func NewHandler(db *database.Postgres, cfg *config.Config) *Handler {
	return &Handler{
		db:  db,
		cfg: cfg,
		mwsClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
