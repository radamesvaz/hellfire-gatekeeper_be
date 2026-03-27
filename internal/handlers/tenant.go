package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/radamesvaz/bakery-app/internal/middleware"
	tenantRepository "github.com/radamesvaz/bakery-app/internal/repository/tenant"
)

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type TenantHandler struct {
	Repo *tenantRepository.Repository
}

type updateBrandingColorsRequest struct {
	PrimaryColor   *string `json:"primary_color"`
	SecondaryColor *string `json:"secondary_color"`
	AccentColor    *string `json:"accent_color"`
}

// GetBranding returns logo + colors for the tenant resolved by TenantFromPathOrHeader (public, no auth).
// Use GET /t/{tenant_slug}/tenant/branding (or X-Tenant-Slug). Response includes tenant_slug for clients.
func (h *TenantHandler) GetBranding(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get tenant from context", http.StatusBadRequest)
		return
	}

	slug, err := middleware.GetTenantSlugFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get tenant slug from context", http.StatusBadRequest)
		return
	}

	branding, err := h.Repo.GetBranding(ctx, tenantID)
	if err != nil {
		http.Error(w, "Failed to get tenant branding", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tenant_id":   tenantID,
		"tenant_slug": slug,
		"branding":    branding,
	})
}

// UpdateBrandingColors updates tenant branding colors using partial JSON payload.
func (h *TenantHandler) UpdateBrandingColors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get tenant from context", http.StatusBadRequest)
		return
	}

	var req updateBrandingColorsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.PrimaryColor == nil && req.SecondaryColor == nil && req.AccentColor == nil {
		http.Error(w, "At least one color is required", http.StatusBadRequest)
		return
	}

	if req.PrimaryColor != nil {
		normalized, ok := normalizeHexColor(*req.PrimaryColor)
		if !ok {
			http.Error(w, "primary_color must use format #RRGGBB", http.StatusBadRequest)
			return
		}
		req.PrimaryColor = &normalized
	}
	if req.SecondaryColor != nil {
		normalized, ok := normalizeHexColor(*req.SecondaryColor)
		if !ok {
			http.Error(w, "secondary_color must use format #RRGGBB", http.StatusBadRequest)
			return
		}
		req.SecondaryColor = &normalized
	}
	if req.AccentColor != nil {
		normalized, ok := normalizeHexColor(*req.AccentColor)
		if !ok {
			http.Error(w, "accent_color must use format #RRGGBB", http.StatusBadRequest)
			return
		}
		req.AccentColor = &normalized
	}

	if err := h.Repo.UpdateBrandingColors(ctx, tenantID, tenantRepository.UpdateBrandingColorsRequest{
		PrimaryColor:   req.PrimaryColor,
		SecondaryColor: req.SecondaryColor,
		AccentColor:    req.AccentColor,
	}); err != nil {
		http.Error(w, "Failed to update tenant branding colors", http.StatusInternalServerError)
		return
	}

	colors, err := h.Repo.GetBrandingColors(ctx, tenantID)
	if err != nil {
		http.Error(w, "Failed to get tenant branding colors after update", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Tenant branding colors updated successfully",
		"tenant_id": tenantID,
		"colors":    colors,
	})
}

func normalizeHexColor(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if !hexColorRegex.MatchString(value) {
		return "", false
	}
	return strings.ToUpper(value), true
}
