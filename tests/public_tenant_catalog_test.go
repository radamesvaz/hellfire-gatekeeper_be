package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/handlers"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	repository "github.com/radamesvaz/bakery-app/internal/repository/products"
	tenantRepository "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicGetAllProductsByTenantSlug(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	productRepo := repository.ProductRepository{DB: db}
	tenantRepo := tenantRepository.Repository{DB: db}
	handler := handlers.ProductHandler{Repo: &productRepo}

	router := mux.NewRouter()
	tPublic := router.PathPrefix("/t/{tenant_slug}").Subrouter()
	tPublic.Use(middleware.TenantFromPathOrHeader(&tenantRepo))
	tPublic.HandleFunc("/products", handler.GetAllProducts).Methods("GET")

	req := httptest.NewRequest("GET", "/t/default/products", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var body struct {
		Items      []map[string]interface{} `json:"items"`
		NextCursor *string                  `json:"next_cursor"`
	}
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Len(t, body.Items, 2)
	assert.Nil(t, body.NextCursor)
	assert.Equal(t, "Suspiros", body.Items[0]["name"])
	assert.Equal(t, "Brownie Clásico", body.Items[1]["name"])
}

func TestPublicGetTenantBrandingBySlug(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	tenantRepo := tenantRepository.Repository{DB: db}
	handler := handlers.TenantHandler{Repo: &tenantRepo}

	router := mux.NewRouter()
	tPublic := router.PathPrefix("/t/{tenant_slug}").Subrouter()
	tPublic.Use(middleware.TenantFromPathOrHeader(&tenantRepo))
	tPublic.HandleFunc("/branding", handler.GetBranding).Methods("GET")

	req := httptest.NewRequest("GET", "/t/default/branding", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var body map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "default", body["tenant_slug"])
	assert.NotNil(t, body["tenant_id"])
	branding, ok := body["branding"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Default Tenant", branding["tenant_name"])
	assert.Contains(t, branding, "logo_url")
	assert.Contains(t, branding, "primary_color")
	assert.Contains(t, branding, "secondary_color")
	assert.Contains(t, branding, "accent_color")
}

func TestPublicTenantBrandingUnknownTenantReturns404(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	tenantRepo := tenantRepository.Repository{DB: db}
	handler := handlers.TenantHandler{Repo: &tenantRepo}

	router := mux.NewRouter()
	tPublic := router.PathPrefix("/t/{tenant_slug}").Subrouter()
	tPublic.Use(middleware.TenantFromPathOrHeader(&tenantRepo))
	tPublic.HandleFunc("/branding", handler.GetBranding).Methods("GET")

	req := httptest.NewRequest("GET", "/t/no-such-tenant/branding", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestPublicGetProductByIDByTenantSlug(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	productRepo := repository.ProductRepository{DB: db}
	tenantRepo := tenantRepository.Repository{DB: db}
	handler := handlers.ProductHandler{Repo: &productRepo}

	router := mux.NewRouter()
	tPublic := router.PathPrefix("/t/{tenant_slug}").Subrouter()
	tPublic.Use(middleware.TenantFromPathOrHeader(&tenantRepo))
	tPublic.HandleFunc("/products/{id}", handler.GetProductByID).Methods("GET")

	req := httptest.NewRequest("GET", "/t/default/products/1", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var product map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &product)
	require.NoError(t, err)
	assert.Equal(t, float64(1), product["id_product"])
	assert.Equal(t, "Brownie Clásico", product["name"])
}

func TestPublicProductsUnknownTenantReturns404(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	productRepo := repository.ProductRepository{DB: db}
	tenantRepo := tenantRepository.Repository{DB: db}
	handler := handlers.ProductHandler{Repo: &productRepo}

	router := mux.NewRouter()
	tPublic := router.PathPrefix("/t/{tenant_slug}").Subrouter()
	tPublic.Use(middleware.TenantFromPathOrHeader(&tenantRepo))
	tPublic.HandleFunc("/products", handler.GetAllProducts).Methods("GET")

	req := httptest.NewRequest("GET", "/t/no-such-tenant/products", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// Legacy public routes (no /t/{slug} prefix) must use TenantFromPathOrHeader so
// X-Tenant-Slug populates context; otherwise handlers return 400.
func TestLegacyPublicProducts_RequiresTenantMiddleware(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	productRepo := repository.ProductRepository{DB: db}
	handler := handlers.ProductHandler{Repo: &productRepo}

	router := mux.NewRouter()
	router.HandleFunc("/products", handler.GetAllProducts).Methods("GET")
	router.HandleFunc("/products/{id}", handler.GetProductByID).Methods("GET")

	listRR := httptest.NewRecorder()
	router.ServeHTTP(listRR, httptest.NewRequest("GET", "/products", nil))
	assert.Equal(t, http.StatusBadRequest, listRR.Code)
	assert.Contains(t, listRR.Body.String(), "tenant context required")

	getRR := httptest.NewRecorder()
	router.ServeHTTP(getRR, httptest.NewRequest("GET", "/products/1", nil))
	assert.Equal(t, http.StatusBadRequest, getRR.Code)
	assert.Contains(t, getRR.Body.String(), "tenant context required")
}

func TestLegacyPublicProducts_WithXTenantSlugHeader(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	productRepo := repository.ProductRepository{DB: db}
	tenantRepo := tenantRepository.Repository{DB: db}
	handler := handlers.ProductHandler{Repo: &productRepo}

	router := mux.NewRouter()
	legacy := router.PathPrefix("").Subrouter()
	legacy.Use(middleware.TenantFromPathOrHeader(&tenantRepo))
	legacy.HandleFunc("/products", handler.GetAllProducts).Methods("GET")
	legacy.HandleFunc("/products/{id}", handler.GetProductByID).Methods("GET")

	listReq := httptest.NewRequest("GET", "/products", nil)
	listReq.Header.Set("X-Tenant-Slug", "default")
	listRR := httptest.NewRecorder()
	router.ServeHTTP(listRR, listReq)
	assert.Equal(t, http.StatusOK, listRR.Code)

	var listBody struct {
		Items []map[string]interface{} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(listRR.Body.Bytes(), &listBody))
	assert.Len(t, listBody.Items, 2)

	getReq := httptest.NewRequest("GET", "/products/1", nil)
	getReq.Header.Set("X-Tenant-Slug", "default")
	getRR := httptest.NewRecorder()
	router.ServeHTTP(getRR, getReq)
	assert.Equal(t, http.StatusOK, getRR.Code)

	var product map[string]interface{}
	require.NoError(t, json.Unmarshal(getRR.Body.Bytes(), &product))
	assert.Equal(t, "Brownie Clásico", product["name"])
}

func TestLegacyPublicProducts_MissingTenantSlugHeaderReturns400(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	productRepo := repository.ProductRepository{DB: db}
	tenantRepo := tenantRepository.Repository{DB: db}
	handler := handlers.ProductHandler{Repo: &productRepo}

	router := mux.NewRouter()
	legacy := router.PathPrefix("").Subrouter()
	legacy.Use(middleware.TenantFromPathOrHeader(&tenantRepo))
	legacy.HandleFunc("/products", handler.GetAllProducts).Methods("GET")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest("GET", "/products", nil))
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "X-Tenant-Slug")
}
