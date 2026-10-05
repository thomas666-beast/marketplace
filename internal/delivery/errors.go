package delivery

import "errors"

var (
	ErrDeliveryNotFound     = errors.New("delivery not found")
	ErrPickupPointNotFound  = errors.New("pickup point not found")
	ErrPickupPointInactive  = errors.New("pickup point is not active")
	ErrDeliveryExists       = errors.New("order already has a delivery")
	ErrInvalidTransition    = errors.New("invalid status transition")
	ErrInvalidPickupCode    = errors.New("invalid pickup code")
	ErrDeliveryNotReady     = errors.New("delivery is not ready for pickup")
	ErrTrackingExists       = errors.New("tracking number already exists")
	ErrNoPickupPointsInCity = errors.New("no active pickup points in this city")
)
