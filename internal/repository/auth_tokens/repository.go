package auth_tokens

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

var (
	ErrTokenNotFound = errors.New("action token not found")
)

type SQLRepository struct {
	DB *sql.DB
}

type CreateTokenInput struct {
	TenantID        uint64
	Email           string
	Purpose         authModel.ActionTokenPurpose
	TokenHash       string
	ExpiresAt       time.Time
	SubjectUserID   *uint64
	CreatedByUserID *uint64
	MetadataJSON    []byte
}

type TokenRecord struct {
	ID            uint64
	TenantID      uint64
	Email         string
	Purpose       authModel.ActionTokenPurpose
	SubjectUserID *uint64
	MetadataJSON  []byte
	ExpiresAt     time.Time
	UsedAt        *time.Time
	RevokedAt     *time.Time
}

func (r *SQLRepository) CreateToken(ctx context.Context, in CreateTokenInput) (uint64, error) {
	var id uint64
	err := r.DB.QueryRowContext(ctx,
		`INSERT INTO auth_action_tokens (
            tenant_id, email, purpose, token_hash, subject_user_id, expires_at, used_at, revoked_at, created_by_user_id, metadata_json
        ) VALUES ($1, $2, $3, $4, $5, $6, NULL, NULL, $7, $8)
        RETURNING id`,
		in.TenantID,
		in.Email,
		string(in.Purpose),
		in.TokenHash,
		toNullUint64(in.SubjectUserID),
		in.ExpiresAt,
		toNullUint64(in.CreatedByUserID),
		toNullBytes(in.MetadataJSON),
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create action token: %w", err)
	}
	return id, nil
}

func (r *SQLRepository) GetTokenForUpdate(ctx context.Context, tx *sql.Tx, tenantID uint64, purpose authModel.ActionTokenPurpose, tokenHash string) (TokenRecord, error) {
	var (
		rec              TokenRecord
		subjectUserIDRaw sql.NullInt64
		usedAtRaw        sql.NullTime
		revokedAtRaw     sql.NullTime
	)

	err := tx.QueryRowContext(ctx,
		`SELECT id, tenant_id, email, purpose, subject_user_id, metadata_json, expires_at, used_at, revoked_at
         FROM auth_action_tokens
         WHERE tenant_id = $1 AND purpose = $2 AND token_hash = $3
         FOR UPDATE`,
		tenantID,
		string(purpose),
		tokenHash,
	).Scan(
		&rec.ID,
		&rec.TenantID,
		&rec.Email,
		&rec.Purpose,
		&subjectUserIDRaw,
		&rec.MetadataJSON,
		&rec.ExpiresAt,
		&usedAtRaw,
		&revokedAtRaw,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TokenRecord{}, ErrTokenNotFound
		}
		return TokenRecord{}, fmt.Errorf("get action token for update: %w", err)
	}

	rec.SubjectUserID = fromNullUint64(subjectUserIDRaw)
	rec.UsedAt = fromNullTime(usedAtRaw)
	rec.RevokedAt = fromNullTime(revokedAtRaw)

	return rec, nil
}

func (r *SQLRepository) GetTokenByIDForUpdate(ctx context.Context, tx *sql.Tx, id uint64) (TokenRecord, error) {
	var (
		rec              TokenRecord
		subjectUserIDRaw sql.NullInt64
		usedAtRaw        sql.NullTime
		revokedAtRaw     sql.NullTime
	)

	err := tx.QueryRowContext(ctx,
		`SELECT id, tenant_id, email, purpose, subject_user_id, metadata_json, expires_at, used_at, revoked_at
         FROM auth_action_tokens
         WHERE id = $1
         FOR UPDATE`,
		id,
	).Scan(
		&rec.ID,
		&rec.TenantID,
		&rec.Email,
		&rec.Purpose,
		&subjectUserIDRaw,
		&rec.MetadataJSON,
		&rec.ExpiresAt,
		&usedAtRaw,
		&revokedAtRaw,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TokenRecord{}, ErrTokenNotFound
		}
		return TokenRecord{}, fmt.Errorf("get action token by id for update: %w", err)
	}

	rec.SubjectUserID = fromNullUint64(subjectUserIDRaw)
	rec.UsedAt = fromNullTime(usedAtRaw)
	rec.RevokedAt = fromNullTime(revokedAtRaw)

	return rec, nil
}

func (r *SQLRepository) ConsumeToken(ctx context.Context, tx *sql.Tx, id uint64) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE auth_action_tokens SET used_at = NOW(), updated_on = NOW() WHERE id = $1`,
		id,
	); err != nil {
		return fmt.Errorf("consume action token: %w", err)
	}
	return nil
}

func (r *SQLRepository) RevokeToken(ctx context.Context, tx *sql.Tx, id uint64) error {
	if _, err := tx.ExecContext(ctx,
		`UPDATE auth_action_tokens SET revoked_at = NOW(), updated_on = NOW() WHERE id = $1`,
		id,
	); err != nil {
		return fmt.Errorf("revoke action token: %w", err)
	}
	return nil
}

func toNullUint64(v *uint64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func toNullBytes(v []byte) interface{} {
	if len(v) == 0 {
		return nil
	}
	return v
}

func fromNullUint64(v sql.NullInt64) *uint64 {
	if !v.Valid || v.Int64 <= 0 {
		return nil
	}
	out := uint64(v.Int64)
	return &out
}

func fromNullTime(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}
