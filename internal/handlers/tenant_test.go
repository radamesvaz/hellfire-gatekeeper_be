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
	"github.com/golang-jwt/jwt/v5"
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
		regexp.QuoteMeta("SELECT name, logo_url, primary_color, secondary_color, accent_color FROM tenants WHERE id = $1"),
	).WithArgs(tenantID).WillReturnRows(
		sqlmock.NewRows([]string{"name", "logo_url", "primary_color", "secondary_color", "accent_color"}).
			AddRow("Café Demo", "https://example.com/logo.png", "#111827", "#374151", "#F59E0B"),
	)

	req := httptest.NewRequest(http.MethodGet, "/t/default/branding", nil)
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
	assert.Equal(t, "Café Demo", branding["tenant_name"])
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

	req := httptest.NewRequest(http.MethodGet, "/t/default/branding", nil)
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
		regexp.QuoteMeta("SELECT name, logo_url, primary_color, secondary_color, accent_color FROM tenants WHERE id = $1"),
	).WithArgs(tenantID).WillReturnRows(
		sqlmock.NewRows([]string{"name", "logo_url", "primary_color", "secondary_color", "accent_color"}).
			AddRow("Tenant Two", nil, "#111827", "#374151", "#F59E0B"),
	)

	req := httptest.NewRequest(http.MethodPatch, "/auth/branding/colors", strings.NewReader(payload))
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

	req := httptest.NewRequest(http.MethodPatch, "/auth/branding/logo", nil)
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

	req := httptest.NewRequest(http.MethodPatch, "/auth/branding/logo", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, uint64(1))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.UploadTenantLogo(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Exactly one logo file is required")
}

func authTenantContext(t *testing.T, tenantID uint64, slug string, roleID float64) context.Context {
	t.Helper()
	claims := jwt.MapClaims{
		"role_id":   roleID,
		"user_id":   float64(1),
		"tenant_id": float64(tenantID),
	}
	ctx := context.WithValue(context.Background(), middleware.UserClaimsKey, claims)
	ctx = context.WithValue(ctx, middleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, middleware.TenantSlugKey, slug)
	return ctx
}

func TestTenantHandler_UpdateTenantDisplayName_AdminSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := &TenantHandler{
		Repo: &tenantRepository.Repository{DB: db},
	}

	tenantID := uint64(2)
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE tenants SET name = $1, updated_on = NOW() WHERE id = $2`,
	)).WithArgs("Nuevo Nombre", tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	payload := `{"tenant_name":"Nuevo Nombre"}`
	req := httptest.NewRequest(http.MethodPatch, "/auth/branding/name", strings.NewReader(payload))
	req = req.WithContext(authTenantContext(t, tenantID, "slug-two", 1))
	rr := httptest.NewRecorder()

	handler.UpdateTenantDisplayName(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Nuevo Nombre")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantHandler_UpdateTenantDisplayName_ClientForbidden(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := &TenantHandler{
		Repo: &tenantRepository.Repository{DB: db},
	}

	tenantID := uint64(2)
	payload := `{"tenant_name":"Nuevo Nombre"}`
	req := httptest.NewRequest(http.MethodPatch, "/auth/branding/name", strings.NewReader(payload))
	req = req.WithContext(authTenantContext(t, tenantID, "slug-two", 2))
	rr := httptest.NewRecorder()

	handler.UpdateTenantDisplayName(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code)
}

func TestTenantHandler_UpdateBrandingColors_InvalidColor(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := &TenantHandler{
		Repo: &tenantRepository.Repository{DB: db},
	}

	tenantID := uint64(2)
	req := httptest.NewRequest(http.MethodPatch, "/auth/branding/colors", strings.NewReader(`{"primary_color":"blue"}`))
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.UpdateBrandingColors(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "primary_color must use format #RRGGBB")
}
