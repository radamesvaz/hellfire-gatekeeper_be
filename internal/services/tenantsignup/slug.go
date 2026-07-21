package tenantsignup

import (
	"fmt"
	"strings"
	"unicode"
)

const MaxTenantSlugLen = 64

// SlugifyTenantName builds a URL-safe slug from a display name (lowercase, hyphens).
// Empty input yields empty string; callers should treat that as invalid.
func SlugifyTenantName(name string) string {
	s := strings.TrimSpace(strings.ToLower(name))
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	prevHyphen := false
	for _, r := range s {
		r = foldLatinRune(r)
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevHyphen = false
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '.':
			if b.Len() > 0 && !prevHyphen {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > MaxTenantSlugLen {
		out = strings.Trim(out[:MaxTenantSlugLen], "-")
	}
	return out
}

// TenantSlugCandidate returns the slug for attempt n (1-based).
// attempt 1 is the base; attempt 2+ appends "-{n}" (truncated so total length ≤ MaxTenantSlugLen).
func TenantSlugCandidate(base string, attempt int) string {
	base = strings.Trim(strings.ToLower(strings.TrimSpace(base)), "-")
	if base == "" {
		base = "tenant"
	}
	if attempt <= 1 {
		if len(base) > MaxTenantSlugLen {
			return strings.Trim(base[:MaxTenantSlugLen], "-")
		}
		return base
	}
	suffix := fmt.Sprintf("-%d", attempt)
	maxBase := MaxTenantSlugLen - len(suffix)
	if maxBase < 1 {
		maxBase = 1
	}
	trimmed := base
	if len(trimmed) > maxBase {
		trimmed = strings.Trim(trimmed[:maxBase], "-")
	}
	if trimmed == "" {
		trimmed = "tenant"
		if len(trimmed) > maxBase {
			trimmed = trimmed[:maxBase]
		}
	}
	return trimmed + suffix
}

func foldLatinRune(r rune) rune {
	switch r {
	case 'á', 'à', 'ä', 'â', 'ã', 'å':
		return 'a'
	case 'é', 'è', 'ë', 'ê':
		return 'e'
	case 'í', 'ì', 'ï', 'î':
		return 'i'
	case 'ó', 'ò', 'ö', 'ô', 'õ':
		return 'o'
	case 'ú', 'ù', 'ü', 'û':
		return 'u'
	case 'ñ':
		return 'n'
	case 'ç':
		return 'c'
	default:
		return r
	}
}
