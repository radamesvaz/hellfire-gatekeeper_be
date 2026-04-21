package bootstrap

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

var (
	ErrTenantSlugExists = errors.New("tenant slug already exists")
	ErrAdminEmailExists = errors.New("admin email already exists in tenant")
)

type Repository struct {
	DB *sql.DB
}

type BootstrapTenantInput struct {
	TenantName   string
	TenantSlug   string
	AdminName    string
	AdminEmail   string
	AdminPhone   string
	PasswordHash string
}

type BootstrapTenantResult struct {
	TenantID uint64
	AdminID  uint64
}

func (r *Repository) BootstrapTenant(ctx context.Context, in BootstrapTenantInput) (BootstrapTenantResult, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return BootstrapTenantResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var tenantID uint64
	err = tx.QueryRowContext(ctx,
		`INSERT INTO tenants (name, slug, is_active, ghost_order_timeout_minutes, subscription_status, plan_code)
		 VALUES ($1, $2, TRUE, 30, 'active', 'basic')
		 RETURNING id`,
		in.TenantName,
		in.TenantSlug,
	).Scan(&tenantID)
	if err != nil {
		if isUniqueViolation(err) {
			return BootstrapTenantResult{}, ErrTenantSlugExists
		}
		return BootstrapTenantResult{}, fmt.Errorf("create tenant: %w", err)
	}

	var adminID uint64
	err = tx.QueryRowContext(ctx,
		`INSERT INTO users (tenant_id, id_role, name, email, phone, password_hash)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id_user`,
		tenantID,
		uModel.UserRoleAdmin,
		in.AdminName,
		in.AdminEmail,
		in.AdminPhone,
		in.PasswordHash,
	).Scan(&adminID)
	if err != nil {
		if isUniqueViolation(err) {
			return BootstrapTenantResult{}, ErrAdminEmailExists
		}
		return BootstrapTenantResult{}, fmt.Errorf("create admin user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return BootstrapTenantResult{}, fmt.Errorf("commit tx: %w", err)
	}
	return BootstrapTenantResult{TenantID: tenantID, AdminID: adminID}, nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && string(pqErr.Code) == "23505"
}
