package delivery

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusCreated          Status = "created"
	StatusAwaitingDispatch Status = "awaiting_dispatch"
	StatusInTransit        Status = "in_transit"
	StatusArrivedAtHub     Status = "arrived_at_hub"
	StatusOutForDelivery   Status = "out_for_delivery"
	StatusReadyForPickup   Status = "ready_for_pickup"
	StatusPickedUp         Status = "picked_up"
	StatusReturned         Status = "returned"
	StatusCancelled        Status = "cancelled"
)

// IsTerminal reports whether no further transitions are expected.
func (s Status) IsTerminal() bool {
	switch s {
	case StatusPickedUp, StatusReturned, StatusCancelled:
		return true
	}
	return false
}

type PickupPointType string

const (
	PickupPointLocker   PickupPointType = "locker"
	PickupPointOffice   PickupPointType = "office"
	PickupPointTerminal PickupPointType = "terminal"
)

type PickupPoint struct {
	ID         string
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
	IsActive   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Delivery struct {
	ID                 string
	OrderID            string
	TrackingNumber     string
	PickupCode         string
	Status             Status
	PickupPointID      *string
	OriginAddressID    *string
	DestinationName    string
	DestinationAddress string
	DestinationCity    string
	DestinationCountry string
	OriginName         string
	OriginAddress      string
	OriginCity         string
	ShippedAt          *time.Time
	ArrivedAt          *time.Time
	ReadyForPickupAt   *time.Time
	PickedUpAt         *time.Time
	ReturnedAt         *time.Time
	CancelledAt        *time.Time
	ExpiresAt          *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type DeliveryEvent struct {
	ID         string
	DeliveryID string
	Status     Status
	Message    string
	Location   string
	OccurredAt time.Time
	CreatedAt  time.Time
}

type DeliveryWithEvents struct {
	Delivery
	Events []DeliveryEvent
}

// CreateInput describes a new delivery.
type CreateInput struct {
	OrderID            string
	PickupPointID      string
	OriginAddressID    *string
	DestinationName    string
	DestinationAddress string
	DestinationCity    string
	DestinationCountry string
	OriginName         string
	OriginAddress      string
	OriginCity         string
	TrackingNumber     string
	PickupCode         string
}
