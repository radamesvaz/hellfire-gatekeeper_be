package auth_tokens

import (
	"context"
	"database/sql"
	stdErrors "errors"
	"fmt"
	"strings"
	"time"

	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	repo "github.com/radamesvaz/bakery-app/internal/repository/auth_tokens"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

const (
	DefaultInviteTTLMinutes        = 60 * 24 * 7
	DefaultPasswordResetTTLMinutes = 60
)

type ActionTokenService struct {
	DB          *sql.DB
	Repo        repo.Repository
	AuthService authService.Service
}

func (s *ActionTokenService) CreateToken(
	ctx context.Context,
	req authModel.CreateActionTokenRequest,
) (authModel.CreateActionTokenResponse, error) {
	purpose := normalizePurpose(req.Purpose)
	if !isAllowedPurpose(purpose) {
		return authModel.CreateActionTokenResponse{}, appErrors.ErrInvalidTokenPurpose
	}
	ttl := req.ExpiresInMinutes
	if ttl <= 0 {
		ttl = defaultTTLByPurpose(purpose)
	}
	expiresAt := time.Now().UTC().Add(time.Duration(ttl) * time.Minute)

	plainToken, hash, err := s.AuthService.GenerateOneTimeToken()
	if err != nil {
		return authModel.CreateActionTokenResponse{}, fmt.Errorf("generate action token: %w", err)
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return authModel.CreateActionTokenResponse{}, fmt.Errorf("begin tx create action token: %w", err)
	}
	defer tx.Rollback()

	id, err := s.Repo.CreateTokenTx(ctx, tx, repo.CreateTokenInput{
		TenantID:        req.TenantID,
		Email:           strings.TrimSpace(req.Email),
		Purpose:         purpose,
		TokenHash:       hash,
		ExpiresAt:       expiresAt,
		SubjectUserID:   req.SubjectUserID,
		CreatedByUserID: req.CreatedByUserID,
		MetadataJSON:    req.MetadataJSON,
	})
	if err != nil {
		return authModel.CreateActionTokenResponse{}, err
	}

	hist := repo.InsertHistoryInput{
		TenantID:          req.TenantID,
		AuthActionTokenID: id,
		Purpose:           purpose,
	}
	switch purpose {
	case authModel.ActionTokenPurposeInvite:
		hist.Action = authModel.ActionTokenInviteCreated
		hist.ModifiedByUserID = req.CreatedByUserID
	case authModel.ActionTokenPurposePasswordReset:
		hist.Action = authModel.ActionTokenPasswordResetIssued
		hist.SubjectUserID = req.SubjectUserID
	}
	if err := s.Repo.InsertHistory(ctx, tx, hist); err != nil {
		return authModel.CreateActionTokenResponse{}, err
	}

	if err := tx.Commit(); err != nil {
		return authModel.CreateActionTokenResponse{}, fmt.Errorf("commit create action token: %w", err)
	}

	return authModel.CreateActionTokenResponse{
		ID:        id,
		Token:     plainToken,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *ActionTokenService) ValidateToken(ctx context.Context, tenantID uint64, purpose authModel.ActionTokenPurpose, plainToken string) (authModel.ActionTokenRecord, error) {
	return s.loadValidatedToken(ctx, nil, tenantID, purpose, plainToken)
}

func (s *ActionTokenService) ConsumeToken(ctx context.Context, tenantID uint64, purpose authModel.ActionTokenPurpose, plainToken string) (authModel.ActionTokenRecord, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return authModel.ActionTokenRecord{}, fmt.Errorf("begin tx consume action token: %w", err)
	}
	defer tx.Rollback()

	rec, err := s.loadValidatedToken(ctx, tx, tenantID, purpose, plainToken)
	if err != nil {
		return authModel.ActionTokenRecord{}, err
	}
	if err := s.Repo.ConsumeToken(ctx, tx, rec.ID); err != nil {
		return authModel.ActionTokenRecord{}, err
	}
	if normalizePurpose(purpose) == authModel.ActionTokenPurposePasswordReset {
		if err := s.Repo.InsertHistory(ctx, tx, repo.InsertHistoryInput{
			TenantID:           rec.TenantID,
			AuthActionTokenID:  rec.ID,
			Purpose:            authModel.ActionTokenPurposePasswordReset,
			Action:             authModel.ActionTokenPasswordResetCompleted,
			SubjectUserID:      rec.SubjectUserID,
			ModifiedByUserID:   nil,
			MetadataJSON:       nil,
		}); err != nil {
			return authModel.ActionTokenRecord{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return authModel.ActionTokenRecord{}, fmt.Errorf("commit consume action token: %w", err)
	}
	now := time.Now().UTC()
	rec.UsedAt = &now
	return rec, nil
}

func (s *ActionTokenService) RevokeToken(ctx context.Context, tokenID uint64) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx revoke action token: %w", err)
	}
	defer tx.Rollback()

	if err := s.Repo.RevokeToken(ctx, tx, tokenID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit revoke action token: %w", err)
	}
	return nil
}

func (s *ActionTokenService) RecordInvitationAccepted(ctx context.Context, tenantID uint64, tokenID uint64, newUserID uint64) error {
	if tenantID == 0 || tokenID == 0 || newUserID == 0 {
		return fmt.Errorf("record invitation accepted: invalid ids")
	}
	return s.Repo.InsertHistory(ctx, nil, repo.InsertHistoryInput{
		TenantID:           tenantID,
		AuthActionTokenID:  tokenID,
		Purpose:            authModel.ActionTokenPurposeInvite,
		Action:             authModel.ActionTokenInviteAccepted,
		SubjectUserID:      &newUserID,
		ModifiedByUserID:   nil,
		MetadataJSON:       nil,
	})
}

func (s *ActionTokenService) RevokeTokenScoped(ctx context.Context, tenantID uint64, purpose authModel.ActionTokenPurpose, tokenID uint64, revokedByUserID *uint64) error {
	purpose = normalizePurpose(purpose)
	if !isAllowedPurpose(purpose) {
		return appErrors.ErrInvalidTokenPurpose
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx scoped revoke action token: %w", err)
	}
	defer tx.Rollback()

	row, err := s.Repo.GetTokenByIDForUpdate(ctx, tx, tokenID)
	if err != nil {
		if stdErrors.Is(err, repo.ErrTokenNotFound) {
			return appErrors.ErrInvalidToken
		}
		return err
	}
	if row.TenantID != tenantID || row.Purpose != purpose {
		return appErrors.ErrInvalidToken
	}
	if row.UsedAt != nil {
		return appErrors.ErrTokenAlreadyConsumed
	}
	if row.RevokedAt != nil {
		return nil
	}

	if err := s.Repo.RevokeToken(ctx, tx, tokenID); err != nil {
		return err
	}
	if revokedByUserID != nil && purpose == authModel.ActionTokenPurposeInvite {
		if err := s.Repo.InsertHistory(ctx, tx, repo.InsertHistoryInput{
			TenantID:           row.TenantID,
			AuthActionTokenID:  tokenID,
			Purpose:            authModel.ActionTokenPurposeInvite,
			Action:             authModel.ActionTokenInviteRevoked,
			ModifiedByUserID:   revokedByUserID,
			SubjectUserID:      nil,
			MetadataJSON:       nil,
		}); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit scoped revoke action token: %w", err)
	}
	return nil
}

func (s *ActionTokenService) GetTokenByIDScoped(ctx context.Context, tenantID uint64, purpose authModel.ActionTokenPurpose, tokenID uint64) (authModel.ActionTokenRecord, error) {
	purpose = normalizePurpose(purpose)
	if !isAllowedPurpose(purpose) {
		return authModel.ActionTokenRecord{}, appErrors.ErrInvalidTokenPurpose
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return authModel.ActionTokenRecord{}, fmt.Errorf("begin tx scoped get action token: %w", err)
	}
	defer tx.Rollback()

	row, err := s.Repo.GetTokenByIDForUpdate(ctx, tx, tokenID)
	if err != nil {
		if stdErrors.Is(err, repo.ErrTokenNotFound) {
			return authModel.ActionTokenRecord{}, appErrors.ErrInvalidToken
		}
		return authModel.ActionTokenRecord{}, err
	}
	if row.TenantID != tenantID || row.Purpose != purpose {
		return authModel.ActionTokenRecord{}, appErrors.ErrInvalidToken
	}

	if err := tx.Commit(); err != nil {
		return authModel.ActionTokenRecord{}, fmt.Errorf("commit scoped get action token: %w", err)
	}

	return authModel.ActionTokenRecord{
		ID:            row.ID,
		TenantID:      row.TenantID,
		Email:         row.Email,
		Purpose:       row.Purpose,
		SubjectUserID: row.SubjectUserID,
		MetadataJSON:  row.MetadataJSON,
		ExpiresAt:     row.ExpiresAt,
		UsedAt:        row.UsedAt,
		RevokedAt:     row.RevokedAt,
	}, nil
}

func (s *ActionTokenService) loadValidatedToken(ctx context.Context, tx *sql.Tx, tenantID uint64, purpose authModel.ActionTokenPurpose, plainToken string) (authModel.ActionTokenRecord, error) {
	purpose = normalizePurpose(purpose)
	if !isAllowedPurpose(purpose) {
		return authModel.ActionTokenRecord{}, appErrors.ErrInvalidTokenPurpose
	}
	tokenHash := s.AuthService.HashOneTimeToken(plainToken)

	ownTx := tx
	var err error
	closeOwnTx := false
	if ownTx == nil {
		ownTx, err = s.DB.BeginTx(ctx, nil)
		if err != nil {
			return authModel.ActionTokenRecord{}, fmt.Errorf("begin tx validate action token: %w", err)
		}
		closeOwnTx = true
	}
	if closeOwnTx {
		defer ownTx.Rollback()
	}

	row, err := s.Repo.GetTokenForUpdate(ctx, ownTx, tenantID, purpose, tokenHash)
	if err != nil {
		if stdErrors.Is(err, repo.ErrTokenNotFound) {
			return authModel.ActionTokenRecord{}, appErrors.ErrInvalidToken
		}
		return authModel.ActionTokenRecord{}, err
	}

	if row.RevokedAt != nil {
		return authModel.ActionTokenRecord{}, appErrors.ErrTokenRevoked
	}
	if row.UsedAt != nil {
		return authModel.ActionTokenRecord{}, appErrors.ErrTokenAlreadyConsumed
	}
	if !row.ExpiresAt.After(time.Now().UTC()) {
		return authModel.ActionTokenRecord{}, appErrors.ErrExpiredToken
	}

	rec := authModel.ActionTokenRecord{
		ID:            row.ID,
		TenantID:      row.TenantID,
		Email:         row.Email,
		Purpose:       row.Purpose,
		SubjectUserID: row.SubjectUserID,
		MetadataJSON:  row.MetadataJSON,
		ExpiresAt:     row.ExpiresAt,
		UsedAt:        row.UsedAt,
		RevokedAt:     row.RevokedAt,
	}

	if closeOwnTx {
		if err := ownTx.Commit(); err != nil {
			return authModel.ActionTokenRecord{}, fmt.Errorf("commit validate action token: %w", err)
		}
	}

	return rec, nil
}

func normalizePurpose(p authModel.ActionTokenPurpose) authModel.ActionTokenPurpose {
	return authModel.ActionTokenPurpose(strings.TrimSpace(strings.ToLower(string(p))))
}

func isAllowedPurpose(p authModel.ActionTokenPurpose) bool {
	return p == authModel.ActionTokenPurposeInvite || p == authModel.ActionTokenPurposePasswordReset
}

func defaultTTLByPurpose(p authModel.ActionTokenPurpose) int {
	switch p {
	case authModel.ActionTokenPurposeInvite:
		return DefaultInviteTTLMinutes
	case authModel.ActionTokenPurposePasswordReset:
		return DefaultPasswordResetTTLMinutes
	default:
		return DefaultPasswordResetTTLMinutes
	}
}
