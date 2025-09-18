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
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
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

func setupPostgreSQLContainer(t *testing.T) (container testcontainers.Container, db *sql.DB, terminate func(), dsn string) {
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
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     dbUser,
			"POSTGRES_PASSWORD": dbPassword,
			"POSTGRES_DB":       dbName,
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithStartupTimeout(60 * time.Second),
	}

	postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Error setting up the generic container: %v", err)
	}

	host, err := postgresC.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}

	ports, err := postgresC.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatal(err)
	}

	usableDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, host, ports.Port(), dbName)

	var dbConn *sql.DB

	// Wait for postgres to be ready
	for i := 0; i < 10; i++ {
		dbConn, err = sql.Open("postgres", usableDSN)
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
		postgresC.Terminate(ctx)
	}

	return postgresC, dbConn, terminate, usableDSN
}

func runMigrations(t *testing.T, dsn string) {
	m, err := migrate.New(
		"file://../migrations",
		dsn,
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
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
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
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
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
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
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
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
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

	// Setup handlers
	productHandler := handlers.ProductHandler{
		Repo: &repository,
	}

	imageService := imagesService.New(testDir)
	imageHandler := handlers.ImageHandler{
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

	// Add routes for both product and image handlers
	authRouter.HandleFunc("/products", productHandler.CreateProduct).Methods("POST")
	authRouter.HandleFunc("/products/{id}/images", imageHandler.AddProductImages).Methods("POST")

	jwt, err := authService.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com")
	if err != nil {
		t.Fatalf("Error creating a JWT for integration testing: %v", err)
	}

	// Step 1: Create product with JSON
	createData := map[string]interface{}{
		"name":        "Pie de parchita con imagen",
		"description": "Base de galleta maria, decorado con merengue suizo",
		"price":       18.0,
		"available":   true,
		"stock":       6,
		"status":      "active",
	}

	jsonData, err := json.Marshal(createData)
	require.NoError(t, err)

	// Send JSON request to create product
	req := httptest.NewRequest("POST", "/auth/products", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify product creation response
	if rr.Code != http.StatusOK {
		t.Logf("Product creation response body: %s", rr.Body.String())
		t.Logf("Product creation response status: %d", rr.Code)
	}
	assert.Equal(t, http.StatusOK, rr.Code)

	// Parse product creation response
	var productResponse map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &productResponse)
	require.NoError(t, err)
	assert.Equal(t, "Product created successfully", productResponse["message"])
	assert.NotNil(t, productResponse["product_id"])
	assert.NotNil(t, productResponse["image_urls"])

	// Get the created product ID
	productID, ok := productResponse["product_id"].(float64)
	require.True(t, ok)
	productIDStr := fmt.Sprintf("%.0f", productID)

	// Verify image_urls is an empty array initially
	imageURLs, ok := productResponse["image_urls"].([]interface{})
	require.True(t, ok)
	assert.Len(t, imageURLs, 0)

	// Step 2: Add images using multipart form
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

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

	// Send multipart request to add images
	imageReq := httptest.NewRequest("POST", fmt.Sprintf("/auth/products/%s/images", productIDStr), &b)
	imageReq.Header.Set("Content-Type", w.FormDataContentType())
	imageReq.Header.Set("Authorization", "Bearer "+jwt)
	imageRr := httptest.NewRecorder()
	router.ServeHTTP(imageRr, imageReq)

	// Verify image addition response
	if imageRr.Code != http.StatusOK {
		t.Logf("Image addition response body: %s", imageRr.Body.String())
		t.Logf("Image addition response status: %d", imageRr.Code)
	}
	assert.Equal(t, http.StatusOK, imageRr.Code)

	// Parse image addition response
	var imageResponse map[string]interface{}
	err = json.Unmarshal(imageRr.Body.Bytes(), &imageResponse)
	require.NoError(t, err)
	assert.Equal(t, "Images added successfully", imageResponse["message"])
	assert.Contains(t, imageResponse, "new_images")
	assert.Contains(t, imageResponse, "all_images")

	// Verify final image URLs count
	finalImageURLs, ok := imageResponse["all_images"].([]interface{})
	require.True(t, ok)
	assert.Len(t, finalImageURLs, 2)

	// Verify images were saved to Cloudinary
	// Check that image URLs are Cloudinary URLs
	for _, imageURLInterface := range finalImageURLs {
		imageURL, ok := imageURLInterface.(string)
		require.True(t, ok, "Image URL should be a string")

		// Check that the URL is a Cloudinary URL
		assert.True(t, strings.Contains(imageURL, "cloudinary.com"),
			"Image URL should be a Cloudinary URL, got: %s", imageURL)
		assert.True(t, strings.Contains(imageURL, "bakery/products"),
			"Image URL should contain bakery/products path, got: %s", imageURL)
	}
}

func TestUpdateProduct(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
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
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
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

func TestUpdateProductWithImages(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
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

	// Setup handlers
	productHandler := handlers.ProductHandler{
		Repo: &repository,
	}

	imageService := imagesService.New(testDir)
	imageHandler := handlers.ImageHandler{
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

	// Add routes for both product and image handlers
	authRouter.HandleFunc("/products/{id}", productHandler.UpdateProduct).Methods("PUT")
	authRouter.HandleFunc("/products/{id}/images", imageHandler.AddProductImages).Methods("POST")

	jwt, err := authService.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com")
	if err != nil {
		t.Fatalf("Error creating a JWT for integration testing: %v", err)
	}

	// Step 1: Update product data with JSON
	updateData := map[string]interface{}{
		"name":        "Brownie Clásico - ACTUALIZADO CON IMÁGENES",
		"description": "Delicioso brownie de chocolate - ahora con imágenes",
		"price":       4.0,
		"available":   true,
		"stock":       8,
		"status":      "active",
		"image_urls":  []string{}, // Empty initially
	}

	jsonData, err := json.Marshal(updateData)
	require.NoError(t, err)

	// Send JSON request to update product
	req := httptest.NewRequest("PUT", "/auth/products/1", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify product update response
	if rr.Code != http.StatusOK {
		t.Logf("Product update response body: %s", rr.Body.String())
		t.Logf("Product update response status: %d", rr.Code)
	}
	assert.Equal(t, http.StatusOK, rr.Code)

	// Parse product update response
	var productResponse map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &productResponse)
	require.NoError(t, err)
	assert.Equal(t, "Product updated successfully", productResponse["message"])

	// Step 2: Add images using multipart form
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Create test image files
	image1Content := []byte("fake updated image 1 content")
	image2Content := []byte("fake updated image 2 content")

	// Add first image
	fw1, err := w.CreateFormFile("images", "updated_image1.jpg")
	require.NoError(t, err)
	_, err = fw1.Write(image1Content)
	require.NoError(t, err)

	// Add second image
	fw2, err := w.CreateFormFile("images", "updated_image2.png")
	require.NoError(t, err)
	_, err = fw2.Write(image2Content)
	require.NoError(t, err)

	w.Close()

	// Send multipart request to add images
	imageReq := httptest.NewRequest("POST", "/auth/products/1/images", &b)
	imageReq.Header.Set("Content-Type", w.FormDataContentType())
	imageReq.Header.Set("Authorization", "Bearer "+jwt)
	imageRr := httptest.NewRecorder()
	router.ServeHTTP(imageRr, imageReq)

	// Verify image addition response
	if imageRr.Code != http.StatusOK {
		t.Logf("Image addition response body: %s", imageRr.Body.String())
		t.Logf("Image addition response status: %d", imageRr.Code)
	}
	assert.Equal(t, http.StatusOK, imageRr.Code)

	// Parse image addition response
	var imageResponse map[string]interface{}
	err = json.Unmarshal(imageRr.Body.Bytes(), &imageResponse)
	require.NoError(t, err)
	assert.Equal(t, "Images added successfully", imageResponse["message"])
	assert.Contains(t, imageResponse, "new_images")
	assert.Contains(t, imageResponse, "all_images")

	// Verify the product was updated in the database
	ctx := context.Background()
	updatedProduct, err := repository.GetProductByID(ctx, 1)
	require.NoError(t, err)

	// Verify images were saved to Cloudinary
	// Check that image URLs are Cloudinary URLs
	for _, imageURL := range updatedProduct.ImageURLs {
		// Check that the URL is a Cloudinary URL
		assert.True(t, strings.Contains(imageURL, "cloudinary.com"),
			"Image URL should be a Cloudinary URL, got: %s", imageURL)
		assert.True(t, strings.Contains(imageURL, "bakery/products"),
			"Image URL should contain bakery/products path, got: %s", imageURL)
	}

	assert.Equal(t, "Brownie Clásico - ACTUALIZADO CON IMÁGENES", updatedProduct.Name)
	assert.Equal(t, "Delicioso brownie de chocolate - ahora con imágenes", updatedProduct.Description)
	assert.Equal(t, 4.0, updatedProduct.Price)
	assert.Equal(t, uint64(8), updatedProduct.Stock)
	assert.Len(t, updatedProduct.ImageURLs, 2) // Should have 2 images after adding them

	// Verify the image URLs have the correct format
	for _, imageURL := range updatedProduct.ImageURLs {
		// Check that the URL starts with the correct path
		expectedPrefix := "/uploads/products/1/"
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

func TestDeleteProductImage(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
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

	// Setup handlers
	productHandler := handlers.ProductHandler{
		Repo: &repository,
	}

	imageService := imagesService.New(testDir)
	imageHandler := handlers.ImageHandler{
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

	// Add routes for both product and image handlers
	authRouter.HandleFunc("/products/{id}", productHandler.UpdateProduct).Methods("PUT")
	authRouter.HandleFunc("/products/{id}/images", imageHandler.AddProductImages).Methods("POST")
	authRouter.HandleFunc("/products/{id}/images", imageHandler.DeleteProductImage).Methods("DELETE")

	jwt, err := authService.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com")
	if err != nil {
		t.Fatalf("Error creating a JWT for integration testing: %v", err)
	}

	// Step 1: Update product data with JSON
	updateData := map[string]interface{}{
		"name":        "Producto para Test de Eliminación",
		"description": "Producto con múltiples imágenes para probar eliminación",
		"price":       5.0,
		"available":   true,
		"stock":       10,
		"status":      "active",
		"image_urls":  []string{}, // Empty initially
	}

	jsonData, err := json.Marshal(updateData)
	require.NoError(t, err)

	// Send JSON request to update product
	req := httptest.NewRequest("PUT", "/auth/products/1", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify product update response
	if rr.Code != http.StatusOK {
		t.Logf("Product update response body: %s", rr.Body.String())
		t.Logf("Product update response status: %d", rr.Code)
	}
	assert.Equal(t, http.StatusOK, rr.Code)

	// Parse product update response
	var productResponse map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &productResponse)
	require.NoError(t, err)
	assert.Equal(t, "Product updated successfully", productResponse["message"])

	// Step 2: Add multiple images using multipart form
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Create test image files
	image1Content := []byte("fake image 1 content for deletion test")
	image2Content := []byte("fake image 2 content for deletion test")
	image3Content := []byte("fake image 3 content for deletion test")

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

	// Add third image
	fw3, err := w.CreateFormFile("images", "test_image3.webp")
	require.NoError(t, err)
	_, err = fw3.Write(image3Content)
	require.NoError(t, err)

	w.Close()

	// Send multipart request to add images
	imageReq := httptest.NewRequest("POST", "/auth/products/1/images", &b)
	imageReq.Header.Set("Content-Type", w.FormDataContentType())
	imageReq.Header.Set("Authorization", "Bearer "+jwt)
	imageRr := httptest.NewRecorder()
	router.ServeHTTP(imageRr, imageReq)

	// Verify image addition response
	if imageRr.Code != http.StatusOK {
		t.Logf("Image addition response body: %s", imageRr.Body.String())
		t.Logf("Image addition response status: %d", imageRr.Code)
	}
	assert.Equal(t, http.StatusOK, imageRr.Code)

	// Parse image addition response
	var imageResponse map[string]interface{}
	err = json.Unmarshal(imageRr.Body.Bytes(), &imageResponse)
	require.NoError(t, err)
	assert.Equal(t, "Images added successfully", imageResponse["message"])
	assert.Contains(t, imageResponse, "new_images")
	assert.Contains(t, imageResponse, "all_images")

	// Get the list of all images
	allImages, ok := imageResponse["all_images"].([]interface{})
	require.True(t, ok)
	assert.Len(t, allImages, 3) // Should have 3 images

	// Verify images were saved to Cloudinary
	// Check that image URLs are Cloudinary URLs
	for _, imageURLInterface := range allImages {
		imageURL, ok := imageURLInterface.(string)
		require.True(t, ok, "Image URL should be a string")

		// Check that the URL is a Cloudinary URL
		assert.True(t, strings.Contains(imageURL, "cloudinary.com"),
			"Image URL should be a Cloudinary URL, got: %s", imageURL)
		assert.True(t, strings.Contains(imageURL, "bakery/products"),
			"Image URL should contain bakery/products path, got: %s", imageURL)
	}

	// Step 3: Delete a specific image (the second one)
	imageToDelete := allImages[1].(string) // Get the second image URL
	t.Logf("Deleting image: %s", imageToDelete)

	// Use query parameter instead of path parameter to avoid issues with slashes in URL
	deleteReq := httptest.NewRequest("DELETE", "/auth/products/1/images?imageUrl="+url.QueryEscape(imageToDelete), nil)
	deleteReq.Header.Set("Authorization", "Bearer "+jwt)
	deleteRr := httptest.NewRecorder()
	router.ServeHTTP(deleteRr, deleteReq)

	// Verify delete response
	if deleteRr.Code != http.StatusOK {
		t.Logf("Delete response body: %s", deleteRr.Body.String())
		t.Logf("Delete response status: %d", deleteRr.Code)
	}
	assert.Equal(t, http.StatusOK, deleteRr.Code)

	// Parse delete response
	var deleteResponse map[string]interface{}
	err = json.Unmarshal(deleteRr.Body.Bytes(), &deleteResponse)
	require.NoError(t, err)
	assert.Equal(t, "Image deleted successfully", deleteResponse["message"])

	// Step 4: Verify the image was deleted from Cloudinary
	// Check that the remaining images are still Cloudinary URLs
	remainingImages, ok := deleteResponse["all_images"].([]interface{})
	require.True(t, ok)
	assert.Len(t, remainingImages, 2) // Should have 2 images left

	// Verify all remaining images are Cloudinary URLs
	for _, imageURLInterface := range remainingImages {
		imageURL, ok := imageURLInterface.(string)
		require.True(t, ok, "Image URL should be a string")

		// Check that the URL is a Cloudinary URL
		assert.True(t, strings.Contains(imageURL, "cloudinary.com"),
			"Image URL should be a Cloudinary URL, got: %s", imageURL)
		assert.True(t, strings.Contains(imageURL, "bakery/products"),
			"Image URL should contain bakery/products path, got: %s", imageURL)
	}

	// Step 5: Verify the image was removed from the database
	ctx := context.Background()
	updatedProduct, err := repository.GetProductByID(ctx, 1)
	require.NoError(t, err)

	assert.Len(t, updatedProduct.ImageURLs, 2) // Should have 2 images left

	// Verify the deleted image is not in the remaining images
	for _, remainingImageURL := range updatedProduct.ImageURLs {
		assert.NotEqual(t, imageToDelete, remainingImageURL, "Deleted image should not be in remaining images")
	}

	// Step 6: Verify the remaining images are still intact
	assert.Len(t, updatedProduct.ImageURLs, 2)

	// Verify the remaining images are Cloudinary URLs
	for _, remainingImageURL := range updatedProduct.ImageURLs {
		// Check that the URL is a Cloudinary URL
		assert.True(t, strings.Contains(remainingImageURL, "cloudinary.com"),
			"Image URL should be a Cloudinary URL, got: %s", remainingImageURL)
		assert.True(t, strings.Contains(remainingImageURL, "bakery/products"),
			"Image URL should contain bakery/products path, got: %s", remainingImageURL)
	}

	t.Logf("Successfully deleted image: %s", imageToDelete)
	t.Logf("Remaining images: %v", updatedProduct.ImageURLs)
}
