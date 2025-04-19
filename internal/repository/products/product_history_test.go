package products

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	pModel "github.com/radamesvaz/bakery-app/model/products"
	"github.com/stretchr/testify/assert"
)

func TestProductRepository_CreateProductHistory(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error setting up the mock: %v", err)
	}

	defer db.Close()

	repo := &ProductRepository{DB: db}

	tests := []struct {
		name          string
		payload       pModel.ProductHistory
		mockError     error
		expectedError bool
		errorStatus   int
	}{
		{
			name: "HAPPY PATH: Creating a product history",
			payload: pModel.ProductHistory{
				IDProduct:   1,
				Name:        "Producto prueba test OK",
				Description: "Esta es la descripcion del producto de prueba",
				Price:       20.3,
				Available:   true,
				Status:      pModel.StatusActive,
				ModifiedBy:  1,
				Action:      pModel.ActionCreate,
			},
			mockError:     nil,
			expectedError: false,
		},
		{
			name: "HAPPY PATH: Updating a product history",
			payload: pModel.ProductHistory{
				IDProduct:   1,
				Name:        "Producto prueba test OK",
				Description: "Esta es la descripcion del producto de prueba",
				Price:       20.3,
				Available:   true,
				Status:      pModel.StatusInactive,
				ModifiedBy:  1,
				Action:      pModel.ActionUpdate,
			},
			mockError:     nil,
			expectedError: false,
		},
		{
			name: "HAPPY PATH: Deleting a product history",
			payload: pModel.ProductHistory{
				IDProduct:   1,
				Name:        "Producto prueba test OK",
				Description: "Esta es la descripcion del producto de prueba",
				Price:       20.3,
				Available:   true,
				Status:      pModel.StatusDeleted,
				ModifiedBy:  1,
				Action:      pModel.ActionDelete,
			},
			mockError:     nil,
			expectedError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectExec(
					regexp.QuoteMeta(
						`INSERT INTO products_history (
							id, 
							name, 
							description, 
							price, 
							available, 
							status, 
							modified_by, 
							action
							) 
							VALUES (
							?,
							?, 
							?, 
							?, 
							?, 
							?, 
							?, 
							?)`,
					),
				).
					WithArgs(
						tt.payload.IDProduct,
						tt.payload.Name,
						tt.payload.Description,
						tt.payload.Price,
						tt.payload.Available,
						tt.payload.Status,
						tt.payload.ModifiedBy,
						tt.payload.Action,
					).
					WillReturnResult(sqlmock.NewResult(0, 0))
			} else {
				mock.ExpectExec(
					regexp.QuoteMeta(
						`INSERT INTO products_history (
							id, 
							name, 
							description, 
							price, 
							available, 
							status, 
							modified_by, 
							action
							) 
							VALUES (
							?,
							?, 
							?, 
							?, 
							?, 
							?, 
							?, 
							?)`,
					),
				).
					WithArgs(
						tt.payload.IDProduct,
						tt.payload.Name,
						tt.payload.Description,
						tt.payload.Price,
						tt.payload.Available,
						tt.payload.Status,
						tt.payload.ModifiedBy,
						tt.payload.Action,
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			}

			err := repo.CreateProductHistory(
				tt.payload.IDProduct,
				tt.payload.Name,
				tt.payload.Description,
				tt.payload.Price,
				tt.payload.Available,
				tt.payload.Status,
				tt.payload.ModifiedBy,
				tt.payload.Action,
			)
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
