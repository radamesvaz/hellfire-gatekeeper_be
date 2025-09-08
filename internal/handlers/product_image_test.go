package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	productsRepository "github.com/radamesvaz/bakery-app/internal/repository/products"
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductHandler_CreateProductWithImages(t *testing.T) {
	// Setup test directory
	testDir := "test_uploads"
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Setting up mock
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &productsRepository.ProductRepository{DB: db}
	imageService := imagesService.New(testDir)
	handler := &ProductHandler{
		Repo:         repo,
		ImageService: imageService,
	}

	// Setup the router
	router := mux.NewRouter()
	router.HandleFunc("/products", handler.CreateProduct).Methods("POST")

	tests := []struct {
		name           string
		productData    map[string]string
		imageFiles     []string // filenames to create
		expectedStatus int
		expectedError  bool
		setupMock      func()
	}{
		{
			name: "HAPPY PATH: Create product with images",
			productData: map[string]string{
				"name":        "Producto con imágenes",
				"description": "Descripción del producto",
				"price":       "25.50",
				"available":   "true",
				"stock":       "10",
				"status":      "active",
			},
			imageFiles:     []string{"test1.jpg", "test2.jpg"},
			expectedStatus: http.StatusOK,
			expectedError:  false,
			setupMock: func() {
				// Mock product creation
				mock.ExpectExec(
					`INSERT INTO products 
					\(name, description, price, available, stock, status, image_urls\) 
					VALUES \(\?, \?, \?, \?, \?, \?, \?\) `,
				).
					WithArgs(
						"Producto con imágenes",
						"Descripción del producto",
						25.50,
						true,
						uint64(10),
						"active",
						sqlmock.AnyArg(),
					).
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Mock history creation
				mock.ExpectExec(
					`INSERT INTO products_history 
					\(id_product, name, description, price, available, stock, status, image_urls, modified_by, action\) 
					VALUES \(\?, \?, \?, \?, \?, \?, \?, \?, \?, \?\)`,
				).
					WithArgs(
						uint64(1),
						"Producto con imágenes",
						"Descripción del producto",
						25.50,
						true,
						uint64(10),
						"active",
						sqlmock.AnyArg(),
						uint64(1),
						"create",
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "HAPPY PATH: Create product without images",
			productData: map[string]string{
				"name":        "Producto sin imágenes",
				"description": "Descripción del producto",
				"price":       "15.00",
				"available":   "true",
				"stock":       "5",
				"status":      "active",
			},
			imageFiles:     []string{},
			expectedStatus: http.StatusOK,
			expectedError:  false,
			setupMock: func() {
				// Mock product creation
				mock.ExpectExec(
					`INSERT INTO products 
					\(name, description, price, available, stock, status, image_urls\) 
					VALUES \(\?, \?, \?, \?, \?, \?, \?\) `,
				).
					WithArgs(
						"Producto sin imágenes",
						"Descripción del producto",
						15.00,
						true,
						uint64(5),
						"active",
						"[]",
					).
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Mock history creation
				mock.ExpectExec(
					`INSERT INTO products_history 
					\(id_product, name, description, price, available, stock, status, image_urls, modified_by, action\) 
					VALUES \(\?, \?, \?, \?, \?, \?, \?, \?, \?, \?\)`,
				).
					WithArgs(
						uint64(1),
						"Producto sin imágenes",
						"Descripción del producto",
						15.00,
						true,
						uint64(5),
						"active",
						"[]",
						uint64(1),
						"create",
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "SAD PATH: Invalid product data",
			productData: map[string]string{
				"name":        "", // Empty name should fail
				"description": "Descripción del producto",
				"price":       "25.50",
				"available":   "true",
				"stock":       "10",
				"status":      "active",
			},
			imageFiles:     []string{},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
			setupMock: func() {
				// No mock expectations for error case
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock expectations
			tt.setupMock()

			// Create multipart form data
			var b bytes.Buffer
			w := multipart.NewWriter(&b)

			// Add product data
			for key, value := range tt.productData {
				fw, err := w.CreateFormField(key)
				require.NoError(t, err)
				_, err = fw.Write([]byte(value))
				require.NoError(t, err)
			}

			// Add image files
			for _, filename := range tt.imageFiles {
				// Create test file
				testFile := filepath.Join(testDir, filename)
				err := os.WriteFile(testFile, []byte("fake image content"), 0644)
				require.NoError(t, err)

				// Add to form
				fw, err := w.CreateFormFile("images", filename)
				require.NoError(t, err)
				file, err := os.Open(testFile)
				require.NoError(t, err)
				_, err = io.Copy(fw, file)
				require.NoError(t, err)
				file.Close()
			}

			w.Close()

			// Create request
			req := httptest.NewRequest("POST", "/products", &b)
			req.Header.Set("Content-Type", w.FormDataContentType())

			// Add user context
			claims := map[string]interface{}{
				"user_id": float64(1),
			}
			ctx := context.WithValue(req.Context(), middleware.UserClaimsKey, claims)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if !tt.expectedError {
				var response map[string]string
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "Product created successfully", response["message"])
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestProductHandler_GetAllProductsWithImages(t *testing.T) {
	// Setup test directory
	testDir := "test_uploads"
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Setting up mock
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &productsRepository.ProductRepository{DB: db}
	imageService := imagesService.New(testDir)
	handler := &ProductHandler{
		Repo:         repo,
		ImageService: imageService,
	}

	// Setup the router
	router := mux.NewRouter()
	router.HandleFunc("/products", handler.GetAllProducts).Methods("GET")

	tests := []struct {
		name           string
		mockRows       *sqlmock.Rows
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "HAPPY PATH: Get products with images",
			mockRows: sqlmock.NewRows([]string{
				"id_product",
				"name",
				"description",
				"price",
				"available",
				"stock",
				"status",
				"image_urls",
				"created_on",
			}).AddRow(
				"1",
				"Producto con imagen",
				"Descripción",
				25.50,
				true,
				10,
				"active",
				`["/uploads/products/1/main.jpg"]`,
				nil,
			).AddRow(
				"2",
				"Producto sin imagen",
				"Descripción",
				15.00,
				true,
				5,
				"active",
				`[]`,
				nil,
			),
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockRows != nil {
				mock.ExpectQuery("SELECT id_product, name, description, price, available, stock, status, image_urls, created_on FROM products").
					WillReturnRows(tt.mockRows)
			}

			req := httptest.NewRequest("GET", "/products", nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if !tt.expectedError {
				var response []map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 2)

				// Check first product has images
				imageURLs, ok := response[0]["image_urls"].([]interface{})
				require.True(t, ok)
				assert.Len(t, imageURLs, 1)
				assert.Equal(t, "/uploads/products/1/main.jpg", imageURLs[0])

				// Check second product has no images
				imageURLs2, ok := response[1]["image_urls"].([]interface{})
				require.True(t, ok)
				assert.Len(t, imageURLs2, 0)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Helper function to create a multipart form with files
func createMultipartForm(productData map[string]string, imageFiles []string) (*bytes.Buffer, string, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add product data
	for key, value := range productData {
		fw, err := w.CreateFormField(key)
		if err != nil {
			return nil, "", err
		}
		_, err = fw.Write([]byte(value))
		if err != nil {
			return nil, "", err
		}
	}

	// Add image files
	for _, filename := range imageFiles {
		fw, err := w.CreateFormFile("images", filename)
		if err != nil {
			return nil, "", err
		}
		_, err = fw.Write([]byte("fake image content"))
		if err != nil {
			return nil, "", err
		}
	}

	err := w.Close()
	if err != nil {
		return nil, "", err
	}

	return &b, w.FormDataContentType(), nil
}
