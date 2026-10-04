package catalog_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/thomas666-beast/marketplace/internal/catalog"
	"github.com/thomas666-beast/marketplace/internal/postgres"
)

func setupCatalogTest(t *testing.T) (*postgres.DB, *catalog.CategoryRepository) {
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

	return db, catalog.NewCategoryRepository(db.Pool)
}

func TestCategories_SeedLoaded(t *testing.T) {
	_, repo := setupCatalogTest(t)
	ctx := context.Background()

	cats, err := repo.ListActive(ctx)
	require.NoError(t, err)
	require.Len(t, cats, 9, "seed migration should create 9 categories")
}

func TestCategories_GetBySlug(t *testing.T) {
	_, repo := setupCatalogTest(t)
	ctx := context.Background()

	c, err := repo.GetBySlug(ctx, "smartphones")
	require.NoError(t, err)
	require.Equal(t, "smartphones", c.Slug)
	require.Equal(t, "Smartphones", c.Name.EN)
	require.Equal(t, "Смартфоны", c.Name.RU)
	require.Equal(t, "Teléfonos", c.Name.ES)
	require.NotNil(t, c.ParentID)
}

func TestCategories_GetBySlug_NotFound(t *testing.T) {
	_, repo := setupCatalogTest(t)
	ctx := context.Background()

	_, err := repo.GetBySlug(ctx, "does-not-exist")
	require.ErrorIs(t, err, catalog.ErrCategoryNotFound)
}

func TestCategories_GetByID_NotFound(t *testing.T) {
	_, repo := setupCatalogTest(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, catalog.ErrCategoryNotFound)
}

func TestCategories_Tree(t *testing.T) {
	_, repo := setupCatalogTest(t)
	ctx := context.Background()

	roots, err := repo.TreeActive(ctx)
	require.NoError(t, err)
	require.Len(t, roots, 5, "5 root categories in seed data")

	// Find the "electronics" root.
	var electronics *catalog.CategoryNode
	for i := range roots {
		if roots[i].Slug == "electronics" {
			electronics = &roots[i]
			break
		}
	}
	require.NotNil(t, electronics, "electronics category must exist")
	require.Len(t, electronics.Children, 2, "electronics has 2 children in seed data")

	// Verify sort_order: smartphones (10) before laptops (20).
	require.Equal(t, "smartphones", electronics.Children[0].Slug)
	require.Equal(t, "laptops", electronics.Children[1].Slug)
}

func TestCategories_LocalizedName(t *testing.T) {
	_, repo := setupCatalogTest(t)
	ctx := context.Background()

	c, err := repo.GetBySlug(ctx, "books")
	require.NoError(t, err)

	require.Equal(t, "Books", c.Name.For("en"))
	require.Equal(t, "Книги", c.Name.For("ru"))
	require.Equal(t, "Libros", c.Name.For("es"))
	require.Equal(t, "Books", c.Name.For("fr"), "unknown locale falls back to English")
	require.Equal(t, "Books", c.Name.For(""), "empty locale falls back to English")
}

func TestCategories_Deactivate(t *testing.T) {
	_, repo := setupCatalogTest(t)
	ctx := context.Background()

	c, err := repo.GetBySlug(ctx, "books")
	require.NoError(t, err)

	require.NoError(t, repo.Deactivate(ctx, c.ID))

	_, err = repo.GetBySlug(ctx, "books")
	require.ErrorIs(t, err, catalog.ErrCategoryNotFound)
}

func TestCategories_Deactivate_NotFound(t *testing.T) {
	_, repo := setupCatalogTest(t)
	ctx := context.Background()

	err := repo.Deactivate(ctx, "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, catalog.ErrCategoryNotFound)
}

func TestCategories_Deactivate_ChildBecomesRoot(t *testing.T) {
	_, repo := setupCatalogTest(t)
	ctx := context.Background()

	electronics, err := repo.GetBySlug(ctx, "electronics")
	require.NoError(t, err)

	require.NoError(t, repo.Deactivate(ctx, electronics.ID))

	roots, err := repo.TreeActive(ctx)
	require.NoError(t, err)

	// Electronics is gone, but its children are now roots.
	slugs := make(map[string]bool)
	for _, r := range roots {
		slugs[r.Slug] = true
	}
	require.True(t, slugs["smartphones"], "orphaned child becomes root")
	require.True(t, slugs["laptops"], "orphaned child becomes root")
	require.False(t, slugs["electronics"], "parent is deactivated")
}
