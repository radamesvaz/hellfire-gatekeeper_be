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
