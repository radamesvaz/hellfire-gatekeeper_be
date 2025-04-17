package products

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/radamesvaz/bakery-app/internal/errors"
	pModel "github.com/radamesvaz/bakery-app/model/products"
	"github.com/stretchr/testify/assert"
)

func TestProductRepository_GetAllProducts(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error setting up the mock: %v", err)
	}

	defer db.Close()

	repo := &ProductRepository{
		DB: db,
	}

	createdOn := sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	tests := []struct {
		name          string
		mockRows      *sqlmock.Rows
		mockError     error
		expected      []pModel.Product
		expectedError bool
	}{
		{
			name: "HAPPY PATH: empty DB",
			mockRows: sqlmock.NewRows([]string{
				"id_product",
				"name",
				"description",
				"price",
				"available",
				"status",
				"created_on",
			}),
			mockError: nil,
			expected:  nil,
		},
		{
			name: "HAPPY PATH: getting all products",
			mockRows: sqlmock.NewRows([]string{
				"id_product",
				"name",
				"description",
				"price",
				"available",
				"stauts",
				"created_on",
			}).AddRow(
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				"active",
				createdOn,
			).AddRow(
				"2",
				"Suspiros",
				"Suspiros para fiesta desc test",
				10,
				false,
				"inactive",
				createdOn,
			),
			mockError: nil,
			expected: []pModel.Product{
				{
					ID:          1,
					Name:        "Torta de chocolate test",
					Description: "Test descripcion de la torta test",
					Price:       30,
					Available:   true,
					Status:      "active",
					CreatedOn:   createdOn,
				},
				{
					ID:          2,
					Name:        "Suspiros",
					Description: "Suspiros para fiesta desc test",
					Price:       10,
					Available:   false,
					Status:      "inactive",
					CreatedOn:   createdOn,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockRows != nil {
				mock.ExpectQuery("SELECT id_product, name, description, price, available, status, created_on FROM products").
					WillReturnRows(tt.mockRows)
			} else {
				mock.ExpectQuery("SELECT id_product, name, description, price, available, status, created_on FROM products").
					WillReturnError(tt.mockError)
			}

			products, err := repo.GetAll()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, products)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}

}

func TestProductRepository_GetProductByID(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error setting up the mock: %v", err)
	}

	defer db.Close()

	repo := &ProductRepository{DB: db}

	createdOn := sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	tests := []struct {
		name               string
		mockRows           *sqlmock.Rows
		mockError          error
		expected           pModel.Product
		expectedError      bool
		idProductForLookup uint64
		errorStatus        int
	}{
		{
			name: "HAPPY PATH: getting a product wiht id 1",
			mockRows: sqlmock.NewRows([]string{
				"id_product",
				"name",
				"description",
				"price",
				"available",
				"stauts",
				"created_on",
			}).AddRow(
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				"active",
				createdOn,
			).AddRow(
				"2",
				"Suspiros",
				"Suspiros para fiesta desc test",
				10,
				false,
				"deleted",
				createdOn,
			),
			mockError: nil,
			expected: pModel.Product{
				ID:          1,
				Name:        "Torta de chocolate test",
				Description: "Test descripcion de la torta test",
				Price:       30,
				Available:   true,
				Status:      "active",
				CreatedOn:   createdOn,
			},
			idProductForLookup: 1,
		},
		{
			name: "SAD PATH: getting a product with nonexisting id",
			mockRows: sqlmock.NewRows([]string{
				"id_product",
				"name",
				"description",
				"price",
				"available",
				"status",
				"created_on",
			}).AddRow(
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				"inactive",
				createdOn,
			),
			mockError:          errors.ErrProductNotFound,
			errorStatus:        404,
			expected:           pModel.Product{},
			idProductForLookup: 9999,
			expectedError:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectQuery(
					"SELECT id_product, name, description, price, available, status, created_on FROM products WHERE id_product = ?",
				).
					WithArgs(tt.idProductForLookup).
					WillReturnError(sql.ErrNoRows)
			} else {
				mock.ExpectQuery(
					"SELECT id_product, name, description, price, available, status, created_on FROM products WHERE id_product = ?",
				).
					WithArgs(tt.idProductForLookup).
					WillReturnRows(tt.mockRows)
			}

			product, err := repo.GetProductByID(tt.idProductForLookup)
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, product)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestProductRepository_CreateProduct(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error setting up the mock: %v", err)
	}

	defer db.Close()

	repo := &ProductRepository{DB: db}

	createdOn := sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	tests := []struct {
		name          string
		payload       pModel.Product
		mockError     error
		expected      pModel.Product
		expectedError bool
		errorStatus   int
	}{
		{
			name: "HAPPY PATH: Creating a product",
			payload: pModel.Product{
				Name:        "Producto prueba test OK",
				Description: "Esta es la descripcion del producto de prueba",
				Price:       20.3,
				Available:   true,
				Status:      "active",
			},
			mockError: nil,
			expected: pModel.Product{
				ID:          1,
				Name:        "Producto prueba test OK",
				Description: "Esta es la descripcion del producto de prueba",
				Price:       20.3,
				Available:   true,
				Status:      "active",
				CreatedOn:   createdOn,
			},
			expectedError: false,
		},
		{
			name: "SAD PATH: Creating a product without a name",
			payload: pModel.Product{
				Name:        "",
				Description: "Esta es la descripcion del producto de prueba",
				Price:       20.3,
				Available:   true,
			},
			expectedError: true,
			mockError:     errors.ErrCreatingProduct,
			errorStatus:   400,
			expected:      pModel.Product{},
		},
		{
			name: "SAD PATH: Creating a product without a description",
			payload: pModel.Product{
				Name:        "Name",
				Description: "",
				Price:       20.3,
				Available:   true,
			},
			expectedError: true,
			mockError:     errors.ErrCreatingProduct,
			errorStatus:   400,
			expected:      pModel.Product{},
		},
		{
			name: "SAD PATH: Creating a product without a price",
			payload: pModel.Product{
				Name:        "Name",
				Description: "Esta es la descripcion del producto de prueba",
				Price:       0,
				Available:   true,
			},
			expectedError: true,
			mockError:     errors.ErrCreatingProduct,
			errorStatus:   400,
			expected:      pModel.Product{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.expectedError {
				mock.ExpectQuery(
					regexp.QuoteMeta(
						`INSERT INTO products 
						(name, description, price, available, status) 
						VALUES (?, ?, ?, ?, ?) 
						RETURNING 
						id_product, 
						name,
						description, 
						price,
						available,
						status, 
						created_on`,
					),
				).
					WithArgs(
						tt.payload.Name,
						tt.payload.Description,
						tt.payload.Price,
						tt.payload.Available,
						tt.payload.Status,
					).
					WillReturnRows(sqlmock.NewRows([]string{
						"id_product", "name", "description", "pice", "available", "status", "created_on",
					}).AddRow(
						1,
						tt.payload.Name,
						tt.payload.Description,
						tt.payload.Price,
						tt.payload.Available,
						tt.payload.Status,
						createdOn.Time,
					))
			}

			product, err := repo.CreateProduct(
				tt.payload.Name,
				tt.payload.Description,
				tt.payload.Price,
				tt.payload.Available,
				tt.payload.Status,
			)
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, product)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestProductRepository_UpdateProductStatus(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error setting up the mock: %v", err)
	}

	defer db.Close()

	repo := &ProductRepository{DB: db}

	createdOn := sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	tests := []struct {
		name               string
		mockRows           *sqlmock.Rows
		idProductForUpdate uint64
		status             pModel.ProductStatus
		expectedError      bool
		mockError          error
		errorStatus        int
	}{
		{
			name:          "HAPPY PATH: deleting a product with ID: 1",
			expectedError: false,
			mockRows: sqlmock.NewRows([]string{
				"id_product",
				"name",
				"description",
				"price",
				"available",
				"status",
				"created_on",
			}).AddRow(
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				"active",
				createdOn,
			).AddRow(
				"2",
				"Suspiros",
				"Suspiros para fiesta desc test",
				10,
				false,
				"deleted",
				createdOn,
			),
			mockError:          nil,
			idProductForUpdate: 1,
			status:             pModel.StatusDeleted,
		},
		{
			name:          "SAD PATH: product ID not found",
			expectedError: true,
			errorStatus:   404,
			mockRows: sqlmock.NewRows([]string{
				"id_product",
				"name",
				"description",
				"price",
				"available",
				"created_on",
			}).AddRow(
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				createdOn,
			).AddRow(
				"2",
				"Suspiros",
				"Suspiros para fiesta desc test",
				10,
				false,
				createdOn,
			),
			mockError:          errors.ErrProductNotFound,
			idProductForUpdate: 99999,
			status:             pModel.StatusInactive,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE products SET status = ? where id_product = ?"),
				).
					WithArgs(tt.status, tt.idProductForUpdate).
					WillReturnResult(sqlmock.NewResult(0, 0))
			} else {
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE products SET status = ? where id_product = ?"),
				).
					WithArgs(tt.status, tt.idProductForUpdate).
					WillReturnResult(sqlmock.NewResult(0, 1))
			}

			err := repo.UpdateProductStatus(tt.idProductForUpdate, tt.status)
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestProductRepository_UpdateProductInvalidStatus(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error setting up the mock: %v", err)
	}

	defer db.Close()

	repo := &ProductRepository{DB: db}

	createdOn := sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	tests := []struct {
		name               string
		mockRows           *sqlmock.Rows
		idProductForUpdate uint64
		status             pModel.ProductStatus
		expectedError      bool
		mockError          error
		errorStatus        int
	}{
		{
			name:          "SAD PATH: Invalid product status",
			expectedError: true,
			errorStatus:   400,
			mockRows: sqlmock.NewRows([]string{
				"id_product",
				"name",
				"description",
				"price",
				"available",
				"created_on",
			}).AddRow(
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				createdOn,
			).AddRow(
				"2",
				"Suspiros",
				"Suspiros para fiesta desc test",
				10,
				false,
				createdOn,
			),
			mockError:          errors.ErrInvalidStatus,
			idProductForUpdate: 1,
			status:             pModel.ProductStatus("invalid"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateProductStatus(tt.idProductForUpdate, tt.status)
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Validates the error to be of *HTTPError type, have the correct status and message
func assertHTTPError(t *testing.T, err error, expectedStatus int, expectedMessage string) {
	httpErr, ok := err.(*errors.HTTPError)

	if assert.True(t, ok, "The error is not HTTP type") {
		assert.Equal(t, expectedStatus, httpErr.StatusCode, "The code status is not as expected")
		assert.EqualError(t, httpErr.Err, expectedMessage, "Mismatch on error message")
	}
}
