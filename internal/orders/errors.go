package orders

import "errors"

var (
	ErrOrderNotFound      = errors.New("order not found")
	ErrProductNotFound    = errors.New("product not found")
	ErrInsufficientStock  = errors.New("insufficient stock")
	ErrEmptyOrder         = errors.New("order has no items")
	ErrInvalidTransition  = errors.New("invalid status transition")
	ErrNotBuyer           = errors.New("you are not the buyer of this order")
	ErrNotSeller          = errors.New("you are not the seller of this order")
	ErrMixedCurrency      = errors.New("all products in an order must share the same currency")
	ErrInvalidQuantity    = errors.New("quantity must be greater than zero")
)
