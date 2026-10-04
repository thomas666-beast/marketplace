package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/thomas666-beast/marketplace/internal/auth"
	"github.com/thomas666-beast/marketplace/internal/catalog"
	"github.com/thomas666-beast/marketplace/internal/i18n"
	"github.com/thomas666-beast/marketplace/internal/orders"
	"github.com/thomas666-beast/marketplace/internal/postgres"
	"github.com/thomas666-beast/marketplace/internal/users"
)

type Server struct {
	httpServer *http.Server
	db         *postgres.DB
	i18n       *i18n.Bundle
	logger     *slog.Logger
	users      *users.Repository
	categories *catalog.CategoryRepository
	products   *catalog.ProductRepository
	orders     *orders.Repository
	tokens     *auth.TokenManager
	mw         *authMiddleware
}

func New(
	port string,
	db *postgres.DB,
	bundle *i18n.Bundle,
	logger *slog.Logger,
	userRepo *users.Repository,
	categoryRepo *catalog.CategoryRepository,
	productRepo *catalog.ProductRepository,
	orderRepo *orders.Repository,
	tokens *auth.TokenManager,
) *Server {
	s := &Server{
		db:         db,
		i18n:       bundle,
		logger:     logger,
		users:      userRepo,
		categories: categoryRepo,
		products:   productRepo,
		orders:     orderRepo,
		tokens:     tokens,
	}
	s.mw = newAuthMiddleware(tokens, userRepo)

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.httpServer = &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return s
}

func (s *Server) Start() error {
	s.logger.Info("http server starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}
