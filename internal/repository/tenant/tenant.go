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
