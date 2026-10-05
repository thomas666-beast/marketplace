package seller

import "errors"

var (
	ErrAddressNotFound = errors.New("address not found")
	ErrNotOwner        = errors.New("you do not own this address")
	ErrNoDefault       = errors.New("no default address set")
	ErrInvalidInput    = errors.New("invalid address input")
)
