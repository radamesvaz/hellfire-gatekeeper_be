package validators

import (
	"testing"

	pModel "github.com/radamesvaz/bakery-app/model/products"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeProductStatus(t *testing.T) {
	got, ok := NormalizeProductStatus("")
	assert.True(t, ok)
	assert.Equal(t, pModel.StatusActive, got)

	got, ok = NormalizeProductStatus(pModel.StatusInactive)
	assert.True(t, ok)
	assert.Equal(t, pModel.StatusInactive, got)

	_, ok = NormalizeProductStatus(pModel.ProductStatus("bogus"))
	assert.False(t, ok)
}

func TestIsNonNegativePrice(t *testing.T) {
	assert.True(t, IsNonNegativePrice(0))
	assert.True(t, IsNonNegativePrice(10.5))
	assert.False(t, IsNonNegativePrice(-0.01))
}
