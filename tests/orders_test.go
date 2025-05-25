package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/handlers"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	ordersRepository "github.com/radamesvaz/bakery-app/internal/repository/orders"
	productRepo "github.com/radamesvaz/bakery-app/internal/repository/products"
	userRepo "github.com/radamesvaz/bakery-app/internal/repository/user"
	"github.com/radamesvaz/bakery-app/internal/services/auth"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
)

func TestGetAllOrders(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupMySQLContainer(t)
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
        "id_user": "Client",
        "status": "delivered",
        "total_price": 40,
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
        "delivery_date": "2025-04-05T00:00:00Z"
    	},
    {
        "id_order": 2,
        "id_user": "Client",
        "status": "pending",
        "total_price": 20,
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
        "delivery_date": "2025-04-20T00:00:00Z"
    },
    {
        "id_order": 3,
        "id_user": "Client",
        "status": "preparing",
        "total_price": 40,
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
        "delivery_date": "2025-04-25T00:00:00Z"
    }
]`,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, expected, rr.Body.String())
}

func TestGetOrderByID(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupMySQLContainer(t)
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
    "id_user": "Client",
    "status": "delivered",
    "total_price": 40,
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
    "delivery_date": "2025-04-05T00:00:00Z"
}`,
	)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, expected, rr.Body.String())
}

func TestCreateOrder(t *testing.T) {
	// setup
	_, db, terminate, dsn := setupMySQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	// Order setup
	orderRepo := &ordersRepository.OrderRepository{DB: db}
	userRepo := &userRepo.UserRepository{DB: db}
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
