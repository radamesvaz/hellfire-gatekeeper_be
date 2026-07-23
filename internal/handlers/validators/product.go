package validators

import (
	pModel "github.com/radamesvaz/bakery-app/model/products"
)

// NormalizeProductStatus defaults empty status to active and validates the result.
func NormalizeProductStatus(status pModel.ProductStatus) (pModel.ProductStatus, bool) {
	if status == "" {
		status = pModel.StatusActive
	}
	switch status {
	case pModel.StatusActive, pModel.StatusInactive, pModel.StatusDeleted:
		return status, true
	default:
		return "", false
	}
}

// IsNonNegativePrice reports whether price is allowed for create/update (>= 0).
func IsNonNegativePrice(price float64) bool {
	return price >= 0
}
