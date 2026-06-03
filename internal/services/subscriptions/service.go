package subscriptions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	tenantRepo "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	tenantModel "github.com/radamesvaz/bakery-app/model/tenant"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

const (
	DefaultGraceDays  = 5
	DefaultPeriodDays = 30
)

var (
	ErrForbidden                 = errors.New("forbidden")
	ErrInvalidSubscriptionStatus = errors.New("invalid subscription status")
)

type Repository interface {
	GetSubscriptionSnapshot(ctx context.Context, tenantID uint64) (tenantRepo.SubscriptionSnapshot, error)
	GetSlugByTenantID(ctx context.Context, tenantID uint64) (string, error)
	UpdateTenantSubscription(ctx context.Context, tenantID uint64, status tenantModel.SubscriptionStatus, periodEnd sql.NullTime, updatePeriodEnd bool) error
	MarkExpiredActiveAsPending(ctx context.Context, now time.Time) (int64, error)
	MarkExpiredPendingAsCanceled(ctx context.Context, now time.Time, graceDays int) (int64, error)
}

type Service struct {
	Repo      Repository
	GraceDays int
}

type TransitionResult struct {
	MarkedPending  int64
	MarkedCanceled int64
}

func NewService(repo Repository, graceDays int) *Service {
	if graceDays <= 0 {
		graceDays = DefaultGraceDays
	}
	return &Service{Repo: repo, GraceDays: graceDays}
}

func (s *Service) GetSubscriptionForTenant(ctx context.Context, tenantID uint64, tenantSlug string, now time.Time) (authModel.SubscriptionContextResponse, error) {

	snapshot, err := s.Repo.GetSubscriptionSnapshot(ctx, tenantID)
	if err != nil {
		return authModel.SubscriptionContextResponse{}, err
	}

	response := authModel.SubscriptionContextResponse{
		TenantID:   tenantID,
		TenantSlug: tenantSlug,
		Subscription: authModel.SubscriptionContext{
			Status:   snapshot.Status,
			PlanCode: snapshot.PlanCode,
		},
	}

	if snapshot.CurrentPeriodEnd.Valid {
		periodEnd := snapshot.CurrentPeriodEnd.Time.UTC()
		response.Subscription.CurrentPeriodEnd = &periodEnd

		graceEnd := periodEnd.AddDate(0, 0, s.GraceDays)
		response.Subscription.GracePeriodEnd = &graceEnd

		if snapshot.Status == tenantModel.SubscriptionStatusPending {
			days := int(graceEnd.Sub(now.UTC()).Hours() / 24)
			if days < 0 {
				days = 0
			}
			response.Subscription.DaysUntilCancel = &days
		}
	}

	return response, nil
}

// AdminUpdateSubscription allows a superadmin (role admin) to change subscription status and period end.
func (s *Service) AdminUpdateSubscription(
	ctx context.Context,
	roleID uint64,
	tenantID uint64,
	req authModel.UpdateTenantSubscriptionRequest,
	now time.Time,
) (authModel.UpdateTenantSubscriptionResponse, error) {
	if roleID != uint64(uModel.UserRoleAdmin) {
		return authModel.UpdateTenantSubscriptionResponse{}, ErrForbidden
	}

	status, ok := tenantModel.ParseSubscriptionStatus(req.SubscriptionStatus)
	if !ok {
		return authModel.UpdateTenantSubscriptionResponse{}, ErrInvalidSubscriptionStatus
	}

	periodEnd, updatePeriodEnd := resolvePeriodEnd(req, status, now.UTC())

	if err := s.Repo.UpdateTenantSubscription(ctx, tenantID, status, periodEnd, updatePeriodEnd); err != nil {
		return authModel.UpdateTenantSubscriptionResponse{}, err
	}

	slug, err := s.Repo.GetSlugByTenantID(ctx, tenantID)
	if err != nil {
		return authModel.UpdateTenantSubscriptionResponse{}, err
	}

	ctxResp, err := s.GetSubscriptionForTenant(ctx, tenantID, slug, now.UTC())
	if err != nil {
		return authModel.UpdateTenantSubscriptionResponse{}, err
	}

	return authModel.UpdateTenantSubscriptionResponse{
		TenantID:     ctxResp.TenantID,
		TenantSlug:   ctxResp.TenantSlug,
		Subscription: ctxResp.Subscription,
	}, nil
}

func resolvePeriodEnd(req authModel.UpdateTenantSubscriptionRequest, status tenantModel.SubscriptionStatus, now time.Time) (sql.NullTime, bool) {
	if req.CurrentPeriodEnd != nil {
		t := req.CurrentPeriodEnd.UTC()
		return sql.NullTime{Time: t, Valid: true}, true
	}
	if req.PeriodDays != nil && *req.PeriodDays > 0 {
		return sql.NullTime{Time: now.AddDate(0, 0, *req.PeriodDays), Valid: true}, true
	}
	if status == tenantModel.SubscriptionStatusActive {
		return sql.NullTime{Time: now.AddDate(0, 0, DefaultPeriodDays), Valid: true}, true
	}
	return sql.NullTime{}, false
}

func (s *Service) ProcessTransitions(ctx context.Context, now time.Time) (TransitionResult, error) {
	markedPending, err := s.Repo.MarkExpiredActiveAsPending(ctx, now.UTC())
	if err != nil {
		return TransitionResult{}, fmt.Errorf("transition active->pending: %w", err)
	}
	markedCanceled, err := s.Repo.MarkExpiredPendingAsCanceled(ctx, now.UTC(), s.GraceDays)
	if err != nil {
		return TransitionResult{}, fmt.Errorf("transition pending->canceled: %w", err)
	}
	return TransitionResult{
		MarkedPending:  markedPending,
		MarkedCanceled: markedCanceled,
	}, nil
}
