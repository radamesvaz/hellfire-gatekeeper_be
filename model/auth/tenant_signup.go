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

type PublicTenantRegisterRequest struct {
	TenantName string `json:"tenant_name"`
	// TenantSlug is optional. When empty, the backend derives it from tenant_name
	// and may append -2, -3, ... if the base slug is already taken.
	TenantSlug  string `json:"tenant_slug"`
	AdminName   string `json:"admin_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Password    string `json:"password"`
	OneTimeCode string `json:"one_time_code"`
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
