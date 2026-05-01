package auth_tokens

import (
	"context"

	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

type Service interface {
	CreateToken(ctx context.Context, req authModel.CreateActionTokenRequest) (authModel.CreateActionTokenResponse, error)
	ValidateToken(ctx context.Context, tenantID uint64, purpose authModel.ActionTokenPurpose, plainToken string) (authModel.ActionTokenRecord, error)
	ConsumeToken(ctx context.Context, tenantID uint64, purpose authModel.ActionTokenPurpose, plainToken string) (authModel.ActionTokenRecord, error)
	RevokeToken(ctx context.Context, tokenID uint64) error
	// RevokeTokenScoped sets revokedByUserID in audit history when non-nil (invite revoke/resend flows).
	RevokeTokenScoped(ctx context.Context, tenantID uint64, purpose authModel.ActionTokenPurpose, tokenID uint64, revokedByUserID *uint64) error
	GetTokenByIDScoped(ctx context.Context, tenantID uint64, purpose authModel.ActionTokenPurpose, tokenID uint64) (authModel.ActionTokenRecord, error)
	RecordInvitationAccepted(ctx context.Context, tenantID uint64, tokenID uint64, newUserID uint64) error
}
