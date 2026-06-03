package middleware

import (
	"context"
	"net/http"

	tenantRepo "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	tenantModel "github.com/radamesvaz/bakery-app/model/tenant"
)

// SubscriptionSnapshotReader loads the current subscription snapshot for a tenant.
type SubscriptionSnapshotReader interface {
	GetSubscriptionSnapshot(ctx context.Context, tenantID uint64) (tenantRepo.SubscriptionSnapshot, error)
}

// RequireOperableSubscription rejects requests when the tenant in context has subscription_status canceled.
// Apply after AuthMiddleware and TenantMiddleware on business /auth routes.
func RequireOperableSubscription(reader SubscriptionSnapshotReader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID, err := GetTenantIDFromContext(r.Context())
			if err != nil || tenantID == 0 {
				http.Error(w, "tenant context missing", http.StatusBadRequest)
				return
			}

			snapshot, err := reader.GetSubscriptionSnapshot(r.Context(), tenantID)
			if err != nil {
				http.Error(w, "could not load tenant subscription", http.StatusInternalServerError)
				return
			}

			if snapshot.Status == tenantModel.SubscriptionStatusCanceled {
				http.Error(w, "tenant subscription is canceled", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
