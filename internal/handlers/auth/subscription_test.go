package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	subscriptionService "github.com/radamesvaz/bakery-app/internal/services/subscriptions"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	tenantModel "github.com/radamesvaz/bakery-app/model/tenant"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSubscriptionService struct {
	response       authModel.SubscriptionContextResponse
	updateResponse authModel.UpdateTenantSubscriptionResponse
	err            error
	updateErr      error
}

func (m *mockSubscriptionService) GetSubscriptionForTenant(ctx context.Context, tenantID uint64, tenantSlug string, now time.Time) (authModel.SubscriptionContextResponse, error) {
	return m.response, m.err
}

func (m *mockSubscriptionService) AdminUpdateSubscription(ctx context.Context, roleID uint64, tenantID uint64, req authModel.UpdateTenantSubscriptionRequest, now time.Time) (authModel.UpdateTenantSubscriptionResponse, error) {
	if m.updateErr != nil {
		return authModel.UpdateTenantSubscriptionResponse{}, m.updateErr
	}
	return m.updateResponse, nil
}

func TestSubscriptionHandler_GetSubscription_Success(t *testing.T) {
	handler := &SubscriptionHandler{
		Service: &mockSubscriptionService{
			response: authModel.SubscriptionContextResponse{
				TenantID:   1,
				TenantSlug: "default",
				Subscription: authModel.SubscriptionContext{
					Status:   tenantModel.SubscriptionStatusPending,
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

func TestSubscriptionHandler_UpdateTenantSubscriptionInternal_Success(t *testing.T) {
	handler := &SubscriptionHandler{
		Service: &mockSubscriptionService{
			updateResponse: authModel.UpdateTenantSubscriptionResponse{
				TenantID:   2,
				TenantSlug: "acme",
				Subscription: authModel.SubscriptionContext{
					Status:   tenantModel.SubscriptionStatusActive,
					PlanCode: "basic",
				},
			},
		},
	}

	body := `{"subscription_status":"active"}`
	req := httptest.NewRequest(http.MethodPatch, "/auth/internal/tenants/2/subscription", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"tenant_id": "2"})
	ctx := context.WithValue(req.Context(), middleware.UserClaimsKey, jwt.MapClaims{
		"user_id": float64(1),
		"role_id": float64(uModel.UserRoleSuperAdmin),
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.UpdateTenantSubscriptionInternal(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp authModel.UpdateTenantSubscriptionResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, tenantModel.SubscriptionStatusActive, resp.Subscription.Status)
}

func TestSubscriptionHandler_UpdateTenantSubscriptionInternal_Forbidden(t *testing.T) {
	handler := &SubscriptionHandler{
		Service: &mockSubscriptionService{
			updateErr: subscriptionService.ErrForbidden,
		},
	}

	req := httptest.NewRequest(http.MethodPatch, "/auth/internal/tenants/2/subscription",
		bytes.NewBufferString(`{"subscription_status":"active"}`))
	req = mux.SetURLVars(req, map[string]string{"tenant_id": "2"})
	ctx := context.WithValue(req.Context(), middleware.UserClaimsKey, jwt.MapClaims{
		"user_id": float64(2),
		"role_id": float64(uModel.UserRoleAdmin),
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler.UpdateTenantSubscriptionInternal(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}
