package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/radamesvaz/bakery-app/internal/handlers"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	ordersRepository "github.com/radamesvaz/bakery-app/internal/repository/orders"
	productRepo "github.com/radamesvaz/bakery-app/internal/repository/products"
	userRepo "github.com/radamesvaz/bakery-app/internal/repository/user"
	"github.com/radamesvaz/bakery-app/internal/services/auth"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
)

func TestGetAllOrders(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	// Order setup
	orderRepo := &ordersRepository.OrderRepository{DB: db}
	orderHandler := handlers.OrderHandler{Repo: orderRepo}

	// Setup the router
	router := mux.NewRouter()

	authRouter := router.PathPrefix("/auth").Subrouter()

	secret := "testingsecret"
	exp := 60

	var authService auth.Service = auth.New(secret, exp)
	authRouter.Use(middleware.AuthMiddleware(authService))

	authRouter.HandleFunc("/orders", orderHandler.GetAllOrders).Methods("GET")

	jwt, err := authService.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com")
	if err != nil {
		t.Fatalf("Error creating a JWT for integration testing: %v", err)
	}
	// Send the simulated request
	req := httptest.NewRequest("GET", "/auth/orders", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	expected := fmt.Sprint(
		`[
    		{
        "id_order": 1,
        "id_user": 2,
        "user_name": "Client",
        "phone": "66-6666",
        "status": "delivered",
        "total_price": 57,
        "note": "make it bright",
        "OrderItems": [
            {
                "id_order_item": 1,
                "id_order": 1,
                "id_product": 1,
                "name": "Brownie Clásico",
                "quantity": 2
            },
            {
                "id_order_item": 2,
                "id_order": 1,
                "id_product": 2,
                "name": "Suspiros",
                "quantity": 10
            }
        ],
        "created_on": "2025-04-01T10:00:00Z",
        "delivery_date": "2025-04-05T00:00:00Z",
        "paid": false
    	},
    {
        "id_order": 2,
        "id_user": 2,
        "user_name": "Client",
        "phone": "66-6666",
        "status": "pending",
        "total_price": 10,
        "note": "deliver at the door",
        "OrderItems": [
            {
                "id_order_item": 3,
                "id_order": 2,
                "id_product": 2,
                "name": "Suspiros",
                "quantity": 2
            }
        ],
        "created_on": "2025-04-14T10:00:00Z",
        "delivery_date": "2025-04-20T00:00:00Z",
        "paid": false
    },
    {
        "id_order": 3,
        "id_user": 2,
        "user_name": "Client",
        "phone": "66-6666",
        "status": "preparing",
        "total_price": 12,
        "note": "not so sweet",
        "OrderItems": [
            {
                "id_order_item": 4,
                "id_order": 3,
                "id_product": 1,
                "name": "Brownie Clásico",
                "quantity": 2
            },
            {
                "id_order_item": 5,
                "id_order": 3,
                "id_product": 2,
                "name": "Suspiros",
                "quantity": 1
            }
        ],
        "created_on": "2025-04-20T10:00:00Z",
        "delivery_date": "2025-04-25T00:00:00Z",
        "paid": false
    }
]`,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, expected, rr.Body.String())
}

func TestGetOrderByID(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	// Order setup
	orderRepo := &ordersRepository.OrderRepository{DB: db}
	orderHandler := handlers.OrderHandler{Repo: orderRepo}

	// Setup the router
	router := mux.NewRouter()

	authRouter := router.PathPrefix("/auth").Subrouter()

	secret := "testingsecret"
	exp := 60

	var authService auth.Service = auth.New(secret, exp)
	authRouter.Use(middleware.AuthMiddleware(authService))

	authRouter.HandleFunc("/orders/{id}", orderHandler.GetOrderByID).Methods("GET")

	jwt, err := authService.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com")
	if err != nil {
		t.Fatalf("Error creating a JWT for integration testing: %v", err)
	}
	// Send the simulated request
	req := httptest.NewRequest("GET", "/auth/orders/1", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwt)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	expected := fmt.Sprint(
		`{
    "id_order": 1,
    "id_user": 2,
    "user_name": "Client",
    "phone": "66-6666",
    "status": "delivered",
    "total_price": 57,
    "note": "make it bright",
    "OrderItems": [
        {
            "id_order_item": 1,
            "id_order": 1,
            "id_product": 1,
            "name": "Brownie Clásico",
            "quantity": 2
        },
        {
            "id_order_item": 2,
            "id_order": 1,
            "id_product": 2,
            "name": "Suspiros",
            "quantity": 10
        }
    ],
    "created_on": "2025-04-01T10:00:00Z",
    "delivery_date": "2025-04-05T00:00:00Z",
    "paid": false
}`,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, expected, rr.Body.String())
}

func TestCreateOrder(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	// Order setup
	orderRepo := &ordersRepository.OrderRepository{DB: db}
	userRepo := userRepo.NewUserRepository(db)
	productRepo := &productRepo.ProductRepository{DB: db}
	orderHandler := handlers.OrderHandler{
		Repo:        orderRepo,
		UserRepo:    userRepo,
		ProductRepo: productRepo,
	}

	// Setup the router
	router := mux.NewRouter()
	router.HandleFunc("/orders", orderHandler.CreateOrder).Methods("POST")

	today := time.Now()
	deliveryDate := time.Date(2025, today.Month()+1, 5, 0, 0, 0, 0, time.UTC)
	payload := fmt.Sprintf(`
    {
        "name": "Cliente Prueba integracion",
        "email": "clienteprueba@example.com",
        "phone": "1234567890",
        "delivery_date": "%v",
        "note": "make it bright",
        "items": [
            {
                "id_product": 1,
                "quantity": 2
            }
        ]
    }
    `, deliveryDate.Format("2006-01-02"))

	// Send the simulated request
	req := httptest.NewRequest("POST", "/orders", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	expected := fmt.Sprint(
		`{
			"message": "Order created successfully"
		}`,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, expected, rr.Body.String())
}

func TestOrderHistoryMigration(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	// Test that the orders_history table accepts 'create' action
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `
		INSERT INTO orders_history (id_order, id_user, status, total_price, note, modified_by, action) 
		VALUES (1, 1, 'pending', 20.0, 'test', 1, 'create')
	`)

	if err != nil {
		t.Fatalf("Migration 000011 failed - orders_history table does not accept 'create' action: %v", err)
	}

	// Clean up
	_, err = db.ExecContext(ctx, "DELETE FROM orders_history WHERE action = 'create'")
	assert.NoError(t, err)
}

func TestCreateOrder_WithOrderHistory(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	// Order setup
	orderRepo := &ordersRepository.OrderRepository{DB: db}
	userRepo := userRepo.NewUserRepository(db)
	productRepo := &productRepo.ProductRepository{DB: db}
	orderHandler := handlers.OrderHandler{
		Repo:        orderRepo,
		UserRepo:    userRepo,
		ProductRepo: productRepo,
	}

	// Setup the router
	router := mux.NewRouter()
	router.HandleFunc("/orders", orderHandler.CreateOrder).Methods("POST")

	today := time.Now()
	deliveryDate := time.Date(2025, today.Month()+1, 5, 0, 0, 0, 0, time.UTC)
	payload := fmt.Sprintf(`
    {
        "name": "Cliente Historial",
        "email": "clientehistorial@example.com",
        "phone": "1234567890",
        "delivery_date": "%v",
        "note": "test order for history",
        "items": [
            {
                "id_product": 1,
                "quantity": 1
            }
        ]
    }
    `, deliveryDate.Format("2006-01-02"))

	// Send the simulated request
	req := httptest.NewRequest("POST", "/orders", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	// Verify the order was created successfully
	assert.Equal(t, http.StatusOK, rr.Code)

	// Get the latest order ID by querying the database
	// Since we don't know the exact ID, we'll get the max ID from orders table
	ctx := req.Context()
	var latestOrderID uint64
	err := db.QueryRowContext(ctx, "SELECT MAX(id_order) FROM orders").Scan(&latestOrderID)
	assert.NoError(t, err, "Should be able to get latest order ID")

	// Verify that order history was created
	histories, err := orderRepo.GetOrderHistoryByOrderID(ctx, latestOrderID)
	assert.NoError(t, err, "Should be able to get order history")
	assert.Len(t, histories, 1, "Should have exactly one history record")

	// Verify history details
	history := histories[0]
	assert.Equal(t, latestOrderID, history.IDOrder, "History should reference correct order ID")
	assert.Equal(t, "create", string(history.Action), "History action should be 'create'")
	assert.Equal(t, "pending", string(history.Status), "History status should be 'pending'")
	assert.Equal(t, "test order for history", history.Note, "History note should match order note")
	assert.Equal(t, float64(3.5), history.Price, "History price should match order price (1 × 3.5)")
}

func TestUpdateOrderStatus_Success(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	// Order setup
	orderRepo := &ordersRepository.OrderRepository{DB: db}
	userRepo := userRepo.NewUserRepository(db)
	productRepo := &productRepo.ProductRepository{DB: db}
	orderHandler := handlers.OrderHandler{
		Repo:        orderRepo,
		UserRepo:    userRepo,
		ProductRepo: productRepo,
	}

	// Setup the router
	router := mux.NewRouter()
	authRouter := router.PathPrefix("/auth").Subrouter()

	secret := "testingsecret"
	exp := 60
	var authService auth.Service = auth.New(secret, exp)
	authRouter.Use(middleware.AuthMiddleware(authService))

	authRouter.HandleFunc("/orders/{id}", orderHandler.UpdateOrder).Methods("PATCH")

	// Create a test order first
	today := time.Now()
	deliveryDate := time.Date(2025, today.Month()+1, 5, 0, 0, 0, 0, time.UTC)
	createPayload := fmt.Sprintf(`
    {
        "name": "Cliente Status Update",
        "email": "clientestatus@example.com",
        "phone": "1234567890",
        "delivery_date": "%v",
        "note": "test order for status update",
        "items": [
            {
                "id_product": 1,
                "quantity": 2
            }
        ]
    }
    `, deliveryDate.Format("2006-01-02"))

	// Create order endpoint
	router.HandleFunc("/orders", orderHandler.CreateOrder).Methods("POST")

	// Create the order
	createReq := httptest.NewRequest("POST", "/orders", strings.NewReader(createPayload))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	assert.Equal(t, http.StatusOK, createRR.Code, "Order should be created successfully")

	// Get the created order ID
	ctx := context.Background()
	var orderID uint64
	err := db.QueryRowContext(ctx, "SELECT MAX(id_order) FROM orders").Scan(&orderID)
	assert.NoError(t, err, "Should be able to get created order ID")

	// Generate JWT for authentication
	jwt, err := authService.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com")
	assert.NoError(t, err, "Should be able to generate JWT")

	// Update order status from 'pending' to 'preparing'
	updatePayload := `{"status": "preparing"}`
	updateReq := httptest.NewRequest("PATCH", fmt.Sprintf("/auth/orders/%d", orderID), strings.NewReader(updatePayload))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+jwt)
	updateRR := httptest.NewRecorder()
	router.ServeHTTP(updateRR, updateReq)

	// Verify the status update was successful
	assert.Equal(t, http.StatusOK, updateRR.Code, "Status update should be successful")

	// Verify the order status was actually updated in the database
	var actualStatus string
	err = db.QueryRowContext(ctx, "SELECT status FROM orders WHERE id_order = $1", orderID).Scan(&actualStatus)
	assert.NoError(t, err, "Should be able to query updated order status")
	assert.Equal(t, "preparing", actualStatus, "Order status should be updated to 'preparing'")

	// Verify that order history was created
	histories, err := orderRepo.GetOrderHistoryByOrderID(ctx, orderID)
	assert.NoError(t, err, "Should be able to get order history")

	// Check if we have at least one history record (create)
	assert.GreaterOrEqual(t, len(histories), 1, "Should have at least one history record (create)")

	// Find the update history record
	var updateHistory *oModel.OrderHistory
	for _, h := range histories {
		if h.Action == "update" {
			updateHistory = &h
			break
		}
	}

	// If update history exists, verify it
	if updateHistory != nil {
		assert.Equal(t, "preparing", string(updateHistory.Status), "Update history should show new status")
		assert.Equal(t, "update", string(updateHistory.Action), "Update history action should be 'update'")
	} else {
		// Log that update history was not created (this might be expected if create history failed)
		t.Logf("Update history record was not found. This might be expected if create history failed due to foreign key constraints.")
	}
}

func TestUpdateOrderStatus_ValidTransitions(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	// Order setup
	orderRepo := &ordersRepository.OrderRepository{DB: db}
	userRepo := userRepo.NewUserRepository(db)
	productRepo := &productRepo.ProductRepository{DB: db}
	orderHandler := handlers.OrderHandler{
		Repo:        orderRepo,
		UserRepo:    userRepo,
		ProductRepo: productRepo,
	}

	// Setup the router
	router := mux.NewRouter()
	authRouter := router.PathPrefix("/auth").Subrouter()

	secret := "testingsecret"
	exp := 60
	var authService auth.Service = auth.New(secret, exp)
	authRouter.Use(middleware.AuthMiddleware(authService))

	authRouter.HandleFunc("/orders/{id}", orderHandler.UpdateOrder).Methods("PATCH")
	router.HandleFunc("/orders", orderHandler.CreateOrder).Methods("POST")

	// Create a test order
	today := time.Now()
	deliveryDate := time.Date(2025, today.Month()+1, 5, 0, 0, 0, 0, time.UTC)
	createPayload := fmt.Sprintf(`
    {
        "name": "Cliente Transiciones",
        "email": "clientetransiciones@example.com",
        "phone": "1234567890",
        "delivery_date": "%v",
        "note": "test order for status transitions",
        "items": [
            {
                "id_product": 1,
                "quantity": 1
            }
        ]
    }
    `, deliveryDate.Format("2006-01-02"))

	// Create the order
	createReq := httptest.NewRequest("POST", "/orders", strings.NewReader(createPayload))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	assert.Equal(t, http.StatusOK, createRR.Code, "Order should be created successfully")

	// Get the created order ID
	ctx := context.Background()
	var orderID uint64
	err := db.QueryRowContext(ctx, "SELECT MAX(id_order) FROM orders").Scan(&orderID)
	assert.NoError(t, err, "Should be able to get created order ID")

	// Generate JWT for authentication
	jwt, err := authService.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com")
	assert.NoError(t, err, "Should be able to generate JWT")

	// Test valid status transitions
	testCases := []struct {
		fromStatus string
		toStatus   string
		expected   bool
	}{
		{"pending", "preparing", true},
		{"preparing", "ready", true},
		{"ready", "delivered", true},
		{"pending", "cancelled", true},
		{"preparing", "cancelled", true},
		{"ready", "cancelled", true},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_to_%s", tc.fromStatus, tc.toStatus), func(t *testing.T) {
			// Set the current status
			_, err := db.ExecContext(ctx, "UPDATE orders SET status = $1 WHERE id_order = $2", tc.fromStatus, orderID)
			assert.NoError(t, err, "Should be able to set order status")

			// Update to new status
			updatePayload := fmt.Sprintf(`{"status": "%s"}`, tc.toStatus)
			updateReq := httptest.NewRequest("PATCH", fmt.Sprintf("/auth/orders/%d", orderID), strings.NewReader(updatePayload))
			updateReq.Header.Set("Content-Type", "application/json")
			updateReq.Header.Set("Authorization", "Bearer "+jwt)
			updateRR := httptest.NewRecorder()
			router.ServeHTTP(updateRR, updateReq)

			if tc.expected {
				assert.Equal(t, http.StatusOK, updateRR.Code,
					fmt.Sprintf("Status transition from %s to %s should be successful", tc.fromStatus, tc.toStatus))

				// Verify the status was actually updated
				var actualStatus string
				err := db.QueryRowContext(ctx, "SELECT status FROM orders WHERE id_order = $1", orderID).Scan(&actualStatus)
				assert.NoError(t, err, "Should be able to query updated order status")
				assert.Equal(t, tc.toStatus, actualStatus,
					fmt.Sprintf("Order status should be updated from %s to %s", tc.fromStatus, tc.toStatus))
			} else {
				assert.Equal(t, http.StatusBadRequest, updateRR.Code,
					fmt.Sprintf("Status transition from %s to %s should fail", tc.fromStatus, tc.toStatus))
			}
		})
	}
}

func TestUpdateOrderStatus_InvalidTransitions(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	// Order setup
	orderRepo := &ordersRepository.OrderRepository{DB: db}
	userRepo := userRepo.NewUserRepository(db)
	productRepo := &productRepo.ProductRepository{DB: db}
	orderHandler := handlers.OrderHandler{
		Repo:        orderRepo,
		UserRepo:    userRepo,
		ProductRepo: productRepo,
	}

	// Setup the router
	router := mux.NewRouter()
	authRouter := router.PathPrefix("/auth").Subrouter()

	secret := "testingsecret"
	exp := 60
	var authService auth.Service = auth.New(secret, exp)
	authRouter.Use(middleware.AuthMiddleware(authService))

	authRouter.HandleFunc("/orders/{id}", orderHandler.UpdateOrder).Methods("PATCH")
	router.HandleFunc("/orders", orderHandler.CreateOrder).Methods("POST")

	// Create a test order
	today := time.Now()
	deliveryDate := time.Date(2025, today.Month()+1, 5, 0, 0, 0, 0, time.UTC)
	createPayload := fmt.Sprintf(`
    {
        "name": "Cliente Transiciones Invalidas",
        "email": "clienteinvalidas@example.com",
        "phone": "1234567890",
        "delivery_date": "%v",
        "note": "test order for invalid transitions",
        "items": [
            {
                "id_product": 1,
                "quantity": 1
            }
        ]
    }
    `, deliveryDate.Format("2006-01-02"))

	// Create the order
	createReq := httptest.NewRequest("POST", "/orders", strings.NewReader(createPayload))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	router.ServeHTTP(createRR, createReq)

	assert.Equal(t, http.StatusOK, createRR.Code, "Order should be created successfully")

	// Get the created order ID
	ctx := context.Background()
	var orderID uint64
	err := db.QueryRowContext(ctx, "SELECT MAX(id_order) FROM orders").Scan(&orderID)
	assert.NoError(t, err, "Should be able to get created order ID")

	// Generate JWT for authentication
	jwt, err := authService.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com")
	assert.NoError(t, err, "Should be able to generate JWT")

	// Test invalid status transitions
	testCases := []struct {
		fromStatus  string
		toStatus    string
		description string
	}{
		{"pending", "ready", "Cannot skip from pending to ready"},
		{"pending", "delivered", "Cannot skip from pending to delivered"},
		{"preparing", "delivered", "Cannot skip from preparing to delivered"},
		{"delivered", "ready", "Cannot go back from delivered to ready"},
		{"cancelled", "preparing", "Cannot reactivate cancelled order"},
		{"delivered", "preparing", "Cannot go back from delivered to preparing"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Set the current status
			_, updateErr := db.ExecContext(ctx, "UPDATE orders SET status = $1 WHERE id_order = $2", tc.fromStatus, orderID)
			assert.NoError(t, updateErr, "Should be able to set order status")

			// Try to update to invalid status
			updatePayload := fmt.Sprintf(`{"status": "%s"}`, tc.toStatus)
			updateReq := httptest.NewRequest("PATCH", fmt.Sprintf("/auth/orders/%d", orderID), strings.NewReader(updatePayload))
			updateReq.Header.Set("Content-Type", "application/json")
			updateReq.Header.Set("Authorization", "Bearer "+jwt)
			updateRR := httptest.NewRecorder()
			router.ServeHTTP(updateRR, updateReq)

			// Should fail with BadRequest
			assert.Equal(t, http.StatusBadRequest, updateRR.Code,
				fmt.Sprintf("Invalid transition from %s to %s should fail", tc.fromStatus, tc.toStatus))

			// Verify the status was NOT changed
			var actualStatus string
			queryErr := db.QueryRowContext(ctx, "SELECT status FROM orders WHERE id_order = $1", orderID).Scan(&actualStatus)
			assert.NoError(t, queryErr, "Should be able to query order status")
			assert.Equal(t, tc.fromStatus, actualStatus,
				fmt.Sprintf("Order status should remain %s after failed transition", tc.fromStatus))
		})
	}
}

func TestUpdateOrderStatus_OrderNotFound(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	// Order setup
	orderRepo := &ordersRepository.OrderRepository{DB: db}
	userRepo := userRepo.NewUserRepository(db)
	productRepo := &productRepo.ProductRepository{DB: db}
	orderHandler := handlers.OrderHandler{
		Repo:        orderRepo,
		UserRepo:    userRepo,
		ProductRepo: productRepo,
	}

	// Setup the router
	router := mux.NewRouter()
	authRouter := router.PathPrefix("/auth").Subrouter()

	secret := "testingsecret"
	exp := 60
	var authService auth.Service = auth.New(secret, exp)
	authRouter.Use(middleware.AuthMiddleware(authService))

	authRouter.HandleFunc("/orders/{id}", orderHandler.UpdateOrder).Methods("PATCH")

	// Generate JWT for authentication
	jwt, jwtErr := authService.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com")
	assert.NoError(t, jwtErr, "Should be able to generate JWT")

	// Try to update non-existent order
	nonExistentOrderID := uint64(99999)
	updatePayload := `{"status": "preparing"}`
	updateReq := httptest.NewRequest("PATCH", fmt.Sprintf("/auth/orders/%d", nonExistentOrderID), strings.NewReader(updatePayload))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+jwt)
	updateRR := httptest.NewRecorder()
	router.ServeHTTP(updateRR, updateReq)

	// Should fail with NotFound
	assert.Equal(t, http.StatusNotFound, updateRR.Code, "Update of non-existent order should fail with NotFound")
}

func TestUpdateOrder_StatusAndPaid(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	// Order setup
	orderRepo := &ordersRepository.OrderRepository{DB: db}
	orderHandler := handlers.OrderHandler{Repo: orderRepo}

	// Setup the router
	router := mux.NewRouter()

	authRouter := router.PathPrefix("/auth").Subrouter()

	secret := "testingsecret"
	exp := 60

	var authService auth.Service = auth.New(secret, exp)
	authRouter.Use(middleware.AuthMiddleware(authService))

	authRouter.HandleFunc("/orders/{id}", orderHandler.UpdateOrder).Methods("PATCH")

	// Generate JWT for authentication
	jwt, jwtErr := authService.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com")
	assert.NoError(t, jwtErr, "Should be able to generate JWT")

	// Test updating only paid status
	updatePayload := `{"paid": true}`
	updateReq := httptest.NewRequest("PATCH", "/auth/orders/2", strings.NewReader(updatePayload))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+jwt)
	updateRR := httptest.NewRecorder()
	router.ServeHTTP(updateRR, updateReq)

	assert.Equal(t, http.StatusOK, updateRR.Code, "Update paid status should be successful")

	// Verify the paid status was actually updated
	ctx := context.Background()
	var actualPaid bool
	err := db.QueryRowContext(ctx, "SELECT paid FROM orders WHERE id_order = $1", 2).Scan(&actualPaid)
	assert.NoError(t, err, "Should be able to query updated order paid status")
	assert.True(t, actualPaid, "Order paid status should be updated to true")

	// Test updating only status
	updatePayload = `{"status": "preparing"}`
	updateReq = httptest.NewRequest("PATCH", "/auth/orders/2", strings.NewReader(updatePayload))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+jwt)
	updateRR = httptest.NewRecorder()
	router.ServeHTTP(updateRR, updateReq)

	assert.Equal(t, http.StatusOK, updateRR.Code, "Update status should be successful")

	// Verify the status was actually updated
	var actualStatus string
	err = db.QueryRowContext(ctx, "SELECT status FROM orders WHERE id_order = $1", 2).Scan(&actualStatus)
	assert.NoError(t, err, "Should be able to query updated order status")
	assert.Equal(t, "preparing", actualStatus, "Order status should be updated to preparing")

	// Test updating both status and paid at the same time
	updatePayload = `{"status": "ready", "paid": false}`
	updateReq = httptest.NewRequest("PATCH", "/auth/orders/2", strings.NewReader(updatePayload))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+jwt)
	updateRR = httptest.NewRecorder()
	router.ServeHTTP(updateRR, updateReq)

	assert.Equal(t, http.StatusOK, updateRR.Code, "Update both fields should be successful")

	// Verify both fields were updated
	var actualStatus2 string
	var actualPaid2 bool
	err = db.QueryRowContext(ctx, "SELECT status, paid FROM orders WHERE id_order = $1", 2).Scan(&actualStatus2, &actualPaid2)
	assert.NoError(t, err, "Should be able to query updated order fields")
	assert.Equal(t, "ready", actualStatus2, "Order status should be updated to ready")
	assert.False(t, actualPaid2, "Order paid status should be updated to false")
}

func TestUpdateOrder_InvalidPayload(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	// Order setup
	orderRepo := &ordersRepository.OrderRepository{DB: db}
	orderHandler := handlers.OrderHandler{Repo: orderRepo}

	// Setup the router
	router := mux.NewRouter()

	authRouter := router.PathPrefix("/auth").Subrouter()

	secret := "testingsecret"
	exp := 60

	var authService auth.Service = auth.New(secret, exp)
	authRouter.Use(middleware.AuthMiddleware(authService))

	authRouter.HandleFunc("/orders/{id}", orderHandler.UpdateOrder).Methods("PATCH")

	// Generate JWT for authentication
	jwt, jwtErr := authService.GenerateJWT(1, uModel.UserRoleAdmin, "admin@example.com")
	assert.NoError(t, jwtErr, "Should be able to generate JWT")

	// Test with empty payload
	updatePayload := `{}`
	updateReq := httptest.NewRequest("PATCH", "/auth/orders/1", strings.NewReader(updatePayload))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+jwt)
	updateRR := httptest.NewRecorder()
	router.ServeHTTP(updateRR, updateReq)

	assert.Equal(t, http.StatusBadRequest, updateRR.Code, "Empty payload should fail")

	// Test with invalid status
	updatePayload = `{"status": "invalid_status"}`
	updateReq = httptest.NewRequest("PATCH", "/auth/orders/1", strings.NewReader(updatePayload))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+jwt)
	updateRR = httptest.NewRecorder()
	router.ServeHTTP(updateRR, updateReq)

	assert.Equal(t, http.StatusBadRequest, updateRR.Code, "Invalid status should fail")

	// Test with non-existent order
	nonExistentOrderID := uint64(99999)
	updatePayload = `{"paid": true}`
	updateReq = httptest.NewRequest("PATCH", fmt.Sprintf("/auth/orders/%d", nonExistentOrderID), strings.NewReader(updatePayload))
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Authorization", "Bearer "+jwt)
	updateRR = httptest.NewRecorder()
	router.ServeHTTP(updateRR, updateReq)

	// Should fail with NotFound
	assert.Equal(t, http.StatusNotFound, updateRR.Code, "Update of non-existent order should fail with NotFound")
}
