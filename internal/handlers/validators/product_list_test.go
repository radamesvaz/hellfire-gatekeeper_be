package validators

import (
	"net/http"
	"testing"

	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProductSearchQuery(t *testing.T) {
	p, err := ParseProductSearchQuery("")
	require.NoError(t, err)
	assert.Nil(t, p)

	p, err = ParseProductSearchQuery("   ")
	require.NoError(t, err)
	assert.Nil(t, p)

	p, err = ParseProductSearchQuery("ab")
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, "ab", *p)

	p, err = ParseProductSearchQuery("  xy  ")
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, "xy", *p)

	_, err = ParseProductSearchQuery("x")
	require.Error(t, err)
	var he *appErrors.HTTPError
	require.ErrorAs(t, err, &he)
	assert.Equal(t, http.StatusBadRequest, he.StatusCode)
}

func TestProductNamePrefixLikePattern(t *testing.T) {
	assert.Equal(t, "foo%", ProductNamePrefixLikePattern("foo"))
	assert.Equal(t, `a\%b%`, ProductNamePrefixLikePattern(`a%b`))
	assert.Equal(t, `a\_b%`, ProductNamePrefixLikePattern(`a_b`))
}
