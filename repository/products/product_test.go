package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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

	repo := &ProductRepository{DB: db}

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
