package orders

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusPendingPayment Status = "pending_payment"
	StatusPaid           Status = "paid"
	StatusShipped        Status = "shipped"
	StatusDelivered      Status = "delivered"
	StatusCompleted      Status = "completed"
	StatusCancelled      Status = "cancelled"
	StatusRefunded       Status = "refunded"
)

type Currency string

const (
	CurrencyRUB Currency = "RUB"
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
)

type Order struct {
	ID              string
	OrderNumber     string
	BuyerID         string
	SellerID        string
	Status          Status
	SubtotalCents   int64
	ShippingCents   int64
	TotalCents      int64
	Currency        Currency
	ShippingAddress json.RawMessage
	Note            string
	CancelledReason *string
	CancelledAt     *time.Time
	PaidAt          *time.Time
	ShippedAt       *time.Time
	DeliveredAt     *time.Time
	CompletedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type OrderItem struct {
	ID             string
	OrderID        string
	ProductID      string
	ProductName    string
	ProductSlug    string
	UnitPriceCents int64
	Quantity       int
	LineTotalCents int64
	CreatedAt      time.Time
}

type OrderWithItems struct {
	Order
	Items []OrderItem
}

// CheckoutItem is the caller's request to buy a specific quantity of a product.
type CheckoutItem struct {
	ProductID string
	Quantity  int
}

// CheckoutInput is everything needed to create an order from one seller.
type CheckoutInput struct {
	BuyerID         string
	SellerID        string
	Items           []CheckoutItem
	ShippingAddress json.RawMessage
	ShippingCents   int64
	Note            string
}

type OrderFilter struct {
	BuyerID  *string
	SellerID *string
	Status   *Status
	Cursor   string
	Limit    int
}

type OrderPage struct {
	Items      []Order
	NextCursor string
}
