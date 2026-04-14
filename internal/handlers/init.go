package handlers

import (
	"encoding/json"
	"net/http"
	"wikilivee/config"
	"wikilivee/internal/database"
)

type Handler struct {
	db  *database.Postgres
	cfg *config.Config
}

func NewHandler(db *database.Postgres, cfg *config.Config) *Handler {
	return &Handler{
		db:  db,
		cfg: cfg,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
