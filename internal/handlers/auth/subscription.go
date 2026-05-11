package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/radamesvaz/bakery-app/internal/middleware"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

type SubscriptionService interface {
	GetSubscriptionForTenant(ctx context.Context, tenantID uint64, tenantSlug string, now time.Time) (authModel.SubscriptionContextResponse, error)
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
