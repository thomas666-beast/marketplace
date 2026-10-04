package catalog_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/thomas666-beast/marketplace/internal/catalog"
	"github.com/thomas666-beast/marketplace/internal/users"
)

// helper: create a seller user and return its ID
func createSeller(t *testing.T, ctx context.Context, userRepo *users.Repository, email string) string {
	t.Helper()
	u, err := userRepo.Create(ctx, users.CreateInput{
		Email:           email,
		PasswordHash:    "hash",
		DisplayName:     "Seller",
		Role:            users.RoleSeller,
		PreferredLocale: users.LocaleEN,
	})
	require.NoError(t, err)
	return u.ID
}

// helper: get the "smartphones" category
func smartphonesCategory(t *testing.T, ctx context.Context, repo *catalog.CategoryRepository) catalog.Category {
	t.Helper()
	c, err := repo.GetBySlug(ctx, "smartphones")
	require.NoError(t, err)
	return c
}

func TestProduct_Create(t *testing.T) {
	db, catRepo := setupCatalogTest(t)
	ctx := context.Background()
	prodRepo := catalog.NewProductRepository(db.Pool)
	userRepo := users.NewRepository(db.Pool)

	sellerID := createSeller(t, ctx, userRepo, "seller1@example.com")
	cat := smartphonesCategory(t, ctx, catRepo)

	p, err := prodRepo.Create(ctx, catalog.CreateProductInput{
		SellerID:      sellerID,
		CategoryID:    cat.ID,
		Name:          "iPhone 15 Pro",
		Description:   "The latest flagship",
		PriceCents:    129900,
		Currency:      catalog.CurrencyUSD,
		StockQuantity: 10,
		Status:        catalog.StatusActive,
	})
	require.NoError(t, err)
	require.NotEmpty(t, p.ID)
	require.Equal(t, "iPhone 15 Pro", p.Name)
	require.NotEmpty(t, p.Slug)
	require.Equal(t, int64(129900), p.PriceCents)
	require.Equal(t, catalog.CurrencyUSD, p.Currency)
	require.Equal(t, 10, p.StockQuantity)
	require.Equal(t, catalog.StatusActive, p.Status)
	require.Nil(t, p.DeletedAt)
}

func TestProduct_GetByID_NotFound(t *testing.T) {
	db, _ := setupCatalogTest(t)
	ctx := context.Background()
	prodRepo := catalog.NewProductRepository(db.Pool)

	_, err := prodRepo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, catalog.ErrProductNotFound)
}

func TestProduct_Update_Owner(t *testing.T) {
	db, catRepo := setupCatalogTest(t)
	ctx := context.Background()
	prodRepo := catalog.NewProductRepository(db.Pool)
	userRepo := users.NewRepository(db.Pool)

	sellerID := createSeller(t, ctx, userRepo, "seller2@example.com")
	cat := smartphonesCategory(t, ctx, catRepo)

	p, err := prodRepo.Create(ctx, catalog.CreateProductInput{
		SellerID: sellerID, CategoryID: cat.ID, Name: "Phone",
		PriceCents: 1000, Currency: catalog.CurrencyUSD,
		Status: catalog.StatusDraft,
	})
	require.NoError(t, err)

	newName := "Updated Phone"
	newPrice := int64(2000)
	updated, err := prodRepo.Update(ctx, p.ID, sellerID, catalog.UpdateProductInput{
		Name:       &newName,
		PriceCents: &newPrice,
	})
	require.NoError(t, err)
	require.Equal(t, "Updated Phone", updated.Name)
	require.Equal(t, int64(2000), updated.PriceCents)
}

func TestProduct_Update_NotOwner(t *testing.T) {
	db, catRepo := setupCatalogTest(t)
	ctx := context.Background()
	prodRepo := catalog.NewProductRepository(db.Pool)
	userRepo := users.NewRepository(db.Pool)

	sellerID := createSeller(t, ctx, userRepo, "seller3@example.com")
	otherID := createSeller(t, ctx, userRepo, "other@example.com")
	cat := smartphonesCategory(t, ctx, catRepo)

	p, err := prodRepo.Create(ctx, catalog.CreateProductInput{
		SellerID: sellerID, CategoryID: cat.ID, Name: "Phone",
		PriceCents: 1000, Currency: catalog.CurrencyUSD,
		Status: catalog.StatusDraft,
	})
	require.NoError(t, err)

	newName := "Hijacked"
	_, err = prodRepo.Update(ctx, p.ID, otherID, catalog.UpdateProductInput{Name: &newName})
	require.ErrorIs(t, err, catalog.ErrNotOwner)
}

func TestProduct_Update_NotFound(t *testing.T) {
	db, _ := setupCatalogTest(t)
	ctx := context.Background()
	prodRepo := catalog.NewProductRepository(db.Pool)

	newName := "x"
	_, err := prodRepo.Update(
		ctx,
		"00000000-0000-0000-0000-000000000000", // valid UUID, does not exist
		"00000000-0000-0000-0000-000000000001", // valid UUID, does not exist
		catalog.UpdateProductInput{Name: &newName},
	)
	require.ErrorIs(t, err, catalog.ErrProductNotFound)
}

func TestProduct_SoftDelete_Owner(t *testing.T) {
	db, catRepo := setupCatalogTest(t)
	ctx := context.Background()
	prodRepo := catalog.NewProductRepository(db.Pool)
	userRepo := users.NewRepository(db.Pool)

	sellerID := createSeller(t, ctx, userRepo, "seller4@example.com")
	cat := smartphonesCategory(t, ctx, catRepo)

	p, err := prodRepo.Create(ctx, catalog.CreateProductInput{
		SellerID: sellerID, CategoryID: cat.ID, Name: "Phone",
		PriceCents: 1000, Currency: catalog.CurrencyUSD,
		Status: catalog.StatusActive,
	})
	require.NoError(t, err)

	require.NoError(t, prodRepo.SoftDelete(ctx, p.ID, sellerID))

	_, err = prodRepo.GetByID(ctx, p.ID)
	require.ErrorIs(t, err, catalog.ErrProductNotFound)
}

func TestProduct_SoftDelete_NotOwner(t *testing.T) {
	db, catRepo := setupCatalogTest(t)
	ctx := context.Background()
	prodRepo := catalog.NewProductRepository(db.Pool)
	userRepo := users.NewRepository(db.Pool)

	sellerID := createSeller(t, ctx, userRepo, "seller5@example.com")
	otherID := createSeller(t, ctx, userRepo, "other5@example.com")
	cat := smartphonesCategory(t, ctx, catRepo)

	p, err := prodRepo.Create(ctx, catalog.CreateProductInput{
		SellerID: sellerID, CategoryID: cat.ID, Name: "Phone",
		PriceCents: 1000, Currency: catalog.CurrencyUSD,
		Status: catalog.StatusActive,
	})
	require.NoError(t, err)

	err = prodRepo.SoftDelete(ctx, p.ID, otherID)
	require.ErrorIs(t, err, catalog.ErrNotOwner)
}

func TestProduct_List_ByCategory(t *testing.T) {
	db, catRepo := setupCatalogTest(t)
	ctx := context.Background()
	prodRepo := catalog.NewProductRepository(db.Pool)
	userRepo := users.NewRepository(db.Pool)

	sellerID := createSeller(t, ctx, userRepo, "seller6@example.com")
	cat := smartphonesCategory(t, ctx, catRepo)

	for i := 0; i < 3; i++ {
		_, err := prodRepo.Create(ctx, catalog.CreateProductInput{
			SellerID: sellerID, CategoryID: cat.ID,
			Name: "Phone " + string(rune('A'+i)),
			PriceCents: int64(1000 + i), Currency: catalog.CurrencyUSD,
			Status: catalog.StatusActive,
		})
		require.NoError(t, err)
	}

	page, err := prodRepo.List(ctx, catalog.ProductFilter{CategoryID: &cat.ID})
	require.NoError(t, err)
	require.Len(t, page.Items, 3)
	require.Empty(t, page.NextCursor)
}

func TestProduct_List_ExcludesDraftForBuyer(t *testing.T) {
	db, catRepo := setupCatalogTest(t)
	ctx := context.Background()
	prodRepo := catalog.NewProductRepository(db.Pool)
	userRepo := users.NewRepository(db.Pool)

	sellerID := createSeller(t, ctx, userRepo, "seller7@example.com")
	cat := smartphonesCategory(t, ctx, catRepo)

	_, err := prodRepo.Create(ctx, catalog.CreateProductInput{
		SellerID: sellerID, CategoryID: cat.ID, Name: "Published",
		PriceCents: 1000, Currency: catalog.CurrencyUSD,
		Status: catalog.StatusActive,
	})
	require.NoError(t, err)
	_, err = prodRepo.Create(ctx, catalog.CreateProductInput{
		SellerID: sellerID, CategoryID: cat.ID, Name: "Draft",
		PriceCents: 1000, Currency: catalog.CurrencyUSD,
		Status: catalog.StatusDraft,
	})
	require.NoError(t, err)

	// Buyer-facing (no seller filter): only active.
	page, err := prodRepo.List(ctx, catalog.ProductFilter{CategoryID: &cat.ID})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, "Published", page.Items[0].Name)

	// Seller-facing (seller filter): both.
	page, err = prodRepo.List(ctx, catalog.ProductFilter{SellerID: &sellerID})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
}

func TestProduct_List_Pagination(t *testing.T) {
	db, catRepo := setupCatalogTest(t)
	ctx := context.Background()
	prodRepo := catalog.NewProductRepository(db.Pool)
	userRepo := users.NewRepository(db.Pool)

	sellerID := createSeller(t, ctx, userRepo, "seller8@example.com")
	cat := smartphonesCategory(t, ctx, catRepo)

	for i := 0; i < 5; i++ {
		_, err := prodRepo.Create(ctx, catalog.CreateProductInput{
			SellerID: sellerID, CategoryID: cat.ID,
			Name: "Product " + string(rune('A'+i)),
			PriceCents: int64(1000 + i), Currency: catalog.CurrencyUSD,
			Status: catalog.StatusActive,
		})
		require.NoError(t, err)
	}

	// First page: 2 items, next cursor present.
	page1, err := prodRepo.List(ctx, catalog.ProductFilter{CategoryID: &cat.ID, Limit: 2})
	require.NoError(t, err)
	require.Len(t, page1.Items, 2)
	require.NotEmpty(t, page1.NextCursor)

	// Second page: 2 items.
	page2, err := prodRepo.List(ctx, catalog.ProductFilter{
		CategoryID: &cat.ID, Limit: 2, Cursor: page1.NextCursor,
	})
	require.NoError(t, err)
	require.Len(t, page2.Items, 2)
	require.NotEmpty(t, page2.NextCursor)

	// Third page: 1 item, no next cursor.
	page3, err := prodRepo.List(ctx, catalog.ProductFilter{
		CategoryID: &cat.ID, Limit: 2, Cursor: page2.NextCursor,
	})
	require.NoError(t, err)
	require.Len(t, page3.Items, 1)
	require.Empty(t, page3.NextCursor)

	// All items are distinct.
	seen := map[string]bool{}
	for _, p := range append(append(page1.Items, page2.Items...), page3.Items...) {
		require.False(t, seen[p.ID], "duplicate product across pages: %s", p.ID)
		seen[p.ID] = true
	}
	require.Len(t, seen, 5, "all 5 products should appear exactly once")
}

func TestProduct_List_Search(t *testing.T) {
	db, catRepo := setupCatalogTest(t)
	ctx := context.Background()
	prodRepo := catalog.NewProductRepository(db.Pool)
	userRepo := users.NewRepository(db.Pool)

	sellerID := createSeller(t, ctx, userRepo, "seller9@example.com")
	cat := smartphonesCategory(t, ctx, catRepo)

	_, err := prodRepo.Create(ctx, catalog.CreateProductInput{
		SellerID: sellerID, CategoryID: cat.ID, Name: "iPhone 15 Pro Max",
		Description: "Latest flagship smartphone",
		PriceCents:  129900, Currency: catalog.CurrencyUSD,
		Status: catalog.StatusActive,
	})
	require.NoError(t, err)

	_, err = prodRepo.Create(ctx, catalog.CreateProductInput{
		SellerID: sellerID, CategoryID: cat.ID, Name: "Samsung Galaxy",
		Description: "Android phone",
		PriceCents:  99900, Currency: catalog.CurrencyUSD,
		Status: catalog.StatusActive,
	})
	require.NoError(t, err)

	page, err := prodRepo.List(ctx, catalog.ProductFilter{Search: "iphone"})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Contains(t, page.Items[0].Name, "iPhone")
}

func TestProduct_List_InvalidCursor(t *testing.T) {
	db, _ := setupCatalogTest(t)
	ctx := context.Background()
	prodRepo := catalog.NewProductRepository(db.Pool)

	_, err := prodRepo.List(ctx, catalog.ProductFilter{Cursor: "not-a-valid-cursor!!!"})
	require.Error(t, err)
}
