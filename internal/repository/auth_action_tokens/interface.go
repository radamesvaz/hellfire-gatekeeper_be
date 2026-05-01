package auth_action_tokens

import (
	"context"
	"database/sql"

	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

type Repository interface {
	CreateToken(ctx context.Context, in CreateTokenInput) (id uint64, err error)
	CreateTokenTx(ctx context.Context, tx *sql.Tx, in CreateTokenInput) (id uint64, err error)
	InsertHistory(ctx context.Context, tx *sql.Tx, in InsertHistoryInput) error
	GetTokenForUpdate(ctx context.Context, tx *sql.Tx, tenantID uint64, purpose authModel.ActionTokenPurpose, tokenHash string) (TokenRecord, error)
	GetTokenByIDForUpdate(ctx context.Context, tx *sql.Tx, id uint64) (TokenRecord, error)
	ConsumeToken(ctx context.Context, tx *sql.Tx, id uint64) error
	RevokeToken(ctx context.Context, tx *sql.Tx, id uint64) error
}

// InsertHistoryInput appends auth_action_tokens_history; tx nil uses DB directly.
type InsertHistoryInput struct {
	TenantID          uint64
	AuthActionTokenID uint64
	Purpose           authModel.ActionTokenPurpose
	Action            authModel.ActionTokenHistoryAction
	ModifiedByUserID  *uint64
	SubjectUserID     *uint64
	MetadataJSON      []byte
}
