package tests

import (
	"bytes"
	"context"
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

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	authHandler "github.com/radamesvaz/bakery-app/internal/handlers/auth"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	tenantRepository "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	tenantSignupRepo "github.com/radamesvaz/bakery-app/internal/repository/tenantsignup"
	"github.com/radamesvaz/bakery-app/internal/repository/user"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	"github.com/radamesvaz/bakery-app/internal/services/email"
	tenantSignupService "github.com/radamesvaz/bakery-app/internal/services/tenantsignup"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingTenantSignupEmailSender struct {
	mu    sync.Mutex
	last  email.TenantSignupCodePayload
	calls int
	err   error
}

func (r *recordingTenantSignupEmailSender) SendPasswordReset(context.Context, email.PasswordResetPayload) error {
	return nil
}

func (r *recordingTenantSignupEmailSender) SendTenantInvitation(context.Context, email.TenantInvitationPayload) error {
	return nil
}

func (r *recordingTenantSignupEmailSender) SendTenantSignupCode(_ context.Context, payload email.TenantSignupCodePayload) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.last = payload
	return r.err
}

type tenantSignupIntegrationEnv struct {
	router           *mux.Router
	db               *sql.DB
	authSvc          authService.Service
	emailSender      *recordingTenantSignupEmailSender
	superadminUserID uint64
	superadminJWT    string
	tenantAdminJWT   string
}

func setupTenantSignupIntegrationEnv(t *testing.T) (*tenantSignupIntegrationEnv, func()) {
	t.Helper()

	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	runMigrations(t, dsn)

	authSvc := authService.New("testingsecret", 60)
	emailSender := &recordingTenantSignupEmailSender{}
	svc := &tenantSignupService.TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authSvc,
		EmailSender: emailSender,
		AppBaseURL:  "https://admin.example.com",
	}
	handler := &authHandler.TenantSignupHandler{Service: svc}
	tenantRepo := &tenantRepository.Repository{DB: db}
	loginHandler := &authHandler.LoginHandler{
		UserRepo:    user.UserRepository{DB: db},
		TenantRepo:  tenantRepo,
		AuthService: authSvc,
	}

	router := mux.NewRouter()
	router.HandleFunc("/login", loginHandler.Login).Methods("POST")
	router.HandleFunc("/public/tenant-register", handler.RegisterTenantWithCode).Methods("POST")
	authRouter := router.PathPrefix("/auth").Subrouter()
	authRouter.Use(middleware.AuthMiddleware(authSvc))
	authRouter.Use(middleware.TenantMiddleware(tenantRepo))
	authRouter.HandleFunc("/internal/tenant-signup-codes", handler.CreateSignupCode).Methods("POST")

	defaultTenantID := uint64(1)
	superadminID := lookupUserIDByEmailAndRole(t, db, "superadmin@example.com", uModel.UserRoleSuperAdmin)
	adminID := lookupUserIDByEmailAndRole(t, db, "admin@example.com", uModel.UserRoleAdmin)

	superadminJWT, err := authSvc.GenerateJWT(superadminID, uModel.UserRoleSuperAdmin, "superadmin@example.com", &defaultTenantID)
	require.NoError(t, err)
	tenantAdminJWT, err := authSvc.GenerateJWT(adminID, uModel.UserRoleAdmin, "admin@example.com", &defaultTenantID)
	require.NoError(t, err)

	return &tenantSignupIntegrationEnv{
			router:           router,
			db:               db,
			authSvc:          authSvc,
			emailSender:      emailSender,
			superadminUserID: superadminID,
			superadminJWT:    superadminJWT,
			tenantAdminJWT:   tenantAdminJWT,
		}, func() {
			terminate()
		}
}

func lookupUserIDByEmailAndRole(t *testing.T, db *sql.DB, email string, role uModel.UserRole) uint64 {
	t.Helper()
	var id uint64
	err := db.QueryRow(
		`SELECT id_user FROM users WHERE tenant_id = 1 AND email = $1 AND id_role = $2 AND deleted_at IS NULL`,
		email,
		role,
	).Scan(&id)
	require.NoError(t, err, "expected seeded user %s with role %d", email, role)
	return id
}

func createSignupCodeInternal(
	t *testing.T,
	env *tenantSignupIntegrationEnv,
	recipientEmail string,
	ttlMinutes int,
	notes string,
) authModel.CreateSignupCodeResponse {
	t.Helper()

	reqBody := fmt.Sprintf(
		`{"email":"%s","expires_in_minutes":%d,"notes":"%s"}`,
		recipientEmail,
		ttlMinutes,
		notes,
	)
	req := httptest.NewRequest(http.MethodPost, "/auth/internal/tenant-signup-codes", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+env.superadminJWT)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	env.router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	var resp authModel.CreateSignupCodeResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Code)
	require.Equal(t, strings.ToLower(recipientEmail), resp.Email)
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

func TestTenantSignupIntegration_CreateSignupCode_ForbiddenForTenantAdmin(t *testing.T) {
	env, cleanup := setupTenantSignupIntegrationEnv(t)
	defer cleanup()

	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/internal/tenant-signup-codes",
		strings.NewReader(`{"email":"invitee@example.com","expires_in_minutes":120,"notes":"tenant admin should be forbidden"}`),
	)
	req.Header.Set("Authorization", "Bearer "+env.tenantAdminJWT)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	env.router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "Forbidden")
}

func TestTenantSignupIntegration_CreateSignupCode_AfterSuperAdminLogin(t *testing.T) {
	env, cleanup := setupTenantSignupIntegrationEnv(t)
	defer cleanup()

	loginReq := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(`{"email":"superadmin@example.com","password":"adminpass"}`),
	)
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	env.router.ServeHTTP(loginRR, loginReq)
	require.Equal(t, http.StatusOK, loginRR.Code, loginRR.Body.String())

	var loginBody authHandler.LoginResponse
	require.NoError(t, json.Unmarshal(loginRR.Body.Bytes(), &loginBody))
	require.NotEmpty(t, loginBody.Token)

	parsed, err := env.authSvc.ValidateToken(loginBody.Token)
	require.NoError(t, err)
	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, float64(uModel.UserRoleSuperAdmin), claims["role_id"])

	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/internal/tenant-signup-codes",
		strings.NewReader(`{"email":"invitee@example.com","expires_in_minutes":60,"notes":"from login flow"}`),
	)
	req.Header.Set("Authorization", "Bearer "+loginBody.Token)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	var resp authModel.CreateSignupCodeResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.Code)
	assert.Equal(t, "invitee@example.com", resp.Email)
	assert.Equal(t, 1, env.emailSender.calls)
	assert.Equal(t, "invitee@example.com", env.emailSender.last.ToEmail)
	assert.Contains(t, env.emailSender.last.RegisterURL, "/tenant-register?code=")
	assert.Contains(t, env.emailSender.last.RegisterURL, resp.Code)
}

func TestTenantSignupIntegration_RegisterWithCode_HappyPath(t *testing.T) {
	env, cleanup := setupTenantSignupIntegrationEnv(t)
	defer cleanup()

	recipient := fmt.Sprintf("invitee-happy-%d@test.com", time.Now().UnixNano())
	createdCode := createSignupCodeInternal(t, env, recipient, 120, "integration happy path")
	assert.Equal(t, 1, env.emailSender.calls)
	assert.Contains(t, env.emailSender.last.RegisterURL, createdCode.Code)

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
		usedAt          sql.NullTime
		usedByTenantID  sql.NullInt64
		usedByUserID    sql.NullInt64
		usedEmail       sql.NullString
		createdBy       sql.NullInt64
		storedRecipient sql.NullString
	)
	err := env.db.QueryRow(
		`SELECT used_at, used_by_tenant_id, used_by_user_id, used_email, created_by_user_id, recipient_email
		 FROM tenant_signup_codes
		 WHERE code_hash = $1`,
		hashOneTimeCodeForTest(createdCode.Code),
	).Scan(&usedAt, &usedByTenantID, &usedByUserID, &usedEmail, &createdBy, &storedRecipient)
	require.NoError(t, err)

	assert.True(t, usedAt.Valid)
	assert.True(t, usedByTenantID.Valid)
	assert.True(t, usedByUserID.Valid)
	assert.True(t, usedEmail.Valid)
	assert.True(t, createdBy.Valid)
	assert.True(t, storedRecipient.Valid)
	assert.Equal(t, recipient, storedRecipient.String)
	assert.Equal(t, int64(resp.TenantID), usedByTenantID.Int64)
	assert.Equal(t, int64(resp.AdminID), usedByUserID.Int64)
	assert.Equal(t, email, usedEmail.String)
	assert.Equal(t, int64(env.superadminUserID), createdBy.Int64)
}

func TestTenantSignupIntegration_RegisterWithCode_ReusedCodeReturns422(t *testing.T) {
	env, cleanup := setupTenantSignupIntegrationEnv(t)
	defer cleanup()

	createdCode := createSignupCodeInternal(t, env, "invitee-reuse@test.com", 120, "integration reused code")

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

func TestTenantSignupIntegration_RegisterWithCode_DuplicateSlugAutoSuffixes(t *testing.T) {
	env, cleanup := setupTenantSignupIntegrationEnv(t)
	defer cleanup()

	// Seeded tenant uses slug "default"; requesting the same slug should succeed as "default-2".
	createdCode := createSignupCodeInternal(t, env, "invitee-dup-slug@test.com", 120, "integration duplicate slug")
	ownerEmail := fmt.Sprintf("owner-collision-%d@test.com", time.Now().UnixNano())
	rr := registerTenantPublic(
		t, env,
		"Default Tenant Collision", "default", "Owner Collision", ownerEmail, "555-5050", "StrongPass123!",
		createdCode.Code,
	)

	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	var resp authModel.PublicTenantRegisterResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "default-2", resp.TenantSlug)
	assert.NotEmpty(t, resp.Token)

	var usedAt sql.NullTime
	err := env.db.QueryRow(
		`SELECT used_at FROM tenant_signup_codes WHERE code_hash = $1`,
		hashOneTimeCodeForTest(createdCode.Code),
	).Scan(&usedAt)
	require.NoError(t, err)
	assert.True(t, usedAt.Valid, "code should be consumed after successful auto-suffix registration")

	var slugCount int
	err = env.db.QueryRow(`SELECT COUNT(*) FROM tenants WHERE slug = $1`, "default-2").Scan(&slugCount)
	require.NoError(t, err)
	assert.Equal(t, 1, slugCount)
}

func TestTenantSignupIntegration_RegisterWithCode_ConcurrentSameCodeOnlyOneSucceeds(t *testing.T) {
	env, cleanup := setupTenantSignupIntegrationEnv(t)
	defer cleanup()

	createdCode := createSignupCodeInternal(t, env, "invitee-race@test.com", 120, "integration concurrent same code")

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
