package httpserver_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDashboard_Empty(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "dash-empty@example.com")

	resp, body := authGet(t, ts.url+"/api/v1/seller/orders/dashboard", sellerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))

	counts := out["counts"].(map[string]any)
	require.Equal(t, float64(0), counts["awaiting_payment"])
	require.Equal(t, float64(0), counts["ready_to_dispatch"])
	require.Equal(t, float64(0), counts["in_delivery"])
	require.Equal(t, float64(0), counts["completed"])
	require.Equal(t, float64(0), counts["cancelled"])

	require.Len(t, out["items"].([]any), 0)
}

func TestDashboard_CountsAcrossStates(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "dash-states@example.com")
	buyerToken := registerBuyer(t, ts, "dash-buyer@example.com")
	catID := getSmartphoneCategoryID(t, ts)
	sellerID := getMyID(t, ts, sellerToken)
	productID := createProduct(t, ts, sellerToken, catID, "D", 1000, 100)

	// Order 1: pending payment.
	_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerID,
		"items":            []map[string]any{{"product_id": productID, "quantity": 1}},
		"shipping_address": map[string]any{"city": "Moscow"},
	})

	// Order 2: paid, not dispatched → ready to dispatch.
	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerID,
		"items":            []map[string]any{{"product_id": productID, "quantity": 1}},
		"shipping_address": map[string]any{"city": "Moscow"},
	})
	var o2 map[string]any
	require.NoError(t, json.Unmarshal(body, &o2))
	_, _ = authJSON(t, http.MethodPost,
		ts.url+"/api/v1/orders/"+o2["id"].(string)+"/pay", buyerToken, nil)

	// Order 3: dispatched → in delivery.
	_, body = authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerID,
		"items":            []map[string]any{{"product_id": productID, "quantity": 1}},
		"shipping_address": map[string]any{"city": "Moscow"},
	})
	var o3 map[string]any
	require.NoError(t, json.Unmarshal(body, &o3))
	order3ID := o3["id"].(string)
	_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/"+order3ID+"/pay", buyerToken, nil)

	addressID := getSellerAddressID(t, ts, sellerToken)
	_, _ = authJSON(t, http.MethodPost,
		ts.url+"/api/v1/seller/orders/"+order3ID+"/dispatch", sellerToken,
		map[string]any{
			"pickup_point_id":   getPickupPointID(t, ts, "MSK-001"),
			"origin_address_id": addressID,
		})

	resp, body := authGet(t, ts.url+"/api/v1/seller/orders/dashboard", sellerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	counts := out["counts"].(map[string]any)

	require.Equal(t, float64(1), counts["awaiting_payment"])
	require.Equal(t, float64(1), counts["ready_to_dispatch"])
	require.Equal(t, float64(1), counts["in_delivery"])
	require.Equal(t, float64(0), counts["completed"])
	require.Equal(t, float64(0), counts["cancelled"])
}

func TestDashboard_FilterReadyToDispatch(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken, _, orderID := makePaidOrder(t, ts)

	resp, body := authGet(t,
		ts.url+"/api/v1/seller/orders/dashboard?filter=ready_to_dispatch", sellerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	items := out["items"].([]any)
	require.Len(t, items, 1)

	first := items[0].(map[string]any)
	order := first["order"].(map[string]any)
	require.Equal(t, orderID, order["id"])
	require.NotContains(t, first, "delivery", "no delivery yet")
}

func TestDashboard_DeliveryIncludedAfterDispatch(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken, _, orderID := makePaidOrder(t, ts)

	addressID := getSellerAddressID(t, ts, sellerToken)
	_, _ = authJSON(t, http.MethodPost,
		ts.url+"/api/v1/seller/orders/"+orderID+"/dispatch", sellerToken,
		map[string]any{
			"pickup_point_id":   getPickupPointID(t, ts, "MSK-001"),
			"origin_address_id": addressID,
		})

	resp, body := authGet(t, ts.url+"/api/v1/seller/orders/dashboard", sellerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	items := out["items"].([]any)
	require.Len(t, items, 1)

	first := items[0].(map[string]any)
	require.Contains(t, first, "delivery")

	d := first["delivery"].(map[string]any)
	require.NotEmpty(t, d["id"])
	require.NotEmpty(t, d["tracking_number"])
	require.Equal(t, "awaiting_dispatch", d["status"])
	require.NotEmpty(t, d["pickup_code"], "seller must see pickup code")
}

func TestDashboard_InvalidFilter(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "dash-invalid@example.com")

	resp, _ := authGet(t, ts.url+"/api/v1/seller/orders/dashboard?filter=bogus", sellerToken)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDashboard_BuyerForbidden(t *testing.T) {
	ts := setupServerFixture(t)
	buyerToken := registerBuyer(t, ts, "dash-nope@example.com")

	resp, _ := authGet(t, ts.url+"/api/v1/seller/orders/dashboard", buyerToken)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestDashboard_RequiresAuth(t *testing.T) {
	ts := setupServerFixture(t)
	resp, _ := authGet(t, ts.url+"/api/v1/seller/orders/dashboard", "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestDashboard_IsolatedPerSeller(t *testing.T) {
	ts := setupServerFixture(t)
	sellerA := registerSeller(t, ts, "dash-seller-a@example.com")
	sellerB := registerSeller(t, ts, "dash-seller-b@example.com")
	buyerToken := registerBuyer(t, ts, "dash-isolated-buyer@example.com")
	catID := getSmartphoneCategoryID(t, ts)

	sellerAID := getMyID(t, ts, sellerA)
	productA := createProduct(t, ts, sellerA, catID, "A", 1000, 10)

	// Order from seller A only.
	_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerAID,
		"items":            []map[string]any{{"product_id": productA, "quantity": 1}},
		"shipping_address": map[string]any{"city": "Moscow"},
	})

	// Seller B sees nothing.
	resp, body := authGet(t, ts.url+"/api/v1/seller/orders/dashboard", sellerB)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Len(t, out["items"].([]any), 0)
	counts := out["counts"].(map[string]any)
	require.Equal(t, float64(0), counts["awaiting_payment"])
	require.Equal(t, float64(0), counts["ready_to_dispatch"])

	// Seller A sees the order.
	resp, body = authGet(t, ts.url+"/api/v1/seller/orders/dashboard", sellerA)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var outA map[string]any
	require.NoError(t, json.Unmarshal(body, &outA))
	require.Len(t, outA["items"].([]any), 1)
}
