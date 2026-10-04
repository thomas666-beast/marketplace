package catalog

import "errors"

var (
	ErrCategoryNotFound = errors.New("category not found")
	ErrCategoryInUse    = errors.New("category has children or products")
	ErrSlugTaken        = errors.New("slug already exists")
	ErrProductNotFound  = errors.New("product not found")
	ErrNotOwner         = errors.New("you do not own this product")
	ErrInvalidStatus    = errors.New("invalid product status")
)
