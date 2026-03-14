package branding

import (
	"context"
	"mime/multipart"
	"regexp"
	"strings"

	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/logger"
	"github.com/radamesvaz/bakery-app/internal/repository/tenant"
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"
	tModel "github.com/radamesvaz/bakery-app/model/tenant"
)

// hexColorRegex matches #RRGGBB (7 chars)
var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// Service provides tenant branding operations.
type Service struct {
	TenantRepo   *tenant.Repository
	ImageService *imagesService.Service
}

// New returns a new branding service.
func New(tenantRepo *tenant.Repository, imageService *imagesService.Service) *Service {
	return &Service{
		TenantRepo:   tenantRepo,
		ImageService: imageService,
	}
}

// GetBranding returns the branding for the given tenant.
func (s *Service) GetBranding(ctx context.Context, tenantID uint64) (tModel.Branding, error) {
	b, err := s.TenantRepo.GetBranding(ctx, tenantID)
	if err != nil {
		return tModel.Branding{}, err
	}
	return tModel.Branding{
		LogoURL:        b.LogoURL,
		LogoWidth:      b.LogoWidth,
		LogoHeight:     b.LogoHeight,
		PrimaryColor:   b.PrimaryColor,
		SecondaryColor: b.SecondaryColor,
		AccentColor:    b.AccentColor,
	}, nil
}

// ValidateHexColor returns nil if s is empty or a valid #RRGGBB hex color.
func ValidateHexColor(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if !hexColorRegex.MatchString(s) {
		return appErrors.ErrInvalidColorFormat
	}
	return nil
}

// NormalizeHexColor returns uppercase #RRGGBB or empty string.
func NormalizeHexColor(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return strings.ToUpper(s)
}

// UpdateColors updates the tenant's brand colors. All three must be valid (empty or #RRGGBB).
func (s *Service) UpdateColors(ctx context.Context, tenantID uint64, primary, secondary, accent string) error {
	if err := ValidateHexColor(primary); err != nil {
		return err
	}
	if err := ValidateHexColor(secondary); err != nil {
		return err
	}
	if err := ValidateHexColor(accent); err != nil {
		return err
	}
	primary = NormalizeHexColor(primary)
	secondary = NormalizeHexColor(secondary)
	accent = NormalizeHexColor(accent)
	return s.TenantRepo.UpdateColors(ctx, tenantID, primary, secondary, accent)
}

// UpdateLogo uploads a new logo and updates the tenant record. Deletes the previous logo file if present.
func (s *Service) UpdateLogo(ctx context.Context, tenantID uint64, file *multipart.FileHeader) (tModel.Branding, error) {
	current, err := s.TenantRepo.GetBranding(ctx, tenantID)
	if err != nil {
		return tModel.Branding{}, err
	}
	logoURL, width, height, err := s.ImageService.SaveTenantLogo(tenantID, file)
	if err != nil {
		return tModel.Branding{}, err
	}
	if current.LogoURL != "" {
		if delErr := s.ImageService.DeleteImage(current.LogoURL); delErr != nil {
			logger.Warn().Err(delErr).Str("logo_url", current.LogoURL).Uint64("tenant_id", tenantID).Msg("failed to delete previous tenant logo file")
		}
	}
	if err := s.TenantRepo.UpdateLogo(ctx, tenantID, logoURL, width, height); err != nil {
		return tModel.Branding{}, err
	}
	return tModel.Branding{
		LogoURL:        logoURL,
		LogoWidth:      width,
		LogoHeight:     height,
		PrimaryColor:   current.PrimaryColor,
		SecondaryColor: current.SecondaryColor,
		AccentColor:    current.AccentColor,
	}, nil
}
