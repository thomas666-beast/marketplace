CREATE TABLE seller_addresses (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    label        TEXT NOT NULL DEFAULT '',
    contact_name TEXT NOT NULL,
    phone        TEXT NOT NULL,
    address      TEXT NOT NULL,
    city         TEXT NOT NULL,
    region       TEXT NOT NULL DEFAULT '',
    postal_code  TEXT NOT NULL DEFAULT '',
    country      TEXT NOT NULL DEFAULT 'RU',
    is_default   BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seller can only see and manage their own addresses.
CREATE INDEX idx_seller_addresses_seller
    ON seller_addresses (seller_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- At most one default address per seller.
CREATE UNIQUE INDEX idx_seller_addresses_one_default
    ON seller_addresses (seller_id)
    WHERE is_default = TRUE AND deleted_at IS NULL;

-- Fast ownership check for the dispatch flow: "does address X belong to seller Y?"
CREATE UNIQUE INDEX idx_seller_addresses_ownership
    ON seller_addresses (seller_id, id)
    WHERE deleted_at IS NULL;

-- Deliveries now reference an origin address when one is set.
ALTER TABLE deliveries
    ADD COLUMN origin_address_id UUID REFERENCES seller_addresses(id) ON DELETE SET NULL;

CREATE INDEX idx_deliveries_origin_address
    ON deliveries (origin_address_id)
    WHERE origin_address_id IS NOT NULL;
