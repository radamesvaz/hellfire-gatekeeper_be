package products

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/radamesvaz/bakery-app/internal/errors"
	pModel "github.com/radamesvaz/bakery-app/model/products"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				"stock",
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
				"stock",
				"stauts",
				"created_on",
			}).AddRow(
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				5,
				"active",
				createdOn,
			).AddRow(
				"2",
				"Suspiros",
				"Suspiros para fiesta desc test",
				10,
				false,
				0,
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
					Stock:       5,
					Status:      "active",
					CreatedOn:   createdOn,
				},
				{
					ID:          2,
					Name:        "Suspiros",
					Description: "Suspiros para fiesta desc test",
					Price:       10,
					Available:   false,
					Stock:       0,
					Status:      "inactive",
					CreatedOn:   createdOn,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockRows != nil {
				mock.ExpectQuery("SELECT id_product, name, description, price, available, stock, status, created_on FROM products").
					WillReturnRows(tt.mockRows)
			} else {
				mock.ExpectQuery("SELECT id_product, name, description, price, available, stock, status, created_on FROM products").
					WillReturnError(tt.mockError)
			}

			products, err := repo.GetAllProducts(context.Background())

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
				"stock",
				"stauts",
				"created_on",
			}).AddRow(
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				5,
				"active",
				createdOn,
			).AddRow(
				"2",
				"Suspiros",
				"Suspiros para fiesta desc test",
				10,
				false,
				0,
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
				Stock:       5,
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
				"stock",
				"status",
				"created_on",
			}).AddRow(
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				1,
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
					"SELECT id_product, name, description, price, available, stock, status, created_on FROM products WHERE id_product = ?",
				).
					WithArgs(tt.idProductForLookup).
					WillReturnError(sql.ErrNoRows)
			} else {
				mock.ExpectQuery(
					"SELECT id_product, name, description, price, available, stock, status, created_on FROM products WHERE id_product = ?",
				).
					WithArgs(tt.idProductForLookup).
					WillReturnRows(tt.mockRows)
			}

			product, err := repo.GetProductByID(context.Background(), tt.idProductForLookup)
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
				Stock:       66,
				Status:      pModel.StatusActive,
			},
			mockError: nil,
			expected: pModel.Product{
				Name:        "Producto prueba test OK",
				Description: "Esta es la descripcion del producto de prueba",
				Price:       20.3,
				Available:   true,
				Stock:       66,
				Status:      pModel.StatusActive,
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
				mock.ExpectExec(
					regexp.QuoteMeta(
						`INSERT INTO products 
						(name, description, price, available, stock, status) 
						VALUES (?, ?, ?, ?, ?, ?) `,
					),
				).
					WithArgs(
						tt.payload.Name,
						tt.payload.Description,
						tt.payload.Price,
						tt.payload.Available,
						tt.expected.Stock,
						tt.payload.Status,
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			}

			product, err := repo.CreateProduct(context.Background(), tt.payload)
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
				"stock",
				"status",
				"created_on",
			}).AddRow(
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				3,
				"active",
				createdOn,
			).AddRow(
				"2",
				"Suspiros",
				"Suspiros para fiesta desc test",
				10,
				false,
				0,
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
				"stock",
				"created_on",
			}).AddRow(
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				5,
				createdOn,
			).AddRow(
				"2",
				"Suspiros",
				"Suspiros para fiesta desc test",
				10,
				false,
				0,
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

			err := repo.UpdateProductStatus(context.Background(), tt.idProductForUpdate, tt.status)
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
			err := repo.UpdateProductStatus(context.Background(), tt.idProductForUpdate, tt.status)
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestProductRepository_UpdateProduct(t *testing.T) {
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
		mockRows      *sqlmock.Rows
		payload       pModel.Product
		expectedError bool
		mockError     error
		errorStatus   int
	}{
		{
			name:          "HAPPY PATH: updating product with ID: 1",
			expectedError: false,
			mockRows: sqlmock.NewRows([]string{
				"id_product",
				"name",
				"description",
				"price",
				"available",
				"stock",
				"status",
				"created_on",
			}).AddRow(
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				5,
				"active",
				createdOn,
			).AddRow(
				"2",
				"Suspiros",
				"Suspiros para fiesta desc test",
				10,
				false,
				0,
				"deleted",
				createdOn,
			),
			mockError: nil,
			payload: pModel.Product{
				ID:          1,
				Name:        "Updated name",
				Description: "Updated description",
				Price:       50,
				Available:   false,
				Stock:       5,
				Status:      pModel.StatusInactive,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE products SET name = ?, description = ?, price = ?, available = ?, stock = ?, status = ? where id_product = ?"),
				).
					WithArgs(tt.payload.Name, tt.payload.Description, tt.payload.Price, tt.payload.Available, tt.payload.Stock, tt.payload.Status, tt.payload.ID).
					WillReturnResult(sqlmock.NewResult(0, 0))
			} else {
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE products SET name = ?, description = ?, price = ?, available = ?, stock = ?, status = ? where id_product = ?"),
				).
					WithArgs(tt.payload.Name, tt.payload.Description, tt.payload.Price, tt.payload.Available, tt.payload.Stock, tt.payload.Status, tt.payload.ID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			}

			err := repo.UpdateProduct(context.Background(), tt.payload)
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestProductRepo_GetProductsByIDs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &ProductRepository{DB: db}

	ids := []uint64{1, 2}
	rows := sqlmock.NewRows([]string{"id", "name", "price", "stock"}).
		AddRow(1, "Torta Chocolate", 12.5, 5).
		AddRow(2, "Torta Vainilla", 10.0, 3)

	query := "SELECT id, name, price, stock FROM products WHERE id IN (?,?)"

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(1, 2).
		WillReturnRows(rows)

	products, err := repo.GetProductsByIDs(context.Background(), ids)
	assert.NoError(t, err)
	assert.Len(t, products, 2)
	assert.Equal(t, "Torta Chocolate", products[0].Name)
	assert.Equal(t, uint64(2), products[1].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// Validates the error to be of *HTTPError type, have the correct status and message
func assertHTTPError(t *testing.T, err error, expectedStatus int, expectedMessage string) {
	httpErr, ok := err.(*errors.HTTPError)

	if assert.True(t, ok, "The error is not HTTP type") {
		assert.Equal(t, expectedStatus, httpErr.StatusCode, "The code status is not as expected")
		assert.EqualError(t, httpErr.Err, expectedMessage, "Mismatch on error message")
	}
}
