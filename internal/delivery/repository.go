package delivery

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

const deliveryColumns = `
	id, order_id, tracking_number, pickup_code, status,
	pickup_point_id, origin_address_id,
	destination_name, destination_address, destination_city, destination_country,
	origin_name, origin_address, origin_city,
	shipped_at, arrived_at, ready_for_pickup_at, picked_up_at, returned_at, cancelled_at,
	expires_at,
	created_at, updated_at
`

func scanDelivery(row pgx.Row) (Delivery, error) {
	var d Delivery
	err := row.Scan(
		&d.ID, &d.OrderID, &d.TrackingNumber, &d.PickupCode, &d.Status,
		&d.PickupPointID, &d.OriginAddressID,
		&d.DestinationName, &d.DestinationAddress, &d.DestinationCity, &d.DestinationCountry,
		&d.OriginName, &d.OriginAddress, &d.OriginCity,
		&d.ShippedAt, &d.ArrivedAt, &d.ReadyForPickupAt, &d.PickedUpAt,
		&d.ReturnedAt, &d.CancelledAt,
		&d.ExpiresAt,
		&d.CreatedAt, &d.UpdatedAt,
	)
	return d, err
}

const deliveryEventColumns = `
	id, delivery_id, status, message, location, occurred_at, created_at
`

func scanDeliveryEvent(row pgx.Row) (DeliveryEvent, error) {
	var e DeliveryEvent
	err := row.Scan(
		&e.ID, &e.DeliveryID, &e.Status, &e.Message, &e.Location,
		&e.OccurredAt, &e.CreatedAt,
	)
	return e, err
}

// Create inserts a new delivery in status "created" and writes the first event.
func (r *Repository) Create(ctx context.Context, in CreateInput) (Delivery, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Delivery{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const insert = `
		INSERT INTO deliveries
			(order_id, tracking_number, pickup_code, status,
			 pickup_point_id, origin_address_id,
			 destination_name, destination_address, destination_city, destination_country,
			 origin_name, origin_address, origin_city)
		VALUES ($1, $2, $3, 'created', $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING ` + deliveryColumns
	d, err := scanDelivery(tx.QueryRow(ctx, insert,
		in.OrderID, in.TrackingNumber, in.PickupCode,
		in.PickupPointID, in.OriginAddressID,
		in.DestinationName, in.DestinationAddress, in.DestinationCity, in.DestinationCountry,
		in.OriginName, in.OriginAddress, in.OriginCity,
	))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				if strings.Contains(pgErr.ConstraintName, "order_id") {
					return Delivery{}, ErrDeliveryExists
				}
				if strings.Contains(pgErr.ConstraintName, "tracking_number") {
					return Delivery{}, ErrTrackingExists
				}
			case "23503":
				if strings.Contains(pgErr.ConstraintName, "pickup_point") {
					return Delivery{}, ErrPickupPointNotFound
				}
			}
		}
		return Delivery{}, fmt.Errorf("insert delivery: %w", err)
	}

	const insertEvent = `
		INSERT INTO delivery_events (delivery_id, status, message, location)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := tx.Exec(ctx, insertEvent,
		d.ID, StatusCreated, "Delivery created", in.OriginCity); err != nil {
		return Delivery{}, fmt.Errorf("insert delivery event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Delivery{}, fmt.Errorf("commit tx: %w", err)
	}
	return d, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (DeliveryWithEvents, error) {
	const q = `SELECT ` + deliveryColumns + ` FROM deliveries WHERE id = $1`

	d, err := scanDelivery(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return DeliveryWithEvents{}, ErrDeliveryNotFound
		}
		return DeliveryWithEvents{}, fmt.Errorf("get delivery: %w", err)
	}

	events, err := r.listEvents(ctx, d.ID)
	if err != nil {
		return DeliveryWithEvents{}, err
	}
	return DeliveryWithEvents{Delivery: d, Events: events}, nil
}

func (r *Repository) GetByTrackingNumber(ctx context.Context, tn string) (DeliveryWithEvents, error) {
	const q = `SELECT ` + deliveryColumns + ` FROM deliveries WHERE tracking_number = $1`

	d, err := scanDelivery(r.pool.QueryRow(ctx, q, tn))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DeliveryWithEvents{}, ErrDeliveryNotFound
		}
		return DeliveryWithEvents{}, fmt.Errorf("get delivery by tracking: %w", err)
	}

	events, err := r.listEvents(ctx, d.ID)
	if err != nil {
		return DeliveryWithEvents{}, err
	}
	return DeliveryWithEvents{Delivery: d, Events: events}, nil
}

func (r *Repository) GetByOrderID(ctx context.Context, orderID string) (DeliveryWithEvents, error) {
	const q = `SELECT ` + deliveryColumns + ` FROM deliveries WHERE order_id = $1`

	d, err := scanDelivery(r.pool.QueryRow(ctx, q, orderID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return DeliveryWithEvents{}, ErrDeliveryNotFound
		}
		return DeliveryWithEvents{}, fmt.Errorf("get delivery by order: %w", err)
	}

	events, err := r.listEvents(ctx, d.ID)
	if err != nil {
		return DeliveryWithEvents{}, err
	}
	return DeliveryWithEvents{Delivery: d, Events: events}, nil
}

func (r *Repository) listEvents(ctx context.Context, deliveryID string) ([]DeliveryEvent, error) {
	const q = `SELECT ` + deliveryEventColumns + `
	           FROM delivery_events WHERE delivery_id = $1
	           ORDER BY occurred_at, id`

	rows, err := r.pool.Query(ctx, q, deliveryID)
	if err != nil {
		return nil, fmt.Errorf("list delivery events: %w", err)
	}
	defer rows.Close()

	var events []DeliveryEvent
	for rows.Next() {
		e, err := scanDeliveryEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// VerifyPickupCode checks that the given code matches the delivery's pickup code.
// Only works when the delivery is in ready_for_pickup status.
func (r *Repository) VerifyPickupCode(ctx context.Context, deliveryID, code string) error {
	const q = `SELECT pickup_code FROM deliveries
	           WHERE id = $1 AND status = 'ready_for_pickup'`

	var stored string
	err := r.pool.QueryRow(ctx, q, deliveryID).Scan(&stored)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return ErrDeliveryNotReady
		}
		return fmt.Errorf("lookup pickup code: %w", err)
	}

	if !constantTimeEqual(stored, code) {
		return ErrInvalidPickupCode
	}
	return nil
}

// transition moves a delivery to a new status and records the event.
// It only succeeds if the current status is in `allowed`.
func (r *Repository) transition(
	ctx context.Context,
	id string,
	allowed []Status,
	to Status,
	message, location string,
	timestampColumn string,
	expiresInDays int,
) (Delivery, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Delivery{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const lockQ = `SELECT status FROM deliveries WHERE id = $1 FOR UPDATE`
	var current Status
	if err := tx.QueryRow(ctx, lockQ, id).Scan(&current); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return Delivery{}, ErrDeliveryNotFound
		}
		return Delivery{}, fmt.Errorf("lock delivery: %w", err)
	}

	ok := false
	for _, s := range allowed {
		if s == current {
			ok = true
			break
		}
	}
	if !ok {
		return Delivery{}, fmt.Errorf("%w: from %s to %s", ErrInvalidTransition, current, to)
	}

	updateQ := `UPDATE deliveries SET status = $1, updated_at = NOW()`
	args := []any{to}
	i := 2
	if timestampColumn != "" {
		updateQ += fmt.Sprintf(", %s = NOW()", timestampColumn)
	}
	if expiresInDays > 0 {
		updateQ += fmt.Sprintf(", expires_at = NOW() + INTERVAL '%d days'", expiresInDays)
	}
	updateQ += fmt.Sprintf(` WHERE id = $%d RETURNING `, i) + deliveryColumns
	args = append(args, id)

	d, err := scanDelivery(tx.QueryRow(ctx, updateQ, args...))
	if err != nil {
		return Delivery{}, fmt.Errorf("update delivery: %w", err)
	}

	const insertEvent = `
		INSERT INTO delivery_events (delivery_id, status, message, location)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := tx.Exec(ctx, insertEvent, d.ID, to, message, location); err != nil {
		return Delivery{}, fmt.Errorf("insert event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Delivery{}, fmt.Errorf("commit tx: %w", err)
	}
	return d, nil
}

func (r *Repository) MarkAwaitingDispatch(ctx context.Context, id, message, location string) (Delivery, error) {
	return r.transition(ctx, id,
		[]Status{StatusCreated},
		StatusAwaitingDispatch,
		message, location,
		"", 0)
}

func (r *Repository) MarkInTransit(ctx context.Context, id, message, location string) (Delivery, error) {
	return r.transition(ctx, id,
		[]Status{StatusAwaitingDispatch},
		StatusInTransit,
		message, location,
		"shipped_at", 0)
}

func (r *Repository) MarkArrivedAtHub(ctx context.Context, id, message, location string) (Delivery, error) {
	return r.transition(ctx, id,
		[]Status{StatusInTransit},
		StatusArrivedAtHub,
		message, location,
		"arrived_at", 0)
}

func (r *Repository) MarkOutForDelivery(ctx context.Context, id, message, location string) (Delivery, error) {
	return r.transition(ctx, id,
		[]Status{StatusArrivedAtHub},
		StatusOutForDelivery,
		message, location,
		"", 0)
}

func (r *Repository) MarkReadyForPickup(ctx context.Context, id, message, location string) (Delivery, error) {
	return r.transition(ctx, id,
		[]Status{StatusOutForDelivery},
		StatusReadyForPickup,
		message, location,
		"ready_for_pickup_at", 0)
}

// MarkPickedUp completes the delivery. Sets expires_at 90 days out.
func (r *Repository) MarkPickedUp(ctx context.Context, id, message, location string) (Delivery, error) {
	return r.transition(ctx, id,
		[]Status{StatusReadyForPickup},
		StatusPickedUp,
		message, location,
		"picked_up_at", 90)
}

// MarkReturned is called when a package is not picked up within the hold period.
func (r *Repository) MarkReturned(ctx context.Context, id, message, location string) (Delivery, error) {
	return r.transition(ctx, id,
		[]Status{StatusReadyForPickup},
		StatusReturned,
		message, location,
		"returned_at", 90)
}

// Cancel can be called from any non-terminal state.
func (r *Repository) Cancel(ctx context.Context, id, message, location string) (Delivery, error) {
	return r.transition(ctx, id,
		[]Status{StatusCreated, StatusAwaitingDispatch, StatusInTransit,
			StatusArrivedAtHub, StatusOutForDelivery, StatusReadyForPickup},
		StatusCancelled,
		message, location,
		"cancelled_at", 0)
}

// constantTimeEqual compares two strings without short-circuiting.
func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

// isInvalidUUID reports whether a pg error is a malformed UUID input.
func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}
