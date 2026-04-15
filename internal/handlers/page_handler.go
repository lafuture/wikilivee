package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"wikilivee/internal/middleware"
	"wikilivee/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type CreatePageRequest struct {
	Title    string `json:"title"`
	Icon     string `json:"icon"`
	ParentId string `json:"parentId"`
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

type SearchRequest struct {
	Text string `json:"text"`
}

type PageAccessResponse struct {
	Owner struct {
		UserID   string `json:"userId"`
		Username string `json:"username"`
		Role     string `json:"role"`
	} `json:"owner"`
	Entries   []models.PageAccessEntry `json:"entries"`
	CanManage bool                     `json:"canManage"`
}

type UpsertPageAccessRequest struct {
	Role string `json:"role"`
}

func (h *Handler) ensureCanEditPage(w http.ResponseWriter, r *http.Request, pageID string) bool {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	username, _ := r.Context().Value(middleware.UsernameKey).(string)
	if userID == "" || username == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return false
	}

	canEdit, err := h.db.CanUserEditPage(r.Context(), pageID, userID, username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return false
	}
	if !canEdit {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return false
	}
	return true
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

func (h *Handler) GetMyPagesHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	username, _ := r.Context().Value(middleware.UsernameKey).(string)
	if userID == "" || username == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	pages, err := h.db.GetPagesForUser(r.Context(), userID, username)
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

	username, _ := r.Context().Value(middleware.UsernameKey).(string)
	page, err := h.db.CreatePage(r.Context(), newID(), req.Title, req.Icon, req.ParentId, username)
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
	if !h.ensureCanEditPage(w, r, id) {
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
	if !h.ensureCanEditPage(w, r, id) {
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

func (h *Handler) GetPageChildrenHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	childrens, err := h.db.GetPageChildren(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
		return
	}

	writeJSON(w, http.StatusOK, childrens)
}

func (h *Handler) GetPageVersionsHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	versions, err := h.db.GetPageVersions(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
		return
	}

	writeJSON(w, http.StatusOK, versions)
}

func (h *Handler) GetPageVersionHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	version := chi.URLParam(r, "version")
	if version == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid version"})
		return
	}

	page, err := h.db.GetPageVersion(r.Context(), id, version)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
		return
	}

	writeJSON(w, http.StatusOK, page)
}

func (h *Handler) RestorePageVersionHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if !h.ensureCanEditPage(w, r, id) {
		return
	}

	version := chi.URLParam(r, "version")
	if version == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid version"})
		return
	}

	newVersion, err := h.db.RestorePage(r.Context(), id, version)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "can not restore the page"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"id": id, "version": newVersion})
}

func (h *Handler) SearchPagesHandler(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	results, err := h.db.SearchPages(r.Context(), req.Text)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}

	if results == nil {
		results = []models.SearchResult{}
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *Handler) GraphPagesHandler(w http.ResponseWriter, r *http.Request) {
	graph, err := h.db.GetPagesGraph(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

type AddCommentRequest struct {
	Text       string `json:"text"`
	AnchorFrom int    `json:"anchorFrom"`
	AnchorTo   int    `json:"anchorTo"`
	AnchorText string `json:"anchorText"`
}

func (h *Handler) GetCommentsHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	comments, err := h.db.GetComments(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	writeJSON(w, http.StatusOK, comments)
}

func (h *Handler) AddCommentHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req AddCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Text == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text required"})
		return
	}

	username, _ := r.Context().Value(middleware.UsernameKey).(string)
	if username == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	comment, err := h.db.CreateComment(r.Context(), id, username, req.Text, req.AnchorFrom, req.AnchorTo, req.AnchorText)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	writeJSON(w, http.StatusCreated, comment)
}

func (h *Handler) DeleteCommentHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	commentID := r.URL.Query().Get("commentId")
	if id == "" || commentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing id or commentId"})
		return
	}
	username, _ := r.Context().Value(middleware.UsernameKey).(string)
	if username == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.db.DeleteComment(r.Context(), id, commentID, username); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only comment author can delete"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SearchUsersHandler(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeJSON(w, http.StatusOK, []models.UserSummary{})
		return
	}
	items, err := h.db.SearchUsers(r.Context(), query, 10)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) GetPageAccessHandler(w http.ResponseWriter, r *http.Request) {
	pageID := chi.URLParam(r, "id")
	if pageID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	page, err := h.db.GetPage(r.Context(), pageID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
		return
	}

	currentUsername, _ := r.Context().Value(middleware.UsernameKey).(string)
	ownerUsername := strings.TrimSpace(page.Author)
	if ownerUsername == "" {
		ownerUsername = currentUsername
	}
	canManage := ownerUsername != "" && currentUsername == ownerUsername
	entries, err := h.db.GetPageAccessEntries(r.Context(), pageID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}

	ownerID := ""
	if ownerUsername != "" {
		if ownerUser, ownerErr := h.db.GetUserByUsername(r.Context(), ownerUsername); ownerErr == nil {
			ownerID = ownerUser.ID
		}
	}

	var response PageAccessResponse
	response.Owner.UserID = ownerID
	response.Owner.Username = ownerUsername
	response.Owner.Role = "owner"
	response.Entries = entries
	response.CanManage = canManage
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) UpsertPageAccessHandler(w http.ResponseWriter, r *http.Request) {
	pageID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")
	if pageID == "" || userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	page, err := h.db.GetPage(r.Context(), pageID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
		return
	}

	currentUsername, _ := r.Context().Value(middleware.UsernameKey).(string)
	ownerUsername := strings.TrimSpace(page.Author)
	if ownerUsername == "" {
		ownerUsername = currentUsername
	}
	if ownerUsername == "" || currentUsername != ownerUsername {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	var req UpsertPageAccessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = "editor"
	}
	if role != "editor" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported role"})
		return
	}

	ownerUser, err := h.db.GetUserByUsername(r.Context(), ownerUsername)
	if err == nil && userID == ownerUser.ID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "owner permission can not be changed"})
		return
	}

	if err := h.db.UpsertPagePermission(r.Context(), pageID, userID, role); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) DeletePageAccessHandler(w http.ResponseWriter, r *http.Request) {
	pageID := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")
	if pageID == "" || userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	page, err := h.db.GetPage(r.Context(), pageID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "page not found"})
		return
	}

	currentUsername, _ := r.Context().Value(middleware.UsernameKey).(string)
	ownerUsername := strings.TrimSpace(page.Author)
	if ownerUsername == "" {
		ownerUsername = currentUsername
	}
	if ownerUsername == "" || currentUsername != ownerUsername {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}

	ownerUser, err := h.db.GetUserByUsername(r.Context(), ownerUsername)
	if err == nil && userID == ownerUser.ID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "owner permission can not be removed"})
		return
	}

	if err := h.db.DeletePagePermission(r.Context(), pageID, userID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database error"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
