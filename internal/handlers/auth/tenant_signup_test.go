package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/lib/pq"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	tenantSignupRepo "github.com/radamesvaz/bakery-app/internal/repository/tenantsignup"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	tenantSignupService "github.com/radamesvaz/bakery-app/internal/services/tenantsignup"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantSignupHandler_CreateSignupCode_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := &tenantSignupService.TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authService.New("testingsecret", 60),
	}
	handler := &TenantSignupHandler{Service: svc}

	mock.ExpectQuery(`INSERT INTO tenant_signup_codes`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), uint64(99), "manual request").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint64(7)))

	body := bytes.NewBufferString(`{"expires_in_minutes":120,"notes":"manual request"}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/internal/tenant-signup-codes", body)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserClaimsKey, jwt.MapClaims{
		"user_id": float64(99),
		"role_id": float64(1),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.CreateSignupCode(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	var resp authModel.CreateSignupCodeResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, uint64(7), resp.ID)
	assert.NotEmpty(t, resp.Code)
	assert.True(t, resp.ExpiresAt.After(time.Now().UTC().Add(100*time.Minute)))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantSignupHandler_RegisterTenantWithCode_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := &tenantSignupService.TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authService.New("testingsecret", 60),
	}
	handler := &TenantSignupHandler{Service: svc}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, expires_at, used_at, revoked_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "expires_at", "used_at", "revoked_at"}).
			AddRow(uint64(5), time.Now().UTC().Add(2*time.Hour), nil, nil))
	mock.ExpectQuery(`INSERT INTO tenants`).
		WithArgs("Acme Bakery", "acme").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint64(10)))
	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(uint64(10), 1, "Owner Admin", "owner@acme.com", "555-0101", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id_user"}).AddRow(uint64(100)))
	mock.ExpectExec(`UPDATE tenant_signup_codes`).
		WithArgs(uint64(5), uint64(10), uint64(100), "owner@acme.com").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	body := bytes.NewBufferString(`{
		"tenant_name":"Acme Bakery",
		"tenant_slug":"acme",
		"admin_name":"Owner Admin",
		"email":"owner@acme.com",
		"phone":"555-0101",
		"password":"StrongPass123!",
		"one_time_code":"ABCD1234-EFGH5678"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/public/tenant-register", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterTenantWithCode(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	var resp authModel.PublicTenantRegisterResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, uint64(10), resp.TenantID)
	assert.Equal(t, uint64(100), resp.AdminID)
	assert.Equal(t, "Acme Bakery", resp.TenantName)
	assert.NotEmpty(t, resp.Token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantSignupHandler_CreateSignupCode_DefaultTTL120(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := &tenantSignupService.TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authService.New("testingsecret", 60),
	}
	handler := &TenantSignupHandler{Service: svc}

	mock.ExpectQuery(`INSERT INTO tenant_signup_codes`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), uint64(99), "").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint64(7)))

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/auth/internal/tenant-signup-codes", body)
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), middleware.UserClaimsKey, jwt.MapClaims{
		"user_id": float64(99),
		"role_id": float64(1),
	})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.CreateSignupCode(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	var resp authModel.CreateSignupCodeResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, uint64(7), resp.ID)
	assert.NotEmpty(t, resp.Code)
	want := time.Now().UTC().Add(120 * time.Minute)
	assert.WithinDuration(t, want, resp.ExpiresAt, 3*time.Minute)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func validRegisterBody(oneTimeCode string) string {
	return `{
		"tenant_name":"Acme Bakery",
		"tenant_slug":"acme",
		"admin_name":"Owner Admin",
		"email":"owner@acme.com",
		"phone":"555-0101",
		"password":"StrongPass123!",
		"one_time_code":"` + oneTimeCode + `"
	}`
}

func TestTenantSignupHandler_RegisterTenantWithCode_WithExpiredOTC_Returns422(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := &tenantSignupService.TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authService.New("testingsecret", 60),
	}
	handler := &TenantSignupHandler{Service: svc}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, expires_at, used_at, revoked_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "expires_at", "used_at", "revoked_at"}).
			AddRow(uint64(5), time.Now().UTC().Add(-time.Hour), nil, nil))
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodPost, "/public/tenant-register",
		bytes.NewBufferString(validRegisterBody("ABCD1234-EFGH5678")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterTenantWithCode(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "Invalid or unavailable one-time code")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantSignupHandler_RegisterTenantWithCode_WithUsedOTC_Returns422(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := &tenantSignupService.TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authService.New("testingsecret", 60),
	}
	handler := &TenantSignupHandler{Service: svc}

	usedAt := time.Now().UTC().Add(-30 * time.Minute)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, expires_at, used_at, revoked_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "expires_at", "used_at", "revoked_at"}).
			AddRow(uint64(5), time.Now().UTC().Add(2*time.Hour), usedAt, nil))
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodPost, "/public/tenant-register",
		bytes.NewBufferString(validRegisterBody("ABCD1234-EFGH5678")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterTenantWithCode(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code, rr.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantSignupHandler_RegisterTenantWithCode_WithRevokedOTC_Returns422(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := &tenantSignupService.TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authService.New("testingsecret", 60),
	}
	handler := &TenantSignupHandler{Service: svc}

	revokedAt := time.Now().UTC().Add(-time.Minute)
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, expires_at, used_at, revoked_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "expires_at", "used_at", "revoked_at"}).
			AddRow(uint64(5), time.Now().UTC().Add(2*time.Hour), nil, revokedAt))
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodPost, "/public/tenant-register",
		bytes.NewBufferString(validRegisterBody("ABCD1234-EFGH5678")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterTenantWithCode(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code, rr.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantSignupHandler_RegisterTenantWithCode_WithUnknownOTC_Returns422(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := &tenantSignupService.TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authService.New("testingsecret", 60),
	}
	handler := &TenantSignupHandler{Service: svc}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, expires_at, used_at, revoked_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodPost, "/public/tenant-register",
		bytes.NewBufferString(validRegisterBody("ZZZZ9999-YYYY8888")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterTenantWithCode(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code, rr.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantSignupHandler_RegisterTenantWithCode_WithDuplicateSlug_Returns409(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := &tenantSignupService.TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authService.New("testingsecret", 60),
	}
	handler := &TenantSignupHandler{Service: svc}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, expires_at, used_at, revoked_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "expires_at", "used_at", "revoked_at"}).
			AddRow(uint64(5), time.Now().UTC().Add(2*time.Hour), nil, nil))
	mock.ExpectQuery(`INSERT INTO tenants`).
		WithArgs("Acme Bakery", "acme").
		WillReturnError(&pq.Error{Code: "23505"})
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodPost, "/public/tenant-register",
		bytes.NewBufferString(validRegisterBody("ABCD1234-EFGH5678")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterTenantWithCode(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "Tenant slug already exists")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantSignupHandler_RegisterTenantWithCode_WithDuplicateAdminEmail_Returns409(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := &tenantSignupService.TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authService.New("testingsecret", 60),
	}
	handler := &TenantSignupHandler{Service: svc}

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, expires_at, used_at, revoked_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "expires_at", "used_at", "revoked_at"}).
			AddRow(uint64(5), time.Now().UTC().Add(2*time.Hour), nil, nil))
	mock.ExpectQuery(`INSERT INTO tenants`).
		WithArgs("Acme Bakery", "acme").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint64(10)))
	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(uint64(10), 1, "Owner Admin", "owner@acme.com", "555-0101", sqlmock.AnyArg()).
		WillReturnError(&pq.Error{Code: "23505"})
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodPost, "/public/tenant-register",
		bytes.NewBufferString(validRegisterBody("ABCD1234-EFGH5678")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.RegisterTenantWithCode(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "Admin email already exists")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Two sequential registrations with the same OTC: only the first succeeds (sqlmock).
// True concurrent FOR UPDATE behavior belongs in integration tests against PostgreSQL.
func TestTenantSignupHandler_RegisterTenantWithCode_WithRaceOnSameOTC_OnlyOneSucceeds(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := &tenantSignupService.TenantSignupService{
		Repo:        &tenantSignupRepo.Repository{DB: db},
		AuthService: authService.New("testingsecret", 60),
	}
	handler := &TenantSignupHandler{Service: svc}

	code := "ABCD1234-EFGH5678"
	body := validRegisterBody(code)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, expires_at, used_at, revoked_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "expires_at", "used_at", "revoked_at"}).
			AddRow(uint64(5), time.Now().UTC().Add(2*time.Hour), nil, nil))
	mock.ExpectQuery(`INSERT INTO tenants`).
		WithArgs("Acme Bakery", "acme").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint64(10)))
	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(uint64(10), 1, "Owner Admin", "owner@acme.com", "555-0101", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id_user"}).AddRow(uint64(100)))
	mock.ExpectExec(`UPDATE tenant_signup_codes`).
		WithArgs(uint64(5), uint64(10), uint64(100), "owner@acme.com").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	req1 := httptest.NewRequest(http.MethodPost, "/public/tenant-register", bytes.NewBufferString(body))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	handler.RegisterTenantWithCode(rr1, req1)
	require.Equal(t, http.StatusCreated, rr1.Code, rr1.Body.String())

	usedAt := time.Now().UTC()
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id, expires_at, used_at, revoked_at`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "expires_at", "used_at", "revoked_at"}).
			AddRow(uint64(5), time.Now().UTC().Add(2*time.Hour), usedAt, nil))
	mock.ExpectRollback()

	req2 := httptest.NewRequest(http.MethodPost, "/public/tenant-register", bytes.NewBufferString(body))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	handler.RegisterTenantWithCode(rr2, req2)

	assert.Equal(t, http.StatusUnprocessableEntity, rr2.Code, rr2.Body.String())
	assert.NoError(t, mock.ExpectationsWereMet())
}
