package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/handlers"
	"github.com/joho/godotenv"
	h "github.com/radamesvaz/bakery-app/internal/handlers"
	"github.com/radamesvaz/bakery-app/internal/handlers/auth"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	ordersRepository "github.com/radamesvaz/bakery-app/internal/repository/orders"
	productsRepository "github.com/radamesvaz/bakery-app/internal/repository/products"
	"github.com/radamesvaz/bakery-app/internal/repository/user"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("⚠ Could not load .env file")
	}

	dbUser := os.Getenv("MYSQL_USER")
	dbPassword := os.Getenv("MYSQL_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("MYSQL_DATABASE")
	secret := os.Getenv("JWT_SECRET")
	expMinutes := os.Getenv("JWT_EXPIRATION_MINUTES")
	port := os.Getenv("PORT")
	exp, err := strconv.Atoi(expMinutes)
	if err != nil {
		fmt.Printf("could not get the expMinutes from env: %v", err)
		panic(err)
	}

	// Set default port if not provided
	if port == "" {
		port = "8080"
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Printf("Could not connect to the DB")
		panic(err)
	}
	defer db.Close()

	// Product setup
	productRepo := &productsRepository.ProductRepository{DB: db}

	// Image service setup
	uploadDir := "uploads"
	imageService := imagesService.New(uploadDir)

	// Product handler (only for product data)
	productHandler := &h.ProductHandler{
		Repo: productRepo,
	}

	// Image handler (only for image management)
	imageHandler := &h.ImageHandler{
		Repo:         productRepo,
		ImageService: imageService,
	}

	// Auth setup
	userRepo := user.UserRepository{DB: db}
	authService := authService.New(secret, exp)
	authHandler := &auth.LoginHandler{
		UserRepo:    userRepo,
		AuthService: *authService,
	}

	// Order setup
	orderRepo := &ordersRepository.OrderRepository{DB: db}
	orderHandler := &h.OrderHandler{
		Repo:        orderRepo,
		UserRepo:    &userRepo,
		ProductRepo: productRepo,
	}

	r := mux.NewRouter()

	// Serve static files (images)
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))

	r.HandleFunc("/products", productHandler.GetAllProducts).Methods("GET")
	r.HandleFunc("/products/{id}", productHandler.GetProductByID).Methods("GET")
	// Auth endpoints
	r.HandleFunc("/login", authHandler.Login).Methods("POST")
	r.HandleFunc("/register", authHandler.Register).Methods("POST")

	// Test middleware endpoint
	auth := r.PathPrefix("/auth").Subrouter()
	auth.Use(middleware.AuthMiddleware(authService))
	auth.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Token válido, acceso permitido")
	}).Methods("GET")

	// Product endpoints (data only)
	auth.HandleFunc("/products", productHandler.CreateProduct).Methods("POST")
	auth.HandleFunc("/products/{id}", productHandler.UpdateProduct).Methods("PUT")
	auth.HandleFunc("/products/{id}", productHandler.UpdateProductStatus).Methods("PATCH")

	// Image endpoints (image management)
	auth.HandleFunc("/products/{id}/images", imageHandler.AddProductImages).Methods("POST")
	auth.HandleFunc("/products/{id}/images", imageHandler.ReplaceProductImages).Methods("PUT")
	auth.HandleFunc("/products/{id}/images", imageHandler.DeleteProductImage).Methods("DELETE")

	// Order endnpoints
	auth.HandleFunc("/orders", orderHandler.GetAllOrders).Methods("GET")
	auth.HandleFunc("/orders/{id}", orderHandler.GetOrderByID).Methods("GET")
	auth.HandleFunc("/orders/{id}", orderHandler.UpdateOrder).Methods("PATCH")
	r.HandleFunc("/orders", orderHandler.CreateOrder).Methods("POST")

	// CORS setup for local FE development
	allowedOrigins := handlers.AllowedOrigins([]string{
		"http://localhost:5173",
		"http://localhost:3000",
		"http://localhost:5000",
	})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	allowedHeaders := handlers.AllowedHeaders([]string{"Authorization", "Content-Type"})

	fmt.Printf("🚀 Servidor corriendo en http://localhost:%s\n", port)
	http.ListenAndServe(":"+port, handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders)(r))
}
