package httpserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/thomas666-beast/marketplace/internal/auth"
	"github.com/thomas666-beast/marketplace/internal/users"
)

type contextKey string

const (
	ctxUserID contextKey = "user_id"
	ctxRole   contextKey = "role"
	ctxLocale contextKey = "locale"
)

type authMiddleware struct {
	tokens *auth.TokenManager
	users  *users.Repository
}

func newAuthMiddleware(tokens *auth.TokenManager, repo *users.Repository) *authMiddleware {
	return &authMiddleware{tokens: tokens, users: repo}
}

func (m *authMiddleware) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			m.unauthorized(w, r)
			return
		}
		tokenString := strings.TrimPrefix(header, "Bearer ")

		claims, err := m.tokens.VerifyAccessToken(tokenString)
		if err != nil {
			m.unauthorized(w, r)
			return
		}

		// Verify the user still exists and is active.
		// This DB hit is worth it: it prevents deactivated users from acting
		// until their token expires.
		u, err := m.users.GetByID(r.Context(), claims.UserID)
		if err != nil {
			m.unauthorized(w, r)
			return
		}
		if !u.IsActive {
			m.unauthorized(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserID, u.ID)
		ctx = context.WithValue(ctx, ctxRole, string(u.Role))
		ctx = context.WithValue(ctx, ctxLocale, string(u.PreferredLocale))

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (m *authMiddleware) unauthorized(w http.ResponseWriter, r *http.Request) {
	locale := localeFromRequest(r)
	messages := map[string]string{
		"en": "You are not authorized",
		"ru": "Вы не авторизованы",
		"es": "No estás autorizado",
	}
	msg, ok := messages[locale]
	if !ok {
		msg = messages["en"]
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized","message":"` + msg + `"}`))
}

func (m *authMiddleware) requireSeller(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(ctxRole).(string)
		if role != "seller" && role != "admin" {
			locale := localeFromRequest(r)
			messages := map[string]string{
				"en": "You do not have permission to perform this action",
				"ru": "У вас нет прав на это действие",
				"es": "No tienes permiso para realizar esta acción",
			}
			msg := messages[locale]
			if msg == "" {
				msg = messages["en"]
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"forbidden","message":"` + msg + `"}`))
			return
		}
		next.ServeHTTP(w, r)
	}
}

