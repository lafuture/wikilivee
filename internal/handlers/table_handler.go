package handlers

import (
	"encoding/json"
	"net/http"
	"wikilivee/internal/models"

	"github.com/go-chi/chi/v5"
)

type CreateTableRequest struct {
	Name    string                   `json:"name"`
	Columns []models.TableColumnSpec `json:"columns"`
}

type UpdateTableRequest struct {
	Name string `json:"name"`
}

type AddColumnRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type UpdateRowRequest struct {
	Values map[string]string `json:"values"`
}

func (h *Handler) GetTablesHandler(w http.ResponseWriter, r *http.Request) {
	tables, err := h.db.GetTables(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	if tables == nil {
		tables = []models.TableSummary{}
	}
	writeJSON(w, http.StatusOK, tables)
}

func (h *Handler) CreateTableHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Columns == nil {
		req.Columns = []models.TableColumnSpec{}
	}

	table, err := h.db.CreateTable(r.Context(), req.Name, req.Columns)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	writeJSON(w, http.StatusCreated, table)
}

func (h *Handler) GetTableHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	table, err := h.db.GetTable(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "table not found"})
		return
	}
	writeJSON(w, http.StatusOK, table)
}

func (h *Handler) UpdateTableHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	table, err := h.db.UpdateTable(r.Context(), id, req.Name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "table not found"})
		return
	}
	writeJSON(w, http.StatusOK, table)
}

func (h *Handler) DeleteTableHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.db.DeleteTable(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "table not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AddColumnHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req AddColumnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Type == "" {
		req.Type = "text"
	}

	col, err := h.db.AddColumn(r.Context(), id, req.Name, req.Type)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	writeJSON(w, http.StatusCreated, col)
}

func (h *Handler) DeleteColumnHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	colId := chi.URLParam(r, "colId")
	if id == "" || colId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.db.DeleteColumn(r.Context(), id, colId); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "column not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AddRowHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	row, err := h.db.AddRow(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	writeJSON(w, http.StatusCreated, row)
}

func (h *Handler) UpdateRowHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rowId := chi.URLParam(r, "rowId")
	if id == "" || rowId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req UpdateRowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	row, err := h.db.UpdateRow(r.Context(), id, rowId, req.Values)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "row not found"})
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (h *Handler) DeleteRowHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rowId := chi.URLParam(r, "rowId")
	if id == "" || rowId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if err := h.db.DeleteRow(r.Context(), id, rowId); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "row not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
