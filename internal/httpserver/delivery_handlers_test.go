package httpserver_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPickupPoints_ListByCity(t *testing.T) {
	ts := setupServerFixture(t)

	resp, body := authGet(t, ts.url+"/api/v1/pickup-points?city=Moscow", "")
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	items := out["items"].([]any)
	require.Len(t, items, 2)

	first := items[0].(map[string]any)
	require.Equal(t, "MSK-001", first["external_id"])
	require.Equal(t, "Moscow", first["city"])
	require.Equal(t, "locker", first["type"])
}

func TestPickupPoints_ListByCity_MissingParam(t *testing.T) {
	ts := setupServerFixture(t)
	resp, _ := authGet(t, ts.url+"/api/v1/pickup-points", "")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPickupPoints_GetByExternalID(t *testing.T) {
	ts := setupServerFixture(t)

	resp, body := authGet(t, ts.url+"/api/v1/pickup-points/SPB-001", "")
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, "Nevsky Locker", out["name"])
	require.Equal(t, "Saint Petersburg", out["city"])
}

func TestPickupPoints_GetByExternalID_NotFound(t *testing.T) {
	ts := setupServerFixture(t)
	resp, _ := authGet(t, ts.url+"/api/v1/pickup-points/NOPE-999", "")
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestPickupPoints_Nearby(t *testing.T) {
	ts := setupServerFixture(t)

	// Near Arbat Locker
	resp, body := authGet(t, ts.url+"/api/v1/pickup-points/nearby?lat=55.751244&lng=37.593409&radius=3", "")
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	items := out["items"].([]any)
	require.GreaterOrEqual(t, len(items), 2)

	first := items[0].(map[string]any)
	require.Equal(t, "MSK-001", first["external_id"], "closest point should be MSK-001")
}

func TestPickupPoints_Nearby_InvalidRadius(t *testing.T) {
	ts := setupServerFixture(t)
	resp, _ := authGet(t, ts.url+"/api/v1/pickup-points/nearby?lat=55&lng=37&radius=999", "")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestPickupPoints_Nearby_MissingCoords(t *testing.T) {
	ts := setupServerFixture(t)
	resp, _ := authGet(t, ts.url+"/api/v1/pickup-points/nearby", "")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// helper: walk an order through to paid, ready for dispatch.
func makePaidOrder(t *testing.T, ts *serverFixture) (sellerToken, buyerToken, orderID string) {
	t.Helper()
	sellerToken = registerSeller(t, ts, "disp-seller@example.com")
	buyerToken = registerBuyer(t, ts, "disp-buyer@example.com")
	catID := getSmartphoneCategoryID(t, ts)
	sellerID := getMyID(t, ts, sellerToken)
	productID := createProduct(t, ts, sellerToken, catID, "Dispatchable", 1000, 5)

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/checkout", buyerToken, map[string]any{
		"seller_id":        sellerID,
		"items":            []map[string]any{{"product_id": productID, "quantity": 1}},
		"shipping_address": map[string]any{"city": "Moscow"},
	})
	var created map[string]any
	require.NoError(t, json.Unmarshal(body, &created))
	orderID = created["id"].(string)

	_, _ = authJSON(t, http.MethodPost, ts.url+"/api/v1/orders/"+orderID+"/pay", buyerToken, nil)
	return sellerToken, buyerToken, orderID
}

func TestDispatchOrder_Success(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken, _, orderID := makePaidOrder(t, ts)

	resp, body := authJSON(t, http.MethodPost,
		ts.url+"/api/v1/seller/orders/"+orderID+"/dispatch", sellerToken,
		map[string]any{
			"pickup_point_id":   getPickupPointID(t, ts, "MSK-001"),
			"origin_address_id": getSellerAddressID(t, ts, sellerToken),
		})
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, "awaiting_dispatch", out["status"])
	require.NotEmpty(t, out["tracking_number"])
	require.Equal(t, "Moscow", out["destination_city"])
}

func TestDispatchOrder_OrderNotPaid(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken := registerSeller(t, ts, "unpaid-seller@example.com")
	buyerToken := registerBuyer(t, ts, "unpaid-buyer@example.com")
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
	orderID := created["id"].(string)

	// Seller creates an address (so the only failure is unpaid order).
	addressID := getSellerAddressID(t, ts, sellerToken)

	resp, _ := authJSON(t, http.MethodPost,
		ts.url+"/api/v1/seller/orders/"+orderID+"/dispatch", sellerToken,
		map[string]any{
			"pickup_point_id":   getPickupPointID(t, ts, "MSK-001"),
			"origin_address_id": addressID,
		})
	require.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestDispatchOrder_NotOwner(t *testing.T) {
	ts := setupServerFixture(t)
	_, _, orderID := makePaidOrder(t, ts)

	otherSellerToken := registerSeller(t, ts, "other-disp-seller@example.com")
	otherAddressID := getSellerAddressID(t, ts, otherSellerToken)

	resp, _ := authJSON(t, http.MethodPost,
		ts.url+"/api/v1/seller/orders/"+orderID+"/dispatch", otherSellerToken,
		map[string]any{
			"pickup_point_id":   getPickupPointID(t, ts, "MSK-001"),
			"origin_address_id": otherAddressID,
		})
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestDispatchOrder_BuyerForbidden(t *testing.T) {
	ts := setupServerFixture(t)
	_, buyerToken, orderID := makePaidOrder(t, ts)

	resp, _ := authJSON(t, http.MethodPost,
		ts.url+"/api/v1/seller/orders/"+orderID+"/dispatch", buyerToken,
		map[string]any{
			"pickup_point_id":   getPickupPointID(t, ts, "MSK-001"),
			"origin_address_id": "00000000-0000-0000-0000-000000000000",
		})
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestTrackDelivery_Public(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken, _, orderID := makePaidOrder(t, ts)

	_, body := authJSON(t, http.MethodPost,
		ts.url+"/api/v1/seller/orders/"+orderID+"/dispatch", sellerToken,
		map[string]any{
	"pickup_point_id":   getPickupPointID(t, ts, "MSK-001"),
	"origin_address_id": getSellerAddressID(t, ts, sellerToken),
})
	var dispatched map[string]any
	require.NoError(t, json.Unmarshal(body, &dispatched))
	tn := dispatched["tracking_number"].(string)

	// No auth required.
	resp, body := authGet(t, ts.url+"/api/v1/delivery/track/"+tn, "")
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.Equal(t, tn, out["tracking_number"])
	require.Equal(t, "awaiting_dispatch", out["status"])
	require.NotContains(t, out, "pickup_code", "public tracking must not leak pickup code")
}

func TestTrackDelivery_NotFound(t *testing.T) {
	ts := setupServerFixture(t)
	resp, _ := authGet(t, ts.url+"/api/v1/delivery/track/XXX-00000000-000000", "")
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestBuyerGetDelivery_HasPickupCode(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken, buyerToken, orderID := makePaidOrder(t, ts)

	_, _ = authJSON(t, http.MethodPost,
		ts.url+"/api/v1/seller/orders/"+orderID+"/dispatch", sellerToken,
		map[string]any{
	"pickup_point_id":   getPickupPointID(t, ts, "MSK-001"),
	"origin_address_id": getSellerAddressID(t, ts, sellerToken),
})

	resp, body := authGet(t, ts.url+"/api/v1/orders/"+orderID+"/delivery", buyerToken)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	require.NotEmpty(t, out["pickup_code"], "buyer must see pickup code")
}

func TestBuyerGetDelivery_NotBuyer(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken, _, orderID := makePaidOrder(t, ts)
	otherBuyer := registerBuyer(t, ts, "nosy-buyer@example.com")

	_, _ = authJSON(t, http.MethodPost,
		ts.url+"/api/v1/seller/orders/"+orderID+"/dispatch", sellerToken,
		map[string]any{
	"pickup_point_id":   getPickupPointID(t, ts, "MSK-001"),
	"origin_address_id": getSellerAddressID(t, ts, sellerToken),
})

	resp, _ := authGet(t, ts.url+"/api/v1/orders/"+orderID+"/delivery", otherBuyer)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAdminTransitions(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken, _, orderID := makePaidOrder(t, ts)

	// Dispatch first.
	_, body := authJSON(t, http.MethodPost,
		ts.url+"/api/v1/seller/orders/"+orderID+"/dispatch", sellerToken,
		map[string]any{
	"pickup_point_id":   getPickupPointID(t, ts, "MSK-001"),
	"origin_address_id": getSellerAddressID(t, ts, sellerToken),
})
	var dispatched map[string]any
	require.NoError(t, json.Unmarshal(body, &dispatched))
	deliveryID := dispatched["id"].(string)

	// We need an admin token. Register a seller, then flip role in DB.
	adminToken := registerSeller(t, ts, "admin-user@example.com")
	makeAdmin(t, ts, adminToken)

	steps := []struct {
		action string
		status string
	}{
		{"in_transit", "in_transit"},
		{"arrived_at_hub", "arrived_at_hub"},
		{"out_for_delivery", "out_for_delivery"},
		{"ready_for_pickup", "ready_for_pickup"},
		{"picked_up", "picked_up"},
	}

	for _, step := range steps {
		resp, body := authJSON(t, http.MethodPost,
			ts.url+"/api/v1/admin/deliveries/"+deliveryID+"/"+step.action, adminToken,
			map[string]any{"message": "step", "location": "hub"})
		require.Equal(t, http.StatusOK, resp.StatusCode,
			"action %s failed: %s", step.action, string(body))

		var out map[string]any
		require.NoError(t, json.Unmarshal(body, &out))
		require.Equal(t, step.status, out["status"])
	}
}

func TestAdminTransitions_NonAdminForbidden(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken, _, orderID := makePaidOrder(t, ts)

	_, body := authJSON(t, http.MethodPost,
		ts.url+"/api/v1/seller/orders/"+orderID+"/dispatch", sellerToken,
		map[string]any{
	"pickup_point_id":   getPickupPointID(t, ts, "MSK-001"),
	"origin_address_id": getSellerAddressID(t, ts, sellerToken),
})
	var dispatched map[string]any
	require.NoError(t, json.Unmarshal(body, &dispatched))
	deliveryID := dispatched["id"].(string)

	resp, _ := authJSON(t, http.MethodPost,
		ts.url+"/api/v1/admin/deliveries/"+deliveryID+"/in_transit", sellerToken,
		map[string]any{})
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func getPickupPointID(t *testing.T, ts *serverFixture, externalID string) string {
	t.Helper()
	resp, body := authGet(t, ts.url+"/api/v1/pickup-points/"+externalID, "")
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	return out["id"].(string)
}

func makeAdmin(t *testing.T, ts *serverFixture, token string) {
	t.Helper()
	resp, body := authGet(t, ts.url+"/api/v1/users/me", token)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var me map[string]any
	require.NoError(t, json.Unmarshal(body, &me))
	userID := me["id"].(string)

	require.NoError(t, ts.db.Exec(
		context.Background(),
		"UPDATE users SET role = 'admin' WHERE id = $1",
		userID,
	))
}

// getSellerAddressID creates an address for the seller and returns its ID.
func getSellerAddressID(t *testing.T, ts *serverFixture, sellerToken string) string {
	t.Helper()
	resp, body := authJSON(t, http.MethodPost,
		ts.url+"/api/v1/seller/addresses", sellerToken,
		map[string]any{
			"label":        "Main Warehouse",
			"contact_name": "Ivan Petrov",
			"phone":        "+79001234567",
			"address":      "1 Warehouse Street",
			"city":         "Moscow",
			"region":       "Moscow",
			"postal_code":  "101000",
			"country":      "RU",
		})
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(body))

	var out map[string]any
	require.NoError(t, json.Unmarshal(body, &out))
	return out["id"].(string)
}

func TestDispatchOrder_MissingOriginAddress(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken, _, orderID := makePaidOrder(t, ts)

	resp, _ := authJSON(t, http.MethodPost,
		ts.url+"/api/v1/seller/orders/"+orderID+"/dispatch", sellerToken,
		map[string]any{"pickup_point_id": getPickupPointID(t, ts, "MSK-001")})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDispatchOrder_InvalidOriginAddress(t *testing.T) {
	ts := setupServerFixture(t)
	sellerToken, _, orderID := makePaidOrder(t, ts)

	resp, _ := authJSON(t, http.MethodPost,
		ts.url+"/api/v1/seller/orders/"+orderID+"/dispatch", sellerToken,
		map[string]any{
			"pickup_point_id":   getPickupPointID(t, ts, "MSK-001"),
			"origin_address_id": "00000000-0000-0000-0000-000000000000",
		})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
