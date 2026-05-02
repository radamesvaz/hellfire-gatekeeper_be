package auth

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
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
		writeJSONAPIError(w, http.StatusInternalServerError, "service_unconfigured", "Invitation service not configured")
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
	roleID, err := middleware.GetUserRoleFromContext(r.Context())
	if err != nil {
		writeJSONAPIError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	createdByUserID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		writeJSONAPIError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var req authModel.CreateTenantInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONAPIError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}
	if !v.IsValidEmail(strings.TrimSpace(req.Email)) {
		writeJSONAPIError(w, http.StatusBadRequest, "invalid_email", "Invalid Email")
		return
	}

	resp, err := h.Service.CreateInvitation(r.Context(), tenantID, tenantSlug, roleID, createdByUserID, req)
	if err != nil {
		if respondInvitationMutationError(w, err) {
			return
		}
		writeJSONAPIError(w, http.StatusInternalServerError, "internal_error", "Could not create invitation")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *InvitationHandler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		writeJSONAPIError(w, http.StatusInternalServerError, "service_unconfigured", "Invitation service not configured")
		return
	}

	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == 0 {
		writeJSONAPIError(w, http.StatusBadRequest, "tenant_context_missing", "tenant context missing")
		return
	}

	var req authModel.AcceptTenantInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONAPIError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}
	if strings.TrimSpace(req.Token) == "" {
		writeJSONAPIError(w, http.StatusBadRequest, "bad_request", "token is required")
		return
	}
	if err := v.ValidatePassword(req.Password); err != nil {
		writeValidatorJSONError(w, err, "Password does not meet security requirements")
		return
	}

	resp, err := h.Service.AcceptInvitation(r.Context(), tenantID, req)
	if err != nil {
		if respondInvitationMutationError(w, err) {
			return
		}
		writeJSONAPIError(w, http.StatusInternalServerError, "internal_error", "Could not accept invitation")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *InvitationHandler) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		writeJSONAPIError(w, http.StatusInternalServerError, "service_unconfigured", "Invitation service not configured")
		return
	}

	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil || tenantID == 0 {
		writeJSONAPIError(w, http.StatusBadRequest, "tenant_context_missing", "tenant context missing")
		return
	}
	roleID, err := middleware.GetUserRoleFromContext(r.Context())
	if err != nil {
		writeJSONAPIError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	revokedByUserID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		writeJSONAPIError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	idRaw := strings.TrimSpace(mux.Vars(r)["id"])
	invitationID, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil || invitationID == 0 {
		writeJSONAPIError(w, http.StatusBadRequest, "bad_request", "invalid invitation id")
		return
	}

	err = h.Service.RevokeInvitation(r.Context(), tenantID, roleID, revokedByUserID, invitationID)
	if err != nil {
		if respondInvitationMutationError(w, err) {
			return
		}
		writeJSONAPIError(w, http.StatusInternalServerError, "internal_error", "Could not revoke invitation")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Invitation revoked successfully"})
}

func (h *InvitationHandler) ResendInvitation(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		writeJSONAPIError(w, http.StatusInternalServerError, "service_unconfigured", "Invitation service not configured")
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
	roleID, err := middleware.GetUserRoleFromContext(r.Context())
	if err != nil {
		writeJSONAPIError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	createdByUserID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		writeJSONAPIError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	idRaw := strings.TrimSpace(mux.Vars(r)["id"])
	invitationID, err := strconv.ParseUint(idRaw, 10, 64)
	if err != nil || invitationID == 0 {
		writeJSONAPIError(w, http.StatusBadRequest, "bad_request", "invalid invitation id")
		return
	}

	resp, err := h.Service.ResendInvitation(r.Context(), tenantID, tenantSlug, roleID, createdByUserID, invitationID)
	if err != nil {
		if respondInvitationMutationError(w, err) {
			return
		}
		writeJSONAPIError(w, http.StatusInternalServerError, "internal_error", "Could not resend invitation")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}
