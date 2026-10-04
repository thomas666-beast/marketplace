package orders_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/thomas666-beast/marketplace/internal/catalog"
	"github.com/thomas666-beast/marketplace/internal/orders"
	"github.com/thomas666-beast/marketplace/internal/postgres"
	"github.com/thomas666-beast/marketplace/internal/users"
)

func setupOrdersTest(t *testing.T) (
	*orders.Repository,
	*catalog.ProductRepository,
	*users.Repository,
	func(email string) string,
	func(sellerID, name string, price int64, stock int) catalog.Product,
) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req, Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)
	dsn := "postgres://test:test@" + host + ":" + port.Port() + "/test?sslmode=disable"

	require.NoError(t, postgres.RunMigrations(dsn))
	db, err := postgres.Connect(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(db.Close)

	orderRepo := orders.NewRepository(db.Pool)
	prodRepo := catalog.NewProductRepository(db.Pool)
	catRepo := catalog.NewCategoryRepository(db.Pool)
	userRepo := users.NewRepository(db.Pool)

	cat, err := catRepo.GetBySlug(ctx, "smartphones")
	require.NoError(t, err)

	mkUser := func(email string) string {
		u, err := userRepo.Create(ctx, users.CreateInput{
			Email: email, PasswordHash: "hash",
			DisplayName: "U", Role: users.RoleBuyer,
			PreferredLocale: users.LocaleEN,
		})
		require.NoError(t, err)
		return u.ID
	}

	mkProduct := func(sellerID, name string, price int64, stock int) catalog.Product {
		p, err := prodRepo.Create(ctx, catalog.CreateProductInput{
			SellerID: sellerID, CategoryID: cat.ID,
			Name: name, PriceCents: price, Currency: catalog.CurrencyUSD,
			StockQuantity: stock, Status: catalog.StatusActive,
		})
		require.NoError(t, err)
		return p
	}

	return orderRepo, prodRepo, userRepo, mkUser, mkProduct
}

func TestCheckout_Success(t *testing.T) {
	repo, prodRepo, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("seller@example.com")
	buyerID := mkUser("buyer@example.com")
	p := mkProduct(sellerID, "iPhone", 1000, 5)

	order, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID:       buyerID,
		SellerID:      sellerID,
		Items:         []orders.CheckoutItem{{ProductID: p.ID, Quantity: 2}},
		ShippingCents: 500,
	})
	require.NoError(t, err)
	require.NotEmpty(t, order.ID)
	require.NotEmpty(t, order.OrderNumber)
	require.Equal(t, orders.StatusPendingPayment, order.Status)
	require.Equal(t, int64(2000), order.SubtotalCents)
	require.Equal(t, int64(500), order.ShippingCents)
	require.Equal(t, int64(2500), order.TotalCents)
	require.Equal(t, orders.CurrencyUSD, order.Currency)
	require.Len(t, order.Items, 1)
	require.Equal(t, "iPhone", order.Items[0].ProductName)
	require.Equal(t, int64(1000), order.Items[0].UnitPriceCents)
	require.Equal(t, 2, order.Items[0].Quantity)
	require.Equal(t, int64(2000), order.Items[0].LineTotalCents)

	// Stock decremented.
	updated, err := prodRepo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, 3, updated.StockQuantity)
}

func TestCheckout_InsufficientStock(t *testing.T) {
	repo, prodRepo, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("seller@example.com")
	buyerID := mkUser("buyer@example.com")
	p := mkProduct(sellerID, "Only One", 1000, 1)

	_, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID:  buyerID,
		SellerID: sellerID,
		Items:    []orders.CheckoutItem{{ProductID: p.ID, Quantity: 5}},
	})
	require.ErrorIs(t, err, orders.ErrInsufficientStock)

	// Stock unchanged.
	updated, err := prodRepo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, 1, updated.StockQuantity)
}

func TestCheckout_ProductNotFound(t *testing.T) {
	repo, _, _, mkUser, _ := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("seller@example.com")
	buyerID := mkUser("buyer@example.com")

	_, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID:  buyerID,
		SellerID: sellerID,
		Items:    []orders.CheckoutItem{{ProductID: "00000000-0000-0000-0000-000000000000", Quantity: 1}},
	})
	require.ErrorIs(t, err, orders.ErrProductNotFound)
}

func TestCheckout_WrongSeller(t *testing.T) {
	repo, _, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	seller1 := mkUser("s1@example.com")
	seller2 := mkUser("s2@example.com")
	buyerID := mkUser("buyer@example.com")
	p := mkProduct(seller1, "P", 1000, 5)

	_, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID:  buyerID,
		SellerID: seller2, // wrong seller
		Items:    []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
	})
	require.ErrorIs(t, err, orders.ErrProductNotFound)
}

func TestCheckout_EmptyOrder(t *testing.T) {
	repo, _, _, mkUser, _ := setupOrdersTest(t)
	ctx := context.Background()

	_, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID: mkUser("b@example.com"), SellerID: mkUser("s@example.com"),
	})
	require.ErrorIs(t, err, orders.ErrEmptyOrder)
}

func TestCheckout_MultipleItems(t *testing.T) {
	repo, _, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("seller@example.com")
	buyerID := mkUser("buyer@example.com")
	p1 := mkProduct(sellerID, "Phone", 1000, 10)
	p2 := mkProduct(sellerID, "Case", 200, 10)

	order, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID:  buyerID,
		SellerID: sellerID,
		Items: []orders.CheckoutItem{
			{ProductID: p1.ID, Quantity: 1},
			{ProductID: p2.ID, Quantity: 3},
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1600), order.SubtotalCents)
	require.Len(t, order.Items, 2)
}

func TestCheckout_InvalidQuantity(t *testing.T) {
	repo, _, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("s@example.com")
	buyerID := mkUser("b@example.com")
	p := mkProduct(sellerID, "P", 1000, 5)

	_, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID:  buyerID,
		SellerID: sellerID,
		Items:    []orders.CheckoutItem{{ProductID: p.ID, Quantity: 0}},
	})
	require.ErrorIs(t, err, orders.ErrInvalidQuantity)
}

func TestOrder_GetByID(t *testing.T) {
	repo, _, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("s@example.com")
	buyerID := mkUser("b@example.com")
	p := mkProduct(sellerID, "P", 1000, 5)

	created, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID: buyerID, SellerID: sellerID,
		Items: []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
	})
	require.NoError(t, err)

	fetched, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, fetched.ID)
	require.Len(t, fetched.Items, 1)
}

func TestOrder_GetByID_NotFound(t *testing.T) {
	repo, _, _, _, _ := setupOrdersTest(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, orders.ErrOrderNotFound)
}

func TestOrder_Lifecycle(t *testing.T) {
	repo, _, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("s@example.com")
	buyerID := mkUser("b@example.com")
	p := mkProduct(sellerID, "P", 1000, 5)

	created, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID: buyerID, SellerID: sellerID,
		Items: []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
	})
	require.NoError(t, err)

	paid, err := repo.MarkPaid(ctx, created.ID, buyerID)
	require.NoError(t, err)
	require.Equal(t, orders.StatusPaid, paid.Status)
	require.NotNil(t, paid.PaidAt)

	shipped, err := repo.MarkShipped(ctx, created.ID, sellerID)
	require.NoError(t, err)
	require.Equal(t, orders.StatusShipped, shipped.Status)
	require.NotNil(t, shipped.ShippedAt)

	delivered, err := repo.MarkDelivered(ctx, created.ID, buyerID)
	require.NoError(t, err)
	require.Equal(t, orders.StatusDelivered, delivered.Status)

	completed, err := repo.Complete(ctx, created.ID, buyerID)
	require.NoError(t, err)
	require.Equal(t, orders.StatusCompleted, completed.Status)
}

func TestOrder_InvalidTransition(t *testing.T) {
	repo, _, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("s@example.com")
	buyerID := mkUser("b@example.com")
	p := mkProduct(sellerID, "P", 1000, 5)

	created, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID: buyerID, SellerID: sellerID,
		Items: []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
	})
	require.NoError(t, err)

	// Try to ship before paid.
	_, err = repo.MarkShipped(ctx, created.ID, sellerID)
	require.ErrorIs(t, err, orders.ErrInvalidTransition)
}

func TestOrder_WrongActor(t *testing.T) {
	repo, _, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("s@example.com")
	buyerID := mkUser("b@example.com")
	otherID := mkUser("other@example.com")
	p := mkProduct(sellerID, "P", 1000, 5)

	created, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID: buyerID, SellerID: sellerID,
		Items: []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
	})
	require.NoError(t, err)

	// Other user cannot mark as paid.
	_, err = repo.MarkPaid(ctx, created.ID, otherID)
	require.ErrorIs(t, err, orders.ErrOrderNotFound, "must not leak order existence")
}

func TestOrder_Cancel(t *testing.T) {
	repo, _, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("s@example.com")
	buyerID := mkUser("b@example.com")
	p := mkProduct(sellerID, "P", 1000, 5)

	created, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID: buyerID, SellerID: sellerID,
		Items: []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
	})
	require.NoError(t, err)

	cancelled, err := repo.Cancel(ctx, created.ID, buyerID, "changed my mind")
	require.NoError(t, err)
	require.Equal(t, orders.StatusCancelled, cancelled.Status)
	require.NotNil(t, cancelled.CancelledReason)
	require.Equal(t, "changed my mind", *cancelled.CancelledReason)
	require.NotNil(t, cancelled.CancelledAt)
}

func TestOrder_Cancel_AfterShipping_Fails(t *testing.T) {
	repo, _, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("s@example.com")
	buyerID := mkUser("b@example.com")
	p := mkProduct(sellerID, "P", 1000, 5)

	created, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID: buyerID, SellerID: sellerID,
		Items: []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
	})
	require.NoError(t, err)

	_, err = repo.MarkPaid(ctx, created.ID, buyerID)
	require.NoError(t, err)
	_, err = repo.MarkShipped(ctx, created.ID, sellerID)
	require.NoError(t, err)

	_, err = repo.Cancel(ctx, created.ID, buyerID, "too late")
	require.ErrorIs(t, err, orders.ErrInvalidTransition)
}

func TestOrder_List_ByBuyer(t *testing.T) {
	repo, _, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("s@example.com")
	buyer1 := mkUser("b1@example.com")
	buyer2 := mkUser("b2@example.com")
	p := mkProduct(sellerID, "P", 1000, 100)

	for i := 0; i < 3; i++ {
		_, err := repo.Checkout(ctx, orders.CheckoutInput{
			BuyerID: buyer1, SellerID: sellerID,
			Items: []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
		})
		require.NoError(t, err)
	}
	_, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID: buyer2, SellerID: sellerID,
		Items: []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
	})
	require.NoError(t, err)

	page, err := repo.List(ctx, orders.OrderFilter{BuyerID: &buyer1})
	require.NoError(t, err)
	require.Len(t, page.Items, 3)

	page, err = repo.List(ctx, orders.OrderFilter{BuyerID: &buyer2})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
}

func TestOrder_List_BySellerStatus(t *testing.T) {
	repo, _, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("s@example.com")
	buyerID := mkUser("b@example.com")
	p := mkProduct(sellerID, "P", 1000, 100)

	o1, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID: buyerID, SellerID: sellerID,
		Items: []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
	})
	require.NoError(t, err)
	_, err = repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID: buyerID, SellerID: sellerID,
		Items: []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
	})
	require.NoError(t, err)

	_, err = repo.MarkPaid(ctx, o1.ID, buyerID)
	require.NoError(t, err)

	paid := orders.StatusPaid
	page, err := repo.List(ctx, orders.OrderFilter{SellerID: &sellerID, Status: &paid})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, orders.StatusPaid, page.Items[0].Status)
}

func TestOrder_List_Pagination(t *testing.T) {
	repo, _, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("s@example.com")
	buyerID := mkUser("b@example.com")
	p := mkProduct(sellerID, "P", 1000, 100)

	for i := 0; i < 5; i++ {
		_, err := repo.Checkout(ctx, orders.CheckoutInput{
			BuyerID: buyerID, SellerID: sellerID,
			Items: []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
		})
		require.NoError(t, err)
	}

	page1, err := repo.List(ctx, orders.OrderFilter{BuyerID: &buyerID, Limit: 2})
	require.NoError(t, err)
	require.Len(t, page1.Items, 2)
	require.NotEmpty(t, page1.NextCursor)

	page2, err := repo.List(ctx, orders.OrderFilter{
		BuyerID: &buyerID, Limit: 2, Cursor: page1.NextCursor,
	})
	require.NoError(t, err)
	require.Len(t, page2.Items, 2)

	page3, err := repo.List(ctx, orders.OrderFilter{
		BuyerID: &buyerID, Limit: 2, Cursor: page2.NextCursor,
	})
	require.NoError(t, err)
	require.Len(t, page3.Items, 1)
	require.Empty(t, page3.NextCursor)
}

func TestCheckout_ConcurrentStock(t *testing.T) {
	repo, prodRepo, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("s@example.com")
	buyer1 := mkUser("b1@example.com")
	buyer2 := mkUser("b2@example.com")
	p := mkProduct(sellerID, "Last One", 1000, 1)

	// Two buyers try to buy the last item concurrently.
	type result struct {
		order orders.OrderWithItems
		err   error
	}
	results := make(chan result, 2)

	go func() {
		o, err := repo.Checkout(ctx, orders.CheckoutInput{
			BuyerID: buyer1, SellerID: sellerID,
			Items: []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
		})
		results <- result{o, err}
	}()
	go func() {
		o, err := repo.Checkout(ctx, orders.CheckoutInput{
			BuyerID: buyer2, SellerID: sellerID,
			Items: []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
		})
		results <- result{o, err}
	}()

	r1 := <-results
	r2 := <-results

	successCount := 0
	if r1.err == nil {
		successCount++
	} else {
		require.ErrorIs(t, r1.err, orders.ErrInsufficientStock)
	}
	if r2.err == nil {
		successCount++
	} else {
		require.ErrorIs(t, r2.err, orders.ErrInsufficientStock)
	}
	require.Equal(t, 1, successCount, "exactly one concurrent checkout should succeed")

	// Stock should be 0.
	updated, err := prodRepo.GetByID(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, 0, updated.StockQuantity)
}

func TestCheckout_ShippingAddressPreserved(t *testing.T) {
	repo, _, _, mkUser, mkProduct := setupOrdersTest(t)
	ctx := context.Background()

	sellerID := mkUser("s@example.com")
	buyerID := mkUser("b@example.com")
	p := mkProduct(sellerID, "P", 1000, 5)

	addr := json.RawMessage(`{"line1":"123 Main St","city":"Moscow","country":"RU"}`)
	order, err := repo.Checkout(ctx, orders.CheckoutInput{
		BuyerID: buyerID, SellerID: sellerID,
		Items:           []orders.CheckoutItem{{ProductID: p.ID, Quantity: 1}},
		ShippingAddress: addr,
	})
	require.NoError(t, err)

	var got map[string]string
	require.NoError(t, json.Unmarshal(order.ShippingAddress, &got))
	require.Equal(t, "Moscow", got["city"])
	require.Equal(t, "RU", got["country"])
}

var _ = time.Second // silence unused import in some editors
