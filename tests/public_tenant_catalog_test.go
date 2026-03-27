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

	var products []map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &products)
	require.NoError(t, err)
	assert.Len(t, products, 2)
	assert.Equal(t, "Brownie Clásico", products[0]["name"])
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
	tPublic.HandleFunc("/tenant/branding", handler.GetBranding).Methods("GET")

	req := httptest.NewRequest("GET", "/t/default/tenant/branding", nil)
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
	tPublic.HandleFunc("/tenant/branding", handler.GetBranding).Methods("GET")

	req := httptest.NewRequest("GET", "/t/no-such-tenant/tenant/branding", nil)
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
