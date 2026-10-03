package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const selectColumns = `
	id, email, password_hash, display_name, role, preferred_locale,
	email_verified, is_active, deleted_at, created_at, updated_at
`

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.DisplayName,
		&u.Role,
		&u.PreferredLocale,
		&u.EmailVerified,
		&u.IsActive,
		&u.DeletedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	return u, err
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (User, error) {
	const q = `
		INSERT INTO users (email, password_hash, display_name, role, preferred_locale)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING ` + selectColumns

	row := r.pool.QueryRow(ctx, q,
		in.Email, in.PasswordHash, in.DisplayName, in.Role, in.PreferredLocale,
	)

	u, err := scanUser(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailExists
		}
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (User, error) {
	const q = `SELECT ` + selectColumns + ` FROM users WHERE id = $1 AND deleted_at IS NULL`

	u, err := scanUser(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (User, error) {
	const q = `SELECT ` + selectColumns + ` FROM users WHERE email = $1 AND deleted_at IS NULL`

	u, err := scanUser(r.pool.QueryRow(ctx, q, email))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	const q = `UPDATE users SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *Repository) UpdateLocale(ctx context.Context, id string, locale Locale) error {
	const q = `UPDATE users SET preferred_locale = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, q, locale, id)
	if err != nil {
		return fmt.Errorf("update locale: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}
