package tenant

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetBranding_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT logo_url, logo_width, logo_height, primary_color, secondary_color, accent_color FROM tenants WHERE id = $1`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"logo_url", "logo_width", "logo_height", "primary_color", "secondary_color", "accent_color"}).
			AddRow("/uploads/tenants/1/logo.png", 200, 100, "#FF0000", "#00FF00", "#0000FF"))

	b, err := repo.GetBranding(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, "/uploads/tenants/1/logo.png", b.LogoURL)
	assert.Equal(t, 200, b.LogoWidth)
	assert.Equal(t, 100, b.LogoHeight)
	assert.Equal(t, "#FF0000", b.PrimaryColor)
	assert.Equal(t, "#00FF00", b.SecondaryColor)
	assert.Equal(t, "#0000FF", b.AccentColor)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetBranding_EmptyBranding(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT logo_url, logo_width, logo_height, primary_color, secondary_color, accent_color FROM tenants WHERE id = $1`)).
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"logo_url", "logo_width", "logo_height", "primary_color", "secondary_color", "accent_color"}).
			AddRow(nil, nil, nil, nil, nil, nil))

	b, err := repo.GetBranding(ctx, 2)
	require.NoError(t, err)
	assert.Empty(t, b.LogoURL)
	assert.Equal(t, 0, b.LogoWidth)
	assert.Equal(t, 0, b.LogoHeight)
	assert.Empty(t, b.PrimaryColor)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetBranding_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT logo_url, logo_width, logo_height, primary_color, secondary_color, accent_color FROM tenants WHERE id = $1`)).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetBranding(ctx, 999)
	require.Error(t, err)
	assert.ErrorIs(t, err, appErrors.ErrTenantNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateColors_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE tenants SET primary_color = $1, secondary_color = $2, accent_color = $3, updated_on = NOW() WHERE id = $4`)).
		WithArgs("#111111", "#222222", "#333333", uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateColors(ctx, 1, "#111111", "#222222", "#333333")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateColors_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE tenants SET primary_color = $1, secondary_color = $2, accent_color = $3, updated_on = NOW() WHERE id = $4`)).
		WithArgs("#111111", "#222222", "#333333", uint64(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.UpdateColors(ctx, 999, "#111111", "#222222", "#333333")
	require.Error(t, err)
	assert.ErrorIs(t, err, appErrors.ErrTenantNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateLogo_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE tenants SET logo_url = $1, logo_width = $2, logo_height = $3, updated_on = NOW() WHERE id = $4`)).
		WithArgs("/uploads/tenants/1/logo.png", 200, 100, uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateLogo(ctx, 1, "/uploads/tenants/1/logo.png", 200, 100)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateLogo_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE tenants SET logo_url = $1, logo_width = $2, logo_height = $3, updated_on = NOW() WHERE id = $4`)).
		WithArgs("/uploads/tenants/999/logo.png", 64, 64, uint64(999)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.UpdateLogo(ctx, 999, "/uploads/tenants/999/logo.png", 64, 64)
	require.Error(t, err)
	assert.ErrorIs(t, err, appErrors.ErrTenantNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
