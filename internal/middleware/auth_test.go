package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubTenantSlugResolver struct {
	slug  string
	err   error
	gotID uint64
}

func (s *stubTenantSlugResolver) GetSlugByTenantID(ctx context.Context, tenantID uint64) (string, error) {
	s.gotID = tenantID
	if s.err != nil {
		return "", s.err
	}
	return s.slug, nil
}

func TestTenantMiddleware_ResolvesSlugFromResolverWhenPathHasNoSlug(t *testing.T) {
	res := &stubTenantSlugResolver{slug: "ratiarius"}
	mw := TenantMiddleware(res)

	var gotSlug string
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		gotSlug, err = GetTenantSlugFromContext(r.Context())
		require.NoError(t, err)
	}))

	req := httptest.NewRequest(http.MethodPost, "/auth/invitations", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserClaimsKey, jwt.MapClaims{
		"user_id":   float64(1),
		"role_id":   float64(1),
		"tenant_id": float64(42),
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "ratiarius", gotSlug)
	assert.Equal(t, uint64(42), res.gotID)
}

func TestTenantMiddleware_PathSlugWinsOverResolver(t *testing.T) {
	res := &stubTenantSlugResolver{slug: "from-db"}
	mw := TenantMiddleware(res)

	var gotSlug string
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		gotSlug, err = GetTenantSlugFromContext(r.Context())
		require.NoError(t, err)
	}))

	req := httptest.NewRequest(http.MethodPost, "/t/acme/auth/ping", nil)
	req = mux.SetURLVars(req, map[string]string{"tenant_slug": "acme"})
	req = req.WithContext(context.WithValue(req.Context(), UserClaimsKey, jwt.MapClaims{
		"user_id":   float64(1),
		"role_id":   float64(1),
		"tenant_id": float64(99),
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "acme", gotSlug)
	assert.Equal(t, uint64(0), res.gotID, "resolver must not run when path provides slug")
}

func TestRequireJWTTenantMatchesContext_AllowsWhenIDsMatch(t *testing.T) {
	mw := RequireJWTTenantMatchesContext()
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	ctx := context.WithValue(context.Background(), TenantIDKey, uint64(7))
	ctx = context.WithValue(ctx, UserClaimsKey, jwt.MapClaims{
		"tenant_id": float64(7),
	})
	req := httptest.NewRequest(http.MethodPost, "/t/acme/auth/invitations", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called)
}

func TestRequireJWTTenantMatchesContext_ForbiddenOnMismatch(t *testing.T) {
	mw := RequireJWTTenantMatchesContext()
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	ctx := context.WithValue(context.Background(), TenantIDKey, uint64(1))
	ctx = context.WithValue(ctx, UserClaimsKey, jwt.MapClaims{
		"tenant_id": float64(2),
	})
	req := httptest.NewRequest(http.MethodPost, "/t/other/auth/invitations", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.False(t, called)
}

func TestTenantMiddleware_ResolverErrorReturns500(t *testing.T) {
	res := &stubTenantSlugResolver{err: assert.AnError}
	mw := TenantMiddleware(res)

	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/auth/invitations", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserClaimsKey, jwt.MapClaims{
		"user_id":   float64(1),
		"role_id":   float64(1),
		"tenant_id": float64(1),
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.False(t, called)
}

func TestGetUserRoleFromContext(t *testing.T) {
	tests := []struct {
		name         string
		claims       jwt.MapClaims
		expectedRole uint64
		expectError  bool
	}{
		{
			name: "Valid admin role",
			claims: jwt.MapClaims{
				"user_id": float64(1),
				"role_id": float64(1), // Admin role
				"email":   "admin@example.com",
			},
			expectedRole: 1,
			expectError:  false,
		},
		{
			name: "Valid client role",
			claims: jwt.MapClaims{
				"user_id": float64(2),
				"role_id": float64(2), // Client role
				"email":   "client@example.com",
			},
			expectedRole: 2,
			expectError:  false,
		},
		{
			name: "Missing role_id in claims",
			claims: jwt.MapClaims{
				"user_id": float64(1),
				"email":   "admin@example.com",
			},
			expectedRole: 0,
			expectError:  true,
		},
		{
			name: "Invalid role_id type",
			claims: jwt.MapClaims{
				"user_id": float64(1),
				"role_id": "invalid", // Should be float64
				"email":   "admin@example.com",
			},
			expectedRole: 0,
			expectError:  true,
		},
		{
			name:         "No claims in context",
			claims:       nil,
			expectedRole: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.claims != nil {
				ctx = context.WithValue(ctx, UserClaimsKey, tt.claims)
			}

			role, err := GetUserRoleFromContext(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, uint64(0), role)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRole, role)
			}
		})
	}
}

func TestGetUserRoleFromContext_AdminRole(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserClaimsKey, jwt.MapClaims{
		"user_id": float64(1),
		"role_id": float64(1), // Admin role
		"email":   "admin@example.com",
	})

	role, err := GetUserRoleFromContext(ctx)

	assert.NoError(t, err)
	assert.Equal(t, uint64(1), role)
}

func TestGetUserRoleFromContext_ClientRole(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserClaimsKey, jwt.MapClaims{
		"user_id": float64(2),
		"role_id": float64(2), // Client role
		"email":   "client@example.com",
	})

	role, err := GetUserRoleFromContext(ctx)

	assert.NoError(t, err)
	assert.Equal(t, uint64(2), role)
}

func TestIsAdminRole(t *testing.T) {
	assert.True(t, IsAdminRole(uint64(uModel.UserRoleAdmin)))
	assert.True(t, IsAdminRole(uint64(uModel.UserRoleSuperAdmin)))
	assert.False(t, IsAdminRole(uint64(uModel.UserRoleClient)))
	assert.False(t, IsAdminRole(0))
}

func TestRequireAdminRole_AllowsAdminAndSuperAdmin(t *testing.T) {
	mw := RequireAdminRole()
	var called bool
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	for _, role := range []float64{1, 3} {
		called = false
		req := httptest.NewRequest(http.MethodPost, "/auth/products", nil)
		req = req.WithContext(context.WithValue(req.Context(), UserClaimsKey, jwt.MapClaims{
			"user_id": float64(1),
			"role_id": role,
		}))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "role %.0f", role)
		assert.True(t, called)
	}
}

func TestRequireAdminRole_RejectsClient(t *testing.T) {
	mw := RequireAdminRole()
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/auth/products", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserClaimsKey, jwt.MapClaims{
		"user_id": float64(2),
		"role_id": float64(2),
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}
