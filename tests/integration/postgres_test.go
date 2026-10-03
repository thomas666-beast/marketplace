package integration

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"

    "github.com/thomas666-beast/marketplace/internal/postgres"
)

func TestPostgresConnectivity(t *testing.T) {
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
    defer func() { _ = container.Terminate(ctx) }()

    host, err := container.Host(ctx)
    require.NoError(t, err)
    port, err := container.MappedPort(ctx, "5432")
    require.NoError(t, err)

    dsn := "postgres://test:test@" + host + ":" + port.Port() + "/test?sslmode=disable"

    db, err := postgres.Connect(ctx, dsn)
    require.NoError(t, err)
    defer db.Close()

    var one int
    err = db.Pool.QueryRow(ctx, "SELECT 1").Scan(&one)
    require.NoError(t, err)
    require.Equal(t, 1, one)
}
