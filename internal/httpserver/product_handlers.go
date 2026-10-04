package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/thomas666-beast/marketplace/internal/catalog"
)

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	filter, ok := s.buildProductFilter(w, r)
	if !ok {
		return
	}

	page, err := s.products.List(r.Context(), filter)
	if err != nil {
		if strings.Contains(err.Error(), "invalid cursor") {
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_cursor", "catalog.invalid_cursor")
			return
		}
		s.logger.Error("list products failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	items := make([]productResponse, 0, len(page.Items))
	for _, p := range page.Items {
		items = append(items, toProductResponse(p))
	}

	s.writeJSON(w, http.StatusOK, productListResponse{
		Items:      items,
		NextCursor: page.NextCursor,
	})
}

func (s *Server) handleGetProduct(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	p, err := s.products.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, catalog.ErrProductNotFound) {
			s.errorResponse(w, r, http.StatusNotFound, "product_not_found", "catalog.product_not_found")
			return
		}
		s.logger.Error("get product failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	// Public read: only active products are visible.
	if p.Status != catalog.StatusActive {
		s.errorResponse(w, r, http.StatusNotFound, "product_not_found", "catalog.product_not_found")
		return
	}

	s.writeJSON(w, http.StatusOK, toProductResponse(p))
}

func (s *Server) handleListMyProducts(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(ctxUserID).(string)

	filter, ok := s.buildProductFilter(w, r)
	if !ok {
		return
	}
	// Force the seller filter — the client cannot spoof this.
	filter.SellerID = &sellerID

	page, err := s.products.List(r.Context(), filter)
	if err != nil {
		if strings.Contains(err.Error(), "invalid cursor") {
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_cursor", "catalog.invalid_cursor")
			return
		}
		s.logger.Error("list my products failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	items := make([]productResponse, 0, len(page.Items))
	for _, p := range page.Items {
		items = append(items, toProductResponse(p))
	}

	s.writeJSON(w, http.StatusOK, productListResponse{
		Items:      items,
		NextCursor: page.NextCursor,
	})
}

func (s *Server) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(ctxUserID).(string)

	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)

	if len(req.Name) < 1 || len(req.Name) > 200 {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_name", "catalog.invalid_name")
		return
	}
	if req.PriceCents < 0 {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_price", "catalog.invalid_price")
		return
	}
	if req.Currency != "RUB" && req.Currency != "USD" && req.Currency != "EUR" {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_currency", "catalog.invalid_currency")
		return
	}
	if req.StockQuantity < 0 {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_stock", "catalog.invalid_stock")
		return
	}
	if req.Status == "" {
		req.Status = string(catalog.StatusDraft)
	}
	if req.Status != string(catalog.StatusDraft) &&
		req.Status != string(catalog.StatusActive) &&
		req.Status != string(catalog.StatusArchived) {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_status", "catalog.invalid_status")
		return
	}
	if req.CategoryID == "" {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}

	// Verify the category exists before inserting.
	if _, err := s.categories.GetByID(r.Context(), req.CategoryID); err != nil {
		if errors.Is(err, catalog.ErrCategoryNotFound) {
			s.errorResponse(w, r, http.StatusBadRequest, "category_not_found", "catalog.category_not_found")
			return
		}
		s.logger.Error("category lookup failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	p, err := s.products.Create(r.Context(), catalog.CreateProductInput{
		SellerID:      sellerID,
		CategoryID:    req.CategoryID,
		Name:          req.Name,
		Description:   req.Description,
		PriceCents:    req.PriceCents,
		Currency:      catalog.Currency(req.Currency),
		StockQuantity: req.StockQuantity,
		Status:        catalog.ProductStatus(req.Status),
	})
	if err != nil {
		s.logger.Error("create product failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	s.writeJSON(w, http.StatusCreated, toProductResponse(p))
}

func (s *Server) handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(ctxUserID).(string)
	id := r.PathValue("id")

	var req updateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}

	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		if len(n) < 1 || len(n) > 200 {
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_name", "catalog.invalid_name")
			return
		}
		req.Name = &n
	}
	if req.PriceCents != nil && *req.PriceCents < 0 {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_price", "catalog.invalid_price")
		return
	}
	if req.Currency != nil {
		c := string(*req.Currency)
		if c != "RUB" && c != "USD" && c != "EUR" {
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_currency", "catalog.invalid_currency")
			return
		}
	}
	if req.StockQuantity != nil && *req.StockQuantity < 0 {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_stock", "catalog.invalid_stock")
		return
	}
	if req.Status != nil {
		st := string(*req.Status)
		if st != "draft" && st != "active" && st != "archived" {
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_status", "catalog.invalid_status")
			return
		}
	}

	input := catalog.UpdateProductInput{
		Name:          req.Name,
		Description:   req.Description,
		PriceCents:    req.PriceCents,
		StockQuantity: req.StockQuantity,
	}
	if req.Currency != nil {
		c := catalog.Currency(*req.Currency)
		input.Currency = &c
	}
	if req.Status != nil {
		st := catalog.ProductStatus(*req.Status)
		input.Status = &st
	}

	p, err := s.products.Update(r.Context(), id, sellerID, input)
	if err != nil {
		switch {
		case errors.Is(err, catalog.ErrProductNotFound):
			s.errorResponse(w, r, http.StatusNotFound, "product_not_found", "catalog.product_not_found")
		case errors.Is(err, catalog.ErrNotOwner):
			// Return 404, not 403 — do not leak that the product exists.
			s.errorResponse(w, r, http.StatusNotFound, "product_not_found", "catalog.product_not_found")
		default:
			s.logger.Error("update product failed", "err", err)
			s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		}
		return
	}

	s.writeJSON(w, http.StatusOK, toProductResponse(p))
}

func (s *Server) handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(ctxUserID).(string)
	id := r.PathValue("id")

	err := s.products.SoftDelete(r.Context(), id, sellerID)
	if err != nil {
		switch {
		case errors.Is(err, catalog.ErrProductNotFound):
			s.errorResponse(w, r, http.StatusNotFound, "product_not_found", "catalog.product_not_found")
		case errors.Is(err, catalog.ErrNotOwner):
			s.errorResponse(w, r, http.StatusNotFound, "product_not_found", "catalog.product_not_found")
		default:
			s.logger.Error("delete product failed", "err", err)
			s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// buildProductFilter parses query parameters into a ProductFilter.
// Returns ok=false if it has already written an error response.
func (s *Server) buildProductFilter(w http.ResponseWriter, r *http.Request) (catalog.ProductFilter, bool) {
	f := catalog.ProductFilter{}
	q := r.URL.Query()

	if cat := q.Get("category_id"); cat != "" {
		f.CategoryID = &cat
	}
	if search := q.Get("q"); search != "" {
		f.Search = strings.TrimSpace(search)
	}
	if cursor := q.Get("cursor"); cursor != "" {
		f.Cursor = cursor
	}
	if limitStr := q.Get("limit"); limitStr != "" {
		n, err := parseLimit(limitStr)
		if err != nil {
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_limit", "catalog.invalid_limit")
			return f, false
		}
		f.Limit = n
	}
	return f, true
}
