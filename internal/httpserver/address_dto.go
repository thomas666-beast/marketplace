package httpserver

import (
	"github.com/thomas666-beast/marketplace/internal/seller"
)

type addressResponse struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	ContactName string `json:"contact_name"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	City        string `json:"city"`
	Region      string `json:"region"`
	PostalCode  string `json:"postal_code"`
	Country     string `json:"country"`
	IsDefault   bool   `json:"is_default"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toAddressResponse(a seller.Address) addressResponse {
	return addressResponse{
		ID:          a.ID,
		Label:       a.Label,
		ContactName: a.ContactName,
		Phone:       a.Phone,
		Address:     a.Address,
		City:        a.City,
		Region:      a.Region,
		PostalCode:  a.PostalCode,
		Country:     a.Country,
		IsDefault:   a.IsDefault,
		CreatedAt:   a.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   a.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

type addressListResponse struct {
	Items []addressResponse `json:"items"`
}

type createAddressRequest struct {
	Label       string `json:"label"`
	ContactName string `json:"contact_name"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	City        string `json:"city"`
	Region      string `json:"region"`
	PostalCode  string `json:"postal_code"`
	Country     string `json:"country"`
	IsDefault   bool   `json:"is_default"`
}

type updateAddressRequest struct {
	Label       *string `json:"label"`
	ContactName *string `json:"contact_name"`
	Phone       *string `json:"phone"`
	Address     *string `json:"address"`
	City        *string `json:"city"`
	Region      *string `json:"region"`
	PostalCode  *string `json:"postal_code"`
	Country     *string `json:"country"`
}
