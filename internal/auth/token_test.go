package auth_test

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/thomas666-beast/marketplace/internal/auth"
)

const testSecret = "test-secret-must-be-at-least-32-bytes-long-ok"

func newManager(t *testing.T) *auth.TokenManager {
	t.Helper()
	m, err := auth.NewTokenManager(testSecret)
	require.NoError(t, err)
	return m
}

func TestNewTokenManager_ShortSecret(t *testing.T) {
	_, err := auth.NewTokenManager("short")
	require.Error(t, err, "short secrets must be rejected")
}

func TestNewTokenManager_ExactMinSecret(t *testing.T) {
	_, err := auth.NewTokenManager(strings.Repeat("a", 32))
	require.NoError(t, err)
}

func TestAccessToken_RoundTrip(t *testing.T) {
	m := newManager(t)

	token, err := m.IssueAccessToken("user-123", "buyer", "ru")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := m.VerifyAccessToken(token)
	require.NoError(t, err)
	require.Equal(t, "user-123", claims.UserID)
	require.Equal(t, "buyer", claims.Role)
	require.Equal(t, "ru", claims.Locale)
	require.Equal(t, "marketplace", claims.Issuer)
	require.Equal(t, "user-123", claims.Subject)
	require.NotEmpty(t, claims.ID)
}

func TestAccessToken_Expired(t *testing.T) {
	m := newManager(t)

	// Handcraft an expired token using the same secret.
	now := time.Now().Add(-2 * time.Hour)
	claims := auth.Claims{
		UserID: "user-123",
		Role:   "buyer",
		Locale: "en",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "marketplace",
			Subject:   "user-123",
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			ID:        "test-jti",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testSecret))
	require.NoError(t, err)

	_, err = m.VerifyAccessToken(signed)
	require.ErrorIs(t, err, auth.ErrExpiredToken)
}

func TestAccessToken_WrongSecret(t *testing.T) {
	m := newManager(t)

	token, err := m.IssueAccessToken("user-123", "buyer", "en")
	require.NoError(t, err)

	other, err := auth.NewTokenManager("different-secret-at-least-32-bytes-long-ok")
	require.NoError(t, err)

	_, err = other.VerifyAccessToken(token)
	require.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestAccessToken_Tampered(t *testing.T) {
	m := newManager(t)

	token, err := m.IssueAccessToken("user-123", "buyer", "en")
	require.NoError(t, err)

	// Flip one character in the signature part.
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	parts[2] = parts[2][:len(parts[2])-1] + "X"
	tampered := strings.Join(parts, ".")

	_, err = m.VerifyAccessToken(tampered)
	require.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestAccessToken_AlgNone_Rejected(t *testing.T) {
	m := newManager(t)

	// Craft a token with alg=none. This is the classic attack.
	// If we ever forget WithValidMethods, this test catches it.
	token := jwt.NewWithClaims(jwt.SigningMethodNone, auth.Claims{
		UserID: "attacker",
		Role:   "admin",
		Locale: "en",
	})
	unsigned, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = m.VerifyAccessToken(unsigned)
	require.ErrorIs(t, err, auth.ErrInvalidToken, "alg=none must be rejected")
}

func TestAccessToken_WrongIssuer(t *testing.T) {
	m := newManager(t)

	// Sign a token with the right secret but wrong issuer.
	claims := auth.Claims{
		UserID: "user-123",
		Role:   "buyer",
		Locale: "en",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "some-other-service",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testSecret))
	require.NoError(t, err)

	_, err = m.VerifyAccessToken(signed)
	require.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestRefreshToken_RoundTrip(t *testing.T) {
	m := newManager(t)

	token, sessionID, expiresAt, err := m.IssueRefreshToken("user-123")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotEmpty(t, sessionID)
	require.WithinDuration(t, time.Now().Add(auth.RefreshTokenTTL), expiresAt, 5*time.Second)

	claims, err := m.VerifyRefreshToken(token)
	require.NoError(t, err)
	require.Equal(t, "user-123", claims.UserID)
	require.Equal(t, sessionID, claims.SessionID)
}

func TestRefreshToken_NotValidAsAccess(t *testing.T) {
	m := newManager(t)

	refreshToken, _, _, err := m.IssueRefreshToken("user-123")
	require.NoError(t, err)

	// A refresh token must NOT verify as an access token.
	_, err = m.VerifyAccessToken(refreshToken)
	require.ErrorIs(t, err, auth.ErrInvalidToken,
		"refresh token must not be usable as access token")
}

func TestAccessToken_NotValidAsRefresh(t *testing.T) {
	m := newManager(t)

	accessToken, err := m.IssueAccessToken("user-123", "buyer", "en")
	require.NoError(t, err)

	// An access token must NOT verify as a refresh token.
	_, err = m.VerifyRefreshToken(accessToken)
	require.ErrorIs(t, err, auth.ErrInvalidToken,
		"access token must not be usable as refresh token")
}

func TestAccessToken_DifferentJTI(t *testing.T) {
	m := newManager(t)

	t1, err := m.IssueAccessToken("user-123", "buyer", "en")
	require.NoError(t, err)
	t2, err := m.IssueAccessToken("user-123", "buyer", "en")
	require.NoError(t, err)

	c1, err := m.VerifyAccessToken(t1)
	require.NoError(t, err)
	c2, err := m.VerifyAccessToken(t2)
	require.NoError(t, err)

	require.NotEqual(t, c1.ID, c2.ID, "each token must have a unique jti")
}
