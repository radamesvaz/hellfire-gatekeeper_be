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
	RevokeTokenScoped(ctx context.Context, tenantID uint64, purpose authModel.ActionTokenPurpose, tokenID uint64) error
	GetTokenByIDScoped(ctx context.Context, tenantID uint64, purpose authModel.ActionTokenPurpose, tokenID uint64) (authModel.ActionTokenRecord, error)
}
