package users_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/thomas666-beast/marketplace/internal/postgres"
	"github.com/thomas666-beast/marketplace/internal/users"
)

func setupTestDB(t *testing.T) (*postgres.DB, *users.Repository) {
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
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
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

	return db, users.NewRepository(db.Pool)
}

func sampleInput(email string) users.CreateInput {
	return users.CreateInput{
		Email:           email,
		PasswordHash:    "$argon2id$v=19$m=65536,t=3,p=4$fake$fake",
		DisplayName:     "Test User",
		Role:            users.RoleBuyer,
		PreferredLocale: users.LocaleEN,
	}
}

func TestCreateUser(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	u, err := repo.Create(ctx, sampleInput("alice@example.com"))
	require.NoError(t, err)
	require.NotEmpty(t, u.ID)
	require.Equal(t, "alice@example.com", u.Email)
	require.Equal(t, users.RoleBuyer, u.Role)
	require.Equal(t, users.LocaleEN, u.PreferredLocale)
	require.False(t, u.EmailVerified)
	require.True(t, u.IsActive)
	require.Nil(t, u.DeletedAt)
	require.WithinDuration(t, time.Now(), u.CreatedAt, 5*time.Second)
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, sampleInput("dup@example.com"))
	require.NoError(t, err)

	_, err = repo.Create(ctx, sampleInput("dup@example.com"))
	require.ErrorIs(t, err, users.ErrEmailExists)
}

func TestCreateUser_CaseInsensitiveEmail(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, sampleInput("Case@Example.com"))
	require.NoError(t, err)

	_, err = repo.Create(ctx, sampleInput("case@example.com"))
	require.ErrorIs(t, err, users.ErrEmailExists, "CITEXT must treat emails case-insensitively")
}

func TestGetByEmail(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, sampleInput("bob@example.com"))
	require.NoError(t, err)

	fetched, err := repo.GetByEmail(ctx, "bob@example.com")
	require.NoError(t, err)
	require.Equal(t, created.ID, fetched.ID)
}

func TestGetByID_NotFound(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, users.ErrUserNotFound)
}

func TestSoftDelete(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, sampleInput("carol@example.com"))
	require.NoError(t, err)

	require.NoError(t, repo.SoftDelete(ctx, created.ID))

	_, err = repo.GetByID(ctx, created.ID)
	require.ErrorIs(t, err, users.ErrUserNotFound, "soft-deleted users must not be retrievable")

	_, err = repo.GetByEmail(ctx, "carol@example.com")
	require.ErrorIs(t, err, users.ErrUserNotFound)
}

func TestSoftDelete_NotFound(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	err := repo.SoftDelete(ctx, "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, users.ErrUserNotFound)
}

func TestUpdateLocale(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, sampleInput("dave@example.com"))
	require.NoError(t, err)

	require.NoError(t, repo.UpdateLocale(ctx, created.ID, users.LocaleRU))

	fetched, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, users.LocaleRU, fetched.PreferredLocale)
}

func TestUpdateLocale_NotFound(t *testing.T) {
	_, repo := setupTestDB(t)
	ctx := context.Background()

	err := repo.UpdateLocale(ctx, "00000000-0000-0000-0000-000000000000", users.LocaleES)
	require.ErrorIs(t, err, users.ErrUserNotFound)
}
