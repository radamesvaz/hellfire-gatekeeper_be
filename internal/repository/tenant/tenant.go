package tenant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	tModel "github.com/radamesvaz/bakery-app/model/tenant"
)

// Repository provides tenant lookup by slug for path/header resolution.
type Repository struct {
	DB *sql.DB
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

// SlugExists returns true if a tenant with the given slug already exists (any status).
func (r *Repository) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.DB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)`,
		slug,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check slug exists: %w", err)
	}
	return exists, nil
}

// Create inserts a new tenant and returns its data. Caller must ensure slug is unique (e.g. via SlugExists).
func (r *Repository) Create(ctx context.Context, in tModel.CreateTenantInput) (tModel.CreateTenantResponse, error) {
	var currentPeriodEnd sql.NullTime
	if in.CurrentPeriodEnd != nil {
		currentPeriodEnd = sql.NullTime{Time: *in.CurrentPeriodEnd, Valid: true}
	}
	var row tModel.CreateTenantResponse
	var outPeriodEnd sql.NullTime
	err := r.DB.QueryRowContext(ctx,
		`INSERT INTO tenants (name, slug, is_active, ghost_order_timeout_minutes, subscription_status, current_period_end, plan_code)
		 VALUES ($1, $2, true, $3, $4, $5, $6)
		 RETURNING id, name, slug, plan_code, subscription_status, ghost_order_timeout_minutes, is_active, current_period_end`,
		in.Name, in.Slug, in.GhostOrderTimeoutMinutes, in.SubscriptionStatus, currentPeriodEnd, in.PlanCode,
	).Scan(&row.ID, &row.Name, &row.Slug, &row.PlanCode, &row.SubscriptionStatus, &row.GhostOrderTimeoutMinutes, &row.IsActive, &outPeriodEnd)
	if err != nil {
		return tModel.CreateTenantResponse{}, fmt.Errorf("create tenant: %w", err)
	}
	if outPeriodEnd.Valid {
		row.CurrentPeriodEnd = &outPeriodEnd.Time
	}
	return row, nil
}

// UpdateSubscription updates subscription_status and optionally current_period_end for the tenant.
// Returns ErrTenantNotFound if the tenant does not exist.
func (r *Repository) UpdateSubscription(ctx context.Context, tenantID uint64, subscriptionStatus string, currentPeriodEnd *time.Time) error {
	var result sql.Result
	var err error
	if currentPeriodEnd != nil {
		result, err = r.DB.ExecContext(ctx,
			`UPDATE tenants SET subscription_status = $1, current_period_end = $2, updated_on = NOW() WHERE id = $3`,
			subscriptionStatus, *currentPeriodEnd, tenantID,
		)
	} else {
		result, err = r.DB.ExecContext(ctx,
			`UPDATE tenants SET subscription_status = $1, updated_on = NOW() WHERE id = $2`,
			subscriptionStatus, tenantID,
		)
	}
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return appErrors.ErrTenantNotFound
	}
	return nil
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

// Branding holds the branding fields for a tenant.
type Branding struct {
	LogoURL      string
	LogoWidth    int
	LogoHeight   int
	PrimaryColor   string
	SecondaryColor string
	AccentColor    string
}

// GetBranding returns the branding (logo + colors) for the given tenant.
func (r *Repository) GetBranding(ctx context.Context, tenantID uint64) (Branding, error) {
	var b Branding
	var logoURL sql.NullString
	var logoWidth, logoHeight sql.NullInt64
	var primary, secondary, accent sql.NullString
	err := r.DB.QueryRowContext(ctx,
		`SELECT logo_url, logo_width, logo_height, primary_color, secondary_color, accent_color
		 FROM tenants WHERE id = $1`,
		tenantID,
	).Scan(&logoURL, &logoWidth, &logoHeight, &primary, &secondary, &accent)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Branding{}, appErrors.ErrTenantNotFound
		}
		return Branding{}, fmt.Errorf("get branding: %w", err)
	}
	if logoURL.Valid {
		b.LogoURL = logoURL.String
	}
	if logoWidth.Valid {
		b.LogoWidth = int(logoWidth.Int64)
	}
	if logoHeight.Valid {
		b.LogoHeight = int(logoHeight.Int64)
	}
	if primary.Valid {
		b.PrimaryColor = primary.String
	}
	if secondary.Valid {
		b.SecondaryColor = secondary.String
	}
	if accent.Valid {
		b.AccentColor = accent.String
	}
	return b, nil
}

// UpdateColors updates primary_color, secondary_color, accent_color and updated_on for the tenant.
func (r *Repository) UpdateColors(ctx context.Context, tenantID uint64, primary, secondary, accent string) error {
	result, err := r.DB.ExecContext(ctx,
		`UPDATE tenants SET primary_color = $1, secondary_color = $2, accent_color = $3, updated_on = NOW() WHERE id = $4`,
		primary, secondary, accent, tenantID,
	)
	if err != nil {
		return fmt.Errorf("update colors: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return appErrors.ErrTenantNotFound
	}
	return nil
}

// UpdateLogo updates logo_url, logo_width, logo_height and updated_on for the tenant.
func (r *Repository) UpdateLogo(ctx context.Context, tenantID uint64, logoURL string, width, height int) error {
	result, err := r.DB.ExecContext(ctx,
		`UPDATE tenants SET logo_url = $1, logo_width = $2, logo_height = $3, updated_on = NOW() WHERE id = $4`,
		logoURL, width, height, tenantID,
	)
	if err != nil {
		return fmt.Errorf("update logo: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return appErrors.ErrTenantNotFound
	}
	return nil
}
