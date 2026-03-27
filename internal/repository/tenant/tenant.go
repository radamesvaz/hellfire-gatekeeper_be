package tenant

import (
	"context"
	"database/sql"
	"fmt"
)

// Repository provides tenant lookup by slug for path/header resolution.
type Repository struct {
	DB *sql.DB
}

type BrandingColors struct {
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`
}

// TenantBranding is the full branding snapshot (logo + palette) for a tenant.
type TenantBranding struct {
	LogoURL        string `json:"logo_url"`
	LogoWidth      *int   `json:"logo_width,omitempty"`
	LogoHeight     *int   `json:"logo_height,omitempty"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`
}

type UpdateBrandingColorsRequest struct {
	PrimaryColor   *string `json:"primary_color"`
	SecondaryColor *string `json:"secondary_color"`
	AccentColor    *string `json:"accent_color"`
}

// GetBySlug returns the tenant id and active status for the given slug.
// Only returns a tenant when it is active, subscription_status is 'active', and the current period has not ended.
// Implements middleware.TenantResolver. Returns (0, false, err) when not found, inactive, or subscription expired.
func (r *Repository) GetBySlug(ctx context.Context, slug string) (id uint64, isActive bool, err error) {
	err = r.DB.QueryRowContext(ctx,
		`SELECT id, is_active FROM tenants
		 WHERE slug = $1
		   AND is_active = true
		   AND subscription_status = 'active'
		   AND (current_period_end IS NULL OR current_period_end > NOW())`,
		slug,
	).Scan(&id, &isActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, false, fmt.Errorf("tenant not found or not available: %s", slug)
		}
		return 0, false, fmt.Errorf("tenant lookup: %w", err)
	}
	return id, true, nil
}

// GetGhostOrderTimeoutMinutes returns the per-tenant ghost order timeout (in minutes)
// from the tenants table. It relies on the snapshot stored in
// tenants.ghost_order_timeout_minutes and does not apply any additional logic.
func (r *Repository) GetGhostOrderTimeoutMinutes(ctx context.Context, tenantID uint64) (int, error) {
	var minutes int
	err := r.DB.QueryRowContext(ctx,
		`SELECT ghost_order_timeout_minutes FROM tenants WHERE id = $1`,
		tenantID,
	).Scan(&minutes)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("tenant not found when reading ghost_order_timeout_minutes: %d", tenantID)
		}
		return 0, fmt.Errorf("reading ghost_order_timeout_minutes for tenant %d: %w", tenantID, err)
	}
	return minutes, nil
}

// ListActiveTenantIDs returns the IDs of tenants that are currently active and
// whose subscription is valid. It mirrors the availability conditions used in
// GetBySlug so that public and background flows see a consistent view of which
// tenants can operate.
func (r *Repository) ListActiveTenantIDs(ctx context.Context) ([]uint64, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id FROM tenants
		 WHERE is_active = true
		   AND subscription_status = 'active'
		   AND (current_period_end IS NULL OR current_period_end > NOW())`,
	)
	if err != nil {
		return nil, fmt.Errorf("list active tenants: %w", err)
	}
	defer rows.Close()

	var ids []uint64
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan active tenant id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active tenant ids: %w", err)
	}
	return ids, nil
}

// GetBranding returns logo fields and branding colors for a tenant in one read.
func (r *Repository) GetBranding(ctx context.Context, tenantID uint64) (TenantBranding, error) {
	var logoURL sql.NullString
	var logoWidth, logoHeight sql.NullInt64
	var primaryColor, secondaryColor, accentColor sql.NullString

	err := r.DB.QueryRowContext(ctx,
		`SELECT logo_url, logo_width, logo_height, primary_color, secondary_color, accent_color
		 FROM tenants
		 WHERE id = $1`,
		tenantID,
	).Scan(&logoURL, &logoWidth, &logoHeight, &primaryColor, &secondaryColor, &accentColor)
	if err != nil {
		if err == sql.ErrNoRows {
			return TenantBranding{}, fmt.Errorf("tenant not found when reading branding: %d", tenantID)
		}
		return TenantBranding{}, fmt.Errorf("reading branding for tenant %d: %w", tenantID, err)
	}

	out := TenantBranding{
		LogoURL:        nullStringToString(logoURL),
		PrimaryColor:   nullStringToString(primaryColor),
		SecondaryColor: nullStringToString(secondaryColor),
		AccentColor:    nullStringToString(accentColor),
	}
	if logoWidth.Valid {
		w := int(logoWidth.Int64)
		out.LogoWidth = &w
	}
	if logoHeight.Valid {
		h := int(logoHeight.Int64)
		out.LogoHeight = &h
	}
	return out, nil
}

// GetBrandingColors returns the branding colors configured for a tenant.
func (r *Repository) GetBrandingColors(ctx context.Context, tenantID uint64) (BrandingColors, error) {
	branding, err := r.GetBranding(ctx, tenantID)
	if err != nil {
		return BrandingColors{}, err
	}
	return BrandingColors{
		PrimaryColor:   branding.PrimaryColor,
		SecondaryColor: branding.SecondaryColor,
		AccentColor:    branding.AccentColor,
	}, nil
}

// UpdateBrandingColors partially updates tenant branding colors.
func (r *Repository) UpdateBrandingColors(ctx context.Context, tenantID uint64, req UpdateBrandingColorsRequest) error {
	result, err := r.DB.ExecContext(ctx,
		`UPDATE tenants
		 SET primary_color = COALESCE($1, primary_color),
		     secondary_color = COALESCE($2, secondary_color),
		     accent_color = COALESCE($3, accent_color),
		     updated_on = NOW()
		 WHERE id = $4`,
		req.PrimaryColor,
		req.SecondaryColor,
		req.AccentColor,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("updating branding colors for tenant %d: %w", tenantID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("reading affected rows for tenant %d branding colors update: %w", tenantID, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found when updating branding colors: %d", tenantID)
	}

	return nil
}

func nullStringToString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
