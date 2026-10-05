package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/thomas666-beast/marketplace/internal/orders"
)

func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	buyerID, _ := r.Context().Value(ctxUserID).(string)

	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}

	if req.SellerID == "" {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}
	if len(req.Items) == 0 {
		s.errorResponse(w, r, http.StatusBadRequest, "empty_order", "orders.empty")
		return
	}
	if len(req.ShippingAddress) == 0 {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_shipping", "orders.invalid_shipping")
		return
	}

	items := make([]orders.CheckoutItem, 0, len(req.Items))
	for _, it := range req.Items {
		if it.ProductID == "" || it.Quantity <= 0 {
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_quantity", "orders.invalid_quantity")
			return
		}
		items = append(items, orders.CheckoutItem{
			ProductID: it.ProductID,
			Quantity:  it.Quantity,
		})
	}

	order, err := s.orders.Checkout(r.Context(), orders.CheckoutInput{
		BuyerID:         buyerID,
		SellerID:        req.SellerID,
		Items:           items,
		ShippingAddress: req.ShippingAddress,
		ShippingCents:   req.ShippingCents,
		Note:            strings.TrimSpace(req.Note),
	})
	if err != nil {
		s.mapCheckoutError(w, r, err)
		return
	}

	s.writeJSON(w, http.StatusCreated, toOrderResponse(order.Order, order.Items))
}

// mapCheckoutError translates domain errors from Checkout into HTTP responses.
func (s *Server) mapCheckoutError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, orders.ErrEmptyOrder):
		s.errorResponse(w, r, http.StatusBadRequest, "empty_order", "orders.empty")
	case errors.Is(err, orders.ErrInvalidQuantity):
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_quantity", "orders.invalid_quantity")
	case errors.Is(err, orders.ErrProductNotFound):
		s.errorResponse(w, r, http.StatusNotFound, "product_not_found", "orders.product_not_found")
	case errors.Is(err, orders.ErrInsufficientStock):
		s.errorResponse(w, r, http.StatusConflict, "insufficient_stock", "orders.insufficient_stock")
	case errors.Is(err, orders.ErrMixedCurrency):
		s.errorResponse(w, r, http.StatusBadRequest, "mixed_currency", "orders.mixed_currency")
	default:
		s.logger.Error("checkout failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
	}
}

func (s *Server) handleListMyOrders(w http.ResponseWriter, r *http.Request) {
	buyerID, _ := r.Context().Value(ctxUserID).(string)

	page, ok := s.listOrders(w, r, orders.OrderFilter{BuyerID: &buyerID})
	if !ok {
		return
	}
	s.writeJSON(w, http.StatusOK, page)
}

func (s *Server) handleListSellerOrders(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(ctxUserID).(string)

	filter := orders.OrderFilter{SellerID: &sellerID}

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		st := orders.Status(statusStr)
		switch st {
		case orders.StatusPendingPayment, orders.StatusPaid, orders.StatusShipped,
			orders.StatusDelivered, orders.StatusCompleted,
			orders.StatusCancelled, orders.StatusRefunded:
			filter.Status = &st
		default:
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_status", "orders.invalid_transition")
			return
		}
	}

	page, ok := s.listOrders(w, r, filter)
	if !ok {
		return
	}
	s.writeJSON(w, http.StatusOK, page)
}

// listOrders parses cursor and limit, runs the query, and writes the response.
// Returns false if it already wrote an error.
func (s *Server) listOrders(w http.ResponseWriter, r *http.Request, filter orders.OrderFilter) (orderListResponse, bool) {
	q := r.URL.Query()
	if cursor := q.Get("cursor"); cursor != "" {
		filter.Cursor = cursor
	}
	if limitStr := q.Get("limit"); limitStr != "" {
		n, err := parseLimit(limitStr)
		if err != nil {
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_limit", "catalog.invalid_limit")
			return orderListResponse{}, false
		}
		filter.Limit = n
	}

	page, err := s.orders.List(r.Context(), filter)
	if err != nil {
		if strings.Contains(err.Error(), "invalid cursor") {
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_cursor", "catalog.invalid_cursor")
			return orderListResponse{}, false
		}
		s.logger.Error("list orders failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return orderListResponse{}, false
	}

	items := make([]orderResponse, 0, len(page.Items))
	for _, o := range page.Items {
		items = append(items, toOrderResponse(o, nil))
	}
	return orderListResponse{Items: items, NextCursor: page.NextCursor}, true
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxUserID).(string)
	id := r.PathValue("id")

	order, err := s.orders.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, orders.ErrOrderNotFound) {
			s.errorResponse(w, r, http.StatusNotFound, "not_found", "orders.not_found")
			return
		}
		s.logger.Error("get order failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	// Only buyer or seller may view.
	if order.BuyerID != userID && order.SellerID != userID {
		s.errorResponse(w, r, http.StatusNotFound, "not_found", "orders.not_found")
		return
	}

	s.writeJSON(w, http.StatusOK, toOrderResponse(order.Order, order.Items))
}

func (s *Server) handlePayOrder(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxUserID).(string)
	id := r.PathValue("id")

	order, err := s.orders.MarkPaid(r.Context(), id, userID)
	if err != nil {
		s.mapOrderTransitionError(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, toOrderResponse(order, nil))
}

func (s *Server) handleShipOrder(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxUserID).(string)
	id := r.PathValue("id")

	order, err := s.orders.MarkShipped(r.Context(), id, userID)
	if err != nil {
		s.mapOrderTransitionError(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, toOrderResponse(order, nil))
}

func (s *Server) handleDeliverOrder(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxUserID).(string)
	id := r.PathValue("id")

	order, err := s.orders.MarkDelivered(r.Context(), id, userID)
	if err != nil {
		s.mapOrderTransitionError(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, toOrderResponse(order, nil))
}

func (s *Server) handleCompleteOrder(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxUserID).(string)
	id := r.PathValue("id")

	order, err := s.orders.Complete(r.Context(), id, userID)
	if err != nil {
		s.mapOrderTransitionError(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, toOrderResponse(order, nil))
}

func (s *Server) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxUserID).(string)
	id := r.PathValue("id")

	var req cancelOrderRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // body is optional

	order, err := s.orders.Cancel(r.Context(), id, userID, strings.TrimSpace(req.Reason))
	if err != nil {
		s.mapOrderTransitionError(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, toOrderResponse(order, nil))
}

// mapOrderTransitionError translates errors from status transitions.
func (s *Server) mapOrderTransitionError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, orders.ErrOrderNotFound):
		s.errorResponse(w, r, http.StatusNotFound, "not_found", "orders.not_found")
	case errors.Is(err, orders.ErrInvalidTransition):
		s.errorResponse(w, r, http.StatusConflict, "invalid_transition", "orders.invalid_transition")
	default:
		s.logger.Error("order transition failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
	}
}

func (s *Server) handleSellerDashboard(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(ctxUserID).(string)

	q := r.URL.Query()
	filter := orders.DashboardFilter(q.Get("filter"))
	if filter == "" {
		filter = orders.DashboardAll
	}

	cursor := q.Get("cursor")
	limit := 20
	if limitStr := q.Get("limit"); limitStr != "" {
		n, err := parseLimit(limitStr)
		if err != nil {
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_limit", "catalog.invalid_limit")
			return
		}
		limit = n
	}

	page, err := s.orders.Dashboard(r.Context(), sellerID, filter, cursor, limit)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "invalid dashboard filter"):
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_filter", "seller.invalid_dashboard_filter")
		case strings.Contains(err.Error(), "invalid cursor"):
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_cursor", "catalog.invalid_cursor")
		default:
			s.logger.Error("dashboard failed", "err", err)
			s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		}
		return
	}

	s.writeJSON(w, http.StatusOK, toDashboardResponse(page))
}
