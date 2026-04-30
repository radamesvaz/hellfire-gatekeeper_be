package auth_tokens

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	repo "github.com/radamesvaz/bakery-app/internal/repository/auth_tokens"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionTokenService_RevokeTokenScoped_TenantMismatch_ReturnsInvalidToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := &ActionTokenService{
		DB:   db,
		Repo: &repo.SQLRepository{DB: db},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, tenant_id, email, purpose, subject_user_id, metadata_json, expires_at, used_at, revoked_at\s+FROM auth_action_tokens\s+WHERE id = \$1\s+FOR UPDATE`).
		WithArgs(uint64(11)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "email", "purpose", "subject_user_id", "metadata_json", "expires_at", "used_at", "revoked_at"}).
			AddRow(uint64(11), uint64(999), "invitee@example.com", "invite", nil, nil, time.Now().UTC().Add(time.Hour), nil, nil))
	mock.ExpectRollback()

	err = svc.RevokeTokenScoped(context.Background(), 10, authModel.ActionTokenPurposeInvite, 11)
	require.Error(t, err)
	assert.ErrorIs(t, err, appErrors.ErrInvalidToken)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestActionTokenService_RevokeTokenScoped_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := &ActionTokenService{
		DB:   db,
		Repo: &repo.SQLRepository{DB: db},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, tenant_id, email, purpose, subject_user_id, metadata_json, expires_at, used_at, revoked_at\s+FROM auth_action_tokens\s+WHERE id = \$1\s+FOR UPDATE`).
		WithArgs(uint64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "email", "purpose", "subject_user_id", "metadata_json", "expires_at", "used_at", "revoked_at"}).
			AddRow(uint64(42), uint64(5), "invitee@example.com", "invite", nil, nil, time.Now().UTC().Add(time.Hour), nil, nil))
	mock.ExpectExec(`UPDATE auth_action_tokens SET revoked_at = NOW\(\), updated_on = NOW\(\) WHERE id = \$1`).
		WithArgs(uint64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = svc.RevokeTokenScoped(context.Background(), 5, authModel.ActionTokenPurposeInvite, 42)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestActionTokenService_GetTokenByIDScoped_TenantMismatch_ReturnsInvalidToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := &ActionTokenService{
		DB:   db,
		Repo: &repo.SQLRepository{DB: db},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, tenant_id, email, purpose, subject_user_id, metadata_json, expires_at, used_at, revoked_at\s+FROM auth_action_tokens\s+WHERE id = \$1\s+FOR UPDATE`).
		WithArgs(uint64(70)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "email", "purpose", "subject_user_id", "metadata_json", "expires_at", "used_at", "revoked_at"}).
			AddRow(uint64(70), uint64(55), "invitee@example.com", "invite", nil, nil, time.Now().UTC().Add(time.Hour), nil, nil))
	mock.ExpectRollback()

	_, err = svc.GetTokenByIDScoped(context.Background(), 5, authModel.ActionTokenPurposeInvite, 70)
	require.Error(t, err)
	assert.ErrorIs(t, err, appErrors.ErrInvalidToken)
	assert.NoError(t, mock.ExpectationsWereMet())
}
