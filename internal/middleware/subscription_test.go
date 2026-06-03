package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	tenantRepo "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	tenantModel "github.com/radamesvaz/bakery-app/model/tenant"
	"github.com/stretchr/testify/assert"
)

type stubSubscriptionReader struct {
	snapshot tenantRepo.SubscriptionSnapshot
	err      error
}

func (s *stubSubscriptionReader) GetSubscriptionSnapshot(ctx context.Context, tenantID uint64) (tenantRepo.SubscriptionSnapshot, error) {
	return s.snapshot, s.err
}

func TestRequireOperableSubscription_AllowsActive(t *testing.T) {
	reader := &stubSubscriptionReader{snapshot: tenantRepo.SubscriptionSnapshot{Status: tenantModel.SubscriptionStatusActive}}
	mw := RequireOperableSubscription(reader)

	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	ctx := context.WithValue(context.Background(), TenantIDKey, uint64(5))
	req := httptest.NewRequest(http.MethodGet, "/auth/products", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called)
}

func TestRequireOperableSubscription_ForbiddenWhenCanceled(t *testing.T) {
	reader := &stubSubscriptionReader{snapshot: tenantRepo.SubscriptionSnapshot{Status: tenantModel.SubscriptionStatusCanceled}}
	mw := RequireOperableSubscription(reader)

	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	ctx := context.WithValue(context.Background(), TenantIDKey, uint64(5))
	req := httptest.NewRequest(http.MethodGet, "/auth/orders", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.False(t, called)
}
