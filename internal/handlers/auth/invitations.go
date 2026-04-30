package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	v "github.com/radamesvaz/bakery-app/internal/handlers/validators"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	invitationService "github.com/radamesvaz/bakery-app/internal/services/invitations"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

type InvitationHandler struct {
	Service invitationService.Service
}

func (h *InvitationHandler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "Invitation service not configured", http.StatusInternalServerError)
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

	var req authModel.CreateTenantInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if !v.IsValidEmail(strings.TrimSpace(req.Email)) {
		http.Error(w, "Invalid Email", http.StatusBadRequest)
		return
	}

	resp, err := h.Service.CreateInvitation(r.Context(), tenantID, tenantSlug, roleID, createdByUserID, req)
	if err != nil {
		switch {
		case errors.Is(err, appErrors.ErrForbidden):
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		case errors.Is(err, appErrors.ErrEmailAlreadyExists):
			http.Error(w, "Email already exists", http.StatusConflict)
			return
		default:
			var httpErr *appErrors.HTTPError
			if errors.As(err, &httpErr) {
				http.Error(w, httpErr.Error(), httpErr.StatusCode)
				return
			}
			http.Error(w, "Could not create invitation", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *InvitationHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "Invitation service not configured", http.StatusInternalServerError)
		return
	}

	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == 0 {
		http.Error(w, "tenant context missing", http.StatusBadRequest)
		return
	}

	var req authModel.AcceptTenantInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Token) == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		return
	}
	if err := v.ValidatePassword(req.Password); err != nil {
		writeValidatorError(w, err, "Password does not meet security requirements")
		return
	}

	resp, err := h.Service.AcceptInvitation(r.Context(), tenantID, req)
	if err != nil {
		switch err {
		case appErrors.ErrInvalidToken, appErrors.ErrExpiredToken, appErrors.ErrTokenAlreadyConsumed, appErrors.ErrTokenRevoked:
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		case appErrors.ErrEmailAlreadyExists:
			http.Error(w, "Email already exists", http.StatusConflict)
			return
		default:
			var httpErr *appErrors.HTTPError
			if errors.As(err, &httpErr) {
				http.Error(w, httpErr.Error(), httpErr.StatusCode)
				return
			}
			http.Error(w, "Could not accept invitation", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *InvitationHandler) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "Invitation service not configured", http.StatusInternalServerError)
		return
	}

	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == 0 {
		http.Error(w, "tenant context missing", http.StatusBadRequest)
		return
	}
	roleID, err := middleware.GetUserRoleFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idRaw := strings.TrimSpace(mux.Vars(r)["id"])
	invitationID, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil || invitationID == 0 {
		http.Error(w, "invalid invitation id", http.StatusBadRequest)
		return
	}

	err = h.Service.RevokeInvitation(r.Context(), tenantID, roleID, invitationID)
	if err != nil {
		switch err {
		case appErrors.ErrForbidden:
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		case appErrors.ErrInvalidToken, appErrors.ErrTokenAlreadyConsumed:
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		default:
			var httpErr *appErrors.HTTPError
			if errors.As(err, &httpErr) {
				http.Error(w, httpErr.Error(), httpErr.StatusCode)
				return
			}
			http.Error(w, "Could not revoke invitation", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Invitation revoked successfully"})
}

func (h *InvitationHandler) ResendInvitation(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		http.Error(w, "Invitation service not configured", http.StatusInternalServerError)
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

	idRaw := strings.TrimSpace(mux.Vars(r)["id"])
	invitationID, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil || invitationID == 0 {
		http.Error(w, "invalid invitation id", http.StatusBadRequest)
		return
	}

	resp, err := h.Service.ResendInvitation(r.Context(), tenantID, tenantSlug, roleID, createdByUserID, invitationID)
	if err != nil {
		switch err {
		case appErrors.ErrForbidden:
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		case appErrors.ErrInvalidToken, appErrors.ErrTokenAlreadyConsumed, appErrors.ErrTokenRevoked:
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		case appErrors.ErrEmailAlreadyExists:
			http.Error(w, "Email already exists", http.StatusConflict)
			return
		default:
			var httpErr *appErrors.HTTPError
			if errors.As(err, &httpErr) {
				http.Error(w, httpErr.Error(), httpErr.StatusCode)
				return
			}
			http.Error(w, "Could not resend invitation", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}
