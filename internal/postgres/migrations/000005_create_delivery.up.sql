-- Pickup points: our own lockers, offices, and terminals.
CREATE TABLE pickup_points (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id     TEXT NOT NULL UNIQUE,
    type            TEXT NOT NULL
                    CHECK (type IN ('locker', 'office', 'terminal')),
    name            TEXT NOT NULL,
    address         TEXT NOT NULL,
    city            TEXT NOT NULL,
    region          TEXT NOT NULL DEFAULT '',
    postal_code     TEXT NOT NULL DEFAULT '',
    country         TEXT NOT NULL DEFAULT 'RU',
    latitude        DOUBLE PRECISION,
    longitude       DOUBLE PRECISION,
    work_hours      JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_pickup_points_city ON pickup_points (city) WHERE is_active = TRUE;
CREATE INDEX idx_pickup_points_coords ON pickup_points (latitude, longitude)
    WHERE is_active = TRUE AND latitude IS NOT NULL;


-- Deliveries: one per order.
CREATE TABLE deliveries (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id              UUID NOT NULL UNIQUE REFERENCES orders(id) ON DELETE RESTRICT,
    tracking_number       TEXT NOT NULL UNIQUE,
    pickup_code           TEXT NOT NULL,
    status                TEXT NOT NULL DEFAULT 'created'
                          CHECK (status IN (
                              'created',
                              'awaiting_dispatch',
                              'in_transit',
                              'arrived_at_hub',
                              'out_for_delivery',
                              'ready_for_pickup',
                              'picked_up',
                              'returned',
                              'cancelled'
                          )),
    pickup_point_id       UUID REFERENCES pickup_points(id) ON DELETE SET NULL,
    destination_name      TEXT NOT NULL,
    destination_address   TEXT NOT NULL,
    destination_city      TEXT NOT NULL,
    destination_country   TEXT NOT NULL,
    origin_name           TEXT NOT NULL DEFAULT '',
    origin_address        TEXT NOT NULL DEFAULT '',
    origin_city           TEXT NOT NULL DEFAULT '',
    shipped_at            TIMESTAMPTZ,
    arrived_at            TIMESTAMPTZ,
    ready_for_pickup_at   TIMESTAMPTZ,
    picked_up_at          TIMESTAMPTZ,
    returned_at           TIMESTAMPTZ,
    cancelled_at          TIMESTAMPTZ,
    expires_at            TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deliveries_status ON deliveries (status, created_at DESC);
CREATE INDEX idx_deliveries_tracking ON deliveries (tracking_number);


-- Delivery events: append-only audit trail.
CREATE TABLE delivery_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id     UUID NOT NULL REFERENCES deliveries(id) ON DELETE CASCADE,
    status          TEXT NOT NULL,
    message         TEXT NOT NULL DEFAULT '',
    location        TEXT NOT NULL DEFAULT '',
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_delivery_events_delivery ON delivery_events (delivery_id, occurred_at);
