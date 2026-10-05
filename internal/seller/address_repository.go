package seller

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AddressRepository struct {
	pool *pgxpool.Pool
}

func NewAddressRepository(pool *pgxpool.Pool) *AddressRepository {
	return &AddressRepository{pool: pool}
}

const addressColumns = `
	id, seller_id, label, contact_name, phone, address, city, region,
	postal_code, country, is_default, deleted_at, created_at, updated_at
`

func scanAddress(row pgx.Row) (Address, error) {
	var a Address
	err := row.Scan(
		&a.ID, &a.SellerID, &a.Label, &a.ContactName, &a.Phone,
		&a.Address, &a.City, &a.Region, &a.PostalCode, &a.Country,
		&a.IsDefault, &a.DeletedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	return a, err
}

// Create inserts a new address. If it is the seller's first address,
// it becomes the default automatically. If IsDefault is true, any
// existing default is cleared first. All within a single transaction.
func (r *AddressRepository) Create(ctx context.Context, in CreateAddressInput) (Address, error) {
	if err := validateCreateInput(in); err != nil {
		return Address{}, err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Address{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Count existing active addresses to detect "first address".
	var existing int
	const countQ = `SELECT COUNT(*) FROM seller_addresses
	                WHERE seller_id = $1 AND deleted_at IS NULL`
	if err := tx.QueryRow(ctx, countQ, in.SellerID).Scan(&existing); err != nil {
		return Address{}, fmt.Errorf("count addresses: %w", err)
	}

	isDefault := in.IsDefault || existing == 0

	// If becoming default, clear any existing default.
	if isDefault {
		const clearQ = `UPDATE seller_addresses SET is_default = FALSE, updated_at = NOW()
		                WHERE seller_id = $1 AND is_default = TRUE AND deleted_at IS NULL`
		if _, err := tx.Exec(ctx, clearQ, in.SellerID); err != nil {
			return Address{}, fmt.Errorf("clear existing default: %w", err)
		}
	}

	country := in.Country
	if country == "" {
		country = "RU"
	}

	const q = `
		INSERT INTO seller_addresses
			(seller_id, label, contact_name, phone, address, city, region,
			 postal_code, country, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING ` + addressColumns
	a, err := scanAddress(tx.QueryRow(ctx, q,
		in.SellerID, strings.TrimSpace(in.Label), strings.TrimSpace(in.ContactName),
		strings.TrimSpace(in.Phone), strings.TrimSpace(in.Address),
		strings.TrimSpace(in.City), strings.TrimSpace(in.Region),
		strings.TrimSpace(in.PostalCode), country, isDefault,
	))
	if err != nil {
		return Address{}, fmt.Errorf("insert address: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Address{}, fmt.Errorf("commit tx: %w", err)
	}
	return a, nil
}

func (r *AddressRepository) ListBySeller(ctx context.Context, sellerID string) ([]Address, error) {
	const q = `SELECT ` + addressColumns + `
	           FROM seller_addresses
	           WHERE seller_id = $1 AND deleted_at IS NULL
	           ORDER BY is_default DESC, created_at DESC`

	rows, err := r.pool.Query(ctx, q, sellerID)
	if err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}
	defer rows.Close()

	var out []Address
	for rows.Next() {
		a, err := scanAddress(rows)
		if err != nil {
			return nil, fmt.Errorf("scan address: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AddressRepository) GetByID(ctx context.Context, id, sellerID string) (Address, error) {
	const q = `SELECT ` + addressColumns + `
	           FROM seller_addresses
	           WHERE id = $1 AND seller_id = $2 AND deleted_at IS NULL`

	a, err := scanAddress(r.pool.QueryRow(ctx, q, id, sellerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Address{}, ErrAddressNotFound
		}
		return Address{}, fmt.Errorf("get address: %w", err)
	}
	return a, nil
}

func (r *AddressRepository) Update(ctx context.Context, id, sellerID string, in UpdateAddressInput) (Address, error) {
	sets := []string{}
	args := []any{}
	i := 1

	if in.Label != nil {
		sets = append(sets, fmt.Sprintf("label = $%d", i))
		args = append(args, strings.TrimSpace(*in.Label))
		i++
	}
	if in.ContactName != nil {
		v := strings.TrimSpace(*in.ContactName)
		if len(v) < 2 || len(v) > 100 {
			return Address{}, ErrInvalidInput
		}
		sets = append(sets, fmt.Sprintf("contact_name = $%d", i))
		args = append(args, v)
		i++
	}
	if in.Phone != nil {
		v := strings.TrimSpace(*in.Phone)
		if len(v) < 5 || len(v) > 30 {
			return Address{}, ErrInvalidInput
		}
		sets = append(sets, fmt.Sprintf("phone = $%d", i))
		args = append(args, v)
		i++
	}
	if in.Address != nil {
		v := strings.TrimSpace(*in.Address)
		if len(v) < 5 || len(v) > 200 {
			return Address{}, ErrInvalidInput
		}
		sets = append(sets, fmt.Sprintf("address = $%d", i))
		args = append(args, v)
		i++
	}
	if in.City != nil {
		v := strings.TrimSpace(*in.City)
		if v == "" || len(v) > 100 {
			return Address{}, ErrInvalidInput
		}
		sets = append(sets, fmt.Sprintf("city = $%d", i))
		args = append(args, v)
		i++
	}
	if in.Region != nil {
		sets = append(sets, fmt.Sprintf("region = $%d", i))
		args = append(args, strings.TrimSpace(*in.Region))
		i++
	}
	if in.PostalCode != nil {
		sets = append(sets, fmt.Sprintf("postal_code = $%d", i))
		args = append(args, strings.TrimSpace(*in.PostalCode))
		i++
	}
	if in.Country != nil {
		v := strings.TrimSpace(*in.Country)
		if v == "" || len(v) > 3 {
			return Address{}, ErrInvalidInput
		}
		sets = append(sets, fmt.Sprintf("country = $%d", i))
		args = append(args, v)
		i++
	}

	if len(sets) == 0 && in.IsDefault == nil {
		return r.GetByID(ctx, id, sellerID)
	}

	// Handle IsDefault separately: it needs to clear existing defaults.
	// We do it inside a transaction to keep the "one default" invariant.
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Address{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Ensure the address exists and belongs to the seller.
	var exists bool
	const existsQ = `SELECT EXISTS(SELECT 1 FROM seller_addresses
	                   WHERE id = $1 AND seller_id = $2 AND deleted_at IS NULL)`
	if err := tx.QueryRow(ctx, existsQ, id, sellerID).Scan(&exists); err != nil {
		return Address{}, fmt.Errorf("check address: %w", err)
	}
	if !exists {
		return Address{}, ErrAddressNotFound
	}

	// If IsDefault is being set to true, clear existing default.
	if in.IsDefault != nil && *in.IsDefault {
		const clearQ = `UPDATE seller_addresses SET is_default = FALSE, updated_at = NOW()
		                WHERE seller_id = $1 AND is_default = TRUE
		                  AND deleted_at IS NULL AND id <> $2`
		if _, err := tx.Exec(ctx, clearQ, sellerID, id); err != nil {
			return Address{}, fmt.Errorf("clear existing default: %w", err)
		}
		sets = append(sets, fmt.Sprintf("is_default = $%d", i))
		args = append(args, true)
		i++
	} else if in.IsDefault != nil && !*in.IsDefault {
		// Setting to false explicitly. If it was the only default, we
		// allow it but the seller ends up with no default. Callers should
		// use SetDefault or Delete instead. We still permit it for flexibility.
		sets = append(sets, fmt.Sprintf("is_default = $%d", i))
		args = append(args, false)
		i++
	}

	sets = append(sets, "updated_at = NOW()")

	q := fmt.Sprintf(`
		UPDATE seller_addresses SET %s
		WHERE id = $%d AND seller_id = $%d AND deleted_at IS NULL
		RETURNING %s
	`, strings.Join(sets, ", "), i, i+1, addressColumns)
	args = append(args, id, sellerID)

	a, err := scanAddress(tx.QueryRow(ctx, q, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Address{}, ErrAddressNotFound
		}
		return Address{}, fmt.Errorf("update address: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Address{}, fmt.Errorf("commit tx: %w", err)
	}
	return a, nil
}

// SetDefault atomically makes this address the default for the seller.
func (r *AddressRepository) SetDefault(ctx context.Context, id, sellerID string) (Address, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Address{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Verify ownership.
	var exists bool
	const existsQ = `SELECT EXISTS(SELECT 1 FROM seller_addresses
	                   WHERE id = $1 AND seller_id = $2 AND deleted_at IS NULL)`
	if err := tx.QueryRow(ctx, existsQ, id, sellerID).Scan(&exists); err != nil {
		return Address{}, fmt.Errorf("check address: %w", err)
	}
	if !exists {
		return Address{}, ErrAddressNotFound
	}

	// Clear any existing default.
	const clearQ = `UPDATE seller_addresses SET is_default = FALSE, updated_at = NOW()
	                WHERE seller_id = $1 AND is_default = TRUE
	                  AND deleted_at IS NULL AND id <> $2`
	if _, err := tx.Exec(ctx, clearQ, sellerID, id); err != nil {
		return Address{}, fmt.Errorf("clear existing default: %w", err)
	}

	// Set the new default.
	const setQ = `UPDATE seller_addresses SET is_default = TRUE, updated_at = NOW()
	              WHERE id = $1 AND seller_id = $2 AND deleted_at IS NULL
	              RETURNING ` + addressColumns
	a, err := scanAddress(tx.QueryRow(ctx, setQ, id, sellerID))
	if err != nil {
		return Address{}, fmt.Errorf("set default: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Address{}, fmt.Errorf("commit tx: %w", err)
	}
	return a, nil
}

// SoftDelete removes an address. If it was the default and other
// addresses exist, the oldest remaining address becomes the new default.
func (r *AddressRepository) SoftDelete(ctx context.Context, id, sellerID string) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Fetch the target address (must be owned).
	const getQ = `SELECT is_default FROM seller_addresses
	              WHERE id = $1 AND seller_id = $2 AND deleted_at IS NULL
	              FOR UPDATE`
	var wasDefault bool
	if err := tx.QueryRow(ctx, getQ, id, sellerID).Scan(&wasDefault); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return ErrAddressNotFound
		}
		return fmt.Errorf("lock address: %w", err)
	}

	// Soft delete.
	const delQ = `UPDATE seller_addresses SET deleted_at = NOW(), is_default = FALSE, updated_at = NOW()
	              WHERE id = $1 AND seller_id = $2`
	if _, err := tx.Exec(ctx, delQ, id, sellerID); err != nil {
		return fmt.Errorf("soft delete address: %w", err)
	}

	// If it was default, promote the oldest remaining address.
	if wasDefault {
		const promoteQ = `
			UPDATE seller_addresses SET is_default = TRUE, updated_at = NOW()
			WHERE id = (
				SELECT id FROM seller_addresses
				WHERE seller_id = $1 AND deleted_at IS NULL
				ORDER BY created_at ASC
				LIMIT 1
			)
		`
		if _, err := tx.Exec(ctx, promoteQ, sellerID); err != nil {
			return fmt.Errorf("promote new default: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// GetDefault returns the seller's default address, if one exists.
func (r *AddressRepository) GetDefault(ctx context.Context, sellerID string) (Address, error) {
	const q = `SELECT ` + addressColumns + `
	           FROM seller_addresses
	           WHERE seller_id = $1 AND is_default = TRUE AND deleted_at IS NULL`

	a, err := scanAddress(r.pool.QueryRow(ctx, q, sellerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Address{}, ErrNoDefault
		}
		return Address{}, fmt.Errorf("get default address: %w", err)
	}
	return a, nil
}

func validateCreateInput(in CreateAddressInput) error {
	if strings.TrimSpace(in.ContactName) == "" || len(in.ContactName) > 100 {
		return ErrInvalidInput
	}
	if strings.TrimSpace(in.Phone) == "" || len(in.Phone) > 30 {
		return ErrInvalidInput
	}
	if strings.TrimSpace(in.Address) == "" || len(in.Address) > 200 {
		return ErrInvalidInput
	}
	if strings.TrimSpace(in.City) == "" || len(in.City) > 100 {
		return ErrInvalidInput
	}
	return nil
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
