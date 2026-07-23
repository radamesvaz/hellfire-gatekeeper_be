package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	productsRepository "github.com/radamesvaz/bakery-app/internal/repository/products"
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductHandler_UpdateProduct_SoftDeleteCleansUpImages(t *testing.T) {
	restore := disableCloudinaryForHandlerTests(t)
	defer restore()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	testDir := t.TempDir()
	productID := uint64(10)
	productDir := filepath.Join(testDir, "products", "10")
	require.NoError(t, os.MkdirAll(productDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(productDir, "main.jpg"), []byte("img"), 0o644))

	handler := &ProductHandler{
		Repo:         &productsRepository.ProductRepository{DB: db},
		ImageService: imagesService.New(testDir),
	}

	tenantID := uint64(1)
	userID := float64(77)

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT id_product, tenant_id, name, description, price, track_inventory, stock, status, image_urls, thumbnail_url, created_on FROM products WHERE tenant_id = $1 AND id_product = $2"),
	).WithArgs(tenantID, productID).WillReturnRows(
		sqlmock.NewRows([]string{
			"id_product", "tenant_id", "name", "description", "price", "track_inventory", "stock", "status", "image_urls", "thumbnail_url", "created_on",
		}).AddRow(
			productID, tenantID, "Cake", "desc", 10.5, true, 3, "active", `["/uploads/products/10/main.jpg"]`, "/uploads/products/10/main.jpg", sql.NullTime{},
		),
	)

	mock.ExpectExec(
		regexp.QuoteMeta("UPDATE products SET name = $1, description = $2, price = $3, stock = $4, status = $5, track_inventory = $6, thumbnail_url = $7 WHERE tenant_id = $8 AND id_product = $9"),
	).WithArgs("Cake", "desc", 10.5, uint64(3), "deleted", true, "/uploads/products/10/main.jpg", tenantID, productID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO products_history")).WithArgs(
		tenantID,
		productID,
		"Cake",
		"desc",
		10.5,
		true,
		uint64(3),
		"deleted",
		sqlmock.AnyArg(),
		"/uploads/products/10/main.jpg",
		uint64(userID),
		"update",
	).WillReturnResult(sqlmock.NewResult(0, 1))

	payload := `{"name":"Cake","description":"desc","price":10.5,"stock":3,"status":"deleted","track_inventory":true}`
	req := httptest.NewRequest(http.MethodPut, "/auth/products/10", strings.NewReader(payload))
	req = mux.SetURLVars(req, map[string]string{"id": "10"})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, middleware.UserClaimsKey, jwt.MapClaims{"user_id": userID})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.UpdateProduct(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var got map[string]string
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, "Product updated successfully", got["message"])

	_, err = os.Stat(productDir)
	assert.True(t, os.IsNotExist(err), "product image directory should be removed on soft-delete via PUT")
	assert.NoError(t, mock.ExpectationsWereMet())
}
