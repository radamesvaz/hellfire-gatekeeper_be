package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	bootstrapRepo "github.com/radamesvaz/bakery-app/internal/repository/bootstrap"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	bootstrapService "github.com/radamesvaz/bakery-app/internal/services/bootstrap"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapTenant_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	authSvc := authService.New("testingsecret", 60)
	svc := &bootstrapService.BootstrapService{
		Repo:        &bootstrapRepo.Repository{DB: db},
		AuthService: authSvc,
	}
	handler := &BootstrapHandler{Service: svc}

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO tenants`).
		WithArgs("Acme Bakery", "acme").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint64(10)))
	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs(uint64(10), 1, "Owner Admin", "owner@acme.com", "555-0101", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id_user"}).AddRow(uint64(100)))
	mock.ExpectCommit()

	body := bytes.NewBufferString(`{
		"tenant_name":"Acme Bakery",
		"tenant_slug":"acme",
		"admin_name":"Owner Admin",
		"email":"owner@acme.com",
		"phone":"555-0101",
		"password":"StrongPass123!"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/setup/bootstrap/tenant", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.BootstrapTenant(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
	var resp authModel.BootstrapTenantResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, uint64(10), resp.TenantID)
	assert.Equal(t, "acme", resp.TenantSlug)
	assert.Equal(t, "Acme Bakery", resp.TenantName)
	assert.Equal(t, "owner@acme.com", resp.AdminEmail)
	assert.NotEmpty(t, resp.Token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

