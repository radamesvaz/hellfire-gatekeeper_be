package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
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
		http.Error(w, "Password reset service not configured", http.StatusInternalServerError)
		return
	}

	var req authModel.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if !v.IsValidEmail(strings.TrimSpace(req.Email)) {
		http.Error(w, "Invalid Email", http.StatusBadRequest)
		return
	}

	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == 0 {
		http.Error(w, "tenant context missing", http.StatusBadRequest)
		return
	}
	tenantSlug, err := middleware.GetTenantSlugFromContext(r.Context())
	if err != nil || strings.TrimSpace(tenantSlug) == "" {
		http.Error(w, "tenant context missing", http.StatusBadRequest)
		return
	}

	if err := h.Service.ForgotPassword(r.Context(), tenantID, tenantSlug, req.Email); err != nil {
		http.Error(w, "Could not process request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(authModel.ForgotPasswordResponse{
		Message: "If the account exists, reset instructions will be sent.",
	})
}

func (h *PasswordResetHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "Password reset service not configured", http.StatusInternalServerError)
		return
	}

	var req authModel.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Token) == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		return
	}
	if err := v.ValidatePassword(req.NewPassword); err != nil {
		if httpErr, ok := err.(*appErrors.HTTPError); ok {
			http.Error(w, httpErr.Error(), httpErr.StatusCode)
			return
		}
		http.Error(w, "Password does not meet security requirements", http.StatusBadRequest)
		return
	}

	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == 0 {
		http.Error(w, "tenant context missing", http.StatusBadRequest)
		return
	}

	err = h.Service.ResetPassword(r.Context(), tenantID, req.Token, req.NewPassword)
	if err != nil {
		switch err {
		case appErrors.ErrInvalidToken, appErrors.ErrExpiredToken, appErrors.ErrTokenAlreadyConsumed, appErrors.ErrTokenRevoked:
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		default:
			http.Error(w, "Could not reset password", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(authModel.ResetPasswordResponse{
		Message: "Password reset successfully",
	})
}
