package httpserver

import (
	"encoding/json"
	"time"

	"github.com/thomas666-beast/marketplace/internal/delivery"
)

type pickupPointResponse struct {
	ID         string          `json:"id"`
	ExternalID string          `json:"external_id"`
	Type       string          `json:"type"`
	Name       string          `json:"name"`
	Address    string          `json:"address"`
	City       string          `json:"city"`
	Region     string          `json:"region"`
	PostalCode string          `json:"postal_code"`
	Country    string          `json:"country"`
	Latitude   *float64        `json:"latitude,omitempty"`
	Longitude  *float64        `json:"longitude,omitempty"`
	WorkHours  json.RawMessage `json:"work_hours"`
}

func toPickupPointResponse(p delivery.PickupPoint) pickupPointResponse {
	return pickupPointResponse{
		ID:         p.ID,
		ExternalID: p.ExternalID,
		Type:       string(p.Type),
		Name:       p.Name,
		Address:    p.Address,
		City:       p.City,
		Region:     p.Region,
		PostalCode: p.PostalCode,
		Country:    p.Country,
		Latitude:   p.Latitude,
		Longitude:  p.Longitude,
		WorkHours:  p.WorkHours,
	}
}

type pickupPointListResponse struct {
	Items []pickupPointResponse `json:"items"`
}

type deliveryEventResponse struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	Location   string `json:"location"`
	OccurredAt string `json:"occurred_at"`
}

func toDeliveryEventResponse(e delivery.DeliveryEvent) deliveryEventResponse {
	return deliveryEventResponse{
		ID:         e.ID,
		Status:     string(e.Status),
		Message:    e.Message,
		Location:   e.Location,
		OccurredAt: e.OccurredAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// trackingResponse is what the public tracking endpoint returns.
// Note: no pickup_code. That is only available to the authenticated buyer.
type trackingResponse struct {
	ID                 string                  `json:"id"`
	TrackingNumber     string                  `json:"tracking_number"`
	Status             string                  `json:"status"`
	DestinationName    string                  `json:"destination_name"`
	DestinationAddress string                  `json:"destination_address"`
	DestinationCity    string                  `json:"destination_city"`
	OriginCity         string                  `json:"origin_city"`
	ShippedAt          *string                 `json:"shipped_at,omitempty"`
	ArrivedAt          *string                 `json:"arrived_at,omitempty"`
	ReadyForPickupAt   *string                 `json:"ready_for_pickup_at,omitempty"`
	PickedUpAt         *string                 `json:"picked_up_at,omitempty"`
	ReturnedAt         *string                 `json:"returned_at,omitempty"`
	CancelledAt        *string                 `json:"cancelled_at,omitempty"`
	CreatedAt          string                  `json:"created_at"`
	UpdatedAt          string                  `json:"updated_at"`
	Events             []deliveryEventResponse `json:"events"`
}

func fmtTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format("2006-01-02T15:04:05Z")
	return &s
}

func toTrackingResponse(d delivery.DeliveryWithEvents) trackingResponse {
	r := trackingResponse{
		ID:                 d.ID,
		TrackingNumber:     d.TrackingNumber,
		Status:             string(d.Status),
		DestinationName:    d.DestinationName,
		DestinationAddress: d.DestinationAddress,
		DestinationCity:    d.DestinationCity,
		OriginCity:         d.OriginCity,
		ShippedAt:          fmtTimePtr(d.ShippedAt),
		ArrivedAt:          fmtTimePtr(d.ArrivedAt),
		ReadyForPickupAt:   fmtTimePtr(d.ReadyForPickupAt),
		PickedUpAt:         fmtTimePtr(d.PickedUpAt),
		ReturnedAt:         fmtTimePtr(d.ReturnedAt),
		CancelledAt:        fmtTimePtr(d.CancelledAt),
		CreatedAt:          d.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          d.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Events:             make([]deliveryEventResponse, 0, len(d.Events)),
	}
	for _, e := range d.Events {
		r.Events = append(r.Events, toDeliveryEventResponse(e))
	}
	return r
}

// buyerDeliveryResponse is what the authenticated buyer sees.
// Includes the pickup_code.
type buyerDeliveryResponse struct {
	trackingResponse
	PickupCode string `json:"pickup_code"`
}

func toBuyerDeliveryResponse(d delivery.DeliveryWithEvents) buyerDeliveryResponse {
	return buyerDeliveryResponse{
		trackingResponse: toTrackingResponse(d),
		PickupCode:       d.PickupCode,
	}
}

type dispatchOrderRequest struct {
	PickupPointID   string `json:"pickup_point_id"`
	OriginAddressID string `json:"origin_address_id"`
	Note            string `json:"note"`
}

type transitionRequest struct {
	Message  string `json:"message"`
	Location string `json:"location"`
}
