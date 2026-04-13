package server

import (
	"context"
	"net/http"
	"time"
	"wikilivee/config"
	"wikilivee/internal/handlers"
	"wikilivee/internal/ws"
)

type Server struct {
	srv *http.Server
}

func NewServer(cfg *config.Config, handler *handlers.Handler, hub *ws.Hub) *Server {
	return &Server{
		srv: &http.Server{
			Addr:              cfg.ListenAddr,
			Handler:           NewRouter(handler, hub, cfg.JWTSecret),
			ReadHeaderTimeout: 5 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
