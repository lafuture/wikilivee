package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"wikilivee/internal/models"

	"github.com/go-chi/chi/v5"
)

// ── request types ────────────────────────────────────────────────────────────

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

// ── APITable response types ───────────────────────────────────────────────────

type atResponse struct {
	Success bool            `json:"success"`
	Code    int             `json:"code"`
	Data    json.RawMessage `json:"data"`
	Message string          `json:"message"`
}

type atNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type atField struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	IsPrimary bool   `json:"isPrimary"`
}

type atRecord struct {
	RecordID string                 `json:"recordId"`
	Fields   map[string]interface{} `json:"fields"`
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *Handler) mwsTablesEnabled() bool {
	return strings.TrimSpace(h.cfg.MWSTablesURL) != ""
}

// atFieldTypeToLocal maps APITable field types to our column types.
func atFieldTypeToLocal(t string) string {
	switch t {
	case "Number", "Currency", "Percent", "AutoNumber", "Rating":
		return "number"
	case "DateTime", "CreatedTime", "LastModifiedTime":
		return "date"
	case "SingleSelect", "MultiSelect":
		return "select"
	default:
		return "text"
	}
}

// localTypeToAT maps our column types to APITable field types.
func localTypeToAT(t string) string {
	switch t {
	case "number":
		return "Number"
	case "date":
		return "DateTime"
	case "select":
		return "SingleSelect"
	default:
		return "SingleText"
	}
}

// atDo performs an authenticated request to the APITable fusion API.
func (h *Handler) atDo(method, path string, body interface{}) ([]byte, int, error) {
	base := strings.TrimSuffix(h.cfg.MWSTablesURL, "/")
	url := base + path

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+h.cfg.MWSTablesAPIKey)

	resp, err := h.mwsClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return data, resp.StatusCode, err
}

// atGet calls GET and returns the parsed atResponse.
func (h *Handler) atGet(path string) (*atResponse, error) {
	data, _, err := h.atDo("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var r atResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// atPost calls POST and returns the parsed atResponse.
func (h *Handler) atPost(path string, body interface{}) (*atResponse, error) {
	data, _, err := h.atDo("POST", path, body)
	if err != nil {
		return nil, err
	}
	var r atResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// atPatch calls PATCH and returns the parsed atResponse.
func (h *Handler) atPatch(path string, body interface{}) (*atResponse, error) {
	data, _, err := h.atDo("PATCH", path, body)
	if err != nil {
		return nil, err
	}
	var r atResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// atDelete calls DELETE and returns the status code.
func (h *Handler) atDelete(path string) (int, error) {
	_, code, err := h.atDo("DELETE", path, nil)
	return code, err
}

// atGetFields returns the fields of a datasheet mapped to TableColumn.
func (h *Handler) atGetFields(datasheetID string) ([]models.TableColumn, error) {
	r, err := h.atGet(fmt.Sprintf("/datasheets/%s/fields", datasheetID))
	if err != nil || !r.Success {
		return nil, fmt.Errorf("fields fetch failed")
	}
	var payload struct {
		Fields []atField `json:"fields"`
	}
	if err := json.Unmarshal(r.Data, &payload); err != nil {
		return nil, err
	}
	cols := make([]models.TableColumn, 0, len(payload.Fields))
	for i, f := range payload.Fields {
		cols = append(cols, models.TableColumn{
			ID:       f.ID,
			TableID:  datasheetID,
			Name:     f.Name,
			Type:     atFieldTypeToLocal(f.Type),
			Position: i,
		})
	}
	return cols, nil
}

// atGetRecords returns records using fieldKey=name, then re-maps keys to field IDs.
func (h *Handler) atGetRecords(datasheetID string, fields []models.TableColumn) ([]models.TableRow, error) {
	r, err := h.atGet(fmt.Sprintf("/datasheets/%s/records?fieldKey=name&pageSize=200", datasheetID))
	if err != nil || !r.Success {
		return nil, fmt.Errorf("records fetch failed")
	}
	var payload struct {
		Records []atRecord `json:"records"`
	}
	if err := json.Unmarshal(r.Data, &payload); err != nil {
		return nil, err
	}

	// name → id index
	nameToID := make(map[string]string, len(fields))
	for _, f := range fields {
		nameToID[f.Name] = f.ID
	}

	rows := make([]models.TableRow, 0, len(payload.Records))
	for _, rec := range payload.Records {
		values := make(map[string]string, len(rec.Fields))
		for name, val := range rec.Fields {
			id := nameToID[name]
			if id == "" {
				id = name
			}
			switch v := val.(type) {
			case string:
				values[id] = v
			case float64:
				values[id] = fmt.Sprintf("%g", v)
			case []interface{}:
				if len(v) > 0 {
					if s, ok := v[0].(string); ok {
						values[id] = s
					}
				}
			default:
				if val != nil {
					values[id] = fmt.Sprintf("%v", val)
				}
			}
		}
		rows = append(rows, models.TableRow{
			ID:      rec.RecordID,
			TableID: datasheetID,
			Values:  values,
		})
	}
	return rows, nil
}

// ── handlers ──────────────────────────────────────────────────────────────────

func (h *Handler) GetTablesHandler(w http.ResponseWriter, r *http.Request) {
	if !h.mwsTablesEnabled() {
		tables, err := h.db.GetTables(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
			return
		}
		if tables == nil {
			tables = []models.TableSummary{}
		}
		writeJSON(w, http.StatusOK, tables)
		return
	}

	spaceID := h.cfg.MWSTablesSpaceID
	resp, err := h.atGet(fmt.Sprintf("/spaces/%s/nodes", spaceID))
	if err != nil || !resp.Success {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "MWS Tables API error"})
		return
	}

	var payload struct {
		Nodes []atNode `json:"nodes"`
	}
	if err := json.Unmarshal(resp.Data, &payload); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "parse error"})
		return
	}

	summaries := make([]models.TableSummary, 0)
	for _, n := range payload.Nodes {
		if n.Type == "Datasheet" {
			summaries = append(summaries, models.TableSummary{ID: n.ID, Name: n.Name})
		}
	}
	writeJSON(w, http.StatusOK, summaries)
}

func (h *Handler) CreateTableHandler(w http.ResponseWriter, r *http.Request) {
	if !h.mwsTablesEnabled() {
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
		return
	}

	var req CreateTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	spaceID := h.cfg.MWSTablesSpaceID
	resp, err := h.atPost(fmt.Sprintf("/spaces/%s/datasheets", spaceID), map[string]interface{}{
		"name": req.Name,
	})
	if err != nil || !resp.Success {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "MWS Tables create failed"})
		return
	}

	var created struct {
		ID        string    `json:"id"`
		CreatedAt int64     `json:"createdAt"`
		Fields    []atField `json:"fields"`
	}
	if err := json.Unmarshal(resp.Data, &created); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "parse error"})
		return
	}

	cols := make([]models.TableColumn, 0, len(created.Fields))
	for i, f := range created.Fields {
		cols = append(cols, models.TableColumn{
			ID:       f.ID,
			TableID:  created.ID,
			Name:     f.Name,
			Type:     atFieldTypeToLocal(f.Type),
			Position: i,
		})
	}

	table := models.Table{
		ID:      created.ID,
		Name:    req.Name,
		Columns: cols,
		Rows:    []models.TableRow{},
	}
	writeJSON(w, http.StatusCreated, table)
}

func (h *Handler) GetTableHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if !h.mwsTablesEnabled() {
		table, err := h.db.GetTable(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "table not found"})
			return
		}
		writeJSON(w, http.StatusOK, table)
		return
	}

	fields, err := h.atGetFields(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "table not found"})
		return
	}
	rows, err := h.atGetRecords(id, fields)
	if err != nil {
		rows = []models.TableRow{}
	}

	// Get name from space nodes
	name := id
	spaceID := h.cfg.MWSTablesSpaceID
	if nodesResp, nerr := h.atGet(fmt.Sprintf("/spaces/%s/nodes", spaceID)); nerr == nil && nodesResp.Success {
		var payload struct {
			Nodes []atNode `json:"nodes"`
		}
		if json.Unmarshal(nodesResp.Data, &payload) == nil {
			for _, n := range payload.Nodes {
				if n.ID == id {
					name = n.Name
					break
				}
			}
		}
	}

	table := models.Table{ID: id, Name: name, Columns: fields, Rows: rows}
	writeJSON(w, http.StatusOK, table)
}

func (h *Handler) UpdateTableHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if !h.mwsTablesEnabled() {
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
		return
	}

	// APITable doesn't expose a rename datasheet endpoint in fusion v1 — return current state
	writeJSON(w, http.StatusOK, models.Table{ID: id, Columns: []models.TableColumn{}, Rows: []models.TableRow{}})
}

func (h *Handler) DeleteTableHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	if !h.mwsTablesEnabled() {
		if err := h.db.DeleteTable(r.Context(), id); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "table not found"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
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

	if !h.mwsTablesEnabled() {
		col, err := h.db.AddColumn(r.Context(), id, req.Name, req.Type)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
			return
		}
		writeJSON(w, http.StatusCreated, col)
		return
	}

	resp, err := h.atPost(fmt.Sprintf("/datasheets/%s/fields", id), map[string]interface{}{
		"name": req.Name,
		"type": localTypeToAT(req.Type),
	})
	if err != nil || !resp.Success {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "MWS add column failed"})
		return
	}

	var field struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(resp.Data, &field)
	col := models.TableColumn{ID: field.ID, TableID: id, Name: req.Name, Type: req.Type}
	writeJSON(w, http.StatusCreated, col)
}

func (h *Handler) DeleteColumnHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	colID := chi.URLParam(r, "colId")

	if !h.mwsTablesEnabled() {
		if err := h.db.DeleteColumn(r.Context(), id, colID); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "column not found"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	code, err := h.atDelete(fmt.Sprintf("/datasheets/%s/fields/%s", id, colID))
	if err != nil || (code != 200 && code != 204) {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "MWS delete column failed"})
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

	if !h.mwsTablesEnabled() {
		row, err := h.db.AddRow(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
			return
		}
		writeJSON(w, http.StatusCreated, row)
		return
	}

	resp, err := h.atPost(
		fmt.Sprintf("/datasheets/%s/records?fieldKey=name", id),
		map[string]interface{}{
			"records": []map[string]interface{}{
				{"fields": map[string]interface{}{}},
			},
		},
	)
	if err != nil || !resp.Success {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "MWS add row failed"})
		return
	}

	var payload struct {
		Records []atRecord `json:"records"`
	}
	_ = json.Unmarshal(resp.Data, &payload)
	row := models.TableRow{Values: map[string]string{}}
	if len(payload.Records) > 0 {
		row.ID = payload.Records[0].RecordID
		row.TableID = id
	}
	writeJSON(w, http.StatusCreated, row)
}

func (h *Handler) UpdateRowHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rowID := chi.URLParam(r, "rowId")

	if !h.mwsTablesEnabled() {
		var req UpdateRowRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		row, err := h.db.UpdateRow(r.Context(), id, rowID, req.Values)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "row not found"})
			return
		}
		writeJSON(w, http.StatusOK, row)
		return
	}

	var req UpdateRowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	// Get field ID→name mapping
	fields, err := h.atGetFields(id)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "MWS fields fetch failed"})
		return
	}
	idToName := make(map[string]string, len(fields))
	for _, f := range fields {
		idToName[f.ID] = f.Name
	}

	// Translate column IDs → field names
	namedFields := make(map[string]interface{}, len(req.Values))
	for colID, val := range req.Values {
		name := idToName[colID]
		if name == "" {
			name = colID
		}
		namedFields[name] = val
	}

	resp, err := h.atPatch(
		fmt.Sprintf("/datasheets/%s/records?fieldKey=name", id),
		map[string]interface{}{
			"records": []map[string]interface{}{
				{"recordId": rowID, "fields": namedFields},
			},
		},
	)
	if err != nil || !resp.Success {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "MWS update row failed"})
		return
	}

	row := models.TableRow{ID: rowID, TableID: id, Values: req.Values}
	writeJSON(w, http.StatusOK, row)
}

func (h *Handler) DeleteRowHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rowID := chi.URLParam(r, "rowId")

	if !h.mwsTablesEnabled() {
		if err := h.db.DeleteRow(r.Context(), id, rowID); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "row not found"})
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	code, err := h.atDelete(fmt.Sprintf("/datasheets/%s/records?recordIds=%s", id, rowID))
	if err != nil || (code != 200 && code != 204) {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "MWS delete row failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
