package model

type BootstrapTenantRequest struct {
	TenantName string `json:"tenant_name"`
	TenantSlug string `json:"tenant_slug"`
	AdminName  string `json:"admin_name"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Password   string `json:"password"`
}

type BootstrapTenantResponse struct {
	Message    string `json:"message"`
	Token      string `json:"token"`
	TenantID   uint64 `json:"tenant_id"`
	TenantSlug string `json:"tenant_slug"`
	TenantName string `json:"tenant_name"`
	AdminID    uint64 `json:"admin_id"`
	AdminEmail string `json:"admin_email"`
}
