package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/radamesvaz/bakery-app/internal/middleware"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSubscriptionService struct {
	response authModel.SubscriptionContextResponse
	err      error
}

func (m *mockSubscriptionService) GetSubscriptionForTenant(ctx context.Context, tenantID uint64, tenantSlug string, now time.Time) (authModel.SubscriptionContextResponse, error) {
	return m.response, m.err
}

func TestSubscriptionHandler_GetSubscription_Success(t *testing.T) {
	handler := &SubscriptionHandler{
		Service: &mockSubscriptionService{
			response: authModel.SubscriptionContextResponse{
				TenantID:   1,
				TenantSlug: "default",
				Subscription: authModel.SubscriptionContext{
					Status:   "pending",
					PlanCode: "basic",
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/auth/subscription", nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, uint64(1))
	ctx = context.WithValue(ctx, middleware.TenantSlugKey, "default")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.GetSubscription(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, float64(1), body["tenant_id"])
	assert.Equal(t, "default", body["tenant_slug"])
}

func TestSubscriptionHandler_GetSubscription_MissingContext(t *testing.T) {
	handler := &SubscriptionHandler{Service: &mockSubscriptionService{}}
	req := httptest.NewRequest(http.MethodGet, "/auth/subscription", nil)
	rr := httptest.NewRecorder()

	handler.GetSubscription(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
