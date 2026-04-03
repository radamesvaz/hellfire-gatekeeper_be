package validators

import (
	"net/http"
	"testing"

	"github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseListLimit(t *testing.T) {
	n, err := ParseListLimit("")
	require.NoError(t, err)
	assert.Equal(t, DefaultListLimit, n)

	n, err = ParseListLimit("1")
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	n, err = ParseListLimit("100")
	require.NoError(t, err)
	assert.Equal(t, 100, n)

	_, err = ParseListLimit("0")
	require.Error(t, err)
	httpErr, ok := err.(*errors.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, httpErr.StatusCode)

	_, err = ParseListLimit("101")
	require.Error(t, err)

	_, err = ParseListLimit("abc")
	require.Error(t, err)
}
