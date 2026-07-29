package model

import "time"

type CreateSignupCodeRequest struct {
	Email            string `json:"email"`
	ExpiresInMinutes int    `json:"expires_in_minutes"`
	Notes            string `json:"notes"`
}

type CreateSignupCodeResponse struct {
	ID        uint64    `json:"id"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
	Email     string    `json:"email"`
	Message   string    `json:"message"`
}

// PublicTenantRegisterRequest is the body for POST /public/tenant-register.
// Slug is derived from TenantName server-side; client-sent tenant_slug is ignored.
// Phone is the admin user contact; WhatsAppPhone is the storefront invoice destination on the tenant.
type PublicTenantRegisterRequest struct {
	TenantName    string `json:"tenant_name"`
	AdminName     string `json:"admin_name"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	WhatsAppPhone string `json:"whatsapp_phone"`
	Password      string `json:"password"`
	OneTimeCode   string `json:"one_time_code"`
}

type PublicTenantRegisterResponse struct {
	Message    string `json:"message"`
	Token      string `json:"token"`
	TenantID   uint64 `json:"tenant_id"`
	TenantSlug string `json:"tenant_slug"`
	TenantName string `json:"tenant_name"`
	AdminID    uint64 `json:"admin_id"`
	AdminEmail string `json:"admin_email"`
}
