package users

import "errors"

var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailExists  = errors.New("email already exists")
	ErrInvalidRole  = errors.New("invalid role")
)
