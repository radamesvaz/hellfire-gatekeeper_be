package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryRateLimiter_Middleware_BlocksWhenLimitExceeded(t *testing.T) {
	limiter := NewInMemoryRateLimiter()
	mw := limiter.Middleware(RateLimitOptions{
		Name:        "forgot_password",
		MaxRequests: 2,
		Window:      time.Minute,
	})

	calls := 0
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/t/acme/auth/password/forgot", nil)
		req.RemoteAddr = "127.0.0.1:4000"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		require.Equal(t, http.StatusOK, rr.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/t/acme/auth/password/forgot", nil)
	req.RemoteAddr = "127.0.0.1:4000"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusTooManyRequests, rr.Code)
	assert.NotEmpty(t, rr.Header().Get("Retry-After"))
	assert.Equal(t, 2, calls)
}

func TestInMemoryRateLimiter_Middleware_ScopesByTenant(t *testing.T) {
	limiter := NewInMemoryRateLimiter()
	mw := limiter.Middleware(RateLimitOptions{
		Name:        "invite_accept",
		MaxRequests: 1,
		Window:      time.Minute,
		ScopeTenant: true,
	})

	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodPost, "/t/a/auth/invitations/accept", nil)
	req1.RemoteAddr = "127.0.0.1:5000"
	req1 = req1.WithContext(context.WithValue(req1.Context(), TenantIDKey, uint64(1)))
	rr1 := httptest.NewRecorder()
	h.ServeHTTP(rr1, req1)
	require.Equal(t, http.StatusOK, rr1.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/t/b/auth/invitations/accept", nil)
	req2.RemoteAddr = "127.0.0.1:5000"
	req2 = req2.WithContext(context.WithValue(req2.Context(), TenantIDKey, uint64(2)))
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	require.Equal(t, http.StatusOK, rr2.Code)
}
