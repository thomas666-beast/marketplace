package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/thomas666-beast/marketplace/internal/auth"
	"github.com/thomas666-beast/marketplace/internal/catalog"
	"github.com/thomas666-beast/marketplace/internal/httpserver"
	"github.com/thomas666-beast/marketplace/internal/i18n"
	"github.com/thomas666-beast/marketplace/internal/orders"
	"github.com/thomas666-beast/marketplace/internal/postgres"
	"github.com/thomas666-beast/marketplace/internal/users"
)

const testSecret = "test-secret-must-be-at-least-32-bytes-long-ok"

type serverFixture struct {
	url string
}

func setupServerFixture(t *testing.T) *serverFixture {
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

	bundle, err := i18n.NewBundle()
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tokens, err := auth.NewTokenManager(testSecret)
	require.NoError(t, err)

	userRepo := users.NewRepository(db.Pool)
	catRepo := catalog.NewCategoryRepository(db.Pool)
	prodRepo := catalog.NewProductRepository(db.Pool)
	orderRepo := orders.NewRepository(db.Pool)

	srv := httpserver.New(
		"8080", db, bundle, logger,
		userRepo, catRepo, prodRepo, orderRepo, tokens,
	)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	return &serverFixture{url: ts.URL}
}

func postJSON(t *testing.T, url string, body any, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	buf, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(buf))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, data
}
