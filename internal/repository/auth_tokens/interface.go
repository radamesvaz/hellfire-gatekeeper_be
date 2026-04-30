package auth_tokens

import (
	"context"
	"database/sql"

	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

type Repository interface {
	CreateToken(ctx context.Context, in CreateTokenInput) (id uint64, err error)
	GetTokenForUpdate(ctx context.Context, tx *sql.Tx, tenantID uint64, purpose authModel.ActionTokenPurpose, tokenHash string) (TokenRecord, error)
	GetTokenByIDForUpdate(ctx context.Context, tx *sql.Tx, id uint64) (TokenRecord, error)
	ConsumeToken(ctx context.Context, tx *sql.Tx, id uint64) error
	RevokeToken(ctx context.Context, tx *sql.Tx, id uint64) error
}
