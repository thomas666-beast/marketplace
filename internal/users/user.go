package users

import "time"

type Role string

const (
	RoleBuyer  Role = "buyer"
	RoleSeller Role = "seller"
	RoleAdmin  Role = "admin"
)

type Locale string

const (
	LocaleEN Locale = "en"
	LocaleRU Locale = "ru"
	LocaleES Locale = "es"
)

type User struct {
	ID              string
	Email           string
	PasswordHash    string
	DisplayName     string
	Role            Role
	PreferredLocale Locale
	EmailVerified   bool
	IsActive        bool
	DeletedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CreateInput struct {
	Email           string
	PasswordHash    string
	DisplayName     string
	Role            Role
	PreferredLocale Locale
}
