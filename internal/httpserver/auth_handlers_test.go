package httpserver_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegister_Success(t *testing.T) {
	ts := setupServerFixture(t)

	resp, body := postJSON(t, ts.url+"/api/v1/auth/register", map[string]any{
		"email":            "alice@example.com",
		"password":         "correct horse battery staple",
		"display_name":     "Alice",
		"role":             "buyer",
		"preferred_locale": "en",
	}, nil)

	require.Equal(t, http.StatusCreated, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.NotEmpty(t, out["access_token"])
	require.NotEmpty(t, out["refresh_token"])
	user := out["user"].(map[string]any)
	require.Equal(t, "alice@example.com", user["email"])
	require.NotContains(t, user, "password_hash")
}

func TestRegister_DuplicateEmail(t *testing.T) {
	ts := setupServerFixture(t)

	payload := map[string]any{
		"email": "dup@example.com", "password": "correct horse battery staple",
		"display_name": "Dup", "role": "buyer", "preferred_locale": "en",
	}

	_, _ = postJSON(t, ts.url+"/api/v1/auth/register", payload, nil)

	resp, body := postJSON(t, ts.url+"/api/v1/auth/register", payload, nil)
	require.Equal(t, http.StatusConflict, resp.StatusCode, string(body))
}

func TestRegister_ShortPassword(t *testing.T) {
	ts := setupServerFixture(t)

	resp, _ := postJSON(t, ts.url+"/api/v1/auth/register", map[string]any{
		"email": "x@example.com", "password": "short",
		"display_name": "X", "role": "buyer", "preferred_locale": "en",
	}, nil)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRegister_InvalidEmail(t *testing.T) {
	ts := setupServerFixture(t)

	resp, _ := postJSON(t, ts.url+"/api/v1/auth/register", map[string]any{
		"email": "not-an-email", "password": "correct horse battery staple",
		"display_name": "X", "role": "buyer", "preferred_locale": "en",
	}, nil)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestRegister_LocalizedError(t *testing.T) {
	ts := setupServerFixture(t)

	resp, body := postJSON(t, ts.url+"/api/v1/auth/register", map[string]any{
		"email":            "x@example.com",
		"password":         "short",
		"display_name":     "Valid Name",
		"role":             "buyer",
		"preferred_locale": "en",
	}, map[string]string{"Accept-Language": "ru"})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var out map[string]string
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, "Пароль должен содержать не менее 8 символов", out["message"])
}

func TestLogin_Success(t *testing.T) {
	ts := setupServerFixture(t)

	_, _ = postJSON(t, ts.url+"/api/v1/auth/register", map[string]any{
		"email": "bob@example.com", "password": "correct horse battery staple",
		"display_name": "Bob", "role": "buyer", "preferred_locale": "en",
	}, nil)

	resp, body := postJSON(t, ts.url+"/api/v1/auth/login", map[string]any{
		"email": "bob@example.com", "password": "correct horse battery staple",
	}, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.NotEmpty(t, out["access_token"])
}

func TestLogin_WrongPassword(t *testing.T) {
	ts := setupServerFixture(t)

	_, _ = postJSON(t, ts.url+"/api/v1/auth/register", map[string]any{
		"email": "carol@example.com", "password": "correct horse battery staple",
		"display_name": "Carol", "role": "buyer", "preferred_locale": "en",
	}, nil)

	resp, _ := postJSON(t, ts.url+"/api/v1/auth/login", map[string]any{
		"email": "carol@example.com", "password": "wrong password here",
	}, nil)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestLogin_UnknownEmail_SameResponse(t *testing.T) {
	ts := setupServerFixture(t)

	resp, _ := postJSON(t, ts.url+"/api/v1/auth/login", map[string]any{
		"email": "ghost@example.com", "password": "whatever long enough",
	}, nil)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMe_RequiresAuth(t *testing.T) {
	ts := setupServerFixture(t)

	resp, err := http.Get(ts.url + "/api/v1/users/me")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestMe_Success(t *testing.T) {
	ts := setupServerFixture(t)

	_, body := postJSON(t, ts.url+"/api/v1/auth/register", map[string]any{
		"email": "dave@example.com", "password": "correct horse battery staple",
		"display_name": "Dave", "role": "seller", "preferred_locale": "es",
	}, nil)

	var reg map[string]any
	require.NoError(t, json.Unmarshal(body, &reg))
	access := reg["access_token"].(string)

	req, _ := http.NewRequest(http.MethodGet, ts.url+"/api/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var me map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&me))
	require.Equal(t, "dave@example.com", me["email"])
	require.Equal(t, "seller", me["role"])
	require.Equal(t, "es", me["preferred_locale"])
}

func TestRefresh_Success(t *testing.T) {
	ts := setupServerFixture(t)

	_, body := postJSON(t, ts.url+"/api/v1/auth/register", map[string]any{
		"email": "eve@example.com", "password": "correct horse battery staple",
		"display_name": "Eve", "role": "buyer", "preferred_locale": "en",
	}, nil)

	var reg map[string]any
	require.NoError(t, json.Unmarshal(body, &reg))
	refresh := reg["refresh_token"].(string)

	resp, body := postJSON(t, ts.url+"/api/v1/auth/refresh", map[string]any{
		"refresh_token": refresh,
	}, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.NotEmpty(t, out["access_token"])
}

func TestRefresh_RejectsAccessToken(t *testing.T) {
	ts := setupServerFixture(t)

	_, body := postJSON(t, ts.url+"/api/v1/auth/register", map[string]any{
		"email": "frank@example.com", "password": "correct horse battery staple",
		"display_name": "Frank", "role": "buyer", "preferred_locale": "en",
	}, nil)

	var reg map[string]any
	require.NoError(t, json.Unmarshal(body, &reg))
	access := reg["access_token"].(string)

	resp, _ := postJSON(t, ts.url+"/api/v1/auth/refresh", map[string]any{
		"refresh_token": access,
	}, nil)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestRefresh_LocalizedError(t *testing.T) {
	ts := setupServerFixture(t)

	resp, body := postJSON(t, ts.url+"/api/v1/auth/refresh", map[string]any{
		"refresh_token": "garbage",
	}, map[string]string{"Accept-Language": "es"})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	var out map[string]string
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, "Tu sesión ha expirado. Por favor inicia sesión de nuevo", out["message"])
}
