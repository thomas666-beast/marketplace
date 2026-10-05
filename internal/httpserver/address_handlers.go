package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/thomas666-beast/marketplace/internal/seller"
)

func (s *Server) handleListMyAddresses(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(ctxUserID).(string)

	list, err := s.addresses.ListBySeller(r.Context(), sellerID)
	if err != nil {
		s.logger.Error("list addresses failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	items := make([]addressResponse, 0, len(list))
	for _, a := range list {
		items = append(items, toAddressResponse(a))
	}
	s.writeJSON(w, http.StatusOK, addressListResponse{Items: items})
}

func (s *Server) handleCreateAddress(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(ctxUserID).(string)

	var req createAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}

	a, err := s.addresses.Create(r.Context(), seller.CreateAddressInput{
		SellerID:    sellerID,
		Label:       req.Label,
		ContactName: req.ContactName,
		Phone:       req.Phone,
		Address:     req.Address,
		City:        req.City,
		Region:      req.Region,
		PostalCode:  req.PostalCode,
		Country:     req.Country,
		IsDefault:   req.IsDefault,
	})
	if err != nil {
		if errors.Is(err, seller.ErrInvalidInput) {
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_address", "seller.invalid_address")
			return
		}
		s.logger.Error("create address failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}

	s.writeJSON(w, http.StatusCreated, toAddressResponse(a))
}

func (s *Server) handleGetAddress(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(ctxUserID).(string)
	id := r.PathValue("id")

	a, err := s.addresses.GetByID(r.Context(), id, sellerID)
	if err != nil {
		if errors.Is(err, seller.ErrAddressNotFound) {
			s.errorResponse(w, r, http.StatusNotFound, "not_found", "seller.address_not_found")
			return
		}
		s.logger.Error("get address failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}
	s.writeJSON(w, http.StatusOK, toAddressResponse(a))
}

func (s *Server) handleUpdateAddress(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(ctxUserID).(string)
	id := r.PathValue("id")

	var req updateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "invalid_request", "errors.invalid_request")
		return
	}

	// Normalize: trim string pointers, treat empty strings as "not provided"
	trimPtr := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		return &v
	}
	req.Label = trimPtr(req.Label)
	req.ContactName = trimPtr(req.ContactName)
	req.Phone = trimPtr(req.Phone)
	req.Address = trimPtr(req.Address)
	req.City = trimPtr(req.City)
	req.Region = trimPtr(req.Region)
	req.PostalCode = trimPtr(req.PostalCode)
	req.Country = trimPtr(req.Country)

	a, err := s.addresses.Update(r.Context(), id, sellerID, seller.UpdateAddressInput{
		Label:       req.Label,
		ContactName: req.ContactName,
		Phone:       req.Phone,
		Address:     req.Address,
		City:        req.City,
		Region:      req.Region,
		PostalCode:  req.PostalCode,
		Country:     req.Country,
	})
	if err != nil {
		switch {
		case errors.Is(err, seller.ErrAddressNotFound):
			s.errorResponse(w, r, http.StatusNotFound, "not_found", "seller.address_not_found")
		case errors.Is(err, seller.ErrInvalidInput):
			s.errorResponse(w, r, http.StatusBadRequest, "invalid_address", "seller.invalid_address")
		default:
			s.logger.Error("update address failed", "err", err)
			s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		}
		return
	}
	s.writeJSON(w, http.StatusOK, toAddressResponse(a))
}

func (s *Server) handleSetDefaultAddress(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(ctxUserID).(string)
	id := r.PathValue("id")

	a, err := s.addresses.SetDefault(r.Context(), id, sellerID)
	if err != nil {
		if errors.Is(err, seller.ErrAddressNotFound) {
			s.errorResponse(w, r, http.StatusNotFound, "not_found", "seller.address_not_found")
			return
		}
		s.logger.Error("set default failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}
	s.writeJSON(w, http.StatusOK, toAddressResponse(a))
}

func (s *Server) handleDeleteAddress(w http.ResponseWriter, r *http.Request) {
	sellerID, _ := r.Context().Value(ctxUserID).(string)
	id := r.PathValue("id")

	err := s.addresses.SoftDelete(r.Context(), id, sellerID)
	if err != nil {
		if errors.Is(err, seller.ErrAddressNotFound) {
			s.errorResponse(w, r, http.StatusNotFound, "not_found", "seller.address_not_found")
			return
		}
		s.logger.Error("delete address failed", "err", err)
		s.errorResponse(w, r, http.StatusInternalServerError, "internal", "errors.internal")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
