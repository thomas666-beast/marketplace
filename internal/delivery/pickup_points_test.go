package delivery_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/thomas666-beast/marketplace/internal/delivery"
	"github.com/thomas666-beast/marketplace/internal/postgres"
)

func setupDeliveryTest(t *testing.T) (*postgres.DB, *delivery.PickupPointRepository) {
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

	return db, delivery.NewPickupPointRepository(db.Pool)
}

func TestPickupPoint_GetByExternalID(t *testing.T) {
	_, repo := setupDeliveryTest(t)
	ctx := context.Background()

	p, err := repo.GetByExternalID(ctx, "MSK-001")
	require.NoError(t, err)
	require.Equal(t, "MSK-001", p.ExternalID)
	require.Equal(t, "Arbat Locker", p.Name)
	require.Equal(t, "Moscow", p.City)
	require.Equal(t, delivery.PickupPointLocker, p.Type)
	require.True(t, p.IsActive)
	require.NotNil(t, p.Latitude)
	require.NotNil(t, p.Longitude)
}

func TestPickupPoint_GetByExternalID_NotFound(t *testing.T) {
	_, repo := setupDeliveryTest(t)
	ctx := context.Background()

	_, err := repo.GetByExternalID(ctx, "XXX-999")
	require.ErrorIs(t, err, delivery.ErrPickupPointNotFound)
}

func TestPickupPoint_GetByID(t *testing.T) {
	_, repo := setupDeliveryTest(t)
	ctx := context.Background()

	byExt, err := repo.GetByExternalID(ctx, "SPB-001")
	require.NoError(t, err)

	byID, err := repo.GetByID(ctx, byExt.ID)
	require.NoError(t, err)
	require.Equal(t, byExt.ID, byID.ID)
}

func TestPickupPoint_GetByID_InvalidUUID(t *testing.T) {
	_, repo := setupDeliveryTest(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "not-a-uuid")
	require.ErrorIs(t, err, delivery.ErrPickupPointNotFound)
}

func TestPickupPoint_ListByCity(t *testing.T) {
	_, repo := setupDeliveryTest(t)
	ctx := context.Background()

	moscow, err := repo.ListByCity(ctx, "Moscow")
	require.NoError(t, err)
	require.Len(t, moscow, 2)

	// Ordered by type then name: locker before office.
	require.Equal(t, delivery.PickupPointLocker, moscow[0].Type)
	require.Equal(t, delivery.PickupPointOffice, moscow[1].Type)
}

func TestPickupPoint_ListByCity_CaseInsensitive(t *testing.T) {
	_, repo := setupDeliveryTest(t)
	ctx := context.Background()

	points, err := repo.ListByCity(ctx, "moscow")
	require.NoError(t, err)
	require.Len(t, points, 2)
}

func TestPickupPoint_ListByCity_Empty(t *testing.T) {
	_, repo := setupDeliveryTest(t)
	ctx := context.Background()

	points, err := repo.ListByCity(ctx, "Nowhere")
	require.NoError(t, err)
	require.Empty(t, points)
}

func TestPickupPoint_ListNearby(t *testing.T) {
	_, repo := setupDeliveryTest(t)
	ctx := context.Background()

	// Coordinates near the center of Moscow.
	// MSK-001 is at 55.751244, 37.593409 (Arbat).
	// MSK-002 is at 55.760673, 37.607670 (Tverskaya).
	// Distance between them ≈ 1.4 km.
	near, err := repo.ListNearby(ctx, delivery.NearbyInput{
		Latitude:   55.751244,
		Longitude:  37.593409,
		RadiusKm:   3,
		MaxResults: 10,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(near), 2)

	// Closest should be MSK-001 itself.
	require.Equal(t, "MSK-001", near[0].ExternalID)

	// MSK-002 should be second (only a couple km away).
	found := false
	for _, p := range near {
		if p.ExternalID == "MSK-002" {
			found = true
			break
		}
	}
	require.True(t, found, "MSK-002 should be within 3 km of Arbat")
}

func TestPickupPoint_ListNearby_SmallRadius(t *testing.T) {
	_, repo := setupDeliveryTest(t)
	ctx := context.Background()

	// 500 m radius around Arbat. Tverskaya is ~1.4 km away, should be excluded.
	near, err := repo.ListNearby(ctx, delivery.NearbyInput{
		Latitude:   55.751244,
		Longitude:  37.593409,
		RadiusKm:   0.5,
		MaxResults: 10,
	})
	require.NoError(t, err)

	for _, p := range near {
		require.NotEqual(t, "MSK-002", p.ExternalID,
			"Tverskaya should be outside a 500 m radius of Arbat")
	}
	require.Len(t, near, 1)
	require.Equal(t, "MSK-001", near[0].ExternalID)
}

func TestPickupPoint_ListNearby_NoResults(t *testing.T) {
	_, repo := setupDeliveryTest(t)
	ctx := context.Background()

	// Middle of the Atlantic.
	near, err := repo.ListNearby(ctx, delivery.NearbyInput{
		Latitude:   0,
		Longitude:  -30,
		RadiusKm:   100,
		MaxResults: 10,
	})
	require.NoError(t, err)
	require.Empty(t, near)
}

func TestPickupPoint_Create(t *testing.T) {
	_, repo := setupDeliveryTest(t)
	ctx := context.Background()

	lat := 55.0
	lon := 37.0
	p, err := repo.Create(ctx, delivery.CreatePickupPointInput{
		ExternalID: "TST-001",
		Type:       delivery.PickupPointLocker,
		Name:       "Test Locker",
		Address:    "1 Test Street",
		City:       "Testville",
		Region:     "Test Oblast",
		PostalCode: "000001",
		Country:    "RU",
		Latitude:   &lat,
		Longitude:  &lon,
		WorkHours:  json.RawMessage(`{"mon_sun":"24/7"}`),
	})
	require.NoError(t, err)
	require.NotEmpty(t, p.ID)
	require.Equal(t, "TST-001", p.ExternalID)
	require.Equal(t, "Test Locker", p.Name)

	// Retrieve it back.
	fetched, err := repo.GetByExternalID(ctx, "TST-001")
	require.NoError(t, err)
	require.Equal(t, p.ID, fetched.ID)
}

func TestPickupPoint_Create_DuplicateExternalID(t *testing.T) {
	_, repo := setupDeliveryTest(t)
	ctx := context.Background()

	in := delivery.CreatePickupPointInput{
		ExternalID: "DUP-001",
		Type:       delivery.PickupPointLocker,
		Name:       "Dup",
		Address:    "1 Dup Street",
		City:       "Dupville",
	}

	_, err := repo.Create(ctx, in)
	require.NoError(t, err)

	_, err = repo.Create(ctx, in)
	require.Error(t, err)
	require.Contains(t, err.Error(), "external_id already exists")
}

func TestPickupPoint_Deactivate(t *testing.T) {
	_, repo := setupDeliveryTest(t)
	ctx := context.Background()

	p, err := repo.GetByExternalID(ctx, "KZN-001")
	require.NoError(t, err)

	require.NoError(t, repo.Deactivate(ctx, p.ID))

	// No longer findable.
	_, err = repo.GetByExternalID(ctx, "KZN-001")
	require.ErrorIs(t, err, delivery.ErrPickupPointNotFound)

	_, err = repo.GetByID(ctx, p.ID)
	require.ErrorIs(t, err, delivery.ErrPickupPointNotFound)

	// Not listed by city.
	points, err := repo.ListByCity(ctx, "Kazan")
	require.NoError(t, err)
	require.Empty(t, points)
}

func TestPickupPoint_Deactivate_NotFound(t *testing.T) {
	_, repo := setupDeliveryTest(t)
	ctx := context.Background()

	err := repo.Deactivate(ctx, "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, delivery.ErrPickupPointNotFound)

	err = repo.Deactivate(ctx, "not-a-uuid")
	require.ErrorIs(t, err, delivery.ErrPickupPointNotFound)
}
