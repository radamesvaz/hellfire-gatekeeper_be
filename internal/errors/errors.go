package errors

import "errors"

var (
	// General error
	ErrDatabaseOperation = errors.New("error while executing a database operation")
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
	ErrUpdatingTheProduct     = errors.New("error updating the product")
	// Order errors
	ErrNoOrdersFound = errors.New("error getting all the orders")
	ErrOrderNotFound = errors.New("error getting the order")
)
