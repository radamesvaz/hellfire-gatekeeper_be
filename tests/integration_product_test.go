package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	assert.Equal(t, http.StatusOK, rr.Code)

	// Parse the response to verify structure
	var products []map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &products)
	require.NoError(t, err)

	// Verify we have 2 products
	assert.Len(t, products, 2)

	// Verify first product structure
	product1 := products[0]
	assert.Equal(t, float64(1), product1["id_product"])
	assert.Equal(t, "Brownie Clásico", product1["name"])
	assert.Equal(t, "Delicioso brownie de chocolate", product1["description"])
	assert.Equal(t, 3.5, product1["price"])
	assert.Equal(t, true, product1["available"])
	assert.Equal(t, float64(6), product1["stock"])
	assert.Equal(t, "active", product1["status"])
	assert.Equal(t, []interface{}{}, product1["image_urls"])

	// Verify second product structure
	product2 := products[1]
	assert.Equal(t, float64(2), product2["id_product"])
	assert.Equal(t, "Suspiros", product2["name"])
	assert.Equal(t, "Suspiros tradicionales", product2["description"])
	assert.Equal(t, float64(5), product2["price"])
	assert.Equal(t, true, product2["available"])
	assert.Equal(t, float64(2), product2["stock"])
	assert.Equal(t, "active", product2["status"])
	assert.Equal(t, []interface{}{}, product2["image_urls"])
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

	assert.Equal(t, http.StatusOK, rr.Code)

	// Parse the response to verify structure
	var product map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &product)
	require.NoError(t, err)

	// Verify product structure
	assert.Equal(t, float64(1), product["id_product"])
	assert.Equal(t, "Brownie Clásico", product["name"])
	assert.Equal(t, "Delicioso brownie de chocolate", product["description"])
	assert.Equal(t, 3.5, product["price"])
	assert.Equal(t, true, product["available"])
	assert.Equal(t, float64(6), product["stock"])
	assert.Equal(t, "active", product["status"])
	assert.Equal(t, []interface{}{}, product["image_urls"])
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

	assert.Equal(t, http.StatusOK, rr.Code)

	// Parse the response to verify structure
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response structure
	assert.Equal(t, "Product created successfully", response["message"])
	assert.NotNil(t, response["product_id"])
	assert.NotNil(t, response["image_urls"])

	// Verify image_urls is an empty array
	imageURLs, ok := response["image_urls"].([]interface{})
	require.True(t, ok)
	assert.Len(t, imageURLs, 0)
}

func TestCreateProductWithImages(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupMySQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	repository := repository.ProductRepository{
		DB: db,
	}

	// Setup test directory for images
	testDir := "test_uploads"
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	imageService := imagesService.New(testDir)
	handler := handlers.ProductHandler{
		Repo:         &repository,
		ImageService: imageService,
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

	// Create a multipart form with product data and images
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add form fields
	w.WriteField("name", "Pie de parchita con imagen")
	w.WriteField("description", "Base de galleta maria, decorado con merengue suizo")
	w.WriteField("price", "18.0")
	w.WriteField("available", "true")
	w.WriteField("stock", "6")
	w.WriteField("status", "active")

	// Create test image files
	image1Content := []byte("fake image 1 content")
	image2Content := []byte("fake image 2 content")

	// Add first image
	fw1, err := w.CreateFormFile("images", "test_image1.jpg")
	require.NoError(t, err)
	_, err = fw1.Write(image1Content)
	require.NoError(t, err)

	// Add second image
	fw2, err := w.CreateFormFile("images", "test_image2.png")
	require.NoError(t, err)
	_, err = fw2.Write(image2Content)
	require.NoError(t, err)

	w.Close()

	// Send the multipart request
	req := httptest.NewRequest("POST", "/auth/products", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+jwt)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify response
	if rr.Code != http.StatusOK {
		t.Logf("Response body: %s", rr.Body.String())
		t.Logf("Response status: %d", rr.Code)
	}
	assert.Equal(t, http.StatusOK, rr.Code)

	// Parse response to get product ID and image URLs
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Product created successfully", response["message"])
	assert.NotNil(t, response["product_id"])
	assert.NotNil(t, response["image_urls"])

	// Verify image URLs are returned
	imageURLs, ok := response["image_urls"].([]interface{})
	require.True(t, ok)
	assert.Len(t, imageURLs, 2)

	// Verify images were saved to disk
	productID := response["product_id"].(float64)
	productDir := filepath.Join(testDir, "products", fmt.Sprintf("%.0f", productID))

	// Check that the directory was created
	_, err = os.Stat(productDir)
	assert.NoError(t, err)

	// Check that image files exist
	files, err := os.ReadDir(productDir)
	require.NoError(t, err)
	assert.Len(t, files, 2)

	// Verify the image URLs in the response have the correct format
	for _, imageURLInterface := range imageURLs {
		imageURL, ok := imageURLInterface.(string)
		require.True(t, ok, "Image URL should be a string")

		// Check that the URL starts with the correct path
		expectedPrefix := fmt.Sprintf("/uploads/products/%.0f/", productID)
		assert.True(t, strings.HasPrefix(imageURL, expectedPrefix),
			"Image URL %s should start with %s", imageURL, expectedPrefix)

		// Check that the URL ends with a valid image extension
		validExtensions := []string{".jpg", ".jpeg", ".png", ".webp"}
		hasValidExtension := false
		for _, ext := range validExtensions {
			if strings.HasSuffix(imageURL, ext) {
				hasValidExtension = true
				break
			}
		}
		assert.True(t, hasValidExtension, "Image URL %s should end with a valid image extension", imageURL)
	}
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

	assert.Equal(t, http.StatusOK, rr.Code)

	// Parse the response to verify structure
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response structure - the actual message is "Product status updated successfully"
	assert.Equal(t, "Product status updated successfully", response["message"])
}
