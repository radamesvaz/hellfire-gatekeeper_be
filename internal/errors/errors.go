package errors

import "errors"

var (
	// Product Errors
	ErrProductNotFound       = errors.New("product not found")
	ErrCouldNotGetTheProduct = errors.New("error getting the requested product")
	// User Errors
	ErrUserNotFound       = errors.New("user not found")
	ErrCouldNotGetTheUser = errors.New("error getting the user")
)
