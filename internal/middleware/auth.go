package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/services/auth"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

// TenantSlugResolver resolves slug from tenant id when the route has no {tenant_slug}
// (for example POST /auth/invitations).
type TenantSlugResolver interface {
	GetSlugByTenantID(ctx context.Context, tenantID uint64) (string, error)
}

type contextKey string

const (
	UserClaimsKey   contextKey = "userClaims"
	TenantIDKey     contextKey = "tenantID"
	TenantSlugKey   contextKey = "tenantSlug"
	IsSuperAdminKey contextKey = "isSuperadmin"
)

func AuthMiddleware(authService auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header missing", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := authService.ValidateToken(tokenString)
			if err != nil || !token.Valid {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Add claims to context if needed later
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TenantMiddleware enriches the context with tenant information derived from
// the authenticated user's claims and the current request path.
//
// - tenantSlug:
//   - If the route defines "{tenant_slug}" (e.g. "/t/{tenant_slug}/..."), it is
//     read from mux.Vars.
//   - Otherwise, when slugResolver is non-nil, slug is loaded from the database
//     using tenant_id from the JWT (canonical slug for that tenant). Lookup errors
//     yield 500.
//   - If slugResolver is nil, falls back to "default" (legacy unit tests).
// - tenantID:
//   - From the "tenant_id" JWT claim when present; otherwise 1 for backwards compatibility.
// - isSuperadmin:
//   - Derived from the "role_id" claim (UserRoleSuperAdmin only).
func TenantMiddleware(slugResolver TenantSlugResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			claims, ok := ctx.Value(UserClaimsKey).(jwt.MapClaims)
			if !ok {
				http.Error(w, "missing user claims in context", http.StatusUnauthorized)
				return
			}

			// Determine if user is platform superadmin based on role_id claim.
			isSuperadmin := false
			if roleIDFloat, ok := claims["role_id"].(float64); ok {
				if uint64(roleIDFloat) == uint64(uModel.UserRoleSuperAdmin) {
					isSuperadmin = true
				}
			}

			// Prefer tenant_id from token claims when present; otherwise fall back
			// to the default tenant (1) for backwards compatibility.
			var tenantID uint64 = 1
			if tenantIDFloat, ok := claims["tenant_id"].(float64); ok && tenantIDFloat > 0 {
				tenantID = uint64(tenantIDFloat)
			}

			vars := mux.Vars(r)
			tenantSlug := strings.TrimSpace(vars["tenant_slug"])
			if tenantSlug == "" {
				if slugResolver != nil {
					s, err := slugResolver.GetSlugByTenantID(r.Context(), tenantID)
					if err != nil || strings.TrimSpace(s) == "" {
						http.Error(w, "could not resolve tenant slug", http.StatusInternalServerError)
						return
					}
					tenantSlug = strings.TrimSpace(s)
				} else {
					tenantSlug = "default"
				}
			}

			ctx = context.WithValue(ctx, TenantSlugKey, tenantSlug)
			ctx = context.WithValue(ctx, TenantIDKey, tenantID)
			ctx = context.WithValue(ctx, IsSuperAdminKey, isSuperadmin)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireJWTTenantMatchesContext ensures the session JWT tenant_id matches the
// tenant already resolved into context (for example by TenantFromPathOrHeader
// on /t/{tenant_slug}/auth/...). Call after tenant resolution and AuthMiddleware.
func RequireJWTTenantMatchesContext() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctxTenantID, err := GetTenantIDFromContext(r.Context())
			if err != nil || ctxTenantID == 0 {
				http.Error(w, "tenant context missing", http.StatusBadRequest)
				return
			}
			claims, ok := r.Context().Value(UserClaimsKey).(jwt.MapClaims)
			if !ok {
				http.Error(w, "missing user claims in context", http.StatusUnauthorized)
				return
			}
			jwtTenantFloat, ok := claims["tenant_id"].(float64)
			if !ok || jwtTenantFloat <= 0 {
				http.Error(w, "token not scoped to a tenant", http.StatusForbidden)
				return
			}
			if uint64(jwtTenantFloat) != ctxTenantID {
				http.Error(w, "token tenant does not match path tenant", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func GetUserIDFromContext(ctx context.Context) (uint64, error) {
	claims, ok := ctx.Value(UserClaimsKey).(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("no claims found in context")
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("user_id not found or invalid type in token claims")
	}

	return uint64(userIDFloat), nil
}

func GetUserRoleFromContext(ctx context.Context) (uint64, error) {
	claims, ok := ctx.Value(UserClaimsKey).(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("no claims found in context")
	}

	roleIDFloat, ok := claims["role_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("role_id not found or invalid type in token claims")
	}

	return uint64(roleIDFloat), nil
}

// IsAdminRole reports whether roleID is tenant admin or platform superadmin.
func IsAdminRole(roleID uint64) bool {
	return roleID == uint64(uModel.UserRoleAdmin) || roleID == uint64(uModel.UserRoleSuperAdmin)
}

// RequireAdminRole allows UserRoleAdmin (1) and UserRoleSuperAdmin (3); otherwise 403.
func RequireAdminRole() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roleID, err := GetUserRoleFromContext(r.Context())
			if err != nil {
				http.Error(w, "Unauthorized: invalid token role", http.StatusUnauthorized)
				return
			}
			if !IsAdminRole(roleID) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func GetTenantIDFromContext(ctx context.Context) (uint64, error) {
	tenantID, ok := ctx.Value(TenantIDKey).(uint64)
	if !ok {
		return 0, fmt.Errorf("tenant_id not found or invalid type in context")
	}
	return tenantID, nil
}

func GetTenantSlugFromContext(ctx context.Context) (string, error) {
	slug, ok := ctx.Value(TenantSlugKey).(string)
	if !ok || slug == "" {
		return "", fmt.Errorf("tenant_slug not found in context")
	}
	return slug, nil
}

func IsSuperadminFromContext(ctx context.Context) bool {
	flag, ok := ctx.Value(IsSuperAdminKey).(bool)
	if !ok {
		return false
	}
	return flag
}
