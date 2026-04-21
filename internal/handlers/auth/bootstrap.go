package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	hErrors "github.com/radamesvaz/bakery-app/internal/errors"
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
		http.Error(w, "Bootstrap service not configured", http.StatusInternalServerError)
		return
	}
	var req authModel.BootstrapTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tenantName, err := v.NormalizeAndValidateTenantDisplayName(req.TenantName)
	if err != nil {
		writeValidatorError(w, err, "Invalid tenant name")
		return
	}
	tenantSlug := strings.TrimSpace(strings.ToLower(req.TenantSlug))
	if tenantSlug == "" {
		http.Error(w, "Tenant slug is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.AdminName) == "" {
		http.Error(w, "Admin name is required", http.StatusBadRequest)
		return
	}
	if !v.IsValidEmail(req.Email) {
		http.Error(w, "Invalid Email", http.StatusBadRequest)
		return
	}
	if err := v.ValidatePassword(req.Password); err != nil {
		writeValidatorError(w, err, "Password does not meet security requirements")
		return
	}

	req.TenantName = tenantName
	req.TenantSlug = tenantSlug
	resp, err := h.Service.BootstrapTenant(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, bootstrapService.ErrTenantSlugExists):
			http.Error(w, "Tenant slug already exists", http.StatusConflict)
			return
		case errors.Is(err, bootstrapService.ErrAdminEmailExists):
			http.Error(w, "Admin email already exists in tenant", http.StatusConflict)
			return
		default:
			http.Error(w, "Failed to create tenant", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func writeValidatorError(w http.ResponseWriter, err error, fallback string) {
	var httpErr *hErrors.HTTPError
	if errors.As(err, &httpErr) {
		http.Error(w, httpErr.Error(), httpErr.StatusCode)
		return
	}
	http.Error(w, fallback, http.StatusBadRequest)
}


