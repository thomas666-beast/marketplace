package seller_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/thomas666-beast/marketplace/internal/postgres"
	"github.com/thomas666-beast/marketplace/internal/seller"
	"github.com/thomas666-beast/marketplace/internal/users"
)

func setupSellerTest(t *testing.T) (*postgres.DB, *seller.AddressRepository, *users.Repository) {
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

	return db, seller.NewAddressRepository(db.Pool), users.NewRepository(db.Pool)
}

func makeSeller(t *testing.T, ctx context.Context, repo *users.Repository, email string) string {
	t.Helper()
	u, err := repo.Create(ctx, users.CreateInput{
		Email: email, PasswordHash: "hash",
		DisplayName: "Seller", Role: users.RoleSeller,
		PreferredLocale: users.LocaleEN,
	})
	require.NoError(t, err)
	return u.ID
}

func sampleAddress(sellerID, label string) seller.CreateAddressInput {
	return seller.CreateAddressInput{
		SellerID:    sellerID,
		Label:       label,
		ContactName: "Ivan Petrov",
		Phone:       "+79001234567",
		Address:     "1 Warehouse Street",
		City:        "Moscow",
		Region:      "Moscow",
		PostalCode:  "101000",
		Country:     "RU",
	}
}

func TestAddress_Create_FirstIsDefault(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerID := makeSeller(t, ctx, userRepo, "s1@example.com")

	a, err := repo.Create(ctx, sampleAddress(sellerID, "Main"))
	require.NoError(t, err)
	require.NotEmpty(t, a.ID)
	require.True(t, a.IsDefault, "first address must become default automatically")
	require.Equal(t, "Main", a.Label)
	require.Equal(t, "Moscow", a.City)
	require.Equal(t, "RU", a.Country)
}

func TestAddress_Create_SecondIsNotDefault(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerID := makeSeller(t, ctx, userRepo, "s2@example.com")

	_, err := repo.Create(ctx, sampleAddress(sellerID, "First"))
	require.NoError(t, err)

	second, err := repo.Create(ctx, sampleAddress(sellerID, "Second"))
	require.NoError(t, err)
	require.False(t, second.IsDefault, "second address must not automatically become default")
}

func TestAddress_Create_ExplicitDefault(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerID := makeSeller(t, ctx, userRepo, "s3@example.com")

	first, err := repo.Create(ctx, sampleAddress(sellerID, "First"))
	require.NoError(t, err)
	require.True(t, first.IsDefault)

	// Second, explicitly default.
	in := sampleAddress(sellerID, "Second")
	in.IsDefault = true
	second, err := repo.Create(ctx, in)
	require.NoError(t, err)
	require.True(t, second.IsDefault)

	// First should no longer be default.
	firstRefreshed, err := repo.GetByID(ctx, first.ID, sellerID)
	require.NoError(t, err)
	require.False(t, firstRefreshed.IsDefault, "old default must be cleared")

	// Only one default exists.
	def, err := repo.GetDefault(ctx, sellerID)
	require.NoError(t, err)
	require.Equal(t, second.ID, def.ID)
}

func TestAddress_Create_InvalidInput(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerID := makeSeller(t, ctx, userRepo, "s4@example.com")

	cases := []seller.CreateAddressInput{
		{SellerID: sellerID, ContactName: "", Phone: "+79001234567", Address: "1 St", City: "Moscow"},
		{SellerID: sellerID, ContactName: "Ivan", Phone: "", Address: "1 St", City: "Moscow"},
		{SellerID: sellerID, ContactName: "Ivan", Phone: "+79001234567", Address: "", City: "Moscow"},
		{SellerID: sellerID, ContactName: "Ivan", Phone: "+79001234567", Address: "1 St", City: ""},
	}
	for i, c := range cases {
		_, err := repo.Create(ctx, c)
		require.ErrorIs(t, err, seller.ErrInvalidInput, "case %d must be rejected", i)
	}
}

func TestAddress_ListBySeller(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerID := makeSeller(t, ctx, userRepo, "s5@example.com")

	for i := 0; i < 3; i++ {
		_, err := repo.Create(ctx, sampleAddress(sellerID, "Addr"+string(rune('A'+i))))
		require.NoError(t, err)
	}

	list, err := repo.ListBySeller(ctx, sellerID)
	require.NoError(t, err)
	require.Len(t, list, 3)
	// Default first, then created_at DESC.
	require.True(t, list[0].IsDefault, "default must be first")
}

func TestAddress_ListBySeller_Empty(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerID := makeSeller(t, ctx, userRepo, "s6@example.com")

	list, err := repo.ListBySeller(ctx, sellerID)
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestAddress_GetByID_NotOwner(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerA := makeSeller(t, ctx, userRepo, "sa@example.com")
	sellerB := makeSeller(t, ctx, userRepo, "sb@example.com")

	a, err := repo.Create(ctx, sampleAddress(sellerA, "A"))
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, a.ID, sellerB)
	require.ErrorIs(t, err, seller.ErrAddressNotFound,
		"cross-tenant access must return not-found")
}

func TestAddress_GetByID_NotFound(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerID := makeSeller(t, ctx, userRepo, "s7@example.com")

	_, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000", sellerID)
	require.ErrorIs(t, err, seller.ErrAddressNotFound)

	_, err = repo.GetByID(ctx, "not-a-uuid", sellerID)
	require.ErrorIs(t, err, seller.ErrAddressNotFound)
}

func TestAddress_Update(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerID := makeSeller(t, ctx, userRepo, "s8@example.com")

	a, err := repo.Create(ctx, sampleAddress(sellerID, "Old"))
	require.NoError(t, err)

	newLabel := "New"
	newCity := "Kazan"
	updated, err := repo.Update(ctx, a.ID, sellerID, seller.UpdateAddressInput{
		Label: &newLabel,
		City:  &newCity,
	})
	require.NoError(t, err)
	require.Equal(t, "New", updated.Label)
	require.Equal(t, "Kazan", updated.City)
	require.Equal(t, "1 Warehouse Street", updated.Address, "unchanged fields preserved")
}

func TestAddress_Update_NotOwner(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerA := makeSeller(t, ctx, userRepo, "sa2@example.com")
	sellerB := makeSeller(t, ctx, userRepo, "sb2@example.com")

	a, err := repo.Create(ctx, sampleAddress(sellerA, "A"))
	require.NoError(t, err)

	newLabel := "Hijack"
	_, err = repo.Update(ctx, a.ID, sellerB, seller.UpdateAddressInput{Label: &newLabel})
	require.ErrorIs(t, err, seller.ErrAddressNotFound)
}

func TestAddress_Update_InvalidInput(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerID := makeSeller(t, ctx, userRepo, "s9@example.com")

	a, err := repo.Create(ctx, sampleAddress(sellerID, "A"))
	require.NoError(t, err)

	empty := ""
	_, err = repo.Update(ctx, a.ID, sellerID, seller.UpdateAddressInput{ContactName: &empty})
	require.ErrorIs(t, err, seller.ErrInvalidInput)
}

func TestAddress_SetDefault(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerID := makeSeller(t, ctx, userRepo, "s10@example.com")

	first, err := repo.Create(ctx, sampleAddress(sellerID, "First"))
	require.NoError(t, err)
	second, err := repo.Create(ctx, sampleAddress(sellerID, "Second"))
	require.NoError(t, err)

	require.True(t, first.IsDefault)

	updated, err := repo.SetDefault(ctx, second.ID, sellerID)
	require.NoError(t, err)
	require.True(t, updated.IsDefault)

	firstRefreshed, err := repo.GetByID(ctx, first.ID, sellerID)
	require.NoError(t, err)
	require.False(t, firstRefreshed.IsDefault)
}

func TestAddress_SetDefault_NotOwner(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerA := makeSeller(t, ctx, userRepo, "sa3@example.com")
	sellerB := makeSeller(t, ctx, userRepo, "sb3@example.com")

	a, err := repo.Create(ctx, sampleAddress(sellerA, "A"))
	require.NoError(t, err)

	_, err = repo.SetDefault(ctx, a.ID, sellerB)
	require.ErrorIs(t, err, seller.ErrAddressNotFound)
}

func TestAddress_SoftDelete_NonDefault(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerID := makeSeller(t, ctx, userRepo, "s11@example.com")

	first, err := repo.Create(ctx, sampleAddress(sellerID, "First"))
	require.NoError(t, err)
	second, err := repo.Create(ctx, sampleAddress(sellerID, "Second"))
	require.NoError(t, err)

	require.NoError(t, repo.SoftDelete(ctx, second.ID, sellerID))

	list, err := repo.ListBySeller(ctx, sellerID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, first.ID, list[0].ID)
	require.True(t, list[0].IsDefault, "default unchanged")
}

func TestAddress_SoftDelete_DefaultPromotes(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerID := makeSeller(t, ctx, userRepo, "s12@example.com")

	first, err := repo.Create(ctx, sampleAddress(sellerID, "First"))
	require.NoError(t, err)
	second, err := repo.Create(ctx, sampleAddress(sellerID, "Second"))
	require.NoError(t, err)

	require.True(t, first.IsDefault)

	// Delete the default.
	require.NoError(t, repo.SoftDelete(ctx, first.ID, sellerID))

	// Second should now be default.
	def, err := repo.GetDefault(ctx, sellerID)
	require.NoError(t, err)
	require.Equal(t, second.ID, def.ID)
}

func TestAddress_SoftDelete_LastAddress(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerID := makeSeller(t, ctx, userRepo, "s13@example.com")

	a, err := repo.Create(ctx, sampleAddress(sellerID, "Only"))
	require.NoError(t, err)

	require.NoError(t, repo.SoftDelete(ctx, a.ID, sellerID))

	// No addresses left.
	list, err := repo.ListBySeller(ctx, sellerID)
	require.NoError(t, err)
	require.Empty(t, list)

	// No default.
	_, err = repo.GetDefault(ctx, sellerID)
	require.ErrorIs(t, err, seller.ErrNoDefault)
}

func TestAddress_SoftDelete_NotOwner(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerA := makeSeller(t, ctx, userRepo, "sa4@example.com")
	sellerB := makeSeller(t, ctx, userRepo, "sb4@example.com")

	a, err := repo.Create(ctx, sampleAddress(sellerA, "A"))
	require.NoError(t, err)

	err = repo.SoftDelete(ctx, a.ID, sellerB)
	require.ErrorIs(t, err, seller.ErrAddressNotFound)
}

func TestAddress_GetDefault_None(t *testing.T) {
	_, repo, userRepo := setupSellerTest(t)
	ctx := context.Background()
	sellerID := makeSeller(t, ctx, userRepo, "s14@example.com")

	_, err := repo.GetDefault(ctx, sellerID)
	require.ErrorIs(t, err, seller.ErrNoDefault)
}
