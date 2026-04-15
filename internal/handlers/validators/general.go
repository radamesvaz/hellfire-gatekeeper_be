package validators

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/radamesvaz/bakery-app/internal/errors"
)

func IsValidEmail(email string) bool {
	re := regexp.MustCompile(`^[\w\.-]+@[\w\.-]+\.\w+$`)
	return re.MatchString(email)
}

// ValidatePassword validates password strength
// Requirements: at least 8 characters, 1 uppercase, 1 lowercase, 1 digit, 1 special char
// Returns ErrWeakPassword if validation fails
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.NewBadRequest(errors.ErrWeakPassword)
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return errors.NewBadRequest(errors.ErrWeakPassword)
	}

	return nil
}

const MaxTenantDisplayNameRunes = 255

// NormalizeAndValidateTenantDisplayName trims whitespace and validates non-empty length for tenants.name (API: tenant_name).
func NormalizeAndValidateTenantDisplayName(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.NewBadRequest(errors.ErrTenantNameRequired)
	}
	if utf8.RuneCountInString(s) > MaxTenantDisplayNameRunes {
		return "", errors.NewBadRequest(errors.ErrTenantNameTooLong)
	}
	return s, nil
}

// ThumbnailURLInImageURLs reports whether thumbnailURL is exactly equal to one of the entries in imageURLs.
func ThumbnailURLInImageURLs(thumbnailURL string, imageURLs []string) bool {
	for _, u := range imageURLs {
		if u == thumbnailURL {
			return true
		}
	}
	return false
}
