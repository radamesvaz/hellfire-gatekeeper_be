package products

import (
	"database/sql"
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
			mockError: nil,
			expected: []pModel.Product{
				{
					ID:          1,
					Name:        "Torta de chocolate test",
					Description: "Test descripcion de la torta test",
					Price:       30,
					Available:   true,
					CreatedOn:   createdOn,
				},
				{
					ID:          2,
					Name:        "Suspiros",
					Description: "Suspiros para fiesta desc test",
					Price:       10,
					Available:   false,
					CreatedOn:   createdOn,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockRows != nil {
				mock.ExpectQuery("SELECT id_product, name, description, price, available, created_on FROM products").
					WillReturnRows(tt.mockRows)
			} else {
				mock.ExpectQuery("SELECT id_product, name, description, price, available, created_on FROM products").
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
			mockError: nil,
			expected: pModel.Product{
				ID:          1,
				Name:        "Torta de chocolate test",
				Description: "Test descripcion de la torta test",
				Price:       30,
				Available:   true,
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
				"created_on",
			}).AddRow(
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				createdOn,
			),
			mockError:          errors.ErrProductNotFound,
			errorStatus:        401,
			expected:           pModel.Product{},
			idProductForLookup: 9999,
			expectedError:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectQuery(
					"SELECT id_product, name, description, price, available, created_on FROM products WHERE id_product = ?",
				).
					WithArgs(tt.idProductForLookup).
					WillReturnError(sql.ErrNoRows)
			} else {
				mock.ExpectQuery(
					"SELECT id_product, name, description, price, available, created_on FROM products WHERE id_product = ?",
				).
					WithArgs(tt.idProductForLookup).
					WillReturnRows(sqlmock.NewRows([]string{
						"id_product",
						"name",
						"description",
						"price",
						"available",
						"created_on",
					}).
						AddRow(
							tt.expected.ID,
							tt.expected.Name,
							tt.expected.Description,
							tt.expected.Price,
							tt.expected.Available,
							createdOn.Time,
						),
					)
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

// Validates the error to be of *HTTPError type, have the correct status and message
func assertHTTPError(t *testing.T, err error, expectedStatus int, expectedMessage string) {
	httpErr, ok := err.(*errors.HTTPError)

	if assert.True(t, ok, "The error is not HTTP type") {
		assert.Equal(t, expectedStatus, httpErr.StatusCode, "The code status is not as expected")
		assert.EqualError(t, httpErr.Err, expectedMessage, "Mismatch on error message")
	}
}
