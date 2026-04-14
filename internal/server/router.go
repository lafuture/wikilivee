package server

import (
	"net/http"
	"os"
	"strings"

	"wikilivee/internal/handlers"
	apimw "wikilivee/internal/middleware"
	"wikilivee/internal/ws"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func staticRoot() string {
	if d := os.Getenv("STATIC_DIR"); d != "" {
		return d
	}
	return ""
}

func NewRouter(handler *handlers.Handler, hub *ws.Hub, jwtSecret string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"http://localhost",
			"http://localhost:*",
			"https://localhost",
			"https://localhost:*",
			"http://127.0.0.1",
			"http://127.0.0.1:*",
			"https://127.0.0.1",
			"https://127.0.0.1:*",
			"http://217.73.116.173",
			"http://217.73.116.173:*",
			"https://217.73.116.173",
			"https://217.73.116.173:*",
			"http://balooai.ru",
			"https://balooai.ru",
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.Post("/api/auth/register", handler.RegisterHandler)
	r.Post("/api/auth/login", handler.LoginHandler)

	r.Group(func(r chi.Router) {
		r.Use(apimw.Auth(jwtSecret))

		r.Get("/api/auth/me", handler.MeHandler)

		r.Get("/api/pages", handler.GetPagesHandler)
		r.Post("/api/pages", handler.CreatePageHandler)

		// Search and graph must be registered BEFORE the {id} patterns
		r.Post("/api/pages/search", handler.SearchPagesHandler)
		r.Get("/api/pages/graph", handler.GraphPagesHandler)

		r.Get("/api/pages/{id}", handler.GetPageHandler)
		r.Put("/api/pages/{id}", handler.SavePageHandler)
		r.Delete("/api/pages/{id}", handler.DeletePageHandler)
		r.Get("/api/pages/{id}/backlinks", handler.GetPageBacklinksHandler)
		r.Get("/api/pages/{id}/children", handler.GetPageChildrenHandler)

		r.Get("/api/pages/{id}/versions", handler.GetPageVersionsHandler)
		r.Get("/api/pages/{id}/versions/{version}", handler.GetPageVersionHandler)
		r.Post("/api/pages/{id}/versions/{version}/restore", handler.RestorePageVersionHandler)

		r.Get("/api/pages/{id}/comments", handler.GetCommentsHandler)
		r.Post("/api/pages/{id}/comments", handler.AddCommentHandler)
		r.Delete("/api/pages/{id}/comments", handler.DeleteCommentHandler)

		r.Get("/api/tables", handler.GetTablesHandler)
		r.Post("/api/tables", handler.CreateTableHandler)
		r.Get("/api/tables/{id}", handler.GetTableHandler)
		r.Put("/api/tables/{id}", handler.UpdateTableHandler)
		r.Delete("/api/tables/{id}", handler.DeleteTableHandler)
		r.Post("/api/tables/{id}/columns", handler.AddColumnHandler)
		r.Delete("/api/tables/{id}/columns/{colId}", handler.DeleteColumnHandler)
		r.Post("/api/tables/{id}/rows", handler.AddRowHandler)
		r.Put("/api/tables/{id}/rows/{rowId}", handler.UpdateRowHandler)
		r.Delete("/api/tables/{id}/rows/{rowId}", handler.DeleteRowHandler)
	})

	r.Get("/ws/pages/{id}", ws.NewHandler(hub))

	root := staticRoot()
	fs := http.FileServer(http.Dir(root))
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/api/") || strings.HasPrefix(req.URL.Path, "/ws/") {
			http.NotFound(w, req)
			return
		}
		if req.Method != http.MethodGet && req.Method != http.MethodHead {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		fs.ServeHTTP(w, req)
	})

	return r
}
