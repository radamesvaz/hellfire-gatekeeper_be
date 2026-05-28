package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	subscriptionService "github.com/radamesvaz/bakery-app/internal/services/subscriptions"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

type SubscriptionService interface {
	GetSubscriptionForTenant(ctx context.Context, tenantID uint64, tenantSlug string, now time.Time) (authModel.SubscriptionContextResponse, error)
	AdminUpdateSubscription(ctx context.Context, roleID uint64, tenantID uint64, req authModel.UpdateTenantSubscriptionRequest, now time.Time) (authModel.UpdateTenantSubscriptionResponse, error)
}

type SubscriptionHandler struct {
	Service SubscriptionService
}

func (h *SubscriptionHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "Subscription service not configured", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil || tenantID == 0 {
		http.Error(w, "tenant context missing", http.StatusBadRequest)
		return
	}
	tenantSlug, err := middleware.GetTenantSlugFromContext(ctx)
	if err != nil {
		http.Error(w, "tenant context missing", http.StatusBadRequest)
		return
	}

	payload, err := h.Service.GetSubscriptionForTenant(ctx, tenantID, tenantSlug, time.Now().UTC())
	if err != nil {
		http.Error(w, "Failed to get subscription context", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}

// UpdateTenantSubscriptionInternal allows superadmin (role admin) to set subscription status and period end.
// PATCH /auth/internal/tenants/{tenant_id}/subscription
func (h *SubscriptionHandler) UpdateTenantSubscriptionInternal(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "Subscription service not configured", http.StatusInternalServerError)
		return
	}

	roleID, err := middleware.GetUserRoleFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tenantIDStr := mux.Vars(r)["tenant_id"]
	tenantID, err := strconv.ParseUint(tenantIDStr, 10, 64)
	if err != nil || tenantID == 0 {
		http.Error(w, "Invalid tenant id", http.StatusBadRequest)
		return
	}

	var req authModel.UpdateTenantSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.SubscriptionStatus == "" {
		http.Error(w, "subscription_status is required", http.StatusBadRequest)
		return
	}

	resp, err := h.Service.AdminUpdateSubscription(r.Context(), roleID, tenantID, req, time.Now().UTC())
	if err != nil {
		switch {
		case errors.Is(err, subscriptionService.ErrForbidden):
			http.Error(w, "Forbidden", http.StatusForbidden)
		case errors.Is(err, subscriptionService.ErrInvalidSubscriptionStatus):
			http.Error(w, "Invalid subscription_status", http.StatusBadRequest)
		default:
			http.Error(w, "Failed to update tenant subscription", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
