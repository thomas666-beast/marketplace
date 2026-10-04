package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// helper: create a product as a seller and return its ID.
func createProduct(t *testing.T, ts *serverFixture, sellerToken, catID, name string, price int64, stock int) string {
	t.Helper()
	resp, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/products", sellerToken, map[string]any{
		"category_id":    catID,
		"name":           name,
		"price_cents":    price,
		"currency":       "USD",
		"stock_quantity": stock,
		"status":         "active",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	return out["id"].(string)
}

// helper: get the seller's user ID by calling /me.
func getMyID(t *testing.T, ts *serverFixture, token string) string {
	t.Helper()
	resp, body := authGet(t, ts.url+"/api/v1/users/me", token)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	return out["id"].(string)
}

func TestCheckout_Success(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "checkout-seller@example.com")
	buyerToken := registerBuyer(t, ts, "checkout-buyer@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	sellerID := getMyID(t, ts, sellerToken)
	productID := createProduct(t, ts, sellerToken, catID, "iPhone", 1000, 5)

	resp, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerID,
		"items":            []map[string]any{{"product_id": productID, "quantity": 2}},
		"shipping_cents":   500,
		"shipping_address": map[string]any{"city": "Moscow", "country": "RU"},
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, "pending_payment", out["status"])
	require.Equal(t, float64(2000), out["subtotal_cents"])
	require.Equal(t, float64(2500), out["total_cents"])
	require.NotEmpty(t, out["order_number"])
	items := out["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, "iPhone", items[0].(map[string]any)["product_name"])
}

func TestCheckout_InsufficientStock(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "oos-seller@example.com")
	buyerToken := registerBuyer(t, ts, "oos-buyer@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	sellerID := getMyID(t, ts, sellerToken)
	productID := createProduct(t, ts, sellerToken, catID, "Only One", 1000, 1)

	resp, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerID,
		"items":            []map[string]any{{"product_id": productID, "quantity": 5}},
		"shipping_address": map[string]any{"city": "Moscow", "country": "RU"},
	})
	require.Equal(t, http.StatusConflict, resp.StatusCode, string(body))

	var out map[string]string
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, "insufficient_stock", out["error"])
}

func TestCheckout_EmptyItems(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "empty-seller@example.com")
	buyerToken := registerBuyer(t, ts, "empty-buyer@example.com")

	sellerID := getMyID(t, ts, sellerToken)

	resp, _ := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerID,
		"items":            []any{},
		"shipping_address": map[string]any{"city": "Moscow"},
	})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestCheckout_RequiresAuth(t *testing.T) {
	ts := setupServerFixture(t)
	resp, _ := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", "", map[string]any{})
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestCheckout_LocalizedError(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "loc-seller@example.com")
	buyerToken := registerBuyer(t, ts, "loc-buyer@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	sellerID := getMyID(t, ts, sellerToken)
	productID := createProduct(t, ts, sellerToken, catID, "One", 1000, 1)

	body := mustJSON(t, map[string]any{
		"seller_id": sellerID,
		"items":     []map[string]any{{"product_id": productID, "quantity": 5}},
		"shipping_address": map[string]any{"city": "Moscow"},
	})
	req, err := http.NewRequest(http.MethodPost, ts.url+"/api/v1/orders/checkout", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+buyerToken)
	req.Header.Set("Accept-Language", "es")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusConflict, resp.StatusCode)

	var out map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	require.Equal(t, "No hay suficiente stock disponible", out["message"])
}

func TestOrders_Lifecycle(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "lc-seller@example.com")
	buyerToken := registerBuyer(t, ts, "lc-buyer@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	sellerID := getMyID(t, ts, sellerToken)
	productID := createProduct(t, ts, sellerToken, catID, "P", 1000, 5)

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerID,
		"items":            []map[string]any{{"product_id": productID, "quantity": 1}},
		"shipping_address": map[string]any{"city": "Moscow"},
	})
	var created map[string]any
	require.NoError(t, json.Unmarshal(body, &created))
	id := created["id"].(string)

	resp, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/"+id+"/pay", buyerToken, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	var paid map[string]any
	require.NoError(t, json.Unmarshal(body, &paid))
	require.Equal(t, "paid", paid["status"])

	resp, body = authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/orders/"+id+"/ship", sellerToken, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	var shipped map[string]any
	require.NoError(t, json.Unmarshal(body, &shipped))
	require.Equal(t, "shipped", shipped["status"])

	resp, body = authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/"+id+"/deliver", buyerToken, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	resp, body = authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/"+id+"/complete", buyerToken, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	var done map[string]any
	require.NoError(t, json.Unmarshal(body, &done))
	require.Equal(t, "completed", done["status"])
}

func TestOrders_BuyerCannotShip(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "bs-seller@example.com")
	buyerToken := registerBuyer(t, ts, "bs-buyer@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	sellerID := getMyID(t, ts, sellerToken)
	productID := createProduct(t, ts, sellerToken, catID, "P", 1000, 5)

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerID,
		"items":            []map[string]any{{"product_id": productID, "quantity": 1}},
		"shipping_address": map[string]any{"city": "Moscow"},
	})
	var created map[string]any
	require.NoError(t, json.Unmarshal(body, &created))
	id := created["id"].(string)

	resp, _ := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/orders/"+id+"/ship", buyerToken, nil)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestOrders_SellerCannotMarkPaid(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "sp-seller@example.com")
	buyerToken := registerBuyer(t, ts, "sp-buyer@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	sellerID := getMyID(t, ts, sellerToken)
	productID := createProduct(t, ts, sellerToken, catID, "P", 1000, 5)

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerID,
		"items":            []map[string]any{{"product_id": productID, "quantity": 1}},
		"shipping_address": map[string]any{"city": "Moscow"},
	})
	var created map[string]any
	require.NoError(t, json.Unmarshal(body, &created))
	id := created["id"].(string)

	resp, _ := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/"+id+"/pay", sellerToken, nil)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestOrders_Get_OtherUserForbidden(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "gf-seller@example.com")
	buyerToken := registerBuyer(t, ts, "gf-buyer@example.com")
	strangerToken := registerBuyer(t, ts, "gf-stranger@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	sellerID := getMyID(t, ts, sellerToken)
	productID := createProduct(t, ts, sellerToken, catID, "P", 1000, 5)

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerID,
		"items":            []map[string]any{{"product_id": productID, "quantity": 1}},
		"shipping_address": map[string]any{"city": "Moscow"},
	})
	var created map[string]any
	require.NoError(t, json.Unmarshal(body, &created))
	id := created["id"].(string)

	resp, _ := authGet(t, ts.url+"/api/v1/orders/"+id, strangerToken)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	resp, _ = authGet(t, ts.url+"/api/v1/orders/"+id, buyerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp, _ = authGet(t, ts.url+"/api/v1/orders/"+id, sellerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestOrders_Cancel(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "c-seller@example.com")
	buyerToken := registerBuyer(t, ts, "c-buyer@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	sellerID := getMyID(t, ts, sellerToken)
	productID := createProduct(t, ts, sellerToken, catID, "P", 1000, 5)

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerID,
		"items":            []map[string]any{{"product_id": productID, "quantity": 1}},
		"shipping_address": map[string]any{"city": "Moscow"},
	})
	var created map[string]any
	require.NoError(t, json.Unmarshal(body, &created))
	id := created["id"].(string)

	resp, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/"+id+"/cancel", buyerToken, map[string]any{
		"reason": "changed my mind",
	})
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var cancelled map[string]any
	require.NoError(t, json.Unmarshal(body, &cancelled))
	require.Equal(t, "cancelled", cancelled["status"])
	require.Equal(t, "changed my mind", cancelled["cancelled_reason"])
}

func TestOrders_ListBuyer(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "lb-seller@example.com")
	buyerToken := registerBuyer(t, ts, "lb-buyer@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	sellerID := getMyID(t, ts, sellerToken)
	productID := createProduct(t, ts, sellerToken, catID, "P", 1000, 100)

	for i := 0; i < 3; i++ {
		_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
			"seller_id":        sellerID,
			"items":            []map[string]any{{"product_id": productID, "quantity": 1}},
			"shipping_address": map[string]any{"city": "Moscow"},
		})
	}

	resp, body := authGet(t, ts.url+"/api/v1/orders", buyerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Len(t, out["items"].([]any), 3)
}

func TestOrders_ListSeller(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "ls-seller@example.com")
	buyerToken := registerBuyer(t, ts, "ls-buyer@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	sellerID := getMyID(t, ts, sellerToken)
	productID := createProduct(t, ts, sellerToken, catID, "P", 1000, 100)

	for i := 0; i < 2; i++ {
		_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
			"seller_id":        sellerID,
			"items":            []map[string]any{{"product_id": productID, "quantity": 1}},
			"shipping_address": map[string]any{"city": "Moscow"},
		})
	}

	resp, body := authGet(t, ts.url+"/api/v1/seller/orders", sellerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Len(t, out["items"].([]any), 2)
}

func TestOrders_ListSeller_FilterByStatus(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "fs-seller@example.com")
	buyerToken := registerBuyer(t, ts, "fs-buyer@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	sellerID := getMyID(t, ts, sellerToken)
	productID := createProduct(t, ts, sellerToken, catID, "P", 1000, 100)

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerID,
		"items":            []map[string]any{{"product_id": productID, "quantity": 1}},
		"shipping_address": map[string]any{"city": "Moscow"},
	})
	var o1 map[string]any
	require.NoError(t, json.Unmarshal(body, &o1))
	id1 := o1["id"].(string)

	_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerID,
		"items":            []map[string]any{{"product_id": productID, "quantity": 1}},
		"shipping_address": map[string]any{"city": "Moscow"},
	})

	_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/"+id1+"/pay", buyerToken, nil)

	resp, body := authGet(t, ts.url+"/api/v1/seller/orders?status=paid", sellerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Len(t, out["items"].([]any), 1)
}

// mustJSON is a tiny helper for building request bodies when using http.NewRequest directly.
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
