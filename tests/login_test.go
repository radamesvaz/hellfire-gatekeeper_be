package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/handlers/auth"
	"github.com/radamesvaz/bakery-app/internal/repository/user"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	"github.com/stretchr/testify/assert"
)

func TestLogin(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupMySQLContainer(t)
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
