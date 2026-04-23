package tests

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	authHandler "github.com/radamesvaz/bakery-app/internal/handlers/auth"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	tenantSignupRepo "github.com/radamesvaz/bakery-app/internal/repository/tenantsignup"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	tenantSignupService "github.com/radamesvaz/bakery-app/internal/services/tenantsignup"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tenantSignupIntegrationEnv struct {
	router   *mux.Router
	db       *sql.DB
	adminJWT string
}

func setupTenantSignupIntegrationEnv(t *testing.T) (*tenantSignupIntegrationEnv, func()) {
	t.Helper()

	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	runMigrations(t, dsn)

	authSvc := authService.New("testingsecret", 60)
	svc := &tenantSignupService.TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authSvc,
	}
	handler := &authHandler.TenantSignupHandler{Service: svc}

	router := mux.NewRouter()
	router.HandleFunc("/public/tenant-register", handler.RegisterTenantWithCode).Methods("POST")
	authRouter := router.PathPrefix("/auth").Subrouter()
	authRouter.Use(middleware.AuthMiddleware(authSvc))
	authRouter.Use(middleware.TenantMiddleware())
	authRouter.HandleFunc("/internal/tenant-signup-codes", handler.CreateSignupCode).Methods("POST")

	defaultTenantID := uint64(1)
	adminJWT, err := authSvc.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com", &defaultTenantID)
	require.NoError(t, err)

	return &tenantSignupIntegrationEnv{
			router:   router,
			db:       db,
			adminJWT: adminJWT,
		}, func() {
			terminate()
		}
}

func createSignupCodeInternal(
	t *testing.T,
	env *tenantSignupIntegrationEnv,
	ttlMinutes int,
	notes string,
) authModel.CreateSignupCodeResponse {
	t.Helper()

	reqBody := fmt.Sprintf(`{"expires_in_minutes":%d,"notes":"%s"}`, ttlMinutes, notes)
	req := httptest.NewRequest(http.MethodPost, "/auth/internal/tenant-signup-codes", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+env.adminJWT)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	env.router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	var resp authModel.CreateSignupCodeResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Code)
	return resp
}

func registerTenantPublic(
	t *testing.T,
	env *tenantSignupIntegrationEnv,
	tenantName,
	tenantSlug,
	adminName,
	email,
	phone,
	password,
	code string,
) *httptest.ResponseRecorder {
	t.Helper()

	payload := fmt.Sprintf(`{
		"tenant_name":"%s",
		"tenant_slug":"%s",
		"admin_name":"%s",
		"email":"%s",
		"phone":"%s",
		"password":"%s",
		"one_time_code":"%s"
	}`, tenantName, tenantSlug, adminName, email, phone, password, code)

	req := httptest.NewRequest(http.MethodPost, "/public/tenant-register", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.router.ServeHTTP(rr, req)
	return rr
}

func hashOneTimeCodeForTest(code string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(strings.ToUpper(code))))
	return hex.EncodeToString(sum[:])
}

func insertSignupCodeForTest(t *testing.T, db *sql.DB, plainCode string, expiresAt time.Time, revoked bool) {
	t.Helper()

	var revokedAt interface{}
	if revoked {
		revokedAt = time.Now().UTC()
	}

	_, err := db.Exec(
		`INSERT INTO tenant_signup_codes (code_hash, expires_at, revoked_at, created_by_user_id, notes)
		 VALUES ($1, $2, $3, $4, $5)`,
		hashOneTimeCodeForTest(plainCode),
		expiresAt,
		revokedAt,
		uint64(1),
		"integration seeded code",
	)
	require.NoError(t, err)
}

func TestTenantSignupIntegration_RegisterWithCode_HappyPath(t *testing.T) {
	env, cleanup := setupTenantSignupIntegrationEnv(t)
	defer cleanup()

	createdCode := createSignupCodeInternal(t, env, 120, "integration happy path")

	slug := fmt.Sprintf("tenant-happy-%d", time.Now().UnixNano())
	email := fmt.Sprintf("owner-happy-%d@test.com", time.Now().UnixNano())

	rr := registerTenantPublic(
		t, env,
		"Tenant Happy", slug, "Owner Happy", email, "555-1010", "StrongPass123!",
		createdCode.Code,
	)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	var resp authModel.PublicTenantRegisterResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "Tenant registered successfully", resp.Message)
	assert.Equal(t, slug, resp.TenantSlug)
	assert.Equal(t, "Tenant Happy", resp.TenantName)
	assert.Equal(t, email, resp.AdminEmail)
	assert.NotEmpty(t, resp.Token)
	assert.NotZero(t, resp.TenantID)
	assert.NotZero(t, resp.AdminID)

	var (
		usedAt         sql.NullTime
		usedByTenantID sql.NullInt64
		usedByUserID   sql.NullInt64
		usedEmail      sql.NullString
		createdBy      sql.NullInt64
	)
	err := env.db.QueryRow(
		`SELECT used_at, used_by_tenant_id, used_by_user_id, used_email, created_by_user_id
		 FROM tenant_signup_codes
		 WHERE code_hash = $1`,
		hashOneTimeCodeForTest(createdCode.Code),
	).Scan(&usedAt, &usedByTenantID, &usedByUserID, &usedEmail, &createdBy)
	require.NoError(t, err)

	assert.True(t, usedAt.Valid)
	assert.True(t, usedByTenantID.Valid)
	assert.True(t, usedByUserID.Valid)
	assert.True(t, usedEmail.Valid)
	assert.True(t, createdBy.Valid)
	assert.Equal(t, int64(resp.TenantID), usedByTenantID.Int64)
	assert.Equal(t, int64(resp.AdminID), usedByUserID.Int64)
	assert.Equal(t, email, usedEmail.String)
	assert.Equal(t, int64(1), createdBy.Int64)
}

func TestTenantSignupIntegration_RegisterWithCode_ReusedCodeReturns422(t *testing.T) {
	env, cleanup := setupTenantSignupIntegrationEnv(t)
	defer cleanup()

	createdCode := createSignupCodeInternal(t, env, 120, "integration reused code")

	firstSlug := fmt.Sprintf("tenant-reuse-a-%d", time.Now().UnixNano())
	firstEmail := fmt.Sprintf("owner-reuse-a-%d@test.com", time.Now().UnixNano())
	first := registerTenantPublic(
		t, env,
		"Tenant Reuse A", firstSlug, "Owner Reuse A", firstEmail, "555-2020", "StrongPass123!",
		createdCode.Code,
	)
	require.Equal(t, http.StatusCreated, first.Code, first.Body.String())

	secondSlug := fmt.Sprintf("tenant-reuse-b-%d", time.Now().UnixNano())
	secondEmail := fmt.Sprintf("owner-reuse-b-%d@test.com", time.Now().UnixNano())
	second := registerTenantPublic(
		t, env,
		"Tenant Reuse B", secondSlug, "Owner Reuse B", secondEmail, "555-2021", "StrongPass123!",
		createdCode.Code,
	)
	assert.Equal(t, http.StatusUnprocessableEntity, second.Code, second.Body.String())
	assert.Contains(t, second.Body.String(), "Invalid or unavailable one-time code")
}

func TestTenantSignupIntegration_RegisterWithCode_ExpiredCodeReturns422(t *testing.T) {
	env, cleanup := setupTenantSignupIntegrationEnv(t)
	defer cleanup()

	expiredCode := "EXPIRED1-CODE0001"
	insertSignupCodeForTest(t, env.db, expiredCode, time.Now().UTC().Add(-5*time.Minute), false)

	slug := fmt.Sprintf("tenant-expired-%d", time.Now().UnixNano())
	email := fmt.Sprintf("owner-expired-%d@test.com", time.Now().UnixNano())
	rr := registerTenantPublic(
		t, env,
		"Tenant Expired", slug, "Owner Expired", email, "555-3030", "StrongPass123!",
		expiredCode,
	)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "Invalid or unavailable one-time code")
}

func TestTenantSignupIntegration_RegisterWithCode_RevokedCodeReturns422(t *testing.T) {
	env, cleanup := setupTenantSignupIntegrationEnv(t)
	defer cleanup()

	revokedCode := "REVOKED1-CODE0002"
	insertSignupCodeForTest(t, env.db, revokedCode, time.Now().UTC().Add(2*time.Hour), true)

	slug := fmt.Sprintf("tenant-revoked-%d", time.Now().UnixNano())
	email := fmt.Sprintf("owner-revoked-%d@test.com", time.Now().UnixNano())
	rr := registerTenantPublic(
		t, env,
		"Tenant Revoked", slug, "Owner Revoked", email, "555-4040", "StrongPass123!",
		revokedCode,
	)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "Invalid or unavailable one-time code")
}

func TestTenantSignupIntegration_RegisterWithCode_DuplicateSlugReturns409(t *testing.T) {
	env, cleanup := setupTenantSignupIntegrationEnv(t)
	defer cleanup()

	createdCode := createSignupCodeInternal(t, env, 120, "integration duplicate slug")
	rr := registerTenantPublic(
		t, env,
		"Default Tenant Collision", "default", "Owner Collision", "owner-collision@test.com", "555-5050", "StrongPass123!",
		createdCode.Code,
	)

	assert.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "Tenant slug already exists")

	var usedAt sql.NullTime
	err := env.db.QueryRow(
		`SELECT used_at FROM tenant_signup_codes WHERE code_hash = $1`,
		hashOneTimeCodeForTest(createdCode.Code),
	).Scan(&usedAt)
	require.NoError(t, err)
	assert.False(t, usedAt.Valid, "code should remain unused when tenant creation fails")
}

func TestTenantSignupIntegration_RegisterWithCode_ConcurrentSameCodeOnlyOneSucceeds(t *testing.T) {
	env, cleanup := setupTenantSignupIntegrationEnv(t)
	defer cleanup()

	createdCode := createSignupCodeInternal(t, env, 120, "integration concurrent same code")

	type concurrentResult struct {
		status int
		body   string
	}

	results := make(chan concurrentResult, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup

	attempt := func(suffix string) {
		defer wg.Done()
		<-start

		slug := fmt.Sprintf("tenant-race-%s-%d", suffix, time.Now().UnixNano())
		email := fmt.Sprintf("owner-race-%s-%d@test.com", suffix, time.Now().UnixNano())
		rr := registerTenantPublic(
			t, env,
			"Tenant Race "+suffix,
			slug,
			"Owner Race "+suffix,
			email,
			"555-6060",
			"StrongPass123!",
			createdCode.Code,
		)
		results <- concurrentResult{status: rr.Code, body: rr.Body.String()}
	}

	wg.Add(2)
	go attempt("A")
	go attempt("B")
	close(start)
	wg.Wait()
	close(results)

	var (
		createdCount int
		invalidCount int
	)
	for res := range results {
		switch res.status {
		case http.StatusCreated:
			createdCount++
		case http.StatusUnprocessableEntity:
			invalidCount++
			assert.Contains(t, res.body, "Invalid or unavailable one-time code")
		default:
			t.Fatalf("unexpected status in concurrent run: %d body=%s", res.status, res.body)
		}
	}

	assert.Equal(t, 1, createdCount, "exactly one concurrent request should create tenant")
	assert.Equal(t, 1, invalidCount, "exactly one concurrent request should fail as OTC unavailable")

	var usedCount int
	err := env.db.QueryRow(
		`SELECT COUNT(*) FROM tenant_signup_codes WHERE code_hash = $1 AND used_at IS NOT NULL`,
		hashOneTimeCodeForTest(createdCode.Code),
	).Scan(&usedCount)
	require.NoError(t, err)
	assert.Equal(t, 1, usedCount, "OTC must be consumed once even under concurrent load")
}
