package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"strings"

	"github.com/thomas666-beast/marketplace/internal/auth"
	"github.com/thomas666-beast/marketplace/internal/users"
)

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.PreferredLocale == "" {
		req.PreferredLocale = "en"
	}
	if req.Role == "" {
		req.Role = string(users.RoleBuyer)
	}

	// Validation order: email -> password -> display_name -> role -> locale.
	// Do not change without updating tests that assume this order.

	if !isValidEmail(req.Email) {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_email", "auth.invalid_email")
		return
	}
	if len(req.Password) < auth.MinPasswordLength {
		s.errorResponse(w, r, http.StatusBadRequest, "password_too_short", "auth.password_too_short")
		return
	}
	if len(req.Password) > auth.MaxPasswordLength {
		s.errorResponse(w, r, http.StatusBadRequest, "password_too_long", "auth.password_too_long")
		return
	}
	if len(req.DisplayName) < 2 || len(req.DisplayName) > 100 {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_display_name", "auth.invalid_display_name")
		return
	}
	if req.Role != string(users.RoleBuyer) && req.Role != string(users.RoleSeller) {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_role", "auth.invalid_role")
		return
	}
	if req.PreferredLocale != "en" && req.PreferredLocale != "ru" && req.PreferredLocale != "es" {
		req.PreferredLocale = "en"
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		// Should not happen — length was already validated above.
		// If it does, it's a programming error worth logging loudly.
		s.logger.Error("hash password failed after validation", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	u, err := s.users.Create(r.Context(), users.CreateInput{
		Email:           req.Email,
		PasswordHash:    hash,
		DisplayName:     req.DisplayName,
		Role:            users.Role(req.Role),
		PreferredLocale: users.Locale(req.PreferredLocale),
	})
	if err != nil {
		if errors.Is(err, users.ErrEmailExists) {
			s.errorResponse(w, r, http.StatusConflict, "email_taken", "auth.email_taken")
			return
		}
		s.logger.Error("create user failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	access, err := s.tokens.IssueAccessToken(u.ID, string(u.Role), string(u.PreferredLocale))
	if err != nil {
		s.logger.Error("issue access token failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}
	refresh, _, _, err := s.tokens.IssueRefreshToken(u.ID)
	if err != nil {
		s.logger.Error("issue refresh token failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	s.writeJSON(w, http.StatusCreated, authResponse{
		User:         toUserResponse(u),
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(auth.AccessTokenTTL.Seconds()),
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// Always return the same error regardless of what failed.
	// This prevents account enumeration.
	invalid := func() {
		s.errorResponse(w, r, http.StatusUnauthorized, "invalid_credentials", "auth.invalid_credentials")
	}

	u, err := s.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		invalid()
		return
	}
	if !u.IsActive {
		invalid()
		return
	}
	if err := auth.VerifyPassword(req.Password, u.PasswordHash); err != nil {
		invalid()
		return
	}

	access, err := s.tokens.IssueAccessToken(u.ID, string(u.Role), string(u.PreferredLocale))
	if err != nil {
		s.logger.Error("issue access token failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}
	refresh, _, _, err := s.tokens.IssueRefreshToken(u.ID)
	if err != nil {
		s.logger.Error("issue refresh token failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	s.writeJSON(w, http.StatusOK, authResponse{
		User:         toUserResponse(u),
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(auth.AccessTokenTTL.Seconds()),
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}

	claims, err := s.tokens.VerifyRefreshToken(req.RefreshToken)
	if err != nil {
		s.errorResponse(w, r, http.StatusUnauthorized, "invalid_token", "auth.invalid_token")
		return
	}

	u, err := s.users.GetByID(r.Context(), claims.UserID)
	if err != nil || !u.IsActive {
		s.errorResponse(w, r, http.StatusUnauthorized, "invalid_token", "auth.invalid_token")
		return
	}

	access, err := s.tokens.IssueAccessToken(u.ID, string(u.Role), string(u.PreferredLocale))
	if err != nil {
		s.logger.Error("issue access token failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	s.writeJSON(w, http.StatusOK, refreshResponse{
		AccessToken: access,
		ExpiresIn:   int(auth.AccessTokenTTL.Seconds()),
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxUserID).(string)
	u, err := s.users.GetByID(r.Context(), userID)
	if err != nil {
		s.errorResponse(w, r, http.StatusNotFound, "not_found", "errors.not_found")
		return
	}
	s.writeJSON(w, http.StatusOK, toUserResponse(u))
}

func isValidEmail(s string) bool {
	_, err := mail.ParseAddress(s)
	if err != nil {
		return false
	}
	// mail.ParseAddress accepts "Name <email>" too; ensure raw form.
	return !strings.ContainsAny(s, " <>\t\n")
}
