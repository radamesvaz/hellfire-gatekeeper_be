package subscriptions

import (
	"context"
	"database/sql"
	"testing"
	"time"

	tenantRepo "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	snapshot          tenantRepo.SubscriptionSnapshot
	pendingCount      int64
	canceledCount     int64
	snapshotErr       error
	pendingErr        error
	canceledErr       error
	lastGraceDays     int
	slug              string
	updateStatus      string
	updatePeriodEnd   sql.NullTime
	updatePeriodEndOK bool
}

func (f *fakeRepo) GetSubscriptionSnapshot(ctx context.Context, tenantID uint64) (tenantRepo.SubscriptionSnapshot, error) {
	return f.snapshot, f.snapshotErr
}

func (f *fakeRepo) GetSlugByTenantID(ctx context.Context, tenantID uint64) (string, error) {
	if f.slug == "" {
		return "acme", nil
	}
	return f.slug, nil
}

func (f *fakeRepo) UpdateTenantSubscription(
	ctx context.Context,
	tenantID uint64,
	status string,
	periodEnd sql.NullTime,
	updatePeriodEnd bool,
) error {
	f.updateStatus = status
	f.updatePeriodEnd = periodEnd
	f.updatePeriodEndOK = updatePeriodEnd
	f.snapshot.Status = status
	if updatePeriodEnd {
		f.snapshot.CurrentPeriodEnd = periodEnd
	}
	return nil
}

func (f *fakeRepo) MarkExpiredActiveAsPending(ctx context.Context, now time.Time) (int64, error) {
	return f.pendingCount, f.pendingErr
}

func (f *fakeRepo) MarkExpiredPendingAsCanceled(ctx context.Context, now time.Time, graceDays int) (int64, error) {
	f.lastGraceDays = graceDays
	return f.canceledCount, f.canceledErr
}

func TestService_GetSubscriptionForTenant_PendingIncludesGraceFields(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	periodEnd := now.AddDate(0, 0, -2)
	repo := &fakeRepo{
		snapshot: tenantRepo.SubscriptionSnapshot{
			Status:           "pending",
			PlanCode:         "basic",
			CurrentPeriodEnd: sql.NullTime{Time: periodEnd, Valid: true},
		},
	}
	svc := NewService(repo, 5)

	response, err := svc.GetSubscriptionForTenant(context.Background(), 10, "acme", now)
	require.NoError(t, err)
	assert.Equal(t, uint64(10), response.TenantID)
	assert.Equal(t, "acme", response.TenantSlug)
	assert.Equal(t, "pending", response.Subscription.Status)
	assert.Equal(t, "basic", response.Subscription.PlanCode)
	require.NotNil(t, response.Subscription.GracePeriodEnd)
	require.NotNil(t, response.Subscription.DaysUntilCancel)
	assert.Equal(t, 3, *response.Subscription.DaysUntilCancel)
}

func TestService_ProcessTransitions_ReturnsCounts(t *testing.T) {
	repo := &fakeRepo{pendingCount: 4, canceledCount: 2}
	svc := NewService(repo, 7)

	result, err := svc.ProcessTransitions(context.Background(), time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, int64(4), result.MarkedPending)
	assert.Equal(t, int64(2), result.MarkedCanceled)
	assert.Equal(t, 7, repo.lastGraceDays)
}

func TestService_AdminUpdateSubscription_ForbiddenForNonAdmin(t *testing.T) {
	svc := NewService(&fakeRepo{}, 5)
	_, err := svc.AdminUpdateSubscription(
		context.Background(),
		2,
		10,
		authModel.UpdateTenantSubscriptionRequest{SubscriptionStatus: "active"},
		time.Now().UTC(),
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestService_AdminUpdateSubscription_ActiveSetsDefault30DayPeriod(t *testing.T) {
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepo{
		snapshot: tenantRepo.SubscriptionSnapshot{Status: "canceled", PlanCode: "basic"},
	}
	svc := NewService(repo, 5)

	resp, err := svc.AdminUpdateSubscription(
		context.Background(),
		1,
		10,
		authModel.UpdateTenantSubscriptionRequest{SubscriptionStatus: "active"},
		now,
	)
	require.NoError(t, err)
	assert.Equal(t, "active", repo.updateStatus)
	assert.True(t, repo.updatePeriodEndOK)
	assert.True(t, repo.updatePeriodEnd.Valid)
	assert.Equal(t, now.AddDate(0, 0, DefaultPeriodDays), repo.updatePeriodEnd.Time)
	assert.Equal(t, "active", resp.Subscription.Status)
}
