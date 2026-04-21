package auth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	v "github.com/radamesvaz/bakery-app/internal/handlers/validators"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	tenantSignupService "github.com/radamesvaz/bakery-app/internal/services/tenantsignup"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

type TenantSignupHandler struct {
	Service tenantSignupService.Service
}

func (h *TenantSignupHandler) CreateSignupCode(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "Tenant signup service not configured", http.StatusInternalServerError)
		return
	}

	roleID, err := middleware.GetUserRoleFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	createdByUserID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	req := authModel.CreateSignupCodeRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.Service.CreateSignupCode(r.Context(), roleID, createdByUserID, req)
	if err != nil {
		if errors.Is(err, tenantSignupService.ErrForbidden) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		http.Error(w, "Could not generate signup code", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *TenantSignupHandler) RegisterTenantWithCode(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "Tenant signup service not configured", http.StatusInternalServerError)
		return
	}

	var req authModel.PublicTenantRegisterRequest
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
	if strings.TrimSpace(req.OneTimeCode) == "" {
		http.Error(w, "One time code is required", http.StatusBadRequest)
		return
	}

	req.TenantName = tenantName
	req.TenantSlug = tenantSlug
	resp, err := h.Service.RegisterTenantWithCode(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, tenantSignupService.ErrInvalidOrUnavailableCode):
			http.Error(w, "Invalid or unavailable one-time code", http.StatusUnprocessableEntity)
			return
		case errors.Is(err, tenantSignupService.ErrTenantSlugExists):
			http.Error(w, "Tenant slug already exists", http.StatusConflict)
			return
		case errors.Is(err, tenantSignupService.ErrAdminEmailExists):
			http.Error(w, "Admin email already exists in tenant", http.StatusConflict)
			return
		default:
			http.Error(w, "Failed to register tenant", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}
