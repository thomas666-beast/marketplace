package httpserver

import "github.com/thomas666-beast/marketplace/internal/users"

type registerRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	DisplayName     string `json:"display_name"`
	Role            string `json:"role"`
	PreferredLocale string `json:"preferred_locale"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authResponse struct {
	User         userResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int          `json:"expires_in"`
}

type refreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type userResponse struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	DisplayName     string `json:"display_name"`
	Role            string `json:"role"`
	PreferredLocale string `json:"preferred_locale"`
	EmailVerified   bool   `json:"email_verified"`
	CreatedAt       string `json:"created_at"`
}

func toUserResponse(u users.User) userResponse {
	return userResponse{
		ID:              u.ID,
		Email:           u.Email,
		DisplayName:     u.DisplayName,
		Role:            string(u.Role),
		PreferredLocale: string(u.PreferredLocale),
		EmailVerified:   u.EmailVerified,
		CreatedAt:       u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
