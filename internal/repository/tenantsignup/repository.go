package tenantsignup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

var (
	ErrInvalidOrUnavailableCode = errors.New("invalid or unavailable one-time code")
	ErrTenantSlugExists         = errors.New("tenant slug already exists")
	ErrAdminEmailExists         = errors.New("admin email already exists in tenant")
)

type Repository struct {
	DB *sql.DB
}

type RegisterTenantWithCodeInput struct {
	CodeHash     string
	TenantName   string
	TenantSlug   string
	AdminName    string
	AdminEmail   string
	AdminPhone   string
	PasswordHash string
}

type RegisterTenantWithCodeResult struct {
	TenantID uint64
	AdminID  uint64
}

func (r *Repository) CreateSignupCode(ctx context.Context, codeHash string, expiresAt time.Time, createdByUserID uint64, notes string) (uint64, error) {
	var id uint64
	err := r.DB.QueryRowContext(
		ctx,
		`INSERT INTO tenant_signup_codes (code_hash, expires_at, created_by_user_id, notes)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		codeHash,
		expiresAt,
		createdByUserID,
		strings.TrimSpace(notes),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create signup code: %w", err)
	}
	return id, nil
}

func (r *Repository) RegisterTenantWithCode(
	ctx context.Context,
	in RegisterTenantWithCodeInput,
) (RegisterTenantWithCodeResult, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return RegisterTenantWithCodeResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var (
		signupCodeID uint64
		expiresAt    time.Time
		usedAt       sql.NullTime
		revokedAt    sql.NullTime
	)
	err = tx.QueryRowContext(
		ctx,
		`SELECT id, expires_at, used_at, revoked_at
		 FROM tenant_signup_codes
		 WHERE code_hash = $1
		 FOR UPDATE`,
		in.CodeHash,
	).Scan(&signupCodeID, &expiresAt, &usedAt, &revokedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RegisterTenantWithCodeResult{}, ErrInvalidOrUnavailableCode
		}
		return RegisterTenantWithCodeResult{}, fmt.Errorf("validate signup code: %w", err)
	}
	if usedAt.Valid || revokedAt.Valid || !expiresAt.After(time.Now().UTC()) {
		return RegisterTenantWithCodeResult{}, ErrInvalidOrUnavailableCode
	}

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
			return RegisterTenantWithCodeResult{}, ErrTenantSlugExists
		}
		return RegisterTenantWithCodeResult{}, fmt.Errorf("create tenant: %w", err)
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
			return RegisterTenantWithCodeResult{}, ErrAdminEmailExists
		}
		return RegisterTenantWithCodeResult{}, fmt.Errorf("create admin user: %w", err)
	}

	if _, err = tx.ExecContext(ctx,
		`UPDATE tenant_signup_codes
		 SET used_at = NOW(),
		     used_by_tenant_id = $2,
		     used_by_user_id = $3,
		     used_email = $4,
		     updated_on = NOW()
		 WHERE id = $1`,
		signupCodeID,
		tenantID,
		adminID,
		in.AdminEmail,
	); err != nil {
		return RegisterTenantWithCodeResult{}, fmt.Errorf("consume signup code: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return RegisterTenantWithCodeResult{}, fmt.Errorf("commit tx: %w", err)
	}

	return RegisterTenantWithCodeResult{
		TenantID: tenantID,
		AdminID:  adminID,
	}, nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && string(pqErr.Code) == "23505"
}
