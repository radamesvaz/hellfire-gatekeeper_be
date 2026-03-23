package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
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

func TestImageHandler_UploadProductThumbnail_Success(t *testing.T) {
	restore := disableCloudinaryForHandlerTests(t)
	defer restore()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	testDir := t.TempDir()
	handler := &ImageHandler{
		Repo:         &productsRepository.ProductRepository{DB: db},
		ImageService: imagesService.New(testDir),
	}

	tenantID := uint64(1)
	productID := uint64(10)
	userID := float64(77)

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT id_product, tenant_id, name, description, price, available, stock, status, image_urls, thumbnail_url, created_on FROM products WHERE tenant_id = $1 AND id_product = $2"),
	).WithArgs(tenantID, productID).WillReturnRows(
		sqlmock.NewRows([]string{
			"id_product", "tenant_id", "name", "description", "price", "available", "stock", "status", "image_urls", "thumbnail_url", "created_on",
		}).AddRow(
			productID, tenantID, "Cake", "desc", 10.5, true, 3, "active", `["/uploads/products/10/main.jpg"]`, "/uploads/products/10/main.jpg", sql.NullTime{},
		),
	)

	mock.ExpectBegin()
	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT image_urls FROM products WHERE tenant_id = $1 AND id_product = $2 FOR UPDATE"),
	).WithArgs(tenantID, productID).WillReturnRows(
		sqlmock.NewRows([]string{"image_urls"}).AddRow(`["/uploads/products/10/main.jpg"]`),
	)
	mock.ExpectExec(
		regexp.QuoteMeta("UPDATE products SET image_urls = $1, thumbnail_url = $2 WHERE tenant_id = $3 AND id_product = $4"),
	).WithArgs(
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		tenantID,
		productID,
	).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO products_history")).WithArgs(
		tenantID,
		productID,
		"Cake",
		"desc",
		10.5,
		true,
		uint64(3),
		"active",
		sqlmock.AnyArg(),
		sqlmock.AnyArg(),
		uint64(userID),
		"update",
	).WillReturnResult(sqlmock.NewResult(0, 1))

	req := newMultipartThumbnailRequest(t, "thumbnail", "thumb.jpg", "image/jpeg", []byte("fake image"))
	req = mux.SetURLVars(req, map[string]string{"id": "10"})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, middleware.UserClaimsKey, jwt.MapClaims{"user_id": userID})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.UploadProductThumbnail(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var got map[string]interface{}
	err = json.Unmarshal(rr.Body.Bytes(), &got)
	require.NoError(t, err)
	assert.Equal(t, "Thumbnail uploaded successfully", got["message"])
	assert.NotEmpty(t, got["thumbnail_url"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestImageHandler_UploadProductThumbnail_InvalidFileType(t *testing.T) {
	restore := disableCloudinaryForHandlerTests(t)
	defer restore()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handler := &ImageHandler{
		Repo:         &productsRepository.ProductRepository{DB: db},
		ImageService: imagesService.New(t.TempDir()),
	}

	tenantID := uint64(1)
	productID := uint64(10)
	userID := float64(77)

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT id_product, tenant_id, name, description, price, available, stock, status, image_urls, thumbnail_url, created_on FROM products WHERE tenant_id = $1 AND id_product = $2"),
	).WithArgs(tenantID, productID).WillReturnRows(
		sqlmock.NewRows([]string{
			"id_product", "tenant_id", "name", "description", "price", "available", "stock", "status", "image_urls", "thumbnail_url", "created_on",
		}).AddRow(
			productID, tenantID, "Cake", "desc", 10.5, true, 3, "active", `[]`, nil, sql.NullTime{},
		),
	)

	req := newMultipartThumbnailRequest(t, "thumbnail", "thumb.txt", "text/plain", []byte("plain text"))
	req = mux.SetURLVars(req, map[string]string{"id": "10"})
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, tenantID)
	ctx = context.WithValue(ctx, middleware.UserClaimsKey, jwt.MapClaims{"user_id": userID})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.UploadProductThumbnail(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid thumbnail type")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func newMultipartThumbnailRequest(t *testing.T, fieldName, filename, _ string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fieldName, filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/auth/products/10/thumbnail", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func disableCloudinaryForHandlerTests(t *testing.T) func() {
	t.Helper()
	origCloud := os.Getenv("CLOUDINARY_CLOUD_NAME")
	origKey := os.Getenv("CLOUDINARY_API_KEY")
	origSecret := os.Getenv("CLOUDINARY_API_SECRET")

	_ = os.Unsetenv("CLOUDINARY_CLOUD_NAME")
	_ = os.Unsetenv("CLOUDINARY_API_KEY")
	_ = os.Unsetenv("CLOUDINARY_API_SECRET")

	return func() {
		if origCloud != "" {
			_ = os.Setenv("CLOUDINARY_CLOUD_NAME", origCloud)
		}
		if origKey != "" {
			_ = os.Setenv("CLOUDINARY_API_KEY", origKey)
		}
		if origSecret != "" {
			_ = os.Setenv("CLOUDINARY_API_SECRET", origSecret)
		}
	}
}
