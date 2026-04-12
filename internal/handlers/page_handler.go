package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"wikilivee/internal/models"

	"github.com/go-chi/chi/v5"
)

type CreatePageRequest struct {
	Title string `json:"title"`
}

type SavePageRequest struct {
	Title   string         `json:"title"`
	Content []models.Block `json:"content"`
	Version int            `json:"version"`
}

type SavePageResponse struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *Handler) GetPagesHandler(w http.ResponseWriter, r *http.Request) {
	pages, err := h.db.GetPages(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	if pages == nil {
		pages = []models.PageSummary{}
	}
	writeJSON(w, http.StatusOK, pages)
}

func (h *Handler) CreatePageHandler(w http.ResponseWriter, r *http.Request) {
	var req CreatePageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	page, err := h.db.CreatePage(r.Context(), newID(), req.Title)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	writeJSON(w, http.StatusCreated, page)
}

func (h *Handler) GetPageHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	page, err := h.db.GetPage(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) SavePageHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req SavePageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	newVersion, err := h.db.SavePage(r.Context(), id, req.Title, req.Content, req.Version)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
		return
	}
	writeJSON(w, http.StatusOK, SavePageResponse{ID: id, Version: newVersion})
}

func (h *Handler) DeletePageHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.db.DeletePage(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetPageBacklinksHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	backlinks, err := h.db.GetPageBacklinks(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	if backlinks == nil {
		backlinks = []models.PageSummary{}
	}
	writeJSON(w, http.StatusOK, backlinks)
}
