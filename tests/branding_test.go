package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/handlers"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	tenantRepository "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	brandingService "github.com/radamesvaz/bakery-app/internal/services/branding"
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranding_GetPublic_Success(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	tenantRepo := &tenantRepository.Repository{DB: db}
	uploadDir := "test_uploads_branding"
	require.NoError(t, os.MkdirAll(uploadDir, 0755))
	defer os.RemoveAll(uploadDir)
	imgSvc := imagesService.New(uploadDir)
	brandingSvc := brandingService.New(tenantRepo, imgSvc)
	brandingHandler := &handlers.BrandingHandler{BrandingService: brandingSvc}

	r := mux.NewRouter()
	tPublic := r.PathPrefix("/t/{tenant_slug}").Subrouter()
	tPublic.Use(middleware.TenantFromPathOrHeader(tenantRepo))
	tPublic.HandleFunc("/branding", brandingHandler.GetBranding).Methods("GET")

	req := httptest.NewRequest("GET", "/t/default/branding", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Contains(t, body, "logo_url")
	assert.Contains(t, body, "logo_width")
	assert.Contains(t, body, "logo_height")
	assert.Contains(t, body, "primary_color")
	assert.Contains(t, body, "secondary_color")
	assert.Contains(t, body, "accent_color")
}

func TestBranding_GetAuthenticated_Success(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	tenantRepo := &tenantRepository.Repository{DB: db}
	uploadDir := "test_uploads_branding_auth"
	require.NoError(t, os.MkdirAll(uploadDir, 0755))
	defer os.RemoveAll(uploadDir)
	imgSvc := imagesService.New(uploadDir)
	brandingSvc := brandingService.New(tenantRepo, imgSvc)
	brandingHandler := &handlers.BrandingHandler{BrandingService: brandingSvc}

	secret := "testingsecret"
	exp := 60
	authSvc := authService.New(secret, exp)
	tenantID := uint64(1)
	jwt, err := authSvc.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com", &tenantID)
	require.NoError(t, err)

	r := mux.NewRouter()
	auth := r.PathPrefix("/auth").Subrouter()
	auth.Use(middleware.AuthMiddleware(authSvc))
	auth.Use(middleware.TenantMiddleware())
	auth.HandleFunc("/tenant/branding", brandingHandler.GetBranding).Methods("GET")

	req := httptest.NewRequest("GET", "/auth/tenant/branding", nil)
	req.Header.Set("Authorization", "Bearer "+jwt)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Contains(t, body, "primary_color")
}

func TestBranding_UpdateColors_Admin_Success(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	tenantRepo := &tenantRepository.Repository{DB: db}
	uploadDir := "test_uploads_branding_patch"
	require.NoError(t, os.MkdirAll(uploadDir, 0755))
	defer os.RemoveAll(uploadDir)
	imgSvc := imagesService.New(uploadDir)
	brandingSvc := brandingService.New(tenantRepo, imgSvc)
	brandingHandler := &handlers.BrandingHandler{BrandingService: brandingSvc}

	secret := "testingsecret"
	exp := 60
	authSvc := authService.New(secret, exp)
	tenantID := uint64(1)
	jwt, err := authSvc.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com", &tenantID)
	require.NoError(t, err)

	r := mux.NewRouter()
	auth := r.PathPrefix("/auth").Subrouter()
	auth.Use(middleware.AuthMiddleware(authSvc))
	auth.Use(middleware.TenantMiddleware())
	auth.HandleFunc("/tenant/branding", brandingHandler.GetBranding).Methods("GET")
	auth.HandleFunc("/tenant/branding/colors", brandingHandler.UpdateColors).Methods("PATCH")

	body := []byte(`{"primary_color":"#FF0000","secondary_color":"#00FF00","accent_color":"#0000FF"}`)
	req := httptest.NewRequest("PATCH", "/auth/tenant/branding/colors", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var res map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
	assert.Equal(t, "#FF0000", res["primary_color"])
	assert.Equal(t, "#00FF00", res["secondary_color"])
	assert.Equal(t, "#0000FF", res["accent_color"])
}

func TestBranding_UpdateColors_Client_Forbidden(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	tenantRepo := &tenantRepository.Repository{DB: db}
	imgSvc := imagesService.New("test_uploads")
	brandingSvc := brandingService.New(tenantRepo, imgSvc)
	brandingHandler := &handlers.BrandingHandler{BrandingService: brandingSvc}

	secret := "testingsecret"
	exp := 60
	authSvc := authService.New(secret, exp)
	tenantID := uint64(1)
	jwt, err := authSvc.GenerateJWT(2, uModel.UserRoleClient, "client@example.com", &tenantID)
	require.NoError(t, err)

	r := mux.NewRouter()
	auth := r.PathPrefix("/auth").Subrouter()
	auth.Use(middleware.AuthMiddleware(authSvc))
	auth.Use(middleware.TenantMiddleware())
	auth.HandleFunc("/tenant/branding/colors", brandingHandler.UpdateColors).Methods("PATCH")

	body := []byte(`{"primary_color":"#FF0000","secondary_color":"#00FF00","accent_color":"#0000FF"}`)
	req := httptest.NewRequest("PATCH", "/auth/tenant/branding/colors", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
