package model

import "time"

// Logo dimension limits (valid for tenant branding logo).
const (
	LogoMinWidth  = 32
	LogoMinHeight = 32
	LogoMaxWidth  = 512
	LogoMaxHeight = 512
)

// Branding is the DTO returned by GET tenant branding (logo + colors).
type Branding struct {
	LogoURL        string `json:"logo_url"`
	LogoWidth      int    `json:"logo_width"`
	LogoHeight     int    `json:"logo_height"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`
}

// UpdateBrandingColorsRequest is the JSON body for PATCH /auth/tenant/branding/colors.
type UpdateBrandingColorsRequest struct {
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`
}

// CreateTenantRequest is the JSON body for POST /admin/tenants.
type CreateTenantRequest struct {
	Name                     string     `json:"name"`
	Slug                     string     `json:"slug"`
	PlanCode                 string     `json:"plan_code"`
	SubscriptionStatus       string     `json:"subscription_status"`
	CurrentPeriodEnd         *time.Time `json:"current_period_end,omitempty"`
	GhostOrderTimeoutMinutes int        `json:"ghost_order_timeout_minutes"`
}

// CreateTenantInput is the domain input for creating a tenant (used by repository).
type CreateTenantInput struct {
	Name                     string
	Slug                     string
	PlanCode                 string
	SubscriptionStatus       string
	CurrentPeriodEnd         *time.Time
	GhostOrderTimeoutMinutes int
}

// UpdateTenantSubscriptionRequest is the JSON body for PATCH /admin/tenants/:id/subscription (superadmin only).
type UpdateTenantSubscriptionRequest struct {
	SubscriptionStatus string     `json:"subscription_status"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end,omitempty"`
}

// CreateTenantResponse is returned after creating a tenant.
type CreateTenantResponse struct {
	ID                       uint64     `json:"id"`
	Name                     string     `json:"name"`
	Slug                     string     `json:"slug"`
	PlanCode                 string     `json:"plan_code"`
	SubscriptionStatus       string     `json:"subscription_status"`
	CurrentPeriodEnd         *time.Time `json:"current_period_end,omitempty"`
	GhostOrderTimeoutMinutes int        `json:"ghost_order_timeout_minutes"`
	IsActive                 bool       `json:"is_active"`
}
