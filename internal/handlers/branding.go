package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/logger"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	brandingService "github.com/radamesvaz/bakery-app/internal/services/branding"
	tModel "github.com/radamesvaz/bakery-app/model/tenant"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

// BrandingHandler handles tenant branding endpoints.
type BrandingHandler struct {
	BrandingService *brandingService.Service
}

// GetBranding returns the branding (logo + colors) for the tenant in context.
// Public: GET /t/{tenant_slug}/branding — for the tenant's purchase/shop page; the FE should always have access here.
// Authenticated: GET /auth/tenant/branding — after login, for the admin experience (login form does not show logo/colors).
func (h *BrandingHandler) GetBranding(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		http.Error(w, "tenant required", http.StatusBadRequest)
		return
	}
	b, err := h.BrandingService.GetBranding(ctx, tenantID)
	if err != nil {
		if errors.Is(err, appErrors.ErrTenantNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		logger.Warn().Err(err).Uint64("tenant_id", tenantID).Msg("get branding")
		http.Error(w, "Failed to get branding", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(b)
}

// UpdateColors updates the tenant's brand colors. Requires admin role.
func (h *BrandingHandler) UpdateColors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		http.Error(w, "tenant required", http.StatusBadRequest)
		return
	}
	userRole, err := middleware.GetUserRoleFromContext(ctx)
	if err != nil || userRole != uint64(uModel.UserRoleAdmin) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	var req tModel.UpdateBrandingColorsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	err = h.BrandingService.UpdateColors(ctx, tenantID, req.PrimaryColor, req.SecondaryColor, req.AccentColor)
	if err != nil {
		writeBrandingError(w, err, tenantID, "update_branding_colors")
		return
	}
	b, _ := h.BrandingService.GetBranding(ctx, tenantID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(b)
}

// UpdateLogo uploads a new tenant logo. Requires admin role. Expects multipart form with field "logo".
func (h *BrandingHandler) UpdateLogo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		http.Error(w, "tenant required", http.StatusBadRequest)
		return
	}
	userRole, err := middleware.GetUserRoleFromContext(ctx)
	if err != nil || userRole != uint64(uModel.UserRoleAdmin) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	files := r.MultipartForm.File["logo"]
	if len(files) == 0 {
		http.Error(w, "No logo file provided", http.StatusBadRequest)
		return
	}
	b, err := h.BrandingService.UpdateLogo(ctx, tenantID, files[0])
	if err != nil {
		writeBrandingError(w, err, tenantID, "update_tenant_logo")
		return
	}
	logger.Info().Uint64("tenant_id", tenantID).Str("logo_url", b.LogoURL).Msg("update_tenant_logo")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(b)
}

func writeBrandingError(w http.ResponseWriter, err error, tenantID uint64, operation string) {
	if errors.Is(err, appErrors.ErrTenantNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var httpErr *appErrors.HTTPError
	if errors.As(err, &httpErr) {
		http.Error(w, httpErr.Error(), httpErr.StatusCode)
		return
	}
	logger.Warn().Err(err).Uint64("tenant_id", tenantID).Str("operation", operation).Msg("branding operation failed")
	http.Error(w, "Failed to update branding", http.StatusInternalServerError)
}
