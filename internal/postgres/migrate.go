package postgres

import (
    "database/sql"
    "errors"
    "fmt"

    "github.com/golang-migrate/migrate/v4"
    migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
    "github.com/golang-migrate/migrate/v4/source/iofs"

    _ "github.com/jackc/pgx/v5/stdlib"
)

func RunMigrations(dsn string) error {
    src, err := iofs.New(migrationsFS, "migrations")
    if err != nil {
        return fmt.Errorf("load migrations from embed: %w", err)
    }

    db, err := sql.Open("pgx", dsn)
    if err != nil {
        return fmt.Errorf("open db for migrations: %w", err)
    }
    defer db.Close()

    driver, err := migratepg.WithInstance(db, &migratepg.Config{})
    if err != nil {
        return fmt.Errorf("create migrate driver: %w", err)
    }

    m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
    if err != nil {
        return fmt.Errorf("create migrate instance: %w", err)
    }

    if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
        return fmt.Errorf("apply migrations: %w", err)
    }

    return nil
}
