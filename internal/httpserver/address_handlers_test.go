package httpserver_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func sampleAddressBody(label string) map[string]any {
	return map[string]any{
		"label":        label,
		"contact_name": "Ivan Petrov",
		"phone":        "+79001234567",
		"address":      "1 Warehouse Street",
		"city":         "Moscow",
		"region":       "Moscow",
		"postal_code":  "101000",
		"country":      "RU",
	}
}

func TestAddresses_CreateAndList(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerSeller(t, ts, "addr-seller-1@example.com")

	resp, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/addresses", token,
		sampleAddressBody("Main"))
	require.Equal(t, http.StatusCreated, resp.StatusCode, string(body))

	var created map[string]any
	require.NoError(t, json.Unmarshal(body, &created))
	require.NotEmpty(t, created["id"])
	require.Equal(t, true, created["is_default"], "first address is default")

	resp, body = authGet(t, ts.url+"/api/v1/seller/addresses", token)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var list map[string]any
	require.NoError(t, json.Unmarshal(body, &list))
	require.Len(t, list["items"].([]any), 1)
}

func TestAddresses_BuyerForbidden(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerBuyer(t, ts, "addr-buyer@example.com")

	resp, _ := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/addresses", token,
		sampleAddressBody("Nope"))
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestAddresses_RequiresAuth(t *testing.T) {
	ts := setupServerFixture(t)
	resp, _ := authGet(t, ts.url+"/api/v1/seller/addresses", "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAddresses_InvalidInput(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerSeller(t, ts, "addr-seller-2@example.com")

	resp, _ := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/addresses", token,
		map[string]any{"contact_name": "", "phone": "", "address": "", "city": ""})
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestAddresses_Get(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerSeller(t, ts, "addr-seller-3@example.com")

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/addresses", token,
		sampleAddressBody("Main"))
	var created map[string]any
	require.NoError(t, json.Unmarshal(body, &created))
	id := created["id"].(string)

	resp, body := authGet(t, ts.url+"/api/v1/seller/addresses/"+id, token)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var got map[string]any
	require.NoError(t, json.Unmarshal(body, &got))
	require.Equal(t, id, got["id"])
	require.Equal(t, "Main", got["label"])
}

func TestAddresses_Get_NotOwner(t *testing.T) {
	ts := setupServerFixture(t)
	owner := registerSeller(t, ts, "addr-owner@example.com")
	other := registerSeller(t, ts, "addr-other@example.com")

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/addresses", owner,
		sampleAddressBody("Main"))
	var created map[string]any
	require.NoError(t, json.Unmarshal(body, &created))
	id := created["id"].(string)

	resp, _ := authGet(t, ts.url+"/api/v1/seller/addresses/"+id, other)
	require.Equal(t, http.StatusNotFound, resp.StatusCode,
		"cross-tenant access must return 404")
}

func TestAddresses_Update(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerSeller(t, ts, "addr-seller-4@example.com")

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/addresses", token,
		sampleAddressBody("Old"))
	var created map[string]any
	require.NoError(t, json.Unmarshal(body, &created))
	id := created["id"].(string)

	resp, body := authJSON(t, http.MethodPatch, ts.url+"/api/v1/seller/addresses/"+id, token,
		map[string]any{"label": "New", "city": "Kazan"})
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var updated map[string]any
	require.NoError(t, json.Unmarshal(body, &updated))
	require.Equal(t, "New", updated["label"])
	require.Equal(t, "Kazan", updated["city"])
	require.Equal(t, "1 Warehouse Street", updated["address"])
}

func TestAddresses_SetDefault(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerSeller(t, ts, "addr-seller-5@example.com")

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/addresses", token,
		sampleAddressBody("First"))
	var first map[string]any
	require.NoError(t, json.Unmarshal(body, &first))

	_, body = authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/addresses", token,
		sampleAddressBody("Second"))
	var second map[string]any
	require.NoError(t, json.Unmarshal(body, &second))

	require.Equal(t, true, first["is_default"])
	require.Equal(t, false, second["is_default"])

	resp, body := authJSON(t, http.MethodPut,
		ts.url+"/api/v1/seller/addresses/"+second["id"].(string)+"/default", token, nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, string(body))

	var updated map[string]any
	require.NoError(t, json.Unmarshal(body, &updated))
	require.Equal(t, true, updated["is_default"])

	// First should no longer be default.
	resp, body = authGet(t, ts.url+"/api/v1/seller/addresses/"+first["id"].(string), token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var firstRefreshed map[string]any
	require.NoError(t, json.Unmarshal(body, &firstRefreshed))
	require.Equal(t, false, firstRefreshed["is_default"])
}

func TestAddresses_Delete(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerSeller(t, ts, "addr-seller-6@example.com")

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/addresses", token,
		sampleAddressBody("Main"))
	var created map[string]any
	require.NoError(t, json.Unmarshal(body, &created))
	id := created["id"].(string)

	resp, _ := authJSON(t, http.MethodDelete, ts.url+"/api/v1/seller/addresses/"+id, token, nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	resp, _ = authGet(t, ts.url+"/api/v1/seller/addresses/"+id, token)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAddresses_Delete_DefaultPromotes(t *testing.T) {
	ts := setupServerFixture(t)
	token := registerSeller(t, ts, "addr-seller-7@example.com")

	_, body := authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/addresses", token,
		sampleAddressBody("First"))
	var first map[string]any
	require.NoError(t, json.Unmarshal(body, &first))

	_, body = authJSON(t, http.MethodPost, ts.url+"/api/v1/seller/addresses", token,
		sampleAddressBody("Second"))
	var second map[string]any
	require.NoError(t, json.Unmarshal(body, &second))

	// Delete the default (first).
	resp, _ := authJSON(t, http.MethodDelete,
		ts.url+"/api/v1/seller/addresses/"+first["id"].(string), token, nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Second should now be default.
	resp, body = authGet(t, ts.url+"/api/v1/seller/addresses/"+second["id"].(string), token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var refreshed map[string]any
	require.NoError(t, json.Unmarshal(body, &refreshed))
	require.Equal(t, true, refreshed["is_default"])
}
