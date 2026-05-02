package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	v "github.com/radamesvaz/bakery-app/internal/handlers/validators"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	passwordresetService "github.com/radamesvaz/bakery-app/internal/services/passwordreset"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

type PasswordResetHandler struct {
	Service passwordresetService.Service
}

func (h *PasswordResetHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		writeJSONAPIError(w, http.StatusInternalServerError, "service_unconfigured", "Password reset service not configured")
		return
	}

	var req authModel.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONAPIError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}
	if !v.IsValidEmail(strings.TrimSpace(req.Email)) {
		writeJSONAPIError(w, http.StatusBadRequest, "invalid_email", "Invalid Email")
		return
	}

	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == 0 {
		writeJSONAPIError(w, http.StatusBadRequest, "tenant_context_missing", "tenant context missing")
		return
	}
	tenantSlug, err := middleware.GetTenantSlugFromContext(r.Context())
	if err != nil || strings.TrimSpace(tenantSlug) == "" {
		writeJSONAPIError(w, http.StatusBadRequest, "tenant_context_missing", "tenant context missing")
		return
	}

	if err := h.Service.ForgotPassword(r.Context(), tenantID, tenantSlug, req.Email); err != nil {
		respondForgotPasswordJSONError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(authModel.ForgotPasswordResponse{
		Message: "If the account exists, reset instructions will be sent.",
	})
}

func (h *PasswordResetHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		writeJSONAPIError(w, http.StatusInternalServerError, "service_unconfigured", "Password reset service not configured")
		return
	}

	var req authModel.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONAPIError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}
	if strings.TrimSpace(req.Token) == "" {
		writeJSONAPIError(w, http.StatusBadRequest, "bad_request", "token is required")
		return
	}
	if err := v.ValidatePassword(req.NewPassword); err != nil {
		writeValidatorJSONError(w, err, "Password does not meet security requirements")
		return
	}

	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == 0 {
		writeJSONAPIError(w, http.StatusBadRequest, "tenant_context_missing", "tenant context missing")
		return
	}

	err = h.Service.ResetPassword(r.Context(), tenantID, req.Token, req.NewPassword)
	if err != nil {
		if respondPasswordResetError(w, err) {
			return
		}
		writeJSONAPIError(w, http.StatusInternalServerError, "internal_error", "Could not reset password")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(authModel.ResetPasswordResponse{
		Message: "Password reset successfully",
	})
}
