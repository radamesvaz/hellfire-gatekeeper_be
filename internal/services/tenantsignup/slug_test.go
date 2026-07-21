package tenantsignup

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugifyTenantName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "spaces to hyphens", in: "Panadería Sol", want: "panaderia-sol"},
		{name: "already slug", in: "acme", want: "acme"},
		{name: "trim and collapse", in: "  Hello   World!! ", want: "hello-world"},
		{name: "empty", in: "   ", want: ""},
		{name: "punctuation stripped", in: "Café & Pasteles", want: "cafe-pasteles"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SlugifyTenantName(tt.in))
		})
	}
}

func TestTenantSlugCandidate(t *testing.T) {
	assert.Equal(t, "panaderia-sol", TenantSlugCandidate("panaderia-sol", 1))
	assert.Equal(t, "panaderia-sol-2", TenantSlugCandidate("panaderia-sol", 2))
	assert.Equal(t, "panaderia-sol-3", TenantSlugCandidate("panaderia-sol", 3))
	assert.Equal(t, "tenant", TenantSlugCandidate("", 1))
	assert.Equal(t, "tenant-2", TenantSlugCandidate("", 2))

	long := strings.Repeat("a", 70)
	got := TenantSlugCandidate(long, 1)
	assert.LessOrEqual(t, len(got), MaxTenantSlugLen)
	got2 := TenantSlugCandidate(long, 2)
	assert.LessOrEqual(t, len(got2), MaxTenantSlugLen)
	assert.True(t, strings.HasSuffix(got2, "-2"))
}
