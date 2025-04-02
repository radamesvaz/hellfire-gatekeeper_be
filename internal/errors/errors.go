package errors

import "errors"

var (
	ErrProductNotFound       = errors.New("product not found")
	ErrCouldNotGetTheProduct = errors.New("error getting the requested product")
)
