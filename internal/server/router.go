package server

import (
	"net/http"
	"os"
	"strings"

	"wikilivee/internal/handlers"
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
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:5174", "http://localhost:3000", "http://217.73.116.173:5173", "http://217.73.116.173:3000", "http://217.73.116.173:8080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.Get("/api/pages", handler.GetPagesHandler)
	r.Post("/api/pages", handler.CreatePageHandler)
	r.Get("/api/pages/{id}", handler.GetPageHandler)
	r.Put("/api/pages/{id}", handler.SavePageHandler)
	r.Delete("/api/pages/{id}", handler.DeletePageHandler)
	r.Get("/api/pages/{id}/backlinks", handler.GetPageBacklinksHandler)
	r.Get("/api/pages/{id}/children", handler.GetPageChildrenHandler)

	r.Get("/api/pages/{id}/versions", handler.GetPageVersionsHandler)
	r.Get("/api/pages/{id}/versions/{version}", handler.GetPageVersionHandler)
	r.Post("/api/pages/{id}/versions/{version}/restore", handler.RestorePageVersionHandler)

	//r.Post("/api/pages/search", handler.SearchPagesHandler)
	//r.Get("/api/pages/graph", handler.GraphPagesHandler)

	//r.Get("/api/pages/{id}/comments", handler.GetCommentsHandler)
	//r.Post("/api/pages/{id}/comments", handler.AddCommentHandler)
	//r.Delete("/api/pages/{id}/comments", handler.DeleteCommentHandler)

	//r.Post("/api/ai/complete", handler.GenerateHandler)
	//r.Post("/api/ai/summarize", handler.SummarizeHandler)
	//r.Post("/api/ai/suggest", handler.SuggestHandler)

	r.Get("/api/tables", handler.GetTablesHandler)
	r.Get("/api/tables/{id}", handler.GetTableHandler)

	r.Get("/ws/pages/{id}", ws.NewHandler(hub))

	r.Get("/openapi.yaml", serveFile("openapi.yaml"))
	r.Get("/swagger-ui.html", serveFile("swagger-ui.html"))
	r.Get("/redoc-static.html", serveFile("redoc-static.html"))

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

func serveFile(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, name)
	}
}
