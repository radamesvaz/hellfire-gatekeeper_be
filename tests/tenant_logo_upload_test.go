package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/handlers"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	tenantRepository "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	"github.com/radamesvaz/bakery-app/internal/services/auth"
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationUploadTenantLogo(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	// Force local storage: setupPostgreSQLContainer loads .env which may enable Cloudinary.
	origCloud := os.Getenv("CLOUDINARY_CLOUD_NAME")
	origKey := os.Getenv("CLOUDINARY_API_KEY")
	origSecret := os.Getenv("CLOUDINARY_API_SECRET")
	_ = os.Unsetenv("CLOUDINARY_CLOUD_NAME")
	_ = os.Unsetenv("CLOUDINARY_API_KEY")
	_ = os.Unsetenv("CLOUDINARY_API_SECRET")
	t.Cleanup(func() {
		if origCloud != "" {
			_ = os.Setenv("CLOUDINARY_CLOUD_NAME", origCloud)
		}
		if origKey != "" {
			_ = os.Setenv("CLOUDINARY_API_KEY", origKey)
		}
		if origSecret != "" {
			_ = os.Setenv("CLOUDINARY_API_SECRET", origSecret)
		}
	})

	testDir := t.TempDir()
	tenantRepo := tenantRepository.Repository{DB: db}
	imageService := imagesService.New(testDir)
	tenantHandler := handlers.TenantHandler{
		Repo:         &tenantRepo,
		ImageService: imageService,
	}

	router := mux.NewRouter()

	authRouter := router.PathPrefix("/auth").Subrouter()
	secret := "testingsecret"
	exp := 60
	authSvc := auth.New(secret, exp)
	authRouter.Use(middleware.AuthMiddleware(authSvc))
	authRouter.Use(middleware.TenantMiddleware())
	authRouter.HandleFunc("/tenant/branding/logo", tenantHandler.UploadTenantLogo).Methods("PATCH")

	tPublic := router.PathPrefix("/t/{tenant_slug}").Subrouter()
	tPublic.Use(middleware.TenantFromPathOrHeader(&tenantRepo))
	tPublic.HandleFunc("/tenant/branding", tenantHandler.GetBranding).Methods("GET")

	tenantID := uint64(1)
	jwt, err := authSvc.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com", &tenantID)
	require.NoError(t, err)

	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	fw, err := mw.CreateFormFile("logo", "logo.png")
	require.NoError(t, err)
	_, err = fw.Write([]byte("fakepngdata"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	uploadReq := httptest.NewRequest(http.MethodPatch, "/auth/tenant/branding/logo", &b)
	uploadReq.Header.Set("Content-Type", mw.FormDataContentType())
	uploadReq.Header.Set("Authorization", "Bearer "+jwt)
	uploadRR := httptest.NewRecorder()
	router.ServeHTTP(uploadRR, uploadReq)

	require.Equal(t, http.StatusOK, uploadRR.Code, uploadRR.Body.String())

	var uploadResp map[string]interface{}
	require.NoError(t, json.Unmarshal(uploadRR.Body.Bytes(), &uploadResp))
	logoURL, ok := uploadResp["logo_url"].(string)
	require.True(t, ok)
	assert.Contains(t, logoURL, "/uploads/tenants/1/")
	assert.Contains(t, logoURL, "logo_")

	getReq := httptest.NewRequest(http.MethodGet, "/t/default/tenant/branding", nil)
	getRR := httptest.NewRecorder()
	router.ServeHTTP(getRR, getReq)
	require.Equal(t, http.StatusOK, getRR.Code)

	var getBody map[string]interface{}
	require.NoError(t, json.Unmarshal(getRR.Body.Bytes(), &getBody))
	branding, ok := getBody["branding"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, logoURL, branding["logo_url"])

	var stored string
	err = db.QueryRowContext(context.Background(), `SELECT logo_url FROM tenants WHERE id = $1`, 1).Scan(&stored)
	require.NoError(t, err)
	assert.Equal(t, logoURL, stored)
}
