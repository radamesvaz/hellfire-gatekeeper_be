package branding

import (
	"bytes"
	"context"
	"database/sql"
	"mime/multipart"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/repository/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateHexColor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  error
	}{
		{"empty", "", nil},
		{"valid lowercase", "#ff0000", nil},
		{"valid uppercase", "#FF0000", nil},
		{"valid mixed", "#Ff00Aa", nil},
		{"missing hash", "ff0000", appErrors.ErrInvalidColorFormat},
		{"short", "#fff", appErrors.ErrInvalidColorFormat},
		{"long", "#ff00000", appErrors.ErrInvalidColorFormat},
		{"invalid char", "#gg0000", appErrors.ErrInvalidColorFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHexColor(tt.input)
			if tt.want == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.want)
			}
		})
	}
}

func TestNormalizeHexColor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"lowercase", "#ff0000", "#FF0000"},
		{"with spaces", "  #ab12ef  ", "#AB12EF"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeHexColor(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestService_GetBranding_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantRepo := &tenant.Repository{DB: db}
	svc := &Service{TenantRepo: tenantRepo}

	ctx := context.Background()
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT logo_url, logo_width, logo_height, primary_color, secondary_color, accent_color FROM tenants WHERE id = $1`)).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"logo_url", "logo_width", "logo_height", "primary_color", "secondary_color", "accent_color"}).
			AddRow("/logo.png", 100, 50, "#A1B2C3", "#D4E5F6", "#000000"))

	b, err := svc.GetBranding(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, "/logo.png", b.LogoURL)
	assert.Equal(t, 100, b.LogoWidth)
	assert.Equal(t, 50, b.LogoHeight)
	assert.Equal(t, "#A1B2C3", b.PrimaryColor)
	assert.Equal(t, "#D4E5F6", b.SecondaryColor)
	assert.Equal(t, "#000000", b.AccentColor)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestService_GetBranding_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantRepo := &tenant.Repository{DB: db}
	svc := &Service{TenantRepo: tenantRepo}

	ctx := context.Background()
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT logo_url, logo_width, logo_height, primary_color, secondary_color, accent_color FROM tenants WHERE id = $1`)).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	_, err = svc.GetBranding(ctx, 999)
	require.Error(t, err)
	assert.ErrorIs(t, err, appErrors.ErrTenantNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestService_UpdateColors_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantRepo := &tenant.Repository{DB: db}
	svc := &Service{TenantRepo: tenantRepo}

	ctx := context.Background()
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE tenants SET primary_color = $1, secondary_color = $2, accent_color = $3, updated_on = NOW() WHERE id = $4`)).
		WithArgs("#111111", "#222222", "#333333", uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = svc.UpdateColors(ctx, 1, "#111111", "#222222", "#333333")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestService_UpdateColors_InvalidHex(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantRepo := &tenant.Repository{DB: db}
	svc := &Service{TenantRepo: tenantRepo}
	ctx := context.Background()

	err = svc.UpdateColors(ctx, 1, "#valid", "#invalid", "#00FF00")
	require.Error(t, err)
	assert.ErrorIs(t, err, appErrors.ErrInvalidColorFormat)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestService_UpdateLogo_TenantNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tenantRepo := &tenant.Repository{DB: db}
	svc := &Service{TenantRepo: tenantRepo}

	ctx := context.Background()
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT logo_url, logo_width, logo_height, primary_color, secondary_color, accent_color FROM tenants WHERE id = $1`)).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	// FileHeader with minimal content; we only need tenant lookup to fail before image is used
	fh := createMinimalMultipartFileHeader("logo.png", "image/png")
	_, err = svc.UpdateLogo(ctx, 999, fh)
	require.Error(t, err)
	assert.ErrorIs(t, err, appErrors.ErrTenantNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

// createMinimalMultipartFileHeader creates a multipart.FileHeader for tests that only need the form field to exist.
func createMinimalMultipartFileHeader(filename, contentType string) *multipart.FileHeader {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, _ := w.CreateFormFile("logo", filename)
	_, _ = fw.Write([]byte("x"))
	boundary := w.Boundary()
	w.Close()
	r := multipart.NewReader(&b, boundary)
	form, err := r.ReadForm(32 << 20)
	if err != nil {
		panic(err)
	}
	files := form.File["logo"]
	if len(files) == 0 {
		panic("no logo file")
	}
	files[0].Header = map[string][]string{"Content-Type": {contentType}}
	return files[0]
}
