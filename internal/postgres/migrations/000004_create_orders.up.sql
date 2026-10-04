-- Sequence for human-readable order numbers.
CREATE SEQUENCE IF NOT EXISTS order_number_seq START 1000;

CREATE TABLE orders (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_number      TEXT NOT NULL UNIQUE
                      DEFAULT ('ORD-' || LPAD(nextval('order_number_seq')::TEXT, 8, '0')),
    buyer_id          UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    seller_id         UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status            TEXT NOT NULL DEFAULT 'pending_payment'
                      CHECK (status IN (
                          'pending_payment',
                          'paid',
                          'shipped',
                          'delivered',
                          'completed',
                          'cancelled',
                          'refunded'
                      )),
    subtotal_cents    BIGINT NOT NULL CHECK (subtotal_cents >= 0),
    shipping_cents    BIGINT NOT NULL DEFAULT 0 CHECK (shipping_cents >= 0),
    total_cents       BIGINT NOT NULL CHECK (total_cents >= 0),
    currency          TEXT NOT NULL CHECK (currency IN ('RUB', 'USD', 'EUR')),
    shipping_address  JSONB NOT NULL DEFAULT '{}'::jsonb,
    note              TEXT NOT NULL DEFAULT '',
    cancelled_reason  TEXT,
    cancelled_at      TIMESTAMPTZ,
    paid_at           TIMESTAMPTZ,
    shipped_at        TIMESTAMPTZ,
    delivered_at      TIMESTAMPTZ,
    completed_at      TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (buyer_id <> seller_id),
    CHECK (total_cents = subtotal_cents + shipping_cents)
);

-- Buyer's order history: newest first, excludes nothing.
CREATE INDEX idx_orders_buyer ON orders (buyer_id, created_at DESC);

-- Seller's order dashboard: newest first, filterable by status.
CREATE INDEX idx_orders_seller_status ON orders (seller_id, status, created_at DESC);

CREATE TABLE order_items (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id          UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id        UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name      TEXT NOT NULL,
    product_slug      TEXT NOT NULL,
    unit_price_cents  BIGINT NOT NULL CHECK (unit_price_cents >= 0),
    quantity          INT NOT NULL CHECK (quantity > 0),
    line_total_cents  BIGINT NOT NULL CHECK (line_total_cents >= 0),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (line_total_cents = unit_price_cents * quantity)
);

CREATE INDEX idx_order_items_order ON order_items (order_id);
CREATE INDEX idx_order_items_product ON order_items (product_id);
