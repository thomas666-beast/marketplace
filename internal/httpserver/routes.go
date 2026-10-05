package httpserver

import "net/http"

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Health
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /welcome", s.handleWelcome)

	// Auth
	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", s.handleRefresh)

	// Users
	mux.HandleFunc("GET /api/v1/users/me", s.mw.requireAuth(s.handleMe))

	// Catalog (public)
	mux.HandleFunc("GET /api/v1/categories", s.handleListCategories)
	mux.HandleFunc("GET /api/v1/categories/{slug}", s.handleGetCategory)
	mux.HandleFunc("GET /api/v1/products", s.handleListProducts)
	mux.HandleFunc("GET /api/v1/products/{slug}", s.handleGetProduct)

	// Seller catalog
	mux.HandleFunc("GET /api/v1/seller/products",
		s.mw.requireAuth(s.mw.requireSeller(s.handleListMyProducts)))
	mux.HandleFunc("POST /api/v1/seller/products",
		s.mw.requireAuth(s.mw.requireSeller(s.handleCreateProduct)))
	mux.HandleFunc("PATCH /api/v1/seller/products/{id}",
		s.mw.requireAuth(s.mw.requireSeller(s.handleUpdateProduct)))
	mux.HandleFunc("DELETE /api/v1/seller/products/{id}",
		s.mw.requireAuth(s.mw.requireSeller(s.handleDeleteProduct)))

	// Buyer orders
	mux.HandleFunc("POST /api/v1/orders/checkout", s.mw.requireAuth(s.handleCheckout))
	mux.HandleFunc("GET /api/v1/orders", s.mw.requireAuth(s.handleListMyOrders))
	mux.HandleFunc("GET /api/v1/orders/{id}", s.mw.requireAuth(s.handleGetOrder))
	mux.HandleFunc("POST /api/v1/orders/{id}/pay", s.mw.requireAuth(s.handlePayOrder))
	mux.HandleFunc("POST /api/v1/orders/{id}/deliver", s.mw.requireAuth(s.handleDeliverOrder))
	mux.HandleFunc("POST /api/v1/orders/{id}/complete", s.mw.requireAuth(s.handleCompleteOrder))
	mux.HandleFunc("POST /api/v1/orders/{id}/cancel", s.mw.requireAuth(s.handleCancelOrder))

	// Seller orders
	mux.HandleFunc("GET /api/v1/seller/orders",
		s.mw.requireAuth(s.mw.requireSeller(s.handleListSellerOrders)))
	mux.HandleFunc("POST /api/v1/seller/orders/{id}/ship",
		s.mw.requireAuth(s.mw.requireSeller(s.handleShipOrder)))

			// Public delivery tracking
	mux.HandleFunc("GET /api/v1/delivery/track/{tracking_number}", s.handleTrackDelivery)

	// Public pickup points
	mux.HandleFunc("GET /api/v1/pickup-points", s.handleListPickupPoints)
	mux.HandleFunc("GET /api/v1/pickup-points/nearby", s.handleListPickupPointsNearby)
	mux.HandleFunc("GET /api/v1/pickup-points/{external_id}", s.handleGetPickupPoint)

	// Buyer: get own delivery with pickup code
	mux.HandleFunc("GET /api/v1/orders/{order_id}/delivery",
		s.mw.requireAuth(s.handleGetMyOrderDelivery))

	// Seller: dispatch an order
	mux.HandleFunc("POST /api/v1/seller/orders/{order_id}/dispatch",
		s.mw.requireAuth(s.mw.requireSeller(s.handleDispatchOrder)))

	// Admin: delivery status transitions
	mux.HandleFunc("POST /api/v1/admin/deliveries/{id}/{action}",
		s.mw.requireAuth(s.mw.requireAdmin(s.handleAdminDeliveryTransition)))

	// Seller addresses
	mux.HandleFunc("GET /api/v1/seller/addresses",
		s.mw.requireAuth(s.mw.requireSeller(s.handleListMyAddresses)))
	mux.HandleFunc("POST /api/v1/seller/addresses",
		s.mw.requireAuth(s.mw.requireSeller(s.handleCreateAddress)))
	mux.HandleFunc("GET /api/v1/seller/addresses/{id}",
		s.mw.requireAuth(s.mw.requireSeller(s.handleGetAddress)))
	mux.HandleFunc("PATCH /api/v1/seller/addresses/{id}",
		s.mw.requireAuth(s.mw.requireSeller(s.handleUpdateAddress)))
	mux.HandleFunc("PUT /api/v1/seller/addresses/{id}/default",
		s.mw.requireAuth(s.mw.requireSeller(s.handleSetDefaultAddress)))
	mux.HandleFunc("DELETE /api/v1/seller/addresses/{id}",
		s.mw.requireAuth(s.mw.requireSeller(s.handleDeleteAddress)))

	// Seller orders dashboard
	mux.HandleFunc("GET /api/v1/seller/orders/dashboard",
		s.mw.requireAuth(s.mw.requireSeller(s.handleSellerDashboard)))
}
