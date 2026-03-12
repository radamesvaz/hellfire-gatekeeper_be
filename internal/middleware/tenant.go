package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// TenantResolver resolves a tenant slug to id and active status.
// Used by TenantFromPathOrHeader for public routes (e.g. create order without auth).
type TenantResolver interface {
	GetBySlug(ctx context.Context, slug string) (id uint64, isActive bool, err error)
}

const headerTenantSlug = "X-Tenant-Slug"

// TenantFromPathOrHeader resolves the tenant from the path variable {tenant_slug}
// or from the X-Tenant-Slug header, then injects TenantID and TenantSlug into the context.
// It does not require authentication. Returns 400 if no slug is provided, 404 if tenant
// is not found or inactive.
func TenantFromPathOrHeader(resolver TenantResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			slug := strings.TrimSpace(vars["tenant_slug"])
			if slug == "" {
				slug = strings.TrimSpace(r.Header.Get(headerTenantSlug))
			}
			// After path and header: require at least one source to provide a slug
			if slug == "" {
				http.Error(w, "tenant required: set path /t/{tenant_slug}/... or header X-Tenant-Slug", http.StatusBadRequest)
				return
			}

			ctx := r.Context()
			id, isActive, err := resolver.GetBySlug(ctx, slug)
			if err != nil || !isActive || id == 0 {
				http.Error(w, "tenant not found or inactive", http.StatusNotFound)
				return
			}

			ctx = context.WithValue(ctx, TenantSlugKey, slug)
			ctx = context.WithValue(ctx, TenantIDKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
