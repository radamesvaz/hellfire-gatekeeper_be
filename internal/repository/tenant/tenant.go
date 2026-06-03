package tenant

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	tenantModel "github.com/radamesvaz/bakery-app/model/tenant"
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
// Logo pixel dimensions are not stored; layout is fixed in CSS and uploads are validated on the server.
// TenantName maps tenants.name (JSON key tenant_name for clients).
type TenantBranding struct {
	TenantName     string `json:"tenant_name"`
	LogoURL        string `json:"logo_url"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`
}

type UpdateBrandingColorsRequest struct {
	PrimaryColor   *string `json:"primary_color"`
	SecondaryColor *string `json:"secondary_color"`
	AccentColor    *string `json:"accent_color"`
}

type SubscriptionSnapshot struct {
	Status           tenantModel.SubscriptionStatus
	PlanCode         string
	CurrentPeriodEnd sql.NullTime
}

// GetSlugByTenantID returns the canonical slug for an existing tenant row.
// Used by authenticated /auth/* routes where the path does not include {tenant_slug}.
func (r *Repository) GetSlugByTenantID(ctx context.Context, tenantID uint64) (string, error) {
	var slug string
	err := r.DB.QueryRowContext(ctx, `SELECT slug FROM tenants WHERE id = $1`, tenantID).Scan(&slug)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("tenant not found when reading slug: %d", tenantID)
		}
		return "", fmt.Errorf("reading tenant slug for %d: %w", tenantID, err)
	}
	return slug, nil
}

// GetBySlug returns the tenant id and active status for the given slug.
// Only returns a tenant when it is active, subscription_status is in ('active','pending'),
// and the current period has not ended.
// Implements middleware.TenantResolver. Returns (0, false, err) when not found, inactive, or subscription expired.
func (r *Repository) GetBySlug(ctx context.Context, slug string) (id uint64, isActive bool, err error) {
	err = r.DB.QueryRowContext(ctx,
		`SELECT id, is_active FROM tenants
		 WHERE slug = $1
		   AND is_active = true
		   AND subscription_status IN ('active', 'pending')
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
		   AND subscription_status IN ('active', 'pending')
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

func (r *Repository) GetSubscriptionSnapshot(ctx context.Context, tenantID uint64) (SubscriptionSnapshot, error) {
	var snapshot SubscriptionSnapshot
	err := r.DB.QueryRowContext(ctx,
		`SELECT subscription_status, COALESCE(plan_code, ''), current_period_end
		 FROM tenants
		 WHERE id = $1`,
		tenantID,
	).Scan(&snapshot.Status, &snapshot.PlanCode, &snapshot.CurrentPeriodEnd)
	if err != nil {
		if err == sql.ErrNoRows {
			return SubscriptionSnapshot{}, fmt.Errorf("tenant not found when reading subscription snapshot: %d", tenantID)
		}
		return SubscriptionSnapshot{}, fmt.Errorf("reading subscription snapshot for tenant %d: %w", tenantID, err)
	}
	return snapshot, nil
}

func (r *Repository) MarkExpiredActiveAsPending(ctx context.Context, now time.Time) (int64, error) {
	result, err := r.DB.ExecContext(ctx,
		`UPDATE tenants
		 SET subscription_status = $2,
		     updated_on = NOW()
		 WHERE is_active = true
		   AND subscription_status = $3
		   AND current_period_end IS NOT NULL
		   AND current_period_end <= $1`,
		now.UTC(),
		tenantModel.SubscriptionStatusPending,
		tenantModel.SubscriptionStatusActive,
	)
	if err != nil {
		return 0, fmt.Errorf("mark expired active subscriptions as pending: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("reading rows affected when marking pending subscriptions: %w", err)
	}
	return affected, nil
}

// UpdateTenantSubscription sets subscription_status and optionally current_period_end for a tenant.
// When updatePeriodEnd is false, current_period_end is left unchanged.
func (r *Repository) UpdateTenantSubscription(
	ctx context.Context,
	tenantID uint64,
	status tenantModel.SubscriptionStatus,
	periodEnd sql.NullTime,
	updatePeriodEnd bool,
) error {
	if !status.Valid() {
		return fmt.Errorf("invalid subscription status: %q", status)
	}
	result, err := r.DB.ExecContext(ctx,
		`UPDATE tenants
		 SET subscription_status = $1,
		     current_period_end = CASE WHEN $4::boolean THEN $2 ELSE current_period_end END,
		     updated_on = NOW()
		 WHERE id = $3`,
		status,
		periodEnd,
		tenantID,
		updatePeriodEnd,
	)
	if err != nil {
		return fmt.Errorf("updating tenant subscription for %d: %w", tenantID, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("reading rows affected for tenant subscription update %d: %w", tenantID, err)
	}
	if n == 0 {
		return fmt.Errorf("tenant not found when updating subscription: %d", tenantID)
	}
	return nil
}

func (r *Repository) MarkExpiredPendingAsCanceled(ctx context.Context, now time.Time, graceDays int) (int64, error) {
	if graceDays < 0 {
		graceDays = 0
	}

	result, err := r.DB.ExecContext(ctx,
		`UPDATE tenants
		 SET subscription_status = $3,
		     updated_on = NOW()
		 WHERE is_active = true
		   AND subscription_status = $4
		   AND current_period_end IS NOT NULL
		   AND (current_period_end + make_interval(days => $2)) <= $1`,
		now.UTC(),
		graceDays,
		tenantModel.SubscriptionStatusCanceled,
		tenantModel.SubscriptionStatusPending,
	)
	if err != nil {
		return 0, fmt.Errorf("mark expired pending subscriptions as canceled: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("reading rows affected when marking canceled subscriptions: %w", err)
	}
	return affected, nil
}

// GetBranding returns tenant display name, logo fields and branding colors for a tenant in one read.
func (r *Repository) GetBranding(ctx context.Context, tenantID uint64) (TenantBranding, error) {
	var displayName string
	var logoURL sql.NullString
	var primaryColor, secondaryColor, accentColor sql.NullString

	err := r.DB.QueryRowContext(ctx,
		`SELECT name, logo_url, primary_color, secondary_color, accent_color
		 FROM tenants
		 WHERE id = $1`,
		tenantID,
	).Scan(&displayName, &logoURL, &primaryColor, &secondaryColor, &accentColor)
	if err != nil {
		if err == sql.ErrNoRows {
			return TenantBranding{}, fmt.Errorf("tenant not found when reading branding: %d", tenantID)
		}
		return TenantBranding{}, fmt.Errorf("reading branding for tenant %d: %w", tenantID, err)
	}

	return TenantBranding{
		TenantName:     displayName,
		LogoURL:        nullStringToString(logoURL),
		PrimaryColor:   nullStringToString(primaryColor),
		SecondaryColor: nullStringToString(secondaryColor),
		AccentColor:    nullStringToString(accentColor),
	}, nil
}

// GetTenantName returns tenants.name for the given tenant id.
func (r *Repository) GetTenantName(ctx context.Context, tenantID uint64) (string, error) {
	var name string
	err := r.DB.QueryRowContext(ctx, `SELECT name FROM tenants WHERE id = $1`, tenantID).Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("tenant not found when reading name: %d", tenantID)
		}
		return "", fmt.Errorf("reading tenant name for %d: %w", tenantID, err)
	}
	return name, nil
}

// UpdateTenantName sets tenants.name (business / display name). Does not change slug.
func (r *Repository) UpdateTenantName(ctx context.Context, tenantID uint64, name string) error {
	result, err := r.DB.ExecContext(ctx,
		`UPDATE tenants SET name = $1, updated_on = NOW() WHERE id = $2`,
		name,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("updating tenant name for %d: %w", tenantID, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("reading rows affected for tenant name update %d: %w", tenantID, err)
	}
	if n == 0 {
		return fmt.Errorf("tenant not found when updating name: %d", tenantID)
	}
	return nil
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

// UpdateTenantLogoURL sets the tenant logo URL (after a successful upload).
func (r *Repository) UpdateTenantLogoURL(ctx context.Context, tenantID uint64, logoURL string) error {
	result, err := r.DB.ExecContext(ctx,
		`UPDATE tenants SET logo_url = $1, updated_on = NOW() WHERE id = $2`,
		logoURL,
		tenantID,
	)
	if err != nil {
		return fmt.Errorf("updating tenant logo for tenant %d: %w", tenantID, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("reading rows affected for tenant logo update %d: %w", tenantID, err)
	}
	if n == 0 {
		return fmt.Errorf("tenant not found when updating logo: %d", tenantID)
	}
	return nil
}

func nullStringToString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
