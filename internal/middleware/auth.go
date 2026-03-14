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
// Current behavior (incremental, backwards compatible):
// - tenantSlug:
//   - If the route defines "{tenant_slug}" (e.g. "/t/{tenant_slug}/..."), it is
//     read from mux.Vars.
//   - Otherwise, it falls back to "default".
// - tenantID:
//   - For now we always assume 1 (default tenant). In later phases this
//     middleware will resolve tenantSlug -> tenant.id from the database.
// - isSuperadmin:
//   - Derived from the "role_id" claim (UserRoleAdmin is treated as superadmin).
func TenantMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			claims, ok := ctx.Value(UserClaimsKey).(jwt.MapClaims)
			if !ok {
				http.Error(w, "missing user claims in context", http.StatusUnauthorized)
				return
			}

			// Determine if user is superadmin based on role_id claim.
			isSuperadmin := false
			if roleIDFloat, ok := claims["role_id"].(float64); ok {
				if uint64(roleIDFloat) == uint64(uModel.UserRoleAdmin) {
					isSuperadmin = true
				}
			}

			vars := mux.Vars(r)
			tenantSlug := vars["tenant_slug"]
			if tenantSlug == "" {
				tenantSlug = "default"
			}

			// Prefer tenant_id from token claims when present; otherwise fall back
			// to the default tenant (1) for backwards compatibility.
			var tenantID uint64 = 1
			if tenantIDFloat, ok := claims["tenant_id"].(float64); ok && tenantIDFloat > 0 {
				tenantID = uint64(tenantIDFloat)
			}

			ctx = context.WithValue(ctx, TenantSlugKey, tenantSlug)
			ctx = context.WithValue(ctx, TenantIDKey, tenantID)
			ctx = context.WithValue(ctx, IsSuperAdminKey, isSuperadmin)

			next.ServeHTTP(w, r.WithContext(ctx))
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
