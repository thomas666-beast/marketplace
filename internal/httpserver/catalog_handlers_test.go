package httpserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// helper: register a seller and return the access token
func registerSeller(t *testing.T, ts *serverFixture, email string) string {
	t.Helper()
	_, body := postJSON(t, ts.url+"/api/v1/auth/register", map[string]any{
		"email":            email,
		"password":         "correct horse battery staple",
		"display_name":     "Seller",
		"role":             "seller",
		"preferred_locale": "en",
	}, nil)

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	return out["access_token"].(string)
}

func registerBuyer(t *testing.T, ts *serverFixture, email string) string {
	t.Helper()
	_, body := postJSON(t, ts.url+"/api/v1/auth/register", map[string]any{
		"email":            email,
		"password":         "correct horse battery staple",
		"display_name":     "Buyer",
		"role":             "buyer",
		"preferred_locale": "en",
	}, nil)

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	return out["access_token"].(string)
}

func authGet(t *testing.T, url, token string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, body
}

func authJSON(t *testing.T, method, url, token string, payload any) (*http.Response, []byte) {
	t.Helper()
	var bodyReader io.Reader
	if payload != nil {
		buf, err := json.Marshal(payload)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, body
}

// getSmartphoneCategoryID fetches the seeded category.
func getSmartphoneCategoryID(t *testing.T, ts *serverFixture) string {
	t.Helper()
	resp, body := authGet(t, ts.url+"/api/v1/categories/smartphones", "")
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	return out["id"].(string)
}

func TestCategories_List(t *testing.T) {
	ts := setupServerFixture(t)

	resp, body := authGet(t, ts.url+"/api/v1/categories", "")
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	items := out["items"].([]any)
	require.Len(t, items, 5, "5 root categories in seed data")

	// Check that a known category's name appears in the correct locale.
	resp, body = authGet(t, ts.url+"/api/v1/categories", "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Request in Russian
	req, _ := http.NewRequest(http.MethodGet, ts.url+"/api/v1/categories", nil)
	req.Header.Set("Accept-Language", "ru")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	var ruOut map[string]any
	require.NoError(t, json.Unmarshal(body, &ruOut))
	ruItems := ruOut["items"].([]any)
	first := ruItems[0].(map[string]any)
	// "Электроника" has sort_order 10 (first root)
	require.Equal(t, "Электроника", first["name"])
}

func TestCategories_GetBySlug(t *testing.T) {
	ts := setupServerFixture(t)

	resp, body := authGet(t, ts.url+"/api/v1/categories/smartphones", "")
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, "smartphones", out["slug"])
	require.Equal(t, "Smartphones", out["name"])
}

func TestCategories_GetBySlug_NotFound(t *testing.T) {
	ts := setupServerFixture(t)

	resp, _ := authGet(t, ts.url+"/api/v1/categories/nope", "")
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestSeller_CreateProduct(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerSeller(t, ts, "s1@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	resp, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/products", token, map[string]any{
		"category_id":    catID,
		"name":           "iPhone 15",
		"description":    "A great phone",
		"price_cents":    129900,
		"currency":       "USD",
		"stock_quantity": 5,
		"status":         "active",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, "iPhone 15", out["name"])
	require.Equal(t, float64(129900), out["price_cents"])
	require.Equal(t, "USD", out["currency"])
	require.Equal(t, "active", out["status"])
	require.NotEmpty(t, out["slug"])
}

func TestSeller_CreateProduct_BuyerForbidden(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerBuyer(t, ts, "b1@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	resp, _ := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/products", token, map[string]any{
		"category_id": catID, "name": "X", "price_cents": 100,
		"currency": "USD", "stock_quantity": 1, "status": "draft",
	})
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestSeller_CreateProduct_LocalizedForbidden(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerBuyer(t, ts, "b2@example.com")

	req, _ := http.NewRequest(http.MethodPost, ts.url+"/api/v1/seller/products",
		bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept-Language", "ru")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	var out map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	require.Equal(t, "У вас нет прав на это действие", out["message"])
}

func TestSeller_UpdateProduct_Own(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerSeller(t, ts, "s2@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/products", token, map[string]any{
		"category_id": catID, "name": "Old Name", "price_cents": 1000,
		"currency": "USD", "stock_quantity": 1, "status": "draft",
	})
	var created map[string]any
	require.NoError(t, json.Unmarshal(body, &created))
	id := created["id"].(string)

	resp, body := authJSON(t, http.MethodPatch, ts.url+"/api/v1/seller/products/"+id, token, map[string]any{
		"name":        "New Name",
		"price_cents": 2000,
	})
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, "New Name", out["name"])
	require.Equal(t, float64(2000), out["price_cents"])
}

func TestSeller_UpdateProduct_NotOwner_404(t *testing.T) {
	ts := setupServerFixture(t)
	token1 := registerSeller(t, ts, "s3@example.com")
	token2 := registerSeller(t, ts, "s4@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/products", token1, map[string]any{
		"category_id": catID, "name": "P", "price_cents": 100,
		"currency": "USD", "stock_quantity": 1, "status": "draft",
	})
	var created map[string]any
	require.NoError(t, json.Unmarshal(body, &created))
	id := created["id"].(string)

	resp, _ := authJSON(t, http.MethodPatch, ts.url+"/api/v1/seller/products/"+id, token2, map[string]any{
		"name": "Hijack",
	})
	require.Equal(t, http.StatusNotFound, resp.StatusCode,
		"non-owner must get 404, not 403, to avoid leaking product existence")
}

func TestSeller_DeleteProduct(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerSeller(t, ts, "s5@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/products", token, map[string]any{
		"category_id": catID, "name": "To Delete", "price_cents": 100,
		"currency": "USD", "stock_quantity": 1, "status": "active",
	})
	var created map[string]any
	require.NoError(t, json.Unmarshal(body, &created))
	id := created["id"].(string)
	slug := created["slug"].(string)

	resp, _ := authJSON(t, http.MethodDelete, ts.url+"/api/v1/seller/products/"+id, token, nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Public fetch should now 404.
	resp, _ = authGet(t, ts.url+"/api/v1/products/"+slug, "")
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestProducts_PublicListing_OnlyActive(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerSeller(t, ts, "s6@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	// One active, one draft.
	_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/products", token, map[string]any{
		"category_id": catID, "name": "Published", "price_cents": 100,
		"currency": "USD", "stock_quantity": 1, "status": "active",
	})
	_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/products", token, map[string]any{
		"category_id": catID, "name": "Hidden", "price_cents": 100,
		"currency": "USD", "stock_quantity": 1, "status": "draft",
	})

	resp, body := authGet(t, ts.url+"/api/v1/products?category_id="+catID, "")
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	items := out["items"].([]any)
	require.Len(t, items, 1, "public listing must not include drafts")
}

func TestProducts_SellerListing_IncludesDrafts(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerSeller(t, ts, "s7@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/products", token, map[string]any{
		"category_id": catID, "name": "A", "price_cents": 100,
		"currency": "USD", "stock_quantity": 1, "status": "active",
	})
	_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/products", token, map[string]any{
		"category_id": catID, "name": "D", "price_cents": 100,
		"currency": "USD", "stock_quantity": 1, "status": "draft",
	})

	resp, body := authGet(t, ts.url+"/api/v1/seller/products", token)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	items := out["items"].([]any)
	require.Len(t, items, 2, "seller sees own drafts")
}

func TestProducts_Pagination(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerSeller(t, ts, "s8@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	for i := 0; i < 5; i++ {
		_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/products", token, map[string]any{
			"category_id": catID, "name": "P" + string(rune('A'+i)),
			"price_cents": 100, "currency": "USD", "stock_quantity": 1, "status": "active",
		})
	}

	resp, body := authGet(t, ts.url+"/api/v1/products?category_id="+catID+"&limit=2", "")
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var p1 map[string]any
	require.NoError(t, json.Unmarshal(body, &p1))
	require.Len(t, p1["items"].([]any), 2)
	cursor := p1["next_cursor"].(string)
	require.NotEmpty(t, cursor)

	resp, body = authGet(t, ts.url+"/api/v1/products?category_id="+catID+"&limit=2&cursor="+cursor, "")
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var p2 map[string]any
	require.NoError(t, json.Unmarshal(body, &p2))
	require.Len(t, p2["items"].([]any), 2)
}

func TestProducts_InvalidLimit(t *testing.T) {
	ts := setupServerFixture(t)
	resp, _ := authGet(t, ts.url+"/api/v1/products?limit=200", "")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestProducts_Search(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerSeller(t, ts, "s9@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/products", token, map[string]any{
		"category_id": catID, "name": "iPhone Pro Max",
		"description": "The best", "price_cents": 100,
		"currency": "USD", "stock_quantity": 1, "status": "active",
	})
	_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/products", token, map[string]any{
		"category_id": catID, "name": "Android Phone",
		"description": "Also good", "price_cents": 100,
		"currency": "USD", "stock_quantity": 1, "status": "active",
	})

	resp, body := authGet(t, ts.url+"/api/v1/products?q=iphone", "")
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	items := out["items"].([]any)
	require.Len(t, items, 1)
	require.Contains(t, items[0].(map[string]any)["name"], "iPhone")
}
