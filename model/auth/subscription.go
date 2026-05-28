package model

import "time"

type SubscriptionContextResponse struct {
	TenantID     uint64              `json:"tenant_id"`
	TenantSlug   string              `json:"tenant_slug"`
	Subscription SubscriptionContext `json:"subscription"`
}

type SubscriptionContext struct {
	Status           string     `json:"status"`
	PlanCode         string     `json:"plan_code"`
	CurrentPeriodEnd *time.Time `json:"current_period_end,omitempty"`
	GracePeriodEnd   *time.Time `json:"grace_period_end,omitempty"`
	DaysUntilCancel  *int       `json:"days_until_cancel,omitempty"`
}

// UpdateTenantSubscriptionRequest is used by superadmin internal API to adjust billing snapshot.
// When subscription_status is "active" and neither current_period_end nor period_days is sent,
// the server sets current_period_end to now + 30 days.
type UpdateTenantSubscriptionRequest struct {
	SubscriptionStatus string     `json:"subscription_status"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end,omitempty"`
	PeriodDays         *int       `json:"period_days,omitempty"`
}

type UpdateTenantSubscriptionResponse struct {
	TenantID     uint64              `json:"tenant_id"`
	TenantSlug   string              `json:"tenant_slug"`
	Subscription SubscriptionContext `json:"subscription"`
}
