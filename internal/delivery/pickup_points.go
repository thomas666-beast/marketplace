package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PickupPointRepository struct {
	pool *pgxpool.Pool
}

func NewPickupPointRepository(pool *pgxpool.Pool) *PickupPointRepository {
	return &PickupPointRepository{pool: pool}
}

const pickupPointColumns = `
	id, external_id, type, name, address, city, region, postal_code, country,
	latitude, longitude, work_hours, is_active, created_at, updated_at
`

func scanPickupPoint(row pgx.Row) (PickupPoint, error) {
	var p PickupPoint
	var workHours []byte
	err := row.Scan(
		&p.ID, &p.ExternalID, &p.Type, &p.Name, &p.Address,
		&p.City, &p.Region, &p.PostalCode, &p.Country,
		&p.Latitude, &p.Longitude, &workHours,
		&p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return PickupPoint{}, err
	}
	p.WorkHours = json.RawMessage(workHours)
	return p, nil
}

// ListByCity returns active pickup points in a city, ordered by type then name.
func (r *PickupPointRepository) ListByCity(ctx context.Context, city string) ([]PickupPoint, error) {
	const q = `
		SELECT ` + pickupPointColumns + `
		FROM pickup_points
		WHERE is_active = TRUE AND LOWER(city) = LOWER($1)
		ORDER BY type, name
	`

	rows, err := r.pool.Query(ctx, q, strings.TrimSpace(city))
	if err != nil {
		return nil, fmt.Errorf("list pickup points by city: %w", err)
	}
	defer rows.Close()

	return collectPickupPoints(rows)
}

type NearbyInput struct {
	Latitude   float64
	Longitude  float64
	RadiusKm   float64
	MaxResults int
}

// ListNearby returns active pickup points within RadiusKm of the given coordinates,
// ordered by distance ascending.
func (r *PickupPointRepository) ListNearby(ctx context.Context, in NearbyInput) ([]PickupPoint, error) {
	if in.MaxResults <= 0 {
		in.MaxResults = 20
	}
	if in.MaxResults > 100 {
		in.MaxResults = 100
	}
	if in.RadiusKm <= 0 {
		in.RadiusKm = 5
	}

	const kmPerDegLat = 111.0
	latDelta := in.RadiusKm / kmPerDegLat

	cosLat := 0.01
	if in.Latitude > -89 && in.Latitude < 89 {
		cosLat = math.Cos(in.Latitude * math.Pi / 180)
		if cosLat < 0.01 {
			cosLat = 0.01
		}
	}
	lonDelta := in.RadiusKm / (kmPerDegLat * cosLat)

	const q = `
		SELECT ` + pickupPointColumns + `
		FROM pickup_points
		WHERE is_active = TRUE
		  AND latitude IS NOT NULL
		  AND longitude IS NOT NULL
		  AND latitude BETWEEN $1 AND $2
		  AND longitude BETWEEN $3 AND $4
		ORDER BY
		  (latitude - $5) * (latitude - $5) +
		  (longitude - $6) * (longitude - $6) * $7
		LIMIT $8
	`

	rows, err := r.pool.Query(ctx, q,
		in.Latitude-latDelta, in.Latitude+latDelta,
		in.Longitude-lonDelta, in.Longitude+lonDelta,
		in.Latitude, in.Longitude,
		cosLat*cosLat,
		in.MaxResults,
	)
	if err != nil {
		return nil, fmt.Errorf("list nearby pickup points: %w", err)
	}
	defer rows.Close()

	return collectPickupPoints(rows)
}

func (r *PickupPointRepository) GetByID(ctx context.Context, id string) (PickupPoint, error) {
	const q = `SELECT ` + pickupPointColumns + `
	           FROM pickup_points WHERE id = $1 AND is_active = TRUE`

	p, err := scanPickupPoint(r.pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || isInvalidUUID(err) {
			return PickupPoint{}, ErrPickupPointNotFound
		}
		return PickupPoint{}, fmt.Errorf("get pickup point: %w", err)
	}
	return p, nil
}

func (r *PickupPointRepository) GetByExternalID(ctx context.Context, externalID string) (PickupPoint, error) {
	const q = `SELECT ` + pickupPointColumns + `
	           FROM pickup_points WHERE external_id = $1 AND is_active = TRUE`

	p, err := scanPickupPoint(r.pool.QueryRow(ctx, q, externalID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PickupPoint{}, ErrPickupPointNotFound
		}
		return PickupPoint{}, fmt.Errorf("get pickup point by external id: %w", err)
	}
	return p, nil
}

type CreatePickupPointInput struct {
	ExternalID string
	Type       PickupPointType
	Name       string
	Address    string
	City       string
	Region     string
	PostalCode string
	Country    string
	Latitude   *float64
	Longitude  *float64
	WorkHours  json.RawMessage
}

func (r *PickupPointRepository) Create(ctx context.Context, in CreatePickupPointInput) (PickupPoint, error) {
	if len(in.WorkHours) == 0 {
		in.WorkHours = json.RawMessage(`{}`)
	}
	if in.Country == "" {
		in.Country = "RU"
	}

	const q = `
		INSERT INTO pickup_points
			(external_id, type, name, address, city, region, postal_code, country,
			 latitude, longitude, work_hours)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING ` + pickupPointColumns
	p, err := scanPickupPoint(r.pool.QueryRow(ctx, q,
		in.ExternalID, in.Type, in.Name, in.Address, in.City, in.Region,
		in.PostalCode, in.Country, in.Latitude, in.Longitude, in.WorkHours,
	))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return PickupPoint{}, fmt.Errorf("pickup point external_id already exists")
		}
		return PickupPoint{}, fmt.Errorf("create pickup point: %w", err)
	}
	return p, nil
}

func (r *PickupPointRepository) Deactivate(ctx context.Context, id string) error {
	const q = `UPDATE pickup_points SET is_active = FALSE, updated_at = NOW()
	           WHERE id = $1 AND is_active = TRUE`

	tag, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		if isInvalidUUID(err) {
			return ErrPickupPointNotFound
		}
		return fmt.Errorf("deactivate pickup point: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPickupPointNotFound
	}
	return nil
}

func collectPickupPoints(rows pgx.Rows) ([]PickupPoint, error) {
	var points []PickupPoint
	for rows.Next() {
		p, err := scanPickupPoint(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pickup point: %w", err)
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pickup points: %w", err)
	}
	return points, nil
}
