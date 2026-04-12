package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetTablesHandler(w http.ResponseWriter, r *http.Request) {
	h.proxyMWS(w, r, fmt.Sprintf("%s/tables", h.cfg.MWSTablesURL))
}

func (h *Handler) GetTableHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.proxyMWS(w, r, fmt.Sprintf("%s/tables/%s", h.cfg.MWSTablesURL, id))
}

func (h *Handler) proxyMWS(w http.ResponseWriter, r *http.Request, url string) {
	if h.cfg.MWSTablesURL == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "MWS Tables API not configured"})
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build request"})
		return
	}

	if h.cfg.MWSTablesAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.cfg.MWSTablesAPIKey)
	}

	resp, err := h.mwsClient.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "MWS Tables API unreachable"})
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
