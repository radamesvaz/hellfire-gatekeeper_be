package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/radamesvaz/bakery-app/internal/logger"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	tenantRepository "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"
)

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type TenantHandler struct {
	Repo         *tenantRepository.Repository
	ImageService *imagesService.Service
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

// UploadTenantLogo accepts one image file (field `logo`) and sets or replaces the tenant logo (PATCH).
// Requires auth; old local/Cloudinary file is best-effort deleted when replaced.
func (h *TenantHandler) UploadTenantLogo(w http.ResponseWriter, r *http.Request) {
	if h.ImageService == nil {
		http.Error(w, "Image service not configured", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()

	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get tenant from context", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(8 << 20); err != nil {
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		http.Error(w, "No files provided", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["logo"]
	if len(files) != 1 {
		http.Error(w, "Exactly one logo file is required", http.StatusBadRequest)
		return
	}

	existing, err := h.Repo.GetBranding(ctx, tenantID)
	if err != nil {
		http.Error(w, "Failed to get tenant branding", http.StatusInternalServerError)
		return
	}
	oldURL := existing.LogoURL

	logoURL, err := h.ImageService.SaveTenantLogo(tenantID, files[0])
	if err != nil {
		if errors.Is(err, imagesService.ErrInvalidTenantLogoType) || errors.Is(err, imagesService.ErrTenantLogoTooLarge) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to save tenant logo", http.StatusInternalServerError)
		return
	}

	if err := h.Repo.UpdateTenantLogoURL(ctx, tenantID, logoURL); err != nil {
		_ = h.ImageService.DeleteImage(logoURL)
		http.Error(w, "Failed to update tenant logo", http.StatusInternalServerError)
		return
	}

	if oldURL != "" && oldURL != logoURL {
		if err := h.ImageService.DeleteImage(oldURL); err != nil {
			logger.Warn().Err(err).
				Str("old_logo_url", oldURL).
				Uint64("tenant_id", tenantID).
				Msg("Failed to delete previous tenant logo file")
		}
	}

	branding, err := h.Repo.GetBranding(ctx, tenantID)
	if err != nil {
		http.Error(w, "Failed to get tenant branding after update", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   "Tenant logo updated successfully",
		"tenant_id": tenantID,
		"logo_url":  logoURL,
		"branding":  branding,
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
