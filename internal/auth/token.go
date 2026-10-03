package auth

import (
    "errors"
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

const (
    AccessTokenTTL  = 15 * time.Minute
    RefreshTokenTTL = 30 * 24 * time.Hour

    Issuer = "marketplace"
)

var (
    ErrInvalidToken = errors.New("invalid token")
    ErrExpiredToken = errors.New("token expired")
)

// Token type markers. Prevents a token issued for one purpose
// from being used for another.
const (
    tokenTypeAccess  = "access"
    tokenTypeRefresh = "refresh"
)

type Claims struct {
    UserID    string `json:"uid"`
    Role      string `json:"role"`
    Locale    string `json:"locale"`
    TokenType string `json:"type"`
    jwt.RegisteredClaims
}

type RefreshClaims struct {
    UserID    string `json:"uid"`
    SessionID string `json:"sid"`
    TokenType string `json:"type"`
    jwt.RegisteredClaims
}

type TokenManager struct {
    secret []byte
}

func NewTokenManager(secret string) (*TokenManager, error) {
    if len(secret) < 32 {
        return nil, fmt.Errorf("jwt secret must be at least 32 bytes, got %d", len(secret))
    }
    return &TokenManager{secret: []byte(secret)}, nil
}

func (m *TokenManager) IssueAccessToken(userID, role, locale string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		Role:      role,
		Locale:    locale,
		TokenType: tokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// VerifyAccessToken parses and validates an access token. Returns the claims.
func (m *TokenManager) VerifyAccessToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	},
		jwt.WithIssuer(Issuer),
		jwt.WithValidMethods([]string{"HS256"}),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.TokenType != tokenTypeAccess {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// IssueRefreshToken creates a long-lived refresh token.
// Returns the signed token and its jti (which must be stored in the DB).
func (m *TokenManager) IssueRefreshToken(userID string) (tokenString string, sessionID string, expiresAt time.Time, err error) {
	now := time.Now()
	sessionID = uuid.NewString()
	expiresAt = now.Add(RefreshTokenTTL)

	claims := RefreshClaims{
		UserID:    userID,
		SessionID: sessionID,
		TokenType: tokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        sessionID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err = token.SignedString(m.secret)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("sign refresh token: %w", err)
	}
	return tokenString, sessionID, expiresAt, nil
}

// VerifyRefreshToken parses and validates a refresh token.
func (m *TokenManager) VerifyRefreshToken(tokenString string) (*RefreshClaims, error) {
	claims := &RefreshClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	},
		jwt.WithIssuer(Issuer),
		jwt.WithValidMethods([]string{"HS256"}),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.TokenType != tokenTypeRefresh {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
