-- Categories: adjacency list. Root categories have parent_id = NULL.
CREATE TABLE categories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id   UUID REFERENCES categories(id) ON DELETE RESTRICT,
    slug        CITEXT NOT NULL UNIQUE,
    name_en     TEXT NOT NULL,
    name_ru     TEXT NOT NULL,
    name_es     TEXT NOT NULL,
    sort_order  INT NOT NULL DEFAULT 0,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CHECK (id <> parent_id)
);

CREATE INDEX idx_categories_parent ON categories (parent_id) WHERE is_active = TRUE;
CREATE INDEX idx_categories_sort ON categories (parent_id, sort_order) WHERE is_active = TRUE;

-- Products: created by sellers, belong to a category.
CREATE TABLE products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id       UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    category_id     UUID NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    slug            CITEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    price_cents     BIGINT NOT NULL CHECK (price_cents >= 0),
    currency        TEXT NOT NULL CHECK (currency IN ('RUB', 'USD', 'EUR')),
    stock_quantity  INT NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    status          TEXT NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'active', 'archived')),
    deleted_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The most important index: catalog listing.
-- Buyers see active, non-deleted products in a category, sorted by recency.
CREATE INDEX idx_products_category_active
    ON products (category_id, created_at DESC)
    WHERE status = 'active' AND deleted_at IS NULL;

-- Seller's own product list (dashboard view).
CREATE INDEX idx_products_seller
    ON products (seller_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- Lookup by slug for URL routing.
CREATE INDEX idx_products_slug
    ON products (slug)
    WHERE deleted_at IS NULL;

-- Full-text search over name + description. Added now so we don't
-- need a migration later just to add it.
CREATE INDEX idx_products_search
    ON products
    USING GIN (to_tsvector('simple', name || ' ' || description))
    WHERE status = 'active' AND deleted_at IS NULL;
