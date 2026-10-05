package delivery_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/thomas666-beast/marketplace/internal/catalog"
	"github.com/thomas666-beast/marketplace/internal/delivery"
	"github.com/thomas666-beast/marketplace/internal/orders"
	"github.com/thomas666-beast/marketplace/internal/postgres"
	"github.com/thomas666-beast/marketplace/internal/users"
	"github.com/thomas666-beast/marketplace/internal/seller"
)

type deliveryTestEnv struct {
	pool       *postgres.DB
	deliveries *delivery.Repository
	points     *delivery.PickupPointRepository
	orders     *orders.Repository
	users      *users.Repository
	products   *catalog.ProductRepository
	categories *catalog.CategoryRepository
	addresses  *seller.AddressRepository
}

func setupDeliveryRepoTest(t *testing.T) deliveryTestEnv {
	t.Helper()
	db, _ := setupDeliveryTest(t)
	return deliveryTestEnv{
		pool:       db,
		deliveries: delivery.NewRepository(db.Pool),
		points:     delivery.NewPickupPointRepository(db.Pool),
		orders:     orders.NewRepository(db.Pool),
		users:      users.NewRepository(db.Pool),
		products:   catalog.NewProductRepository(db.Pool),
		categories: catalog.NewCategoryRepository(db.Pool),
		addresses:  seller.NewAddressRepository(db.Pool),
	}
}

func makeBuyerAndSeller(t *testing.T, env deliveryTestEnv) (buyerID, sellerID string) {
	t.Helper()
	ctx := context.Background()

	b, err := env.users.Create(ctx, users.CreateInput{
		Email: "buyer@delivery.test", PasswordHash: "hash",
		DisplayName: "Buyer", Role: users.RoleBuyer, PreferredLocale: users.LocaleEN,
	})
	require.NoError(t, err)

	s, err := env.users.Create(ctx, users.CreateInput{
		Email: "seller@delivery.test", PasswordHash: "hash",
		DisplayName: "Seller", Role: users.RoleSeller, PreferredLocale: users.LocaleEN,
	})
	require.NoError(t, err)

	return b.ID, s.ID
}

func makeOrder(t *testing.T, env deliveryTestEnv, buyerID, sellerID string) string {
	t.Helper()
	ctx := context.Background()

	cat, err := env.categories.GetBySlug(ctx, "smartphones")
	require.NoError(t, err)

	p, err := env.products.Create(ctx, catalog.CreateProductInput{
		SellerID:      sellerID,
		CategoryID:    cat.ID,
		Name:          "Delivery Test Product",
		PriceCents:    1000,
		Currency:      catalog.CurrencyUSD,
		StockQuantity: 10,
		Status:        catalog.StatusActive,
	})
	require.NoError(t, err)

	order, err := env.orders.Checkout(ctx, orders.CheckoutInput{
		BuyerID:  buyerID,
		SellerID: sellerID,
		Items:    []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
	})
	require.NoError(t, err)

	return order.ID
}

func makeDelivery(t *testing.T, env deliveryTestEnv, orderID string) delivery.Delivery {
	t.Helper()
	ctx := context.Background()

	point, err := env.points.GetByExternalID(ctx, "MSK-001")
	require.NoError(t, err)

	// Get any seller's address to use as origin. The first seller in the DB.
	var sellerID string
	err = env.pool.Pool.QueryRow(ctx,
		`SELECT id FROM users WHERE role = 'seller' LIMIT 1`).Scan(&sellerID)
	require.NoError(t, err)

	addresses, err := env.addresses.ListBySeller(ctx, sellerID)
	require.NoError(t, err)

	var originAddrID string
	if len(addresses) == 0 {
		// Create one on the fly.
		a, err := env.addresses.Create(ctx, seller.CreateAddressInput{
			SellerID:    sellerID,
			Label:       "Test Warehouse",
			ContactName: "Ivan",
			Phone:       "+79001234567",
			Address:     "1 Warehouse St",
			City:        "Moscow",
			Country:     "RU",
		})
		require.NoError(t, err)
		originAddrID = a.ID
	} else {
		originAddrID = addresses[0].ID
	}

	tn, err := delivery.GenerateTrackingNumber(point.City, time.Now())
	require.NoError(t, err)
	code, err := delivery.GeneratePickupCode()
	require.NoError(t, err)

	d, err := env.deliveries.Create(ctx, delivery.CreateInput{
		OrderID:            orderID,
		PickupPointID:      point.ID,
		OriginAddressID:    &originAddrID,
		DestinationName:    point.Name,
		DestinationAddress: point.Address,
		DestinationCity:    point.City,
		DestinationCountry: point.Country,
		OriginName:         "Test Warehouse",
		OriginAddress:      "1 Warehouse St",
		OriginCity:         "Moscow",
		TrackingNumber:     tn,
		PickupCode:         code,
	})
	require.NoError(t, err)
	return d
}

func TestDelivery_Create(t *testing.T) {
	env := setupDeliveryRepoTest(t)
	buyerID, sellerID := makeBuyerAndSeller(t, env)
	orderID := makeOrder(t, env, buyerID, sellerID)

	d := makeDelivery(t, env, orderID)

	require.NotEmpty(t, d.ID)
	require.Equal(t, orderID, d.OrderID)
	require.NotEmpty(t, d.TrackingNumber)
	require.NotEmpty(t, d.PickupCode)
	require.Equal(t, delivery.StatusCreated, d.Status)
	require.Equal(t, "Arbat Locker", d.DestinationName)
	require.Equal(t, "Moscow", d.DestinationCity)
	require.Nil(t, d.ShippedAt)
	require.Nil(t, d.PickedUpAt)
}

func TestDelivery_Create_DuplicateOrder(t *testing.T) {
	env := setupDeliveryRepoTest(t)
	buyerID, sellerID := makeBuyerAndSeller(t, env)
	orderID := makeOrder(t, env, buyerID, sellerID)

	_ = makeDelivery(t, env, orderID)

	ctx := context.Background()
	point, _ := env.points.GetByExternalID(ctx, "MSK-001")
	addrs, err := env.addresses.ListBySeller(ctx, sellerID)
	require.NoError(t, err)
	require.NotEmpty(t, addrs)

	tn, _ := delivery.GenerateTrackingNumber("Moscow", time.Now())
	code, _ := delivery.GeneratePickupCode()

	_, err = env.deliveries.Create(ctx, delivery.CreateInput{
		OrderID: orderID, PickupPointID: point.ID,
		OriginAddressID:    &addrs[0].ID,
		DestinationName:    point.Name, DestinationAddress: point.Address,
		DestinationCity:    point.City, DestinationCountry: point.Country,
		OriginName: "Test", OriginAddress: "1 St", OriginCity: "Moscow",
		TrackingNumber: tn, PickupCode: code,
	})
	require.ErrorIs(t, err, delivery.ErrDeliveryExists)
}

func TestDelivery_GetByID(t *testing.T) {
	env := setupDeliveryRepoTest(t)
	buyerID, sellerID := makeBuyerAndSeller(t, env)
	orderID := makeOrder(t, env, buyerID, sellerID)
	created := makeDelivery(t, env, orderID)

	ctx := context.Background()
	got, err := env.deliveries.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, delivery.StatusCreated, got.Status)
	require.Len(t, got.Events, 1)
	require.Equal(t, delivery.StatusCreated, got.Events[0].Status)
}

func TestDelivery_GetByID_NotFound(t *testing.T) {
	env := setupDeliveryRepoTest(t)
	ctx := context.Background()

	_, err := env.deliveries.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, delivery.ErrDeliveryNotFound)

	_, err = env.deliveries.GetByID(ctx, "not-a-uuid")
	require.ErrorIs(t, err, delivery.ErrDeliveryNotFound)
}

func TestDelivery_GetByTrackingNumber(t *testing.T) {
	env := setupDeliveryRepoTest(t)
	buyerID, sellerID := makeBuyerAndSeller(t, env)
	orderID := makeOrder(t, env, buyerID, sellerID)
	created := makeDelivery(t, env, orderID)

	ctx := context.Background()
	got, err := env.deliveries.GetByTrackingNumber(ctx, created.TrackingNumber)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
}

func TestDelivery_GetByOrderID(t *testing.T) {
	env := setupDeliveryRepoTest(t)
	buyerID, sellerID := makeBuyerAndSeller(t, env)
	orderID := makeOrder(t, env, buyerID, sellerID)
	created := makeDelivery(t, env, orderID)

	ctx := context.Background()
	got, err := env.deliveries.GetByOrderID(ctx, orderID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
}

func TestDelivery_FullLifecycle(t *testing.T) {
	env := setupDeliveryRepoTest(t)
	buyerID, sellerID := makeBuyerAndSeller(t, env)
	orderID := makeOrder(t, env, buyerID, sellerID)
	d := makeDelivery(t, env, orderID)

	ctx := context.Background()

	d, err := env.deliveries.MarkAwaitingDispatch(ctx, d.ID, "Preparing for dispatch", "Moscow")
	require.NoError(t, err)
	require.Equal(t, delivery.StatusAwaitingDispatch, d.Status)

	d, err = env.deliveries.MarkInTransit(ctx, d.ID, "Picked up by courier", "Moscow")
	require.NoError(t, err)
	require.Equal(t, delivery.StatusInTransit, d.Status)
	require.NotNil(t, d.ShippedAt)

	d, err = env.deliveries.MarkArrivedAtHub(ctx, d.ID, "Arrived at sorting center", "Moscow Hub")
	require.NoError(t, err)
	require.Equal(t, delivery.StatusArrivedAtHub, d.Status)
	require.NotNil(t, d.ArrivedAt)

	d, err = env.deliveries.MarkOutForDelivery(ctx, d.ID, "Out for delivery", "Moscow")
	require.NoError(t, err)
	require.Equal(t, delivery.StatusOutForDelivery, d.Status)

	d, err = env.deliveries.MarkReadyForPickup(ctx, d.ID, "Ready for pickup", "Arbat Locker")
	require.NoError(t, err)
	require.Equal(t, delivery.StatusReadyForPickup, d.Status)
	require.NotNil(t, d.ReadyForPickupAt)

	d, err = env.deliveries.MarkPickedUp(ctx, d.ID, "Picked up by buyer", "Arbat Locker")
	require.NoError(t, err)
	require.Equal(t, delivery.StatusPickedUp, d.Status)
	require.NotNil(t, d.PickedUpAt)
	require.NotNil(t, d.ExpiresAt, "expires_at must be set on pickup")

	withEvents, err := env.deliveries.GetByID(ctx, d.ID)
	require.NoError(t, err)
	require.Len(t, withEvents.Events, 7)

	for i := 1; i < len(withEvents.Events); i++ {
		require.False(t, withEvents.Events[i].OccurredAt.Before(withEvents.Events[i-1].OccurredAt),
			"events must be in chronological order")
	}
}

func TestDelivery_InvalidTransition(t *testing.T) {
	env := setupDeliveryRepoTest(t)
	buyerID, sellerID := makeBuyerAndSeller(t, env)
	orderID := makeOrder(t, env, buyerID, sellerID)
	d := makeDelivery(t, env, orderID)

	ctx := context.Background()

	_, err := env.deliveries.MarkInTransit(ctx, d.ID, "", "")
	require.ErrorIs(t, err, delivery.ErrInvalidTransition)

	_, err = env.deliveries.MarkReadyForPickup(ctx, d.ID, "", "")
	require.ErrorIs(t, err, delivery.ErrInvalidTransition)
}

func TestDelivery_Cancel(t *testing.T) {
	env := setupDeliveryRepoTest(t)
	buyerID, sellerID := makeBuyerAndSeller(t, env)
	orderID := makeOrder(t, env, buyerID, sellerID)
	d := makeDelivery(t, env, orderID)

	ctx := context.Background()

	d, err := env.deliveries.Cancel(ctx, d.ID, "Buyer changed mind", "Moscow")
	require.NoError(t, err)
	require.Equal(t, delivery.StatusCancelled, d.Status)
	require.NotNil(t, d.CancelledAt)
}

func TestDelivery_Cancel_AfterTerminal_Fails(t *testing.T) {
	env := setupDeliveryRepoTest(t)
	buyerID, sellerID := makeBuyerAndSeller(t, env)
	orderID := makeOrder(t, env, buyerID, sellerID)
	d := makeDelivery(t, env, orderID)

	ctx := context.Background()

	_, err := env.deliveries.Cancel(ctx, d.ID, "", "")
	require.NoError(t, err)

	_, err = env.deliveries.Cancel(ctx, d.ID, "", "")
	require.ErrorIs(t, err, delivery.ErrInvalidTransition)
}

func TestDelivery_Returned(t *testing.T) {
	env := setupDeliveryRepoTest(t)
	buyerID, sellerID := makeBuyerAndSeller(t, env)
	orderID := makeOrder(t, env, buyerID, sellerID)
	d := makeDelivery(t, env, orderID)
	ctx := context.Background()

	_, err := env.deliveries.MarkAwaitingDispatch(ctx, d.ID, "", "")
	require.NoError(t, err)
	_, err = env.deliveries.MarkInTransit(ctx, d.ID, "", "")
	require.NoError(t, err)
	_, err = env.deliveries.MarkArrivedAtHub(ctx, d.ID, "", "")
	require.NoError(t, err)
	_, err = env.deliveries.MarkOutForDelivery(ctx, d.ID, "", "")
	require.NoError(t, err)
	_, err = env.deliveries.MarkReadyForPickup(ctx, d.ID, "", "")
	require.NoError(t, err)

	d, err = env.deliveries.MarkReturned(ctx, d.ID, "Not picked up in 7 days", "Arbat Locker")
	require.NoError(t, err)
	require.Equal(t, delivery.StatusReturned, d.Status)
	require.NotNil(t, d.ReturnedAt)
	require.NotNil(t, d.ExpiresAt)
}

func TestDelivery_VerifyPickupCode(t *testing.T) {
	env := setupDeliveryRepoTest(t)
	buyerID, sellerID := makeBuyerAndSeller(t, env)
	orderID := makeOrder(t, env, buyerID, sellerID)
	d := makeDelivery(t, env, orderID)
	ctx := context.Background()

	err := env.deliveries.VerifyPickupCode(ctx, d.ID, d.PickupCode)
	require.ErrorIs(t, err, delivery.ErrDeliveryNotReady)

	_, err = env.deliveries.MarkAwaitingDispatch(ctx, d.ID, "", "")
	require.NoError(t, err)
	_, err = env.deliveries.MarkInTransit(ctx, d.ID, "", "")
	require.NoError(t, err)
	_, err = env.deliveries.MarkArrivedAtHub(ctx, d.ID, "", "")
	require.NoError(t, err)
	_, err = env.deliveries.MarkOutForDelivery(ctx, d.ID, "", "")
	require.NoError(t, err)
	_, err = env.deliveries.MarkReadyForPickup(ctx, d.ID, "", "")
	require.NoError(t, err)

	require.NoError(t, env.deliveries.VerifyPickupCode(ctx, d.ID, d.PickupCode))
	require.ErrorIs(t, env.deliveries.VerifyPickupCode(ctx, d.ID, "WRONG1"), delivery.ErrInvalidPickupCode)
}
