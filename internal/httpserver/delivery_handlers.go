package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/thomas666-beast/marketplace/internal/delivery"
	"github.com/thomas666-beast/marketplace/internal/orders"
	"github.com/thomas666-beast/marketplace/internal/seller"
)

// --- Public: tracking ---

func (s *Server) handleTrackDelivery(w http.ResponseWriter, r *http.Request) {
	tn := strings.TrimSpace(r.PathValue("tracking_number"))
	if tn == "" {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}

	d, err := s.deliveries.GetByTrackingNumber(r.Context(), tn)
	if err != nil {
		if errors.Is(err, delivery.ErrDeliveryNotFound) {
			s.errorResponse(w, r, http.StatusNotFound, "not_found", "delivery.not_found")
			return
		}
		s.logger.Error("track delivery failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}
	s.writeJSON(w, http.StatusOK, toTrackingResponse(d))
}

// --- Public: pickup points ---

func (s *Server) handleListPickupPoints(w http.ResponseWriter, r *http.Request) {
	city := strings.TrimSpace(r.URL.Query().Get("city"))
	if city == "" {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}

	points, err := s.pickupPoints.ListByCity(r.Context(), city)
	if err != nil {
		s.logger.Error("list pickup points failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	items := make([]pickupPointResponse, 0, len(points))
	for _, p := range points {
		items = append(items, toPickupPointResponse(p))
	}
	s.writeJSON(w, http.StatusOK, pickupPointListResponse{Items: items})
}

func (s *Server) handleListPickupPointsNearby(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	latStr := q.Get("lat")
	lngStr := q.Get("lng")
	if latStr == "" || lngStr == "" {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil || lat < -90 || lat > 90 {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil || lng < -180 || lng > 180 {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}

	radius := 5.0
	if rStr := q.Get("radius"); rStr != "" {
		v, err := strconv.ParseFloat(rStr, 64)
		if err != nil || v < 0.1 || v > 50 {
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_radius", "delivery.invalid_radius")
			return
		}
		radius = v
	}

	limit := 20
	if lStr := q.Get("limit"); lStr != "" {
		n, err := parseLimit(lStr)
		if err != nil {
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_limit", "catalog.invalid_limit")
			return
		}
		limit = n
	}

	points, err := s.pickupPoints.ListNearby(r.Context(), delivery.NearbyInput{
		Latitude:   lat,
		Longitude:  lng,
		RadiusKm:   radius,
		MaxResults: limit,
	})
	if err != nil {
		s.logger.Error("nearby pickup points failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	items := make([]pickupPointResponse, 0, len(points))
	for _, p := range points {
		items = append(items, toPickupPointResponse(p))
	}
	s.writeJSON(w, http.StatusOK, pickupPointListResponse{Items: items})
}

func (s *Server) handleGetPickupPoint(w http.ResponseWriter, r *http.Request) {
	externalID := r.PathValue("external_id")
	p, err := s.pickupPoints.GetByExternalID(r.Context(), externalID)
	if err != nil {
		if errors.Is(err, delivery.ErrPickupPointNotFound) {
			s.errorResponse(w, r, http.StatusNotFound, "not_found", "delivery.pickup_point_not_found")
			return
		}
		s.logger.Error("get pickup point failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}
	s.writeJSON(w, http.StatusOK, toPickupPointResponse(p))
}

// --- Buyer: get own delivery with pickup code ---

func (s *Server) handleGetMyOrderDelivery(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxUserID).(string)
	orderID := r.PathValue("order_id")

	// Verify the caller is the buyer of this order.
	order, err := s.orders.GetByID(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, orders.ErrOrderNotFound) {
			s.errorResponse(w, r, http.StatusNotFound, "not_found", "orders.not_found")
			return
		}
		s.logger.Error("get order failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}
	if order.BuyerID != userID {
		// 404, not 403 — do not leak order existence.
		s.errorResponse(w, r, http.StatusNotFound, "not_found", "orders.not_found")
		return
	}

	d, err := s.deliveries.GetByOrderID(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, delivery.ErrDeliveryNotFound) {
			s.errorResponse(w, r, http.StatusNotFound, "not_found", "delivery.not_found")
			return
		}
		s.logger.Error("get delivery failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	s.writeJSON(w, http.StatusOK, toBuyerDeliveryResponse(d))
}

// --- Seller: dispatch order (create delivery) ---

func (s *Server) handleDispatchOrder(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(ctxUserID).(string)
	orderID := r.PathValue("order_id")

	var req dispatchOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}
	if strings.TrimSpace(req.PickupPointID) == "" {
		s.errorResponse(w, r, http.StatusBadRequest, "missing_pickup_point", "delivery.missing_pickup_point")
		return
	}
	if strings.TrimSpace(req.OriginAddressID) == "" {
		s.errorResponse(w, r, http.StatusBadRequest, "missing_origin_address", "seller.invalid_address")
		return
	}

	// Load the order and verify seller ownership + status.
	order, err := s.orders.GetByID(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, orders.ErrOrderNotFound) {
			s.errorResponse(w, r, http.StatusNotFound, "not_found", "orders.not_found")
			return
		}
		s.logger.Error("get order failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}
	if order.SellerID != sellerID {
		s.errorResponse(w, r, http.StatusNotFound, "not_found", "orders.not_found")
		return
	}
	if order.Status != orders.StatusPaid {
		s.errorResponse(w, r, http.StatusConflict, "order_not_paid", "delivery.order_not_paid")
		return
	}

	// Load and verify the origin address (ownership checked by the repository).
	origin, err := s.addresses.GetByID(r.Context(), req.OriginAddressID, sellerID)
	if err != nil {
		if errors.Is(err, seller.ErrAddressNotFound) {
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_origin_address", "seller.invalid_address")
			return
		}
		s.logger.Error("get origin address failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	// Load the destination pickup point.
	point, err := s.pickupPoints.GetByID(r.Context(), req.PickupPointID)
	if err != nil {
		if errors.Is(err, delivery.ErrPickupPointNotFound) {
			s.errorResponse(w, r, http.StatusBadRequest, "pickup_point_not_found", "delivery.pickup_point_not_found")
			return
		}
		s.logger.Error("get pickup point failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	// Generate tracking number and pickup code.
	tn, err := delivery.GenerateTrackingNumber(point.City, timeNow())
	if err != nil {
		s.logger.Error("generate tracking number failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}
	code, err := delivery.GeneratePickupCode()
	if err != nil {
		s.logger.Error("generate pickup code failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	// Snapshot the origin from the seller's address.
	originLabel := origin.Label
	if originLabel == "" {
		originLabel = "Warehouse"
	}

	d, err := s.deliveries.Create(r.Context(), delivery.CreateInput{
		OrderID:            orderID,
		PickupPointID:      point.ID,
		OriginAddressID:    &origin.ID,
		DestinationName:    point.Name,
		DestinationAddress: point.Address,
		DestinationCity:    point.City,
		DestinationCountry: point.Country,
		OriginName:         originLabel,
		OriginAddress:      origin.Address,
		OriginCity:         origin.City,
		TrackingNumber:     tn,
		PickupCode:         code,
	})
	if err != nil {
		switch {
		case errors.Is(err, delivery.ErrDeliveryExists):
			s.errorResponse(w, r, http.StatusConflict, "already_exists", "delivery.already_exists")
		case errors.Is(err, delivery.ErrTrackingExists):
			s.logger.Error("tracking collision", "err", err)
			s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		default:
			s.logger.Error("create delivery failed", "err", err)
			s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		}
		return
	}

	d, err = s.deliveries.MarkAwaitingDispatch(r.Context(), d.ID, strings.TrimSpace(req.Note), point.City)
	if err != nil {
		s.logger.Error("mark awaiting dispatch failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	_, err = s.orders.MarkShipped(r.Context(), orderID, sellerID)
	if err != nil {
		s.logger.Error("mark order shipped failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	s.writeJSON(w, http.StatusCreated, toTrackingResponse(delivery.DeliveryWithEvents{Delivery: d}))
}
// --- Admin/operator: status transitions ---

func (s *Server) handleAdminDeliveryTransition(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req transitionRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // body optional

	action := r.PathValue("action")

	var d delivery.Delivery
	var err error

	switch action {
	case "in_transit":
		d, err = s.deliveries.MarkInTransit(r.Context(), id, req.Message, req.Location)
	case "arrived_at_hub":
		d, err = s.deliveries.MarkArrivedAtHub(r.Context(), id, req.Message, req.Location)
	case "out_for_delivery":
		d, err = s.deliveries.MarkOutForDelivery(r.Context(), id, req.Message, req.Location)
	case "ready_for_pickup":
		d, err = s.deliveries.MarkReadyForPickup(r.Context(), id, req.Message, req.Location)
	case "picked_up":
		d, err = s.deliveries.MarkPickedUp(r.Context(), id, req.Message, req.Location)
	case "returned":
		d, err = s.deliveries.MarkReturned(r.Context(), id, req.Message, req.Location)
	case "cancel":
		d, err = s.deliveries.Cancel(r.Context(), id, req.Message, req.Location)
	default:
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}

	if err != nil {
		switch {
		case errors.Is(err, delivery.ErrDeliveryNotFound):
			s.errorResponse(w, r, http.StatusNotFound, "not_found", "delivery.not_found")
		case errors.Is(err, delivery.ErrInvalidTransition):
			s.errorResponse(w, r, http.StatusConflict, "invalid_transition", "delivery.invalid_transition")
		default:
			s.logger.Error("delivery transition failed", "err", err)
			s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		}
		return
	}

	s.writeJSON(w, http.StatusOK, toTrackingResponse(delivery.DeliveryWithEvents{Delivery: d}))
}

func timeNow() time.Time {
	return time.Now()
}
