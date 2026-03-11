package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/radamesvaz/bakery-app/internal/handlers/auth"
	"github.com/radamesvaz/bakery-app/internal/repository/user"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	authService := authService.New("testingsecret", 60)
	repository := user.UserRepository{
		DB: db,
	}
	handler := auth.LoginHandler{
		UserRepo:    repository,
		AuthService: *authService,
	}

	// Setup the router
	router := mux.NewRouter()
	router.HandleFunc("/login", handler.Login).Methods("POST")

	//
	payload := `{
		"email": "admin@example.com",
		"password": "adminpass"
	  }`

	// Send the simulated request
	req := httptest.NewRequest("POST", "/login", strings.NewReader(payload))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "token")
}

func TestLogin_SoftDeletedUserCannotLogin(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	// Soft delete the seeded admin user so that GetUserByEmail treats it as not found.
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `UPDATE users SET deleted_at = NOW() WHERE email = 'admin@example.com'`)
	require.NoError(t, err)

	authSvc := authService.New("testingsecret", 60)
	repository := user.UserRepository{
		DB: db,
	}
	handler := auth.LoginHandler{
		UserRepo:    repository,
		AuthService: *authSvc,
	}

	// Setup the router
	router := mux.NewRouter()
	router.HandleFunc("/login", handler.Login).Methods("POST")

	payload := `{
		"email": "admin@example.com",
		"password": "adminpass"
	  }`

	req := httptest.NewRequest("POST", "/login", strings.NewReader(payload))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// With soft delete, the login handler should not allow authentication and
	// will return a 500 with "User not found" (current behavior).
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "User not found")
}
