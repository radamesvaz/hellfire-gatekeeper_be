package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/handlers"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	repository "github.com/radamesvaz/bakery-app/internal/repository/products"
	"github.com/radamesvaz/bakery-app/internal/services/auth"
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
)

func TestCreateProductWithImagesIntegration(t *testing.T) {
	// Setup test directory
	testDir := "test_uploads"
	err := os.MkdirAll(testDir, 0755)
	assert.NoError(t, err)
	defer os.RemoveAll(testDir)

	// setup
	_, db, terminate, dsn := setupMySQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	repository := repository.ProductRepository{
		DB: db,
	}

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

	// Create test image file
	testImagePath := filepath.Join(testDir, "test.jpg")
	err = os.WriteFile(testImagePath, []byte("fake image content"), 0644)
	assert.NoError(t, err)

	// Create multipart form data
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add product data
	fields := map[string]string{
		"name":        "Producto con imagen",
		"description": "Descripción del producto",
		"price":       "25.50",
		"available":   "true",
		"stock":       "10",
		"status":      "active",
	}

	for key, value := range fields {
		fw, err := w.CreateFormField(key)
		assert.NoError(t, err)
		_, err = fw.Write([]byte(value))
		assert.NoError(t, err)
	}

	// Add image file
	fw, err := w.CreateFormFile("images", "test.jpg")
	assert.NoError(t, err)
	file, err := os.Open(testImagePath)
	assert.NoError(t, err)
	_, err = io.Copy(fw, file)
	assert.NoError(t, err)
	file.Close()

	w.Close()

	// Create request
	req := httptest.NewRequest("POST", "/auth/products", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+jwt)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Product created successfully", response["message"])

	// Verify image was saved
	productID := response["product_id"].(float64)
	productDir := filepath.Join(testDir, "products", fmt.Sprintf("%.0f", productID))
	_, err = os.Stat(productDir)
	assert.NoError(t, err, "Product directory should exist")

	// Verify image files exist
	files, err := os.ReadDir(productDir)
	assert.NoError(t, err)
	assert.Greater(t, len(files), 0, "Should have at least one image file")
}

func TestGetProductsWithImagesIntegration(t *testing.T) {
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

	// Verify response includes image_urls field
	var response []map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Greater(t, len(response), 0, "Should have products")

	// Check that each product has image_urls field
	for _, product := range response {
		_, hasImageURLs := product["image_urls"]
		assert.True(t, hasImageURLs, "Product should have image_urls field")
	}
}
