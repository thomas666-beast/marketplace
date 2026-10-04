package catalog

import "time"

type ProductStatus string

const (
	StatusDraft    ProductStatus = "draft"
	StatusActive   ProductStatus = "active"
	StatusArchived ProductStatus = "archived"
)

type Currency string

const (
	CurrencyRUB Currency = "RUB"
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
)

type Product struct {
	ID            string
	SellerID      string
	CategoryID    string
	Slug          string
	Name          string
	Description   string
	PriceCents    int64
	Currency      Currency
	StockQuantity int
	Status        ProductStatus
	DeletedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CreateProductInput struct {
	SellerID      string
	CategoryID    string
	Name          string
	Description   string
	PriceCents    int64
	Currency      Currency
	StockQuantity int
	Status        ProductStatus
}

type UpdateProductInput struct {
	Name          *string
	Description   *string
	PriceCents    *int64
	Currency      *Currency
	StockQuantity *int
	Status        *ProductStatus
}

// ProductFilter drives listing queries.
// All fields are optional. Zero value means "no filter".
type ProductFilter struct {
	CategoryID *string
	SellerID   *string
	Search     string
	// Cursor for keyset pagination. Empty means "first page".
	Cursor string
	// Limit for a single page. Defaults to 20, capped at 100.
	Limit int
}

// ProductPage is what a listing endpoint returns.
type ProductPage struct {
	Items      []Product
	NextCursor string
}
