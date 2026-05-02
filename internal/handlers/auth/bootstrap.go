package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	v "github.com/radamesvaz/bakery-app/internal/handlers/validators"
	bootstrapService "github.com/radamesvaz/bakery-app/internal/services/bootstrap"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

type BootstrapHandler struct {
	Service bootstrapService.Service
}

// BootstrapTenant creates tenant + initial admin in one transaction.
// This endpoint is intended for controlled/internal onboarding only.
func (h *BootstrapHandler) BootstrapTenant(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		writeJSONAPIError(w, http.StatusInternalServerError, "service_unconfigured", "Bootstrap service not configured")
		return
	}
	var req authModel.BootstrapTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONAPIError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	tenantName, err := v.NormalizeAndValidateTenantDisplayName(req.TenantName)
	if err != nil {
		writeValidatorJSONError(w, err, "Invalid tenant name")
		return
	}
	tenantSlug := strings.TrimSpace(strings.ToLower(req.TenantSlug))
	if tenantSlug == "" {
		writeJSONAPIError(w, http.StatusBadRequest, "bad_request", "Tenant slug is required")
		return
	}
	if strings.TrimSpace(req.AdminName) == "" {
		writeJSONAPIError(w, http.StatusBadRequest, "bad_request", "Admin name is required")
		return
	}
	if !v.IsValidEmail(req.Email) {
		writeJSONAPIError(w, http.StatusBadRequest, "invalid_email", "Invalid Email")
		return
	}
	if err := v.ValidatePassword(req.Password); err != nil {
		writeValidatorJSONError(w, err, "Password does not meet security requirements")
		return
	}

	req.TenantName = tenantName
	req.TenantSlug = tenantSlug
	resp, err := h.Service.BootstrapTenant(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, bootstrapService.ErrTenantSlugExists):
			writeJSONAPIError(w, http.StatusConflict, "tenant_slug_exists", "Tenant slug already exists")
			return
		case errors.Is(err, bootstrapService.ErrAdminEmailExists):
			writeJSONAPIError(w, http.StatusConflict, "admin_email_exists", "Admin email already exists in tenant")
			return
		default:
			writeJSONAPIError(w, http.StatusInternalServerError, "internal_error", "Failed to create tenant")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

