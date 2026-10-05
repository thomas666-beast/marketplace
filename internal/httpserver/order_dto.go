package httpserver

import (
	"encoding/json"

	"github.com/thomas666-beast/marketplace/internal/orders"
)

type checkoutItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type checkoutRequest struct {
	SellerID        string                `json:"seller_id"`
	Items           []checkoutItemRequest `json:"items"`
	ShippingAddress json.RawMessage       `json:"shipping_address"`
	ShippingCents   int64                 `json:"shipping_cents"`
	Note            string                `json:"note"`
}

type cancelOrderRequest struct {
	Reason string `json:"reason"`
}

type orderItemResponse struct {
	ID             string `json:"id"`
	ProductID      string `json:"product_id"`
	ProductName    string `json:"product_name"`
	ProductSlug    string `json:"product_slug"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	Quantity       int    `json:"quantity"`
	LineTotalCents int64  `json:"line_total_cents"`
}

func toOrderItemResponse(it orders.OrderItem) orderItemResponse {
	return orderItemResponse{
		ID:             it.ID,
		ProductID:      it.ProductID,
		ProductName:    it.ProductName,
		ProductSlug:    it.ProductSlug,
		UnitPriceCents: it.UnitPriceCents,
		Quantity:       it.Quantity,
		LineTotalCents: it.LineTotalCents,
	}
}

type orderResponse struct {
	ID              string              `json:"id"`
	OrderNumber     string              `json:"order_number"`
	BuyerID         string              `json:"buyer_id"`
	SellerID        string              `json:"seller_id"`
	Status          string              `json:"status"`
	SubtotalCents   int64               `json:"subtotal_cents"`
	ShippingCents   int64               `json:"shipping_cents"`
	TotalCents      int64               `json:"total_cents"`
	Currency        string              `json:"currency"`
	ShippingAddress json.RawMessage     `json:"shipping_address"`
	Note            string              `json:"note"`
	CancelledReason *string             `json:"cancelled_reason,omitempty"`
	CancelledAt     *string             `json:"cancelled_at,omitempty"`
	PaidAt          *string             `json:"paid_at,omitempty"`
	ShippedAt       *string             `json:"shipped_at,omitempty"`
	DeliveredAt     *string             `json:"delivered_at,omitempty"`
	CompletedAt     *string             `json:"completed_at,omitempty"`
	CreatedAt       string              `json:"created_at"`
	UpdatedAt       string              `json:"updated_at"`
	Items           []orderItemResponse `json:"items,omitempty"`
}

func tsPtr(t *interface{ IsZero() bool }) *string { return nil } // placeholder, not used

func formatTime(t interface{ Format(string) string }) string {
	return t.Format("2006-01-02T15:04:05Z")
}

func toOrderResponse(o orders.Order, items []orders.OrderItem) orderResponse {
	r := orderResponse{
		ID:              o.ID,
		OrderNumber:     o.OrderNumber,
		BuyerID:         o.BuyerID,
		SellerID:        o.SellerID,
		Status:          string(o.Status),
		SubtotalCents:   o.SubtotalCents,
		ShippingCents:   o.ShippingCents,
		TotalCents:      o.TotalCents,
		Currency:        string(o.Currency),
		ShippingAddress: o.ShippingAddress,
		Note:            o.Note,
		CancelledReason: o.CancelledReason,
		CreatedAt:       o.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       o.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if o.CancelledAt != nil {
		s := o.CancelledAt.UTC().Format("2006-01-02T15:04:05Z")
		r.CancelledAt = &s
	}
	if o.PaidAt != nil {
		s := o.PaidAt.UTC().Format("2006-01-02T15:04:05Z")
		r.PaidAt = &s
	}
	if o.ShippedAt != nil {
		s := o.ShippedAt.UTC().Format("2006-01-02T15:04:05Z")
		r.ShippedAt = &s
	}
	if o.DeliveredAt != nil {
		s := o.DeliveredAt.UTC().Format("2006-01-02T15:04:05Z")
		r.DeliveredAt = &s
	}
	if o.CompletedAt != nil {
		s := o.CompletedAt.UTC().Format("2006-01-02T15:04:05Z")
		r.CompletedAt = &s
	}
	if items != nil {
		r.Items = make([]orderItemResponse, 0, len(items))
		for _, it := range items {
			r.Items = append(r.Items, toOrderItemResponse(it))
		}
	}
	return r
}

type orderListResponse struct {
	Items      []orderResponse `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

type dashboardDeliveryInfo struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Tracking     string `json:"tracking_number"`
	PickupCode   string `json:"pickup_code,omitempty"`
}

type dashboardRowResponse struct {
	Order    orderResponse         `json:"order"`
	Delivery *dashboardDeliveryInfo `json:"delivery,omitempty"`
}

type dashboardCountsResponse struct {
	AwaitingPayment int `json:"awaiting_payment"`
	ReadyToDispatch int `json:"ready_to_dispatch"`
	InDelivery      int `json:"in_delivery"`
	Completed       int `json:"completed"`
	Cancelled       int `json:"cancelled"`
}

type dashboardResponse struct {
	Counts     dashboardCountsResponse `json:"counts"`
	Items      []dashboardRowResponse  `json:"items"`
	NextCursor string                  `json:"next_cursor,omitempty"`
}

func toDashboardResponse(page orders.DashboardPage) dashboardResponse {
	out := dashboardResponse{
		Counts: dashboardCountsResponse{
			AwaitingPayment: page.Counts.AwaitingPayment,
			ReadyToDispatch: page.Counts.ReadyToDispatch,
			InDelivery:      page.Counts.InDelivery,
			Completed:       page.Counts.Completed,
			Cancelled:       page.Counts.Cancelled,
		},
		Items:      make([]dashboardRowResponse, 0, len(page.Items)),
		NextCursor: page.NextCursor,
	}
	for _, row := range page.Items {
		r := dashboardRowResponse{
			Order: toOrderResponse(row.Order, nil),
		}
		if row.DeliveryID != nil {
			r.Delivery = &dashboardDeliveryInfo{
				ID:         *row.DeliveryID,
				Status:     derefString(row.DeliveryStatus),
				Tracking:   derefString(row.DeliveryTracking),
				PickupCode: derefString(row.DeliveryPickupCode),
			}
		}
		out.Items = append(out.Items, r)
	}
	return out
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
