package tests

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/radamesvaz/bakery-app/internal/handlers"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	repository "github.com/radamesvaz/bakery-app/internal/repository/products"
	"github.com/radamesvaz/bakery-app/internal/services/auth"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupMySQLContainer(t *testing.T) (container testcontainers.Container, db *sql.DB, terminate func(), dsn string) {
	ctx := context.Background()
	err := godotenv.Load("../.env")
	if err != nil {
		fmt.Printf("⚠ Could not load .env file: %v", err)
	}

	dbUser := os.Getenv("MYSQL_USER")
	dbPassword := os.Getenv("MYSQL_PASSWORD")
	dbName := os.Getenv("MYSQL_DATABASE")
	port := os.Getenv("DB_PORT")

	if dbUser == "" || dbPassword == "" || dbName == "" || port == "" {
		t.Fatal("Missing env variables: MYSQL_USER, MYSQL_PASSWORD, MYSQL_DATABASE, DB_PORT")
	}

	req := testcontainers.ContainerRequest{
		Image:        "mysql:8.0",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": os.Getenv("MYSQL_ROOT_PASSWORD"),
			"MYSQL_USER":          os.Getenv("MYSQL_USER"),
			"MYSQL_PASSWORD":      os.Getenv("MYSQL_PASSWORD"),
			"MYSQL_DATABASE":      os.Getenv("MYSQL_DATABASE"),
		},
		// WaitingFor: wait.ForListeningPort("3306/tcp").WithStartupTimeout(30 * time.Second),
		WaitingFor: wait.ForLog("MySQL Community Server - GPL").WithStartupTimeout(60 * time.Second),
	}

	mysqlC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Error setting up the generic container: %v", err)
	}

	host, err := mysqlC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}

	ports, err := mysqlC.MappedPort(ctx, "3306")
	if err != nil {
		t.Fatal(err)
	}

	usableDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUser, dbPassword, host, ports.Port(), dbName)

	var dbConn *sql.DB

	// Wait for mysql to be ready
	for i := 0; i < 10; i++ {
		dbConn, err = sql.Open("mysql", usableDSN)
		if err == nil {
			err = dbConn.Ping()
			if err == nil {
				break
			}
		}
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		t.Fatalf("Could not connect to the database: %v", err)
	}

	// Cleanup function when tests end
	terminate = func() {
		dbConn.Close()
		mysqlC.Terminate(ctx)
	}

	return mysqlC, dbConn, terminate, usableDSN
}

func runMigrations(t *testing.T, dsn string) {
	m, err := migrate.New(
		"file://../migrations",
		"mysql://"+dsn,
	)
	if err != nil {
		t.Fatalf("Error initializing migration: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("Error running migration: %v", err)
	}
}

func TestGetAllProducts(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupMySQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	repository := repository.ProductRepository{
		DB: db,
	}
	handler := handlers.ProductHandler{
		Repo: &repository,
	}

	// Setup the router
	router := mux.NewRouter()
	router.HandleFunc("/products", handler.GetAllProducts).Methods("GET")

	// Send the simulated request
	req := httptest.NewRequest("GET", "/products", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	expected := fmt.Sprint(
		`[{
			"id_product": 1,
			"name": "Brownie Clásico",
			"description": "Delicioso brownie de chocolate",
			"price": 3.5,
			"available": true,
			"stock": 6,
			"status": "active",
			"created_on": "2025-04-14T10:00:00Z"
		},
		{
			"id_product": 2,
			"name": "Suspiros",
			"description": "Suspiros tradicionales",
			"price": 5,
			"available": true,
			"stock": 2,
			"status": "active",
			"created_on": "2025-04-14T10:00:00Z"
		}]`,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, expected, rr.Body.String())
}

func TestGetProductByID(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupMySQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	repository := repository.ProductRepository{
		DB: db,
	}
	handler := handlers.ProductHandler{
		Repo: &repository,
	}

	// Setup the router
	router := mux.NewRouter()
	router.HandleFunc("/products/{id}", handler.GetProductByID).Methods("GET")

	// Send the simulated request
	req := httptest.NewRequest("GET", "/products/1", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	expected := fmt.Sprint(
		`{
			"id_product": 1,
			"name": "Brownie Clásico",
			"description": "Delicioso brownie de chocolate",
			"price": 3.5,
			"available": true,
			"stock": 6,
			"status": "active",
			"created_on": "2025-04-14T10:00:00Z"
		}`,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, expected, rr.Body.String())
}

func TestCreateProduct(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupMySQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	repository := repository.ProductRepository{
		DB: db,
	}
	handler := handlers.ProductHandler{
		Repo: &repository,
	}

	// Setup the router
	router := mux.NewRouter()

	authRouter := router.PathPrefix("/auth").Subrouter()

	secret := "testingsecret"
	exp := 60

	var authService auth.Service = auth.New(secret, exp)
	authRouter.Use(middleware.AuthMiddleware(authService))

	authRouter.HandleFunc("/products", handler.CreateProduct).Methods("POST")

	jwt, err := authService.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com")
	if err != nil {
		t.Fatalf("Error creating a JWT for integration testing: %v", err)
	}

	payload := `{
		"name": "Pie de parchita",
		"description": "Base de galleta maria, decorado con merengue suizo",
		"price": 18.0,
		"available": true,
		"stock": 6,
		"status": "active"
	  }`

	// Send the simulated request
	req := httptest.NewRequest("POST", "/auth/products", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	expected := fmt.Sprint(
		`{
			"message": "Product created successfully"
		}`,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, expected, rr.Body.String())
}

func TestUpdateProduct(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupMySQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	repository := repository.ProductRepository{
		DB: db,
	}
	handler := handlers.ProductHandler{
		Repo: &repository,
	}

	// Setup the router
	router := mux.NewRouter()

	authRouter := router.PathPrefix("/auth").Subrouter()

	secret := "testingsecret"
	exp := 60

	var authService auth.Service = auth.New(secret, exp)
	authRouter.Use(middleware.AuthMiddleware(authService))

	authRouter.HandleFunc("/products/{id}", handler.UpdateProduct).Methods("PUT")

	jwt, err := authService.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com")
	if err != nil {
		t.Fatalf("Error creating a JWT for integration testing: %v", err)
	}

	payload := `{
		"name": "Pie de parchita - ACTUALIZADO",
		"description": "Base de galleta maria, decorado con merengue suizo - actualizado",
		"price": 18.0,
		"available": true,
		"stock": 6,
		"status": "active"
	  }`

	// Send the simulated request
	req := httptest.NewRequest("PUT", "/auth/products/1", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	expected := fmt.Sprint(
		`{
			"message": "Product updated successfully"
		}`,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, expected, rr.Body.String())
}

func TestDeleteProduct(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupMySQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	repository := repository.ProductRepository{
		DB: db,
	}
	handler := handlers.ProductHandler{
		Repo: &repository,
	}

	// Setup the router
	router := mux.NewRouter()

	authRouter := router.PathPrefix("/auth").Subrouter()

	secret := "testingsecret"
	exp := 60

	var authService auth.Service = auth.New(secret, exp)
	authRouter.Use(middleware.AuthMiddleware(authService))

	authRouter.HandleFunc("/products/{id}", handler.UpdateProductStatus).Methods("PATCH")

	jwt, err := authService.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com")
	if err != nil {
		t.Fatalf("Error creating a JWT for integration testing: %v", err)
	}

	payload := `{
		"status": "deleted"
	  }`

	// Send the simulated request
	req := httptest.NewRequest("PATCH", "/auth/products/1", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	expected := fmt.Sprint(
		`{
			"message": "Product updated successfully"
		}`,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, expected, rr.Body.String())
}
