package products

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/radamesvaz/bakery-app/internal/errors"
	pModel "github.com/radamesvaz/bakery-app/model/products"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductRepository_UpdateProductImages(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &ProductRepository{DB: db}

	tests := []struct {
		name          string
		productID     uint64
		imageURLs     []string
		mockError     error
		expectedError bool
		errorStatus   int
	}{
		{
			name:      "HAPPY PATH: Update product with images",
			productID: 1,
			imageURLs: []string{
				"/uploads/products/1/main.jpg",
				"/uploads/products/1/gallery_1.jpg",
			},
			expectedError: false,
		},
		{
			name:          "HAPPY PATH: Update product with empty images",
			productID:     1,
			imageURLs:     []string{},
			expectedError: false,
		},
		{
			name:          "SAD PATH: Product not found",
			productID:     999,
			imageURLs:     []string{"/uploads/products/999/main.jpg"},
			expectedError: true,
			mockError:     errors.ErrProductNotFound,
			errorStatus:   404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert imageURLs to JSON
			imageURLsJSON, err := json.Marshal(tt.imageURLs)
			require.NoError(t, err)

			if tt.expectedError {
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE products SET image_urls = ? WHERE id_product = ?"),
				).
					WithArgs(string(imageURLsJSON), tt.productID).
					WillReturnResult(sqlmock.NewResult(0, 0))
			} else {
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE products SET image_urls = ? WHERE id_product = ?"),
				).
					WithArgs(string(imageURLsJSON), tt.productID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			}

			err = repo.UpdateProductImages(context.Background(), tt.productID, tt.imageURLs)
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestProductRepository_CreateProductWithImages(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &ProductRepository{DB: db}

	tests := []struct {
		name          string
		product       pModel.Product
		imageURLs     []string
		mockError     error
		expected      pModel.Product
		expectedError bool
		errorStatus   int
	}{
		{
			name: "HAPPY PATH: Creating a product with images",
			product: pModel.Product{
				Name:        "Producto con imágenes",
				Description: "Descripción del producto",
				Price:       25.50,
				Available:   true,
				Stock:       10,
				Status:      pModel.StatusActive,
			},
			imageURLs: []string{
				"/uploads/products/1/main.jpg",
				"/uploads/products/1/gallery_1.jpg",
			},
			expected: pModel.Product{
				ID:          1,
				Name:        "Producto con imágenes",
				Description: "Descripción del producto",
				Price:       25.50,
				Available:   true,
				Stock:       10,
				Status:      pModel.StatusActive,
				ImageURLs: []string{
					"/uploads/products/1/main.jpg",
					"/uploads/products/1/gallery_1.jpg",
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert imageURLs to JSON
			imageURLsJSON, err := json.Marshal(tt.imageURLs)
			require.NoError(t, err)

			// Mock the INSERT query
			mock.ExpectExec(
				regexp.QuoteMeta(
					`INSERT INTO products 
					(name, description, price, available, stock, status, image_urls) 
					VALUES (?, ?, ?, ?, ?, ?, ?) `,
				),
			).
				WithArgs(
					tt.product.Name,
					tt.product.Description,
					tt.product.Price,
					tt.product.Available,
					tt.product.Stock,
					tt.product.Status,
					string(imageURLsJSON),
				).
				WillReturnResult(sqlmock.NewResult(1, 1))

			product, err := repo.CreateProductWithImages(context.Background(), tt.product, tt.imageURLs)
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Name, product.Name)
				assert.Equal(t, tt.expected.ImageURLs, product.ImageURLs)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestProductRepository_GetAllProductsWithImages(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &ProductRepository{DB: db}

	tests := []struct {
		name          string
		mockRows      *sqlmock.Rows
		mockError     error
		expected      []pModel.Product
		expectedError bool
	}{
		{
			name: "HAPPY PATH: Getting products with images",
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
			expected: []pModel.Product{
				{
					ID:          1,
					Name:        "Producto con imagen",
					Description: "Descripción",
					Price:       25.50,
					Available:   true,
					Stock:       10,
					Status:      "active",
					ImageURLs:   []string{"/uploads/products/1/main.jpg"},
				},
				{
					ID:          2,
					Name:        "Producto sin imagen",
					Description: "Descripción",
					Price:       15.00,
					Available:   true,
					Stock:       5,
					Status:      "active",
					ImageURLs:   []string{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockRows != nil {
				mock.ExpectQuery("SELECT id_product, name, description, price, available, stock, status, image_urls, created_on FROM products").
					WillReturnRows(tt.mockRows)
			} else {
				mock.ExpectQuery("SELECT id_product, name, description, price, available, stock, status, image_urls, created_on FROM products").
					WillReturnError(tt.mockError)
			}

			products, err := repo.GetAllProducts(context.Background())
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(tt.expected), len(products))
				if len(products) > 0 {
					assert.Equal(t, tt.expected[0].ImageURLs, products[0].ImageURLs)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
