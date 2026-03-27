package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	tenantRepository "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantHandler_GetBranding_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := &TenantHandler{
		Repo: &tenantRepository.Repository{DB: db},
	}

	tenantID := uint64(1)
	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT logo_url, primary_color, secondary_color, accent_color FROM tenants WHERE id = $1"),
	).WithArgs(tenantID).WillReturnRows(
		sqlmock.NewRows([]string{"logo_url", "primary_color", "secondary_color", "accent_color"}).
			AddRow("https://example.com/logo.png", "#111827", "#374151", "#F59E0B"),
	)

	req := httptest.NewRequest(http.MethodGet, "/t/default/tenant/branding", nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, middleware.TenantSlugKey, "default")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetBranding(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, float64(tenantID), body["tenant_id"])
	assert.Equal(t, "default", body["tenant_slug"])
	branding, ok := body["branding"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "https://example.com/logo.png", branding["logo_url"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantHandler_GetBranding_MissingSlug(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := &TenantHandler{
		Repo: &tenantRepository.Repository{DB: db},
	}

	req := httptest.NewRequest(http.MethodGet, "/t/default/tenant/branding", nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, uint64(1))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetBranding(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTenantHandler_UpdateBrandingColors_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := &TenantHandler{
		Repo: &tenantRepository.Repository{DB: db},
	}

	tenantID := uint64(2)
	payload := `{"primary_color":"#111827","accent_color":"#f59e0b"}`

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE tenants
		 SET primary_color = COALESCE($1, primary_color),
		     secondary_color = COALESCE($2, secondary_color),
		     accent_color = COALESCE($3, accent_color),
		     updated_on = NOW()
		 WHERE id = $4`,
	)).WithArgs("#111827", nil, "#F59E0B", tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT logo_url, primary_color, secondary_color, accent_color FROM tenants WHERE id = $1"),
	).WithArgs(tenantID).WillReturnRows(
		sqlmock.NewRows([]string{"logo_url", "primary_color", "secondary_color", "accent_color"}).
			AddRow(nil, "#111827", "#374151", "#F59E0B"),
	)

	req := httptest.NewRequest(http.MethodPatch, "/auth/tenant/branding/colors", strings.NewReader(payload))
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.UpdateBrandingColors(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Tenant branding colors updated successfully")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantHandler_UploadTenantLogo_NoImageService(t *testing.T) {
	handler := &TenantHandler{
		Repo:         &tenantRepository.Repository{},
		ImageService: nil,
	}

	req := httptest.NewRequest(http.MethodPatch, "/auth/tenant/branding/logo", nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, uint64(1))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.UploadTenantLogo(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestTenantHandler_UploadTenantLogo_WrongFieldName(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, err := w.CreateFormFile("not_logo", "x.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("x"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	handler := &TenantHandler{
		Repo:         &tenantRepository.Repository{DB: db},
		ImageService: imagesService.New(t.TempDir()),
	}

	req := httptest.NewRequest(http.MethodPatch, "/auth/tenant/branding/logo", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, uint64(1))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.UploadTenantLogo(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Exactly one logo file is required")
}

func TestTenantHandler_UpdateBrandingColors_InvalidColor(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := &TenantHandler{
		Repo: &tenantRepository.Repository{DB: db},
	}

	tenantID := uint64(2)
	req := httptest.NewRequest(http.MethodPatch, "/auth/tenant/branding/colors", strings.NewReader(`{"primary_color":"blue"}`))
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.UpdateBrandingColors(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "primary_color must use format #RRGGBB")
}
