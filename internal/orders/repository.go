package orders

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const orderColumns = `
	id, order_number, buyer_id, seller_id, status,
	subtotal_cents, shipping_cents, total_cents, currency,
	shipping_address, note,
	cancelled_reason, cancelled_at,
	paid_at, shipped_at, delivered_at, completed_at,
	created_at, updated_at
`

func scanOrder(row pgx.Row) (Order, error) {
	var o Order
	var addr []byte
	err := row.Scan(
		&o.ID, &o.OrderNumber, &o.BuyerID, &o.SellerID, &o.Status,
		&o.SubtotalCents, &o.ShippingCents, &o.TotalCents, &o.Currency,
		&addr, &o.Note,
		&o.CancelledReason, &o.CancelledAt,
		&o.PaidAt, &o.ShippedAt, &o.DeliveredAt, &o.CompletedAt,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return Order{}, err
	}
	o.ShippingAddress = json.RawMessage(addr)
	return o, nil
}

const orderItemColumns = `
	id, order_id, product_id, product_name, product_slug,
	unit_price_cents, quantity, line_total_cents, created_at
`

func scanOrderItem(row pgx.Row) (OrderItem, error) {
	var it OrderItem
	err := row.Scan(
		&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.ProductSlug,
		&it.UnitPriceCents, &it.Quantity, &it.LineTotalCents, &it.CreatedAt,
	)
	return it, err
}

// Checkout creates an order for one seller in a single transaction.
//
// Steps (all-or-nothing):
//  1. Lock the referenced products with SELECT ... FOR UPDATE
//  2. Verify each product belongs to the seller, is active, and has stock
//  3. Compute subtotal and total
//  4. Insert the order
//  5. Insert order_items (with price snapshots)
//  6. Decrement stock on each product
//  7. Return the full order with items
func (r *Repository) Checkout(ctx context.Context, in CheckoutInput) (OrderWithItems, error) {
	if len(in.Items) == 0 {
		return OrderWithItems{}, ErrEmptyOrder
	}
	for _, it := range in.Items {
		if it.Quantity <= 0 {
			return OrderWithItems{}, ErrInvalidQuantity
		}
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return OrderWithItems{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Collect product IDs and preserve order for deterministic output.
	ids := make([]string, len(in.Items))
	for i, it := range in.Items {
		ids[i] = it.ProductID
	}

	// Lock and fetch products.
	const productQuery = `
		SELECT id, seller_id, name, slug, price_cents, currency, stock_quantity
		FROM products
		WHERE id = ANY($1) AND status = 'active' AND deleted_at IS NULL
		FOR UPDATE
	`
	rows, err := tx.Query(ctx, productQuery, ids)
	if err != nil {
		return OrderWithItems{}, fmt.Errorf("lock products: %w", err)
	}

	type lockedProduct struct {
		ID       string
		SellerID string
		Name     string
		Slug     string
		Price    int64
		Currency Currency
		Stock    int
	}
	locked := make(map[string]lockedProduct, len(ids))
	for rows.Next() {
		var p lockedProduct
		if err := rows.Scan(&p.ID, &p.SellerID, &p.Name, &p.Slug, &p.Price, &p.Currency, &p.Stock); err != nil {
			rows.Close()
			return OrderWithItems{}, fmt.Errorf("scan locked product: %w", err)
		}
		locked[p.ID] = p
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return OrderWithItems{}, fmt.Errorf("iterate locked products: %w", err)
	}

	// Verify all requested products exist and belong to the seller,
	// share currency, and have sufficient stock.
	var orderCurrency Currency
	for _, item := range in.Items {
		p, ok := locked[item.ProductID]
		if !ok {
			return OrderWithItems{}, ErrProductNotFound
		}
		if p.SellerID != in.SellerID {
			// Do not leak which seller owns it.
			return OrderWithItems{}, ErrProductNotFound
		}
		if orderCurrency == "" {
			orderCurrency = p.Currency
		} else if p.Currency != orderCurrency {
			return OrderWithItems{}, ErrMixedCurrency
		}
		if p.Stock < item.Quantity {
			return OrderWithItems{}, fmt.Errorf("%w: product %s has %d, requested %d",
				ErrInsufficientStock, p.Name, p.Stock, item.Quantity)
		}
	}

	// Compute totals.
	var subtotal int64
	for _, item := range in.Items {
		p := locked[item.ProductID]
		subtotal += p.Price * int64(item.Quantity)
	}
	total := subtotal + in.ShippingCents

	address := in.ShippingAddress
	if len(address) == 0 {
		address = json.RawMessage(`{}`)
	}

	// Insert order.
	const insertOrder = `
		INSERT INTO orders
			(buyer_id, seller_id, subtotal_cents, shipping_cents, total_cents,
			 currency, shipping_address, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING ` + orderColumns
	order, err := scanOrder(tx.QueryRow(ctx, insertOrder,
		in.BuyerID, in.SellerID, subtotal, in.ShippingCents, total,
		orderCurrency, address, in.Note,
	))
	if err != nil {
		return OrderWithItems{}, fmt.Errorf("insert order: %w", err)
	}

	// Insert order items and decrement stock.
	const insertItem = `
		INSERT INTO order_items
			(order_id, product_id, product_name, product_slug,
			 unit_price_cents, quantity, line_total_cents)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + orderItemColumns

	const decrementStock = `
		UPDATE products
		SET stock_quantity = stock_quantity - $1, updated_at = NOW()
		WHERE id = $2
	`

	items := make([]OrderItem, 0, len(in.Items))
	for _, item := range in.Items {
		p := locked[item.ProductID]
		lineTotal := p.Price * int64(item.Quantity)

		it, err := scanOrderItem(tx.QueryRow(ctx, insertItem,
			order.ID, p.ID, p.Name, p.Slug, p.Price, item.Quantity, lineTotal,
		))
		if err != nil {
			return OrderWithItems{}, fmt.Errorf("insert order item: %w", err)
		}
		items = append(items, it)

		if _, err := tx.Exec(ctx, decrementStock, item.Quantity, p.ID); err != nil {
			return OrderWithItems{}, fmt.Errorf("decrement stock: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return OrderWithItems{}, fmt.Errorf("commit tx: %w", err)
	}

	return OrderWithItems{Order: order, Items: items}, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (OrderWithItems, error) {
	const q = `SELECT ` + orderColumns + ` FROM orders WHERE id = $1`

	o, err := scanOrder(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return OrderWithItems{}, ErrOrderNotFound
		}
		return OrderWithItems{}, fmt.Errorf("get order: %w", err)
	}

	items, err := r.listItems(ctx, o.ID)
	if err != nil {
		return OrderWithItems{}, err
	}
	return OrderWithItems{Order: o, Items: items}, nil
}

func (r *Repository) listItems(ctx context.Context, orderID string) ([]OrderItem, error) {
	const q = `SELECT ` + orderItemColumns + `
	           FROM order_items WHERE order_id = $1 ORDER BY created_at`

	rows, err := r.pool.Query(ctx, q, orderID)
	if err != nil {
		return nil, fmt.Errorf("list order items: %w", err)
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		it, err := scanOrderItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// List returns a page of orders matching the filter.
func (r *Repository) List(ctx context.Context, f OrderFilter) (OrderPage, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	where := []string{"TRUE"}
	args := []any{}
	i := 1

	if f.BuyerID != nil {
		where = append(where, fmt.Sprintf("buyer_id = $%d", i))
		args = append(args, *f.BuyerID)
		i++
	}
	if f.SellerID != nil {
		where = append(where, fmt.Sprintf("seller_id = $%d", i))
		args = append(args, *f.SellerID)
		i++
	}
	if f.Status != nil {
		where = append(where, fmt.Sprintf("status = $%d", i))
		args = append(args, *f.Status)
		i++
	}
	if f.Cursor != "" {
		createdAt, id, err := decodeCursor(f.Cursor)
		if err != nil {
			return OrderPage{}, fmt.Errorf("invalid cursor: %w", err)
		}
		where = append(where, fmt.Sprintf("(created_at, id) < ($%d, $%d)", i, i+1))
		args = append(args, createdAt, id)
		i += 2
	}

	q := fmt.Sprintf(`
		SELECT %s FROM orders
		WHERE %s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d
	`, orderColumns, strings.Join(where, " AND "), i)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return OrderPage{}, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var items []Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return OrderPage{}, fmt.Errorf("scan order: %w", err)
		}
		items = append(items, o)
	}
	if err := rows.Err(); err != nil {
		return OrderPage{}, fmt.Errorf("iterate orders: %w", err)
	}

	page := OrderPage{Items: items}
	if len(items) > limit {
		last := items[limit-1]
		page.Items = items[:limit]
		page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	return page, nil
}

// transition applies a status change if the current status is in `allowed`.
// Returns ErrInvalidTransition if not allowed, ErrOrderNotFound if not found.
func (r *Repository) transition(
	ctx context.Context,
	id string,
	actorColumn string,
	actorID string,
	allowed []Status,
	to Status,
	extraSQL string,
) (Order, error) {
	// Verify current status and actor in a single query.
	checkQ := fmt.Sprintf(
		`SELECT status FROM orders WHERE id = $1 AND %s = $2`, actorColumn)
	var current Status
	if err := r.pool.QueryRow(ctx, checkQ, id, actorID).Scan(&current); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Order{}, ErrOrderNotFound
		}
		return Order{}, fmt.Errorf("check status: %w", err)
	}

	ok := false
	for _, s := range allowed {
		if s == current {
			ok = true
			break
		}
	}
	if !ok {
		return Order{}, fmt.Errorf("%w: from %s to %s", ErrInvalidTransition, current, to)
	}

	updateQ := fmt.Sprintf(`
		UPDATE orders SET status = $1, updated_at = NOW()%s
		WHERE id = $2 AND %s = $3
		RETURNING %s
	`, extraSQL, actorColumn, orderColumns)

	o, err := scanOrder(r.pool.QueryRow(ctx, updateQ, to, id, actorID))
	if err != nil {
		return Order{}, fmt.Errorf("update order: %w", err)
	}
	return o, nil
}

// MarkPaid transitions pending_payment -> paid. Called by the seller
// (or payment webhook, in a future step).
func (r *Repository) MarkPaid(ctx context.Context, id, actorID string) (Order, error) {
	return r.transition(ctx, id, "buyer_id", actorID,
		[]Status{StatusPendingPayment}, StatusPaid, ", paid_at = NOW()")
}

// MarkShipped transitions paid -> shipped. Only the seller may ship.
func (r *Repository) MarkShipped(ctx context.Context, id, sellerID string) (Order, error) {
	return r.transition(ctx, id, "seller_id", sellerID,
		[]Status{StatusPaid}, StatusShipped, ", shipped_at = NOW()")
}

// MarkDelivered transitions shipped -> delivered. Only the buyer may confirm delivery.
func (r *Repository) MarkDelivered(ctx context.Context, id, buyerID string) (Order, error) {
	return r.transition(ctx, id, "buyer_id", buyerID,
		[]Status{StatusShipped}, StatusDelivered, ", delivered_at = NOW()")
}

// Complete transitions delivered -> completed. Buyer confirms receipt.
func (r *Repository) Complete(ctx context.Context, id, buyerID string) (Order, error) {
	return r.transition(ctx, id, "buyer_id", buyerID,
		[]Status{StatusDelivered}, StatusCompleted, ", completed_at = NOW()")
}

// Cancel transitions any non-terminal state -> cancelled.
// The buyer may cancel before shipping.
func (r *Repository) Cancel(ctx context.Context, id, buyerID, reason string) (Order, error) {
	const checkQ = `SELECT status FROM orders WHERE id = $1 AND buyer_id = $2`
	var current Status
	if err := r.pool.QueryRow(ctx, checkQ, id, buyerID).Scan(&current); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Order{}, ErrOrderNotFound
		}
		return Order{}, fmt.Errorf("check status: %w", err)
	}

	switch current {
	case StatusPendingPayment, StatusPaid:
		// allowed
	default:
		return Order{}, fmt.Errorf("%w: cannot cancel from %s", ErrInvalidTransition, current)
	}

	const q = `
		UPDATE orders
		SET status = 'cancelled',
		    cancelled_reason = $1,
		    cancelled_at = NOW(),
		    updated_at = NOW()
		WHERE id = $2 AND buyer_id = $3
		RETURNING ` + orderColumns
	o, err := scanOrder(r.pool.QueryRow(ctx, q, reason, id, buyerID))
	if err != nil {
		return Order{}, fmt.Errorf("cancel order: %w", err)
	}
	return o, nil
}

func isInvalidUUID(err error) bool {
	if err == nil {
		return false
	}
	// pgx wraps PG errors; check string contains the SQLSTATE.
	return strings.Contains(err.Error(), "22P02") ||
		strings.Contains(err.Error(), "invalid input syntax for type uuid")
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
