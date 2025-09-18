package tests

import (
	"encoding/json"
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
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister_Success(t *testing.T) {
	// Setup
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
	router.HandleFunc("/register", handler.Register).Methods("POST")

	// Test payload with valid data
	payload := `{
		"name": "Test Admin",
		"email": "admin@test.com",
		"phone": "555-1234",
		"password": "TestPassword123!"
	}`

	// Send the request
	req := httptest.NewRequest("POST", "/register", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, rr.Code)

	// Parse response
	var response auth.RegisterResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response contains token and message
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, "Admin user registered successfully", response.Message)

	// Verify user was created in database with admin role
	createdUser, err := repository.GetUserByEmail("admin@test.com")
	require.NoError(t, err)
	assert.Equal(t, "Test Admin", createdUser.Name)
	assert.Equal(t, "admin@test.com", createdUser.Email)
	assert.Equal(t, "555-1234", createdUser.Phone)
	assert.Equal(t, uModel.UserRoleAdmin, createdUser.IDRole)

	// Verify password was hashed (not plain text)
	assert.NotEqual(t, "TestPassword123!", createdUser.Password)
	assert.NotEmpty(t, createdUser.Password)
}

func TestRegister_EmailAlreadyExists(t *testing.T) {
	// Setup
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
	router.HandleFunc("/register", handler.Register).Methods("POST")

	// First registration
	payload1 := `{
		"name": "First Admin",
		"email": "duplicate@test.com",
		"phone": "555-1111",
		"password": "FirstPassword123!"
	}`

	req1 := httptest.NewRequest("POST", "/register", strings.NewReader(payload1))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	router.ServeHTTP(rr1, req1)

	assert.Equal(t, http.StatusCreated, rr1.Code)

	// Second registration with same email
	payload2 := `{
		"name": "Second Admin",
		"email": "duplicate@test.com",
		"phone": "555-2222",
		"password": "SecondPassword123!"
	}`

	req2 := httptest.NewRequest("POST", "/register", strings.NewReader(payload2))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	// Should return conflict
	assert.Equal(t, http.StatusConflict, rr2.Code)
	assert.Contains(t, rr2.Body.String(), "Email already exists")
}

func TestRegister_WeakPassword(t *testing.T) {
	// Setup
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
	router.HandleFunc("/register", handler.Register).Methods("POST")

	testCases := []struct {
		name        string
		password    string
		expectedMsg string
	}{
		{
			name:        "too short",
			password:    "Pass1!",
			expectedMsg: "password does not meet security requirements",
		},
		{
			name:        "no uppercase",
			password:    "password123!",
			expectedMsg: "password does not meet security requirements",
		},
		{
			name:        "no lowercase",
			password:    "PASSWORD123!",
			expectedMsg: "password does not meet security requirements",
		},
		{
			name:        "no digits",
			password:    "Password!",
			expectedMsg: "password does not meet security requirements",
		},
		{
			name:        "no special chars",
			password:    "Password123",
			expectedMsg: "password does not meet security requirements",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Replace spaces with hyphens for valid email format
			emailName := strings.ReplaceAll(tc.name, " ", "-")
			payload := `{
				"name": "Test Admin",
				"email": "weak-pass-` + emailName + `@test.com",
				"phone": "555-1234",
				"password": "` + tc.password + `"
			}`

			req := httptest.NewRequest("POST", "/register", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedMsg)
		})
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	// Setup
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
	router.HandleFunc("/register", handler.Register).Methods("POST")

	testCases := []struct {
		email       string
		expectedMsg string
	}{
		{
			email:       "invalid-email",
			expectedMsg: "Invalid Email",
		},
		{
			email:       "@invalid.com",
			expectedMsg: "Invalid Email",
		},
		{
			email:       "test@",
			expectedMsg: "Invalid Email",
		},
		{
			email:       "test.com",
			expectedMsg: "Invalid Email",
		},
		{
			email:       "",
			expectedMsg: "Email is required",
		},
	}

	for _, tc := range testCases {
		t.Run("invalid email: "+tc.email, func(t *testing.T) {
			payload := `{
				"name": "Test Admin",
				"email": "` + tc.email + `",
				"phone": "555-1234",
				"password": "ValidPassword123!"
			}`

			req := httptest.NewRequest("POST", "/register", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), tc.expectedMsg)
		})
	}
}

func TestRegister_InvalidJSON(t *testing.T) {
	// Setup
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
	router.HandleFunc("/register", handler.Register).Methods("POST")

	// Invalid JSON payload
	payload := `{
		"name": "Test Admin",
		"email": "test@example.com"
		"phone": "555-1234",
		"password": "ValidPassword123!"
	}`

	req := httptest.NewRequest("POST", "/register", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid request")
}

func TestRegister_MissingFields(t *testing.T) {
	// Setup
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
	router.HandleFunc("/register", handler.Register).Methods("POST")

	testCases := []struct {
		name    string
		payload string
	}{
		{
			name: "missing name",
			payload: `{
				"email": "test@example.com",
				"phone": "555-1234",
				"password": "ValidPassword123!"
			}`,
		},
		{
			name: "missing email",
			payload: `{
				"name": "Test Admin",
				"phone": "555-1234",
				"password": "ValidPassword123!"
			}`,
		},
		{
			name: "missing password",
			payload: `{
				"name": "Test Admin",
				"email": "test@example.com",
				"phone": "555-1234"
			}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/register", strings.NewReader(tc.payload))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			// Should fail with bad request due to missing fields or empty values
			assert.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}
