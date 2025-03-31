package errors

import "net/http"

type HTTPError struct {
	Err        error
	StatusCode int
}

func (e *HTTPError) Error() string {
	return e.Err.Error()
}

func NewNotFound(err error) *HTTPError {
	return &HTTPError{
		Err:        err,
		StatusCode: http.StatusNotFound,
	}
}
