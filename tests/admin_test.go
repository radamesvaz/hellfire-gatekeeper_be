package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/handlers"
	"github.com/radamesvaz/bakery-app/internal/handlers/auth"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	tenantRepository "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	userRepository "github.com/radamesvaz/bakery-app/internal/repository/user"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildAdminRouter builds a router with /admin/login (public) and /admin/* protected (JWT + superadmin).
func buildAdminRouter(t *testing.T, authSvc *authService.AuthService, authHandler *auth.LoginHandler, adminHandler *handlers.AdminHandler) *mux.Router {
	t.Helper()
	r := mux.NewRouter()
	r.HandleFunc("/admin/login", authHandler.AdminLogin).Methods("POST")
	adminProtected := r.PathPrefix("/admin").Subrouter()
	adminProtected.Use(middleware.AuthMiddleware(authSvc))
	adminProtected.Use(middleware.TenantMiddleware())
	adminProtected.Use(middleware.SuperadminRequired)
	adminProtected.HandleFunc("/tenants", adminHandler.CreateTenant).Methods("POST")
	adminProtected.HandleFunc("/tenants/{id}/subscription", adminHandler.UpdateTenantSubscription).Methods("PATCH")
	return r
}

func TestAdmin_AdminLogin_Success(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	tenantRepo := &tenantRepository.Repository{DB: db}
	userRepo := userRepository.UserRepository{DB: db}
	authSvc := authService.New("testsecret", 60)
	authHandler := &auth.LoginHandler{UserRepo: userRepo, AuthService: *authSvc}
	adminHandler := &handlers.AdminHandler{TenantRepo: tenantRepo}
	r := buildAdminRouter(t, authSvc, authHandler, adminHandler)

	body := `{"email":"admin@example.com","password":"adminpass"}`
	req := httptest.NewRequest("POST", "/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var res struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
	assert.NotEmpty(t, res.Token)
}

func TestAdmin_AdminLogin_NonAdmin_Unauthorized(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	tenantRepo := &tenantRepository.Repository{DB: db}
	userRepo := userRepository.UserRepository{DB: db}
	authSvc := authService.New("testsecret", 60)
	authHandler := &auth.LoginHandler{UserRepo: userRepo, AuthService: *authSvc}
	adminHandler := &handlers.AdminHandler{TenantRepo: tenantRepo}
	r := buildAdminRouter(t, authSvc, authHandler, adminHandler)

	// client@example.com is role client in seed
	body := `{"email":"client@example.com","password":"any"}`
	req := httptest.NewRequest("POST", "/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAdmin_AdminLogin_InvalidPassword_Unauthorized(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	tenantRepo := &tenantRepository.Repository{DB: db}
	userRepo := userRepository.UserRepository{DB: db}
	authSvc := authService.New("testsecret", 60)
	authHandler := &auth.LoginHandler{UserRepo: userRepo, AuthService: *authSvc}
	adminHandler := &handlers.AdminHandler{TenantRepo: tenantRepo}
	r := buildAdminRouter(t, authSvc, authHandler, adminHandler)

	body := `{"email":"admin@example.com","password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/admin/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAdmin_CreateTenant_Success(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	tenantRepo := &tenantRepository.Repository{DB: db}
	userRepo := userRepository.UserRepository{DB: db}
	authSvc := authService.New("testsecret", 60)
	authHandler := &auth.LoginHandler{UserRepo: userRepo, AuthService: *authSvc}
	adminHandler := &handlers.AdminHandler{TenantRepo: tenantRepo}
	r := buildAdminRouter(t, authSvc, authHandler, adminHandler)

	// 1) Admin login
	loginBody := `{"email":"admin@example.com","password":"adminpass"}`
	loginReq := httptest.NewRequest("POST", "/admin/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	r.ServeHTTP(loginRR, loginReq)
	require.Equal(t, http.StatusOK, loginRR.Code)
	var loginRes struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(loginRR.Body.Bytes(), &loginRes))
	require.NotEmpty(t, loginRes.Token)

	// 2) Create tenant
	createBody := `{"name":"New Shop","slug":"new-shop"}`
	createReq := httptest.NewRequest("POST", "/admin/tenants", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+loginRes.Token)
	createRR := httptest.NewRecorder()
	r.ServeHTTP(createRR, createReq)

	assert.Equal(t, http.StatusCreated, createRR.Code)
	var tenant map[string]interface{}
	require.NoError(t, json.Unmarshal(createRR.Body.Bytes(), &tenant))
	assert.Equal(t, "New Shop", tenant["name"])
	assert.Equal(t, "new-shop", tenant["slug"])
	assert.NotZero(t, tenant["id"])
}

func TestAdmin_CreateTenant_WithoutToken_Unauthorized(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	tenantRepo := &tenantRepository.Repository{DB: db}
	userRepo := userRepository.UserRepository{DB: db}
	authSvc := authService.New("testsecret", 60)
	authHandler := &auth.LoginHandler{UserRepo: userRepo, AuthService: *authSvc}
	adminHandler := &handlers.AdminHandler{TenantRepo: tenantRepo}
	r := buildAdminRouter(t, authSvc, authHandler, adminHandler)

	createBody := `{"name":"New Shop","slug":"new-shop"}`
	createReq := httptest.NewRequest("POST", "/admin/tenants", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	r.ServeHTTP(createRR, createReq)

	assert.Equal(t, http.StatusUnauthorized, createRR.Code)
}

func TestAdmin_CreateTenant_WithClientToken_Forbidden(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	tenantRepo := &tenantRepository.Repository{DB: db}
	userRepo := userRepository.UserRepository{DB: db}
	authSvc := authService.New("testsecret", 60)
	authHandler := &auth.LoginHandler{UserRepo: userRepo, AuthService: *authSvc}
	adminHandler := &handlers.AdminHandler{TenantRepo: tenantRepo}
	r := buildAdminRouter(t, authSvc, authHandler, adminHandler)

	// Token with role client (not superadmin)
	tenantID := uint64(1)
	clientToken, err := authSvc.GenerateJWT(2, uModel.UserRoleClient, "client@example.com", &tenantID)
	require.NoError(t, err)

	createBody := `{"name":"New Shop","slug":"new-shop"}`
	createReq := httptest.NewRequest("POST", "/admin/tenants", strings.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+clientToken)
	createRR := httptest.NewRecorder()
	r.ServeHTTP(createRR, createReq)

	assert.Equal(t, http.StatusForbidden, createRR.Code)
}

func TestAdmin_UpdateTenantSubscription_Success(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	tenantRepo := &tenantRepository.Repository{DB: db}
	userRepo := userRepository.UserRepository{DB: db}
	authSvc := authService.New("testsecret", 60)
	authHandler := &auth.LoginHandler{UserRepo: userRepo, AuthService: *authSvc}
	adminHandler := &handlers.AdminHandler{TenantRepo: tenantRepo}
	r := buildAdminRouter(t, authSvc, authHandler, adminHandler)

	// Admin login
	loginBody := `{"email":"admin@example.com","password":"adminpass"}`
	loginReq := httptest.NewRequest("POST", "/admin/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	r.ServeHTTP(loginRR, loginReq)
	require.Equal(t, http.StatusOK, loginRR.Code)
	var loginRes struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(loginRR.Body.Bytes(), &loginRes))

	// PATCH subscription for tenant 1 (default tenant from seed)
	patchBody := []byte(`{"subscription_status":"canceled"}`)
	patchReq := httptest.NewRequest("PATCH", "/admin/tenants/1/subscription", bytes.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Authorization", "Bearer "+loginRes.Token)
	patchRR := httptest.NewRecorder()
	r.ServeHTTP(patchRR, patchReq)

	assert.Equal(t, http.StatusOK, patchRR.Code)
	var res map[string]interface{}
	require.NoError(t, json.Unmarshal(patchRR.Body.Bytes(), &res))
	assert.Equal(t, "subscription updated", res["message"])
}

func TestAdmin_UpdateTenantSubscription_WithoutToken_Unauthorized(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	tenantRepo := &tenantRepository.Repository{DB: db}
	userRepo := userRepository.UserRepository{DB: db}
	authSvc := authService.New("testsecret", 60)
	authHandler := &auth.LoginHandler{UserRepo: userRepo, AuthService: *authSvc}
	adminHandler := &handlers.AdminHandler{TenantRepo: tenantRepo}
	r := buildAdminRouter(t, authSvc, authHandler, adminHandler)

	patchBody := []byte(`{"subscription_status":"active"}`)
	patchReq := httptest.NewRequest("PATCH", "/admin/tenants/1/subscription", bytes.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchRR := httptest.NewRecorder()
	r.ServeHTTP(patchRR, patchReq)

	assert.Equal(t, http.StatusUnauthorized, patchRR.Code)
}

func TestAdmin_UpdateTenantSubscription_NotFound(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	tenantRepo := &tenantRepository.Repository{DB: db}
	userRepo := userRepository.UserRepository{DB: db}
	authSvc := authService.New("testsecret", 60)
	authHandler := &auth.LoginHandler{UserRepo: userRepo, AuthService: *authSvc}
	adminHandler := &handlers.AdminHandler{TenantRepo: tenantRepo}
	r := buildAdminRouter(t, authSvc, authHandler, adminHandler)

	// Admin login
	loginBody := `{"email":"admin@example.com","password":"adminpass"}`
	loginReq := httptest.NewRequest("POST", "/admin/login", strings.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	r.ServeHTTP(loginRR, loginReq)
	require.Equal(t, http.StatusOK, loginRR.Code)
	var loginRes struct {
		Token string `json:"token"`
	}
	require.NoError(t, json.Unmarshal(loginRR.Body.Bytes(), &loginRes))

	// PATCH non-existent tenant
	patchBody := []byte(`{"subscription_status":"active"}`)
	patchReq := httptest.NewRequest("PATCH", "/admin/tenants/99999/subscription", bytes.NewReader(patchBody))
	patchReq.Header.Set("Content-Type", "application/json")
	patchReq.Header.Set("Authorization", "Bearer "+loginRes.Token)
	patchRR := httptest.NewRecorder()
	r.ServeHTTP(patchRR, patchReq)

	assert.Equal(t, http.StatusNotFound, patchRR.Code)
}
