package errors

import "errors"

var (
	// User Errors
	ErrUserNotFound       = errors.New("user not found")
	ErrCouldNotGetTheUser = errors.New("error getting the user")
	// Product Errors
	ErrProductNotFound        = errors.New("product not found")
	ErrCouldNotGetTheProduct  = errors.New("error getting the requested product")
	ErrCreatingProduct        = errors.New("error creating product")
	ErrCreatingProductHistory = errors.New("error creating product history")
	ErrInvalidStatus          = errors.New("error invalid Status")
	ErrUpdatingProductStatus  = errors.New("error updating the product status")
)
