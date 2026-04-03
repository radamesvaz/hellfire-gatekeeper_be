package validators

import (
	"fmt"
	"strings"

	"github.com/radamesvaz/bakery-app/internal/errors"
)

const (
	// MinProductSearchQLen is the minimum rune length for the `q` query param (prefix search).
	MinProductSearchQLen = 2
)

// ParseProductSearchQuery parses the optional `q` param for product list.
// Empty or whitespace-only returns (nil, nil): no name filter.
// Non-empty strings shorter than MinProductSearchQLen return a bad request error.
func ParseProductSearchQuery(q string) (*string, error) {
	trimmed := strings.TrimSpace(q)
	if trimmed == "" {
		return nil, nil
	}
	if len([]rune(trimmed)) < MinProductSearchQLen {
		return nil, errors.NewBadRequest(fmt.Errorf("q must be at least %d characters", MinProductSearchQLen))
	}
	return &trimmed, nil
}

// ProductNamePrefixLikePattern builds a LIKE/ILIKE pattern for case-insensitive prefix match,
// escaping %, _ and \ in user input.
func ProductNamePrefixLikePattern(prefix string) string {
	s := strings.ReplaceAll(prefix, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return strings.ToLower(s) + "%"
}
