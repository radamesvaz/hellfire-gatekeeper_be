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
				"image_urls",
				"thumbnail_url",
				"created_on",
			}),
			mockError: nil,
			expected:  nil,
		},
		{
			name: "HAPPY PATH: getting all products",
			mockRows: sqlmock.NewRows([]string{
				"id_product",
				"tenant_id",
				"name",
				"description",
				"price",
				"available",
				"stock",
				"status",
				"image_urls",
				"thumbnail_url",
				"created_on",
			}).AddRow(
				"1",
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				5,
				"active",
				"[]",
				sql.NullString{Valid: false},
				createdOn,
			).AddRow(
				"2",
				"1",
				"Suspiros",
				"Suspiros para fiesta desc test",
				10,
				false,
				0,
				"inactive",
				"[]",
				sql.NullString{Valid: false},
				createdOn,
			),
			mockError: nil,
			expected: []pModel.Product{
				{
					ID:           1,
					TenantID:     1,
					Name:         "Torta de chocolate test",
					Description:  "Test descripcion de la torta test",
					Price:        30,
					Available:    true,
					Stock:        5,
					Status:       "active",
					ImageURLs:    []string{},
					ThumbnailURL: "",
					CreatedOn:    createdOn,
				},
				{
					ID:           2,
					TenantID:     1,
					Name:         "Suspiros",
					Description:  "Suspiros para fiesta desc test",
					Price:        10,
					Available:    false,
					Stock:        0,
					Status:       "inactive",
					ImageURLs:    []string{},
					ThumbnailURL: "",
					CreatedOn:    createdOn,
				},
			},
		},
	}
	const tenantID = uint64(1)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockRows != nil {
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT id_product, tenant_id, name, description, price, available, stock, status, image_urls, thumbnail_url, created_on FROM products WHERE tenant_id = $1"),
				).
					WithArgs(tenantID).
					WillReturnRows(tt.mockRows)
			} else {
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT id_product, tenant_id, name, description, price, available, stock, status, image_urls, thumbnail_url, created_on FROM products WHERE tenant_id = $1"),
				).
					WithArgs(tenantID).
					WillReturnError(tt.mockError)
			}

			products, err := repo.GetAllProducts(context.Background(), tenantID)

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
				"tenant_id",
				"name",
				"description",
				"price",
				"available",
				"stock",
				"status",
				"image_urls",
				"thumbnail_url",
				"created_on",
			}).AddRow(
				"1",
				"1",
				"Torta de chocolate test",
				"Test descripcion de la torta test",
				30,
				true,
				5,
				"active",
				"[]",
				sql.NullString{Valid: false},
				createdOn,
			),
			mockError: nil,
			expected: pModel.Product{
				ID:           1,
				TenantID:     1,
				Name:         "Torta de chocolate test",
				Description:  "Test descripcion de la torta test",
				Price:        30,
				Available:    true,
				Stock:        5,
				Status:       "active",
				ImageURLs:    []string{},
				ThumbnailURL: "",
				CreatedOn:    createdOn,
			},
			idProductForLookup: 1,
		},
		{
			name: "SAD PATH: getting a product with nonexisting id",
			mockRows: sqlmock.NewRows([]string{
				"id_product",
				"tenant_id",
				"name",
				"description",
				"price",
				"available",
				"stock",
				"status",
				"image_urls",
				"thumbnail_url",
				"created_on",
			}),
			mockError:          errors.ErrProductNotFound,
			errorStatus:        404,
			expected:           pModel.Product{},
			idProductForLookup: 9999,
			expectedError:      true,
		},
	}
	const tenantID2 = uint64(1)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT id_product, tenant_id, name, description, price, available, stock, status, image_urls, thumbnail_url, created_on FROM products WHERE tenant_id = $1 AND id_product = $2"),
				).
					WithArgs(tenantID2, tt.idProductForLookup).
					WillReturnError(sql.ErrNoRows)
			} else {
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT id_product, tenant_id, name, description, price, available, stock, status, image_urls, thumbnail_url, created_on FROM products WHERE tenant_id = $1 AND id_product = $2"),
				).
					WithArgs(tenantID2, tt.idProductForLookup).
					WillReturnRows(tt.mockRows)
			}

			product, err := repo.GetProductByID(context.Background(), tenantID2, tt.idProductForLookup)
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
				TenantID:     1,
				Name:         "Producto prueba test OK",
				Description:  "Esta es la descripcion del producto de prueba",
				Price:        20.3,
				Available:    true,
				Stock:        66,
				Status:       pModel.StatusActive,
				ImageURLs:    []string{},
				ThumbnailURL: "",
			},
			mockError: nil,
			expected: pModel.Product{
				ID:           1,
				TenantID:     1,
				Name:         "Producto prueba test OK",
				Description:  "Esta es la descripcion del producto de prueba",
				Price:        20.3,
				Available:    true,
				Stock:        66,
				Status:       pModel.StatusActive,
				ImageURLs:    []string{},
				ThumbnailURL: "",
			},
			expectedError: false,
		},
		{
			name: "SAD PATH: Creating a product without a name",
			payload: pModel.Product{
				Name:         "",
				Description:  "Esta es la descripcion del producto de prueba",
				Price:        20.3,
				Available:    true,
				ImageURLs:    []string{},
				ThumbnailURL: "",
			},
			expectedError: true,
			mockError:     errors.ErrCreatingProduct,
			errorStatus:   400,
			expected:      pModel.Product{},
		},
		{
			name: "SAD PATH: Creating a product without a description",
			payload: pModel.Product{
				Name:         "Name",
				Description:  "",
				Price:        20.3,
				Available:    true,
				ImageURLs:    []string{},
				ThumbnailURL: "",
			},
			expectedError: true,
			mockError:     errors.ErrCreatingProduct,
			errorStatus:   400,
			expected:      pModel.Product{},
		},
		{
			name: "SAD PATH: Creating a product without a price",
			payload: pModel.Product{
				Name:         "Name",
				Description:  "Esta es la descripcion del producto de prueba",
				Price:        0,
				Available:    true,
				ImageURLs:    []string{},
				ThumbnailURL: "",
			},
			expectedError: true,
			mockError:     errors.ErrCreatingProduct,
			errorStatus:   400,
			expected:      pModel.Product{},
		},
	}
	const tenantID = uint64(1)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.expectedError {
				mock.ExpectQuery(
					regexp.QuoteMeta(
						`INSERT INTO products 
		(tenant_id, name, description, price, available, stock, status, image_urls, thumbnail_url) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id_product`,
					),
				).
					WithArgs(
						tenantID,
						tt.payload.Name,
						tt.payload.Description,
						tt.payload.Price,
						tt.payload.Available,
						tt.expected.Stock,
						tt.payload.Status,
						"[]", // JSON marshaled empty array for image_urls
						nil,
					).
					WillReturnRows(sqlmock.NewRows([]string{"id_product"}).AddRow(1))
			}

			product, err := repo.CreateProduct(context.Background(), tenantID, tt.payload)
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
	const tenantID3 = uint64(1)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE products SET status = $1 WHERE tenant_id = $2 AND id_product = $3"),
				).
					WithArgs(tt.status, tenantID3, tt.idProductForUpdate).
					WillReturnResult(sqlmock.NewResult(0, 0))
			} else {
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE products SET status = $1 WHERE tenant_id = $2 AND id_product = $3"),
				).
					WithArgs(tt.status, tenantID3, tt.idProductForUpdate).
					WillReturnResult(sqlmock.NewResult(0, 1))
			}

			err := repo.UpdateProductStatus(context.Background(), tenantID3, tt.idProductForUpdate, tt.status)
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
	const tenantIDInvalid = uint64(1)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateProductStatus(context.Background(), tenantIDInvalid, tt.idProductForUpdate, tt.status)
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
				ID:           1,
				Name:         "Updated name",
				Description:  "Updated description",
				Price:        50,
				Available:    false,
				Stock:        5,
				Status:       pModel.StatusInactive,
				ThumbnailURL: "",
			},
		},
	}
	const tenantID4 = uint64(1)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE products SET name = $1, description = $2, price = $3, available = $4, stock = $5, status = $6, thumbnail_url = $7 WHERE tenant_id = $8 AND id_product = $9"),
				).
					WithArgs(tt.payload.Name, tt.payload.Description, tt.payload.Price, tt.payload.Available, tt.payload.Stock, tt.payload.Status, nil, tenantID4, tt.payload.ID).
					WillReturnResult(sqlmock.NewResult(0, 0))
			} else {
				mock.ExpectExec(
					regexp.QuoteMeta("UPDATE products SET name = $1, description = $2, price = $3, available = $4, stock = $5, status = $6, thumbnail_url = $7 WHERE tenant_id = $8 AND id_product = $9"),
				).
					WithArgs(tt.payload.Name, tt.payload.Description, tt.payload.Price, tt.payload.Available, tt.payload.Stock, tt.payload.Status, nil, tenantID4, tt.payload.ID).
					WillReturnResult(sqlmock.NewResult(0, 1))
			}

			err := repo.UpdateProduct(context.Background(), tenantID4, tt.payload)
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestProductRepository_UpdateProductImages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &ProductRepository{DB: db}

	const tenantID5 = uint64(1)
	mock.ExpectExec(
		regexp.QuoteMeta("UPDATE products SET image_urls = $1, thumbnail_url = $2 WHERE tenant_id = $3 AND id_product = $4"),
	).
		WithArgs(`["a","b"]`, "a", tenantID5, uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateProductImages(context.Background(), tenantID5, 1, []string{"a", "b"}, "a")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_UpdateProductThumbnail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &ProductRepository{DB: db}
	const tenantID6 = uint64(1)

	mock.ExpectExec(
		regexp.QuoteMeta("UPDATE products SET thumbnail_url = $1 WHERE tenant_id = $2 AND id_product = $3"),
	).
		WithArgs("a", tenantID6, uint64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateProductThumbnail(context.Background(), tenantID6, 1, "a")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_PrependImageAndSetThumbnail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &ProductRepository{DB: db}
	const tenantID7 = uint64(1)
	const productID = uint64(2)
	newURL := "/uploads/products/2/thumbnails/thumbnail.jpg"

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT image_urls FROM products WHERE tenant_id = $1 AND id_product = $2 FOR UPDATE")).
		WithArgs(tenantID7, productID).
		WillReturnRows(sqlmock.NewRows([]string{"image_urls"}).AddRow(`["/uploads/products/2/main.jpg","/uploads/products/2/gallery.jpg"]`))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE products SET image_urls = $1, thumbnail_url = $2 WHERE tenant_id = $3 AND id_product = $4")).
		WithArgs(`["/uploads/products/2/thumbnails/thumbnail.jpg","/uploads/products/2/main.jpg","/uploads/products/2/gallery.jpg"]`, newURL, tenantID7, productID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	updatedURLs, err := repo.PrependImageAndSetThumbnail(context.Background(), tenantID7, productID, newURL)
	require.NoError(t, err)
	assert.Equal(t, []string{newURL, "/uploads/products/2/main.jpg", "/uploads/products/2/gallery.jpg"}, updatedURLs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepo_GetProductsByIDs(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &ProductRepository{DB: db}

	const tenantID8 = uint64(1)
	ids := []uint64{1, 2}
	rows := sqlmock.NewRows([]string{"id_product", "tenant_id", "name", "price", "stock"}).
		AddRow(1, tenantID8, "Torta Chocolate", 12.5, 5).
		AddRow(2, tenantID8, "Torta Vainilla", 10.0, 3)

	query := "SELECT id_product, tenant_id, name, price, stock FROM products WHERE tenant_id = $1 AND id_product IN ($2,$3)"

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(tenantID8, 1, 2).
		WillReturnRows(rows)

	products, err := repo.GetProductsByIDs(context.Background(), tenantID8, ids)
	assert.NoError(t, err)
	assert.Len(t, products, 2)
	assert.Equal(t, "Torta Chocolate", products[0].Name)
	assert.Equal(t, uint64(2), products[1].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_DecrementProductStockTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &ProductRepository{DB: db}
	ctx := context.Background()

	const tenantID9 = uint64(1)

	t.Run("success_decrements_stock", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta("UPDATE products SET stock = stock - $1 WHERE tenant_id = $2 AND id_product = $3 AND stock >= $1")).
			WithArgs(3, tenantID9, 1).
			WillReturnResult(sqlmock.NewResult(0, 1))

		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		rows, err := repo.DecrementProductStockTx(ctx, tx, tenantID9, 1, 3)
		require.NoError(t, err)
		assert.Equal(t, int64(1), rows)
		_ = tx.Rollback()
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("zero_rows_when_insufficient_stock", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta("UPDATE products SET stock = stock - $1 WHERE tenant_id = $2 AND id_product = $3 AND stock >= $1")).
			WithArgs(10, tenantID9, 1).
			WillReturnResult(sqlmock.NewResult(0, 0))

		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		rows, err := repo.DecrementProductStockTx(ctx, tx, tenantID9, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(0), rows)
		_ = tx.Rollback()
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("zero_quantity_returns_success", func(t *testing.T) {
		mock.ExpectBegin()
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		rows, err := repo.DecrementProductStockTx(ctx, tx, tenantID9, 1, 0)
		require.NoError(t, err)
		assert.Equal(t, int64(1), rows)
		_ = tx.Rollback()
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// Validates the error to be of *HTTPError type, have the correct status and message
func assertHTTPError(t *testing.T, err error, expectedStatus int, expectedMessage string) {
	httpErr, ok := err.(*errors.HTTPError)

	if assert.True(t, ok, "The error is not HTTP type") {
		assert.Equal(t, expectedStatus, httpErr.StatusCode, "The code status is not as expected")
		assert.EqualError(t, httpErr.Err, expectedMessage, "Mismatch on error message")
	}
}
