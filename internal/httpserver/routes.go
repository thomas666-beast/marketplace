package httpserver

import "net/http"

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Health and demo
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /welcome", s.handleWelcome)

	// Auth (public)
	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", s.handleRefresh)

	// Users (protected)
	mux.HandleFunc("GET /api/v1/users/me", s.mw.requireAuth(s.handleMe))
}
