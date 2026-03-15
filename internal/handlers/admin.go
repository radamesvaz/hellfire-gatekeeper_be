package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/logger"
	tenantRepo "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	tModel "github.com/radamesvaz/bakery-app/model/tenant"
)

// AdminHandler handles admin-only endpoints (e.g. create tenant). Requires JWT with superadmin role (AuthMiddleware + SuperadminRequired).
type AdminHandler struct {
	TenantRepo *tenantRepo.Repository
}

// slugRegex allows lowercase letters, numbers, and hyphens; 1–64 chars.
var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$|^[a-z0-9]$`)

// CreateTenant handles POST /admin/tenants. Creates a new tenant; slug must be unique.
func (h *AdminHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req tModel.CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", "INVALID_BODY", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Slug = strings.TrimSpace(strings.ToLower(req.Slug))
	if req.Name == "" {
		writeJSONError(w, "name is required", "VALIDATION_ERROR", http.StatusBadRequest)
		return
	}
	if req.Slug == "" {
		writeJSONError(w, "slug is required", "VALIDATION_ERROR", http.StatusBadRequest)
		return
	}
	if len(req.Slug) > 64 {
		writeJSONError(w, "slug must be at most 64 characters", "VALIDATION_ERROR", http.StatusBadRequest)
		return
	}
	if !slugRegex.MatchString(req.Slug) {
		writeJSONError(w, "slug must contain only lowercase letters, numbers, and hyphens", "VALIDATION_ERROR", http.StatusBadRequest)
		return
	}

	if req.PlanCode == "" {
		req.PlanCode = "basic"
	}
	if req.SubscriptionStatus == "" {
		req.SubscriptionStatus = "active"
	}
	if req.GhostOrderTimeoutMinutes <= 0 {
		req.GhostOrderTimeoutMinutes = 30
	}

	ctx := r.Context()
	exists, err := h.TenantRepo.SlugExists(ctx, req.Slug)
	if err != nil {
		logger.Warn().Err(err).Str("slug", req.Slug).Msg("check slug exists")
		writeJSONError(w, "Failed to check slug", "INTERNAL_ERROR", http.StatusInternalServerError)
		return
	}
	if exists {
		writeJSONError(w, "tenant slug already exists", "TENANT_SLUG_EXISTS", http.StatusConflict)
		return
	}

	in := tModel.CreateTenantInput{
		Name:                     req.Name,
		Slug:                     req.Slug,
		PlanCode:                 req.PlanCode,
		SubscriptionStatus:       req.SubscriptionStatus,
		CurrentPeriodEnd:         req.CurrentPeriodEnd,
		GhostOrderTimeoutMinutes: req.GhostOrderTimeoutMinutes,
	}
	created, err := h.TenantRepo.Create(ctx, in)
	if err != nil {
		logger.Warn().Err(err).Str("slug", req.Slug).Msg("create tenant")
		writeJSONError(w, "Failed to create tenant", "INTERNAL_ERROR", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// Allowed subscription statuses for superadmin updates (must match DB constraint / usage).
var allowedSubscriptionStatuses = map[string]bool{
	"active":   true,
	"canceled": true,
	"past_due": true,
	"trialing": true,
}

// UpdateTenantSubscription handles PATCH /admin/tenants/:id/subscription. Superadmin only.
func (h *AdminHandler) UpdateTenantSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		w.Header().Set("Allow", "PATCH")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	vars := mux.Vars(r)
	idStr := vars["id"]
	if idStr == "" {
		writeJSONError(w, "tenant id is required", "VALIDATION_ERROR", http.StatusBadRequest)
		return
	}
	tenantID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || tenantID == 0 {
		writeJSONError(w, "invalid tenant id", "VALIDATION_ERROR", http.StatusBadRequest)
		return
	}

	var req tModel.UpdateTenantSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Invalid request body", "INVALID_BODY", http.StatusBadRequest)
		return
	}
	req.SubscriptionStatus = strings.TrimSpace(strings.ToLower(req.SubscriptionStatus))
	if req.SubscriptionStatus == "" {
		writeJSONError(w, "subscription_status is required", "VALIDATION_ERROR", http.StatusBadRequest)
		return
	}
	if !allowedSubscriptionStatuses[req.SubscriptionStatus] {
		writeJSONError(w, "subscription_status must be one of: active, canceled, past_due, trialing", "VALIDATION_ERROR", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	err = h.TenantRepo.UpdateSubscription(ctx, tenantID, req.SubscriptionStatus, req.CurrentPeriodEnd)
	if err != nil {
		if errors.Is(err, appErrors.ErrTenantNotFound) {
			writeJSONError(w, "tenant not found", "TENANT_NOT_FOUND", http.StatusNotFound)
			return
		}
		logger.Warn().Err(err).Uint64("tenant_id", tenantID).Msg("update tenant subscription")
		writeJSONError(w, "Failed to update subscription", "INTERNAL_ERROR", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "subscription updated"})
}

func writeJSONError(w http.ResponseWriter, message, code string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message, "code": code})
}
