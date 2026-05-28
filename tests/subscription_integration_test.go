package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/handlers"
	authHandlers "github.com/radamesvaz/bakery-app/internal/handlers/auth"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	tenantRepository "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	subscriptionService "github.com/radamesvaz/bakery-app/internal/services/subscriptions"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubscription_ProcessTransitions_Integration checks active->pending and pending->canceled
// across two job runs. If current_period_end is far enough in the past that grace already
// elapsed, a single ProcessTransitions applies both SQL updates in one run (second run then
// affects zero rows), which is why we use a near-recent period_end first, then move the end date.
func TestSubscription_ProcessTransitions_Integration(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	ctx := context.Background()

	graceDays := 5
	_, err := db.ExecContext(ctx, `
		UPDATE tenants
		SET subscription_status = 'active',
		    current_period_end = NOW() - INTERVAL '1 hour'
		WHERE slug = 'default'`)
	require.NoError(t, err)

	repo := &tenantRepository.Repository{DB: db}
	svc := subscriptionService.NewService(repo, graceDays)

	firstRun, err := svc.ProcessTransitions(ctx, time.Now().UTC())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, firstRun.MarkedPending, int64(1))
	assert.Equal(t, int64(0), firstRun.MarkedCanceled,
		"grace period still open: tenant should remain pending until period_end + grace_days")

	var statusAfterFirst string
	err = db.QueryRowContext(ctx,
		`SELECT subscription_status FROM tenants WHERE slug = 'default'`).Scan(&statusAfterFirst)
	require.NoError(t, err)
	assert.Equal(t, "pending", statusAfterFirst)

	_, err = db.ExecContext(ctx, `
		UPDATE tenants
		SET current_period_end = NOW() - INTERVAL '10 days'
		WHERE slug = 'default'`)
	require.NoError(t, err)

	secondRun, err := svc.ProcessTransitions(ctx, time.Now().UTC())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, secondRun.MarkedCanceled, int64(1))

	var statusFinal string
	err = db.QueryRowContext(ctx,
		`SELECT subscription_status FROM tenants WHERE slug = 'default'`).Scan(&statusFinal)
	require.NoError(t, err)
	assert.Equal(t, "canceled", statusFinal)
}

func TestSubscription_GetSubscriptionEndpoint_Integration(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	ctx := context.Background()
	_, err := db.ExecContext(ctx, `
		UPDATE tenants
		SET subscription_status = 'pending',
		    current_period_end = NOW() - INTERVAL '2 days',
		    plan_code = 'basic'
		WHERE id = 1`)
	require.NoError(t, err)

	tenantRepo := &tenantRepository.Repository{DB: db}
	subSvc := subscriptionService.NewService(tenantRepo, 5)
	handler := &authHandlers.SubscriptionHandler{Service: subSvc}

	secret := "testingsecret"
	authSvc := authService.New(secret, 60)
	tenantID := uint64(1)
	jwt, err := authSvc.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com", &tenantID)
	require.NoError(t, err)

	router := mux.NewRouter()
	authRouter := router.PathPrefix("/auth").Subrouter()
	authRouter.Use(middleware.AuthMiddleware(authSvc))
	authRouter.Use(middleware.TenantMiddleware(tenantRepo))
	authRouter.Use(middleware.RequireOperableSubscription(tenantRepo))
	authRouter.HandleFunc("/subscription", handler.GetSubscription).Methods("GET")
	authRouter.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/auth/subscription", nil)
	req.Header.Set("Authorization", "Bearer "+jwt)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, "default", body["tenant_slug"])

	subscription, ok := body["subscription"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "pending", subscription["status"])
	assert.Equal(t, "basic", subscription["plan_code"])
	assert.NotNil(t, subscription["grace_period_end"])
	assert.NotNil(t, subscription["days_until_cancel"])
}

func TestSubscription_CanceledTenant_PublicRoute404_Integration(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	ctx := context.Background()
	_, err := db.ExecContext(ctx, `
		UPDATE tenants SET subscription_status = 'canceled' WHERE slug = 'default'`)
	require.NoError(t, err)

	tenantRepo := &tenantRepository.Repository{DB: db}
	handler := handlers.TenantHandler{Repo: tenantRepo}

	router := mux.NewRouter()
	tPublic := router.PathPrefix("/t/{tenant_slug}").Subrouter()
	tPublic.Use(middleware.TenantFromPathOrHeader(tenantRepo))
	tPublic.HandleFunc("/branding", handler.GetBranding).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/t/default/branding", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestSubscription_CanceledTenant_AuthRoute403_Integration(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	ctx := context.Background()
	_, err := db.ExecContext(ctx, `
		UPDATE tenants SET subscription_status = 'canceled' WHERE id = 1`)
	require.NoError(t, err)

	tenantRepo := &tenantRepository.Repository{DB: db}
	secret := "testingsecret"
	authSvc := authService.New(secret, 60)
	tenantID := uint64(1)
	jwt, err := authSvc.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com", &tenantID)
	require.NoError(t, err)

	router := mux.NewRouter()
	authRouter := router.PathPrefix("/auth").Subrouter()
	authRouter.Use(middleware.AuthMiddleware(authSvc))
	authRouter.Use(middleware.TenantMiddleware(tenantRepo))
	authRouter.Use(middleware.RequireOperableSubscription(tenantRepo))
	authRouter.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/auth/ping", nil)
	req.Header.Set("Authorization", "Bearer "+jwt)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestSubscription_PendingTenant_AuthPing200_Integration(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	ctx := context.Background()
	_, err := db.ExecContext(ctx, `
		UPDATE tenants
		SET subscription_status = 'pending',
		    current_period_end = NOW() - INTERVAL '1 day'
		WHERE id = 1`)
	require.NoError(t, err)

	tenantRepo := &tenantRepository.Repository{DB: db}
	secret := "testingsecret"
	authSvc := authService.New(secret, 60)
	tenantID := uint64(1)
	jwt, err := authSvc.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com", &tenantID)
	require.NoError(t, err)

	router := mux.NewRouter()
	authRouter := router.PathPrefix("/auth").Subrouter()
	authRouter.Use(middleware.AuthMiddleware(authSvc))
	authRouter.Use(middleware.TenantMiddleware(tenantRepo))
	authRouter.Use(middleware.RequireOperableSubscription(tenantRepo))
	authRouter.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/auth/ping", nil)
	req.Header.Set("Authorization", "Bearer "+jwt)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
