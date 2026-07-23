package errors

import "net/http"

type HTTPError struct {
	Err        error
	StatusCode int
}

func (e *HTTPError) Error() string {
	return e.Err.Error()
}

// Unwrap allows errors.Is / errors.As to match the wrapped sentinel.
func (e *HTTPError) Unwrap() error {
	return e.Err
}

func NewInternalServerError(err error) *HTTPError {
	return &HTTPError{
		Err:        err,
		StatusCode: http.StatusInternalServerError,
	}
}

func NewNotFound(err error) *HTTPError {
	return &HTTPError{
		Err:        err,
		StatusCode: http.StatusNotFound,
	}
}

func NewBadRequest(err error) *HTTPError {
	return &HTTPError{
		Err:        err,
		StatusCode: http.StatusBadRequest,
	}
}

func NewConflict(err error) *HTTPError {
	return &HTTPError{
		Err:        err,
		StatusCode: http.StatusConflict,
	}
}
