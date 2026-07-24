package errors

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPError_Unwrap_ErrorsIsMatchesWrappedSentinel(t *testing.T) {
	err := NewNotFound(ErrProductNotFound)

	assert.True(t, errors.Is(err, ErrProductNotFound))
	assert.False(t, errors.Is(err, ErrUserNotFound))
}

func TestHTTPError_Unwrap_ErrorsAsYieldsStatus(t *testing.T) {
	err := NewNotFound(ErrProductNotFound)

	var he *HTTPError
	require.True(t, errors.As(err, &he))
	assert.Equal(t, http.StatusNotFound, he.StatusCode)
	assert.Equal(t, ErrProductNotFound.Error(), he.Error())
}

func TestHTTPError_BadRequestAndInternalUnwrap(t *testing.T) {
	bad := NewBadRequest(ErrInvalidStatus)
	assert.True(t, errors.Is(bad, ErrInvalidStatus))

	internal := NewInternalServerError(ErrDatabaseOperation)
	assert.True(t, errors.Is(internal, ErrDatabaseOperation))
}

func TestHTTPError_NewConflict(t *testing.T) {
	err := NewConflict(ErrConflict)

	assert.True(t, errors.Is(err, ErrConflict))
	var he *HTTPError
	require.True(t, errors.As(err, &he))
	assert.Equal(t, http.StatusConflict, he.StatusCode)
}
