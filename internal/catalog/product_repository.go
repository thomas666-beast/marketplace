package catalog

import (
    "context"
    "encoding/base64"
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgx/v5/pgxpool"
)

type ProductRepository struct {
    pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
    return &ProductRepository{pool: pool}
}

const productColumns = `
    id, seller_id, category_id, slug, name, description,
    price_cents, currency, stock_quantity, status,
    deleted_at, created_at, updated_at
`

func scanProduct(row pgx.Row) (Product, error) {
    var p Product
    err := row.Scan(
        &p.ID,
        &p.SellerID,
        &p.CategoryID,
        &p.Slug,
        &p.Name,
        &p.Description,
        &p.PriceCents,
        &p.Currency,
        &p.StockQuantity,
        &p.Status,
        &p.DeletedAt,
        &p.CreatedAt,
        &p.UpdatedAt,
    )
    return p, err
}

// isInvalidUUID reports whether a pg error is a malformed UUID input.
// PostgreSQL error code 22P02 = invalid_text_representation.
func isInvalidUUID(err error) bool {
    var pgErr *pgconn.PgError
    return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}

// slugify produces a URL-friendly slug from a name.
// Appends a short suffix to guarantee uniqueness.
func slugify(name string) string {
    lower := strings.ToLower(name)
    var b strings.Builder
    prevDash := false
    for _, r := range lower {
        switch {
        case r >= 'a' && r <= 'z':
            b.WriteRune(r)
            prevDash = false
        case r >= '0' && r <= '9':
            b.WriteRune(r)
            prevDash = false
        case r == ' ' || r == '-' || r == '_' || r == '.':
            if !prevDash && b.Len() > 0 {
                b.WriteByte('-')
                prevDash = true
            }
        }
    }
    s := strings.Trim(b.String(), "-")
    if s == "" {
        s = "product"
    }
    if len(s) > 60 {
        s = s[:60]
    }
    suffix := fmt.Sprintf("-%d", time.Now().UnixNano()%1000000)
    return s + suffix
}

func (r *ProductRepository) Create(ctx context.Context, in CreateProductInput) (Product, error) {
    const q = `
        INSERT INTO products
            (seller_id, category_id, slug, name, description,
             price_cents, currency, stock_quantity, status)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING ` + productColumns

    slug := slugify(in.Name)

    p, err := scanProduct(r.pool.QueryRow(ctx, q,
        in.SellerID,
        in.CategoryID,
        slug,
        in.Name,
        in.Description,
        in.PriceCents,
        in.Currency,
        in.StockQuantity,
        in.Status,
    ))
    if err != nil {
        return Product{}, fmt.Errorf("create product: %w", err)
    }
    return p, nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id string) (Product, error) {
    const q = `SELECT ` + productColumns + `
               FROM products WHERE id = $1 AND deleted_at IS NULL`

    p, err := scanProduct(r.pool.QueryRow(ctx, q, id))
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
            return Product{}, ErrProductNotFound
        }
        return Product{}, fmt.Errorf("get product: %w", err)
    }
    return p, nil
}

func (r *ProductRepository) GetBySlug(ctx context.Context, slug string) (Product, error) {
    const q = `SELECT ` + productColumns + `
               FROM products WHERE slug = $1 AND deleted_at IS NULL`

    p, err := scanProduct(r.pool.QueryRow(ctx, q, slug))
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return Product{}, ErrProductNotFound
        }
        return Product{}, fmt.Errorf("get product by slug: %w", err)
    }
    return p, nil
}

// Update applies a partial update. Only the owner may update.
func (r *ProductRepository) Update(ctx context.Context, id, sellerID string, in UpdateProductInput) (Product, error) {
    sets := []string{}
    args := []any{}
    i := 1

    if in.Name != nil {
        sets = append(sets, fmt.Sprintf("name = $%d", i))
        args = append(args, *in.Name)
        i++
    }
    if in.Description != nil {
        sets = append(sets, fmt.Sprintf("description = $%d", i))
        args = append(args, *in.Description)
        i++
    }
    if in.PriceCents != nil {
        sets = append(sets, fmt.Sprintf("price_cents = $%d", i))
        args = append(args, *in.PriceCents)
        i++
    }
    if in.Currency != nil {
        sets = append(sets, fmt.Sprintf("currency = $%d", i))
        args = append(args, *in.Currency)
        i++
    }
    if in.StockQuantity != nil {
        sets = append(sets, fmt.Sprintf("stock_quantity = $%d", i))
        args = append(args, *in.StockQuantity)
        i++
    }
    if in.Status != nil {
        sets = append(sets, fmt.Sprintf("status = $%d", i))
        args = append(args, *in.Status)
        i++
    }

    if len(sets) == 0 {
        return r.GetByID(ctx, id)
    }

    sets = append(sets, "updated_at = NOW()")

    q := fmt.Sprintf(`
        UPDATE products SET %s
        WHERE id = $%d AND seller_id = $%d AND deleted_at IS NULL
        RETURNING %s
    `, strings.Join(sets, ", "), i, i+1, productColumns)

    args = append(args, id, sellerID)

    p, err := scanProduct(r.pool.QueryRow(ctx, q, args...))
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
            return Product{}, r.distinguishNotFoundFromNotOwner(ctx, id, sellerID)
        }
        return Product{}, fmt.Errorf("update product: %w", err)
    }
    return p, nil
}

// distinguishNotFoundFromNotOwner runs a follow-up check to determine why
// an update or delete failed. Used only for error reporting, not for auth.
func (r *ProductRepository) distinguishNotFoundFromNotOwner(ctx context.Context, id, sellerID string) error {
    var ownerID string
    err := r.pool.QueryRow(ctx,
        `SELECT seller_id FROM products WHERE id = $1 AND deleted_at IS NULL`,
        id,
    ).Scan(&ownerID)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
            return ErrProductNotFound
        }
        return fmt.Errorf("lookup product owner: %w", err)
    }
    if ownerID != sellerID {
        return ErrNotOwner
    }
    return ErrProductNotFound
}

// SoftDelete marks the product as deleted. Only the owner may delete.
func (r *ProductRepository) SoftDelete(ctx context.Context, id, sellerID string) error {
    const q = `UPDATE products SET deleted_at = NOW(), updated_at = NOW()
               WHERE id = $1 AND seller_id = $2 AND deleted_at IS NULL`

    tag, err := r.pool.Exec(ctx, q, id, sellerID)
    if err != nil {
        if isInvalidUUID(err) {
            return ErrProductNotFound
        }
        return fmt.Errorf("soft delete product: %w", err)
    }
    if tag.RowsAffected() == 0 {
        return r.distinguishNotFoundFromNotOwner(ctx, id, sellerID)
    }
    return nil
}

// List returns a page of products matching the filter.
func (r *ProductRepository) List(ctx context.Context, f ProductFilter) (ProductPage, error) {
    limit := f.Limit
    if limit <= 0 {
        limit = 20
    }
    if limit > 100 {
        limit = 100
    }

    where := []string{"deleted_at IS NULL"}
    args := []any{}
    i := 1

    if f.CategoryID != nil {
        where = append(where, fmt.Sprintf("category_id = $%d", i))
        args = append(args, *f.CategoryID)
        i++
    }
    if f.SellerID != nil {
        where = append(where, fmt.Sprintf("seller_id = $%d", i))
        args = append(args, *f.SellerID)
        i++
    } else {
        // Buyer-facing listing: only active products.
        where = append(where, "status = 'active'")
    }
    if f.Search != "" {
        where = append(where, fmt.Sprintf(
            "to_tsvector('simple', name || ' ' || description) @@ plainto_tsquery('simple', $%d)", i))
        args = append(args, f.Search)
        i++
    }

    // Cursor: keyset on (created_at, id) DESC.
    if f.Cursor != "" {
        createdAt, id, err := decodeCursor(f.Cursor)
        if err != nil {
            return ProductPage{}, fmt.Errorf("invalid cursor: %w", err)
        }
        where = append(where, fmt.Sprintf("(created_at, id) < ($%d, $%d)", i, i+1))
        args = append(args, createdAt, id)
        i += 2
    }

    q := fmt.Sprintf(`
        SELECT %s
        FROM products
        WHERE %s
        ORDER BY created_at DESC, id DESC
        LIMIT $%d
    `, productColumns, strings.Join(where, " AND "), i)

    // Fetch limit+1 to detect whether there's a next page.
    args = append(args, limit+1)

    rows, err := r.pool.Query(ctx, q, args...)
    if err != nil {
        return ProductPage{}, fmt.Errorf("list products: %w", err)
    }
    defer rows.Close()

    var items []Product
    for rows.Next() {
        p, err := scanProduct(rows)
        if err != nil {
            return ProductPage{}, fmt.Errorf("scan product: %w", err)
        }
        items = append(items, p)
    }
    if err := rows.Err(); err != nil {
        return ProductPage{}, fmt.Errorf("iterate products: %w", err)
    }

    page := ProductPage{Items: items}
    if len(items) > limit {
        last := items[limit-1]
        page.Items = items[:limit]
        page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
    }
    return page, nil
}

func encodeCursor(createdAt time.Time, id string) string {
    raw := createdAt.UTC().Format(time.RFC3339Nano) + "|" + id
    return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(cursor string) (time.Time, string, error) {
    raw, err := base64.RawURLEncoding.DecodeString(cursor)
    if err != nil {
        return time.Time{}, "", err
    }
    parts := strings.SplitN(string(raw), "|", 2)
    if len(parts) != 2 {
        return time.Time{}, "", fmt.Errorf("malformed cursor")
    }
    t, err := time.Parse(time.RFC3339Nano, parts[0])
    if err != nil {
        return time.Time{}, "", err
    }
    return t, parts[1], nil
}
