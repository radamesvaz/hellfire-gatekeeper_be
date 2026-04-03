package validators

import (
	"fmt"
	"strconv"

	"github.com/radamesvaz/bakery-app/internal/errors"
)

const (
	DefaultListLimit = 20
	MaxListLimit     = 100
)

// ParseListLimit parses the "limit" query param for cursor pagination.
// Empty string defaults to DefaultListLimit. Values must be in [1, MaxListLimit].
func ParseListLimit(s string) (int, error) {
	if s == "" {
		return DefaultListLimit, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > MaxListLimit {
		return 0, errors.NewBadRequest(fmt.Errorf("limit must be between 1 and %d", MaxListLimit))
	}
	return n, nil
}
