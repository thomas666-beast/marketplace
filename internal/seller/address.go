package seller

import "time"

type Address struct {
	ID          string
	SellerID    string
	Label       string
	ContactName string
	Phone       string
	Address     string
	City        string
	Region      string
	PostalCode  string
	Country     string
	IsDefault   bool
	DeletedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateAddressInput struct {
	SellerID    string
	Label       string
	ContactName string
	Phone       string
	Address     string
	City        string
	Region      string
	PostalCode  string
	Country     string
	IsDefault   bool
}

type UpdateAddressInput struct {
	Label       *string
	ContactName *string
	Phone       *string
	Address     *string
	City        *string
	Region      *string
	PostalCode  *string
	Country     *string
	IsDefault   *bool
}
