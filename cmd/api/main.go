package main

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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

	// Prefer full connection URL if provided (e.g., DATABASE_URL from Supabase/Render)
	databaseURL := os.Getenv("DATABASE_URL")

	// Fallback to discrete variables with PG* compatibility
	dbUser := firstNonEmpty(os.Getenv("POSTGRES_USER"), os.Getenv("PGUSER"))
	dbPassword := firstNonEmpty(os.Getenv("POSTGRES_PASSWORD"), os.Getenv("PGPASSWORD"))
	dbHost := firstNonEmpty(os.Getenv("DB_HOST"), os.Getenv("POSTGRES_HOST"), os.Getenv("PGHOST"))
	dbPort := firstNonEmpty(os.Getenv("DB_PORT"), os.Getenv("POSTGRES_PORT"), os.Getenv("PGPORT"), "5432")
	dbName := firstNonEmpty(os.Getenv("POSTGRES_DB"), os.Getenv("PGDATABASE"))
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

	// Validate required database configuration
	if databaseURL == "" {
		if dbHost == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			fmt.Println("❌ Missing required database environment variables")
			panic("Database configuration incomplete")
		}
	}

	// Warn if using default database name
	if dbName == "postgres" {
		fmt.Println("⚠️ Warning: Using default 'postgres' database name. Make sure this is correct for your Supabase setup.")
	}

	// Debug: Show what IPs are being resolved (only when host is known)
	if databaseURL == "" && dbHost != "" {
		fmt.Printf("🔍 Resolving hostname: %s\n", dbHost)
		ips, err := net.LookupIP(dbHost)
		if err != nil {
			fmt.Printf("⚠️ DNS lookup failed: %v\n", err)
		} else {
			fmt.Printf("📍 Resolved IPs: ")
			for i, ip := range ips {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s", ip.String())
			}
			fmt.Printf("\n")
		}
	}

	// Build DSN
	var dsn string
	if databaseURL != "" {
		// Assume DATABASE_URL already contains proper sslmode settings
		dsn = databaseURL
		fmt.Printf("🔗 Connecting to DB using DATABASE_URL\n")
	} else {
		sslMode := "require"
		lowerHost := strings.ToLower(dbHost)
		if lowerHost == "localhost" || lowerHost == "127.0.0.1" || lowerHost == "::1" {
			sslMode = "disable"
		}
		dsn = fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=30 fallback_application_name=hellfire-gatekeeper",
			dbHost, dbPort, dbUser, dbPassword, dbName, sslMode,
		)
		fmt.Printf("🔗 Connecting to DB: host=%s port=%s user=%s dbname=%s sslmode=%s\n", dbHost, dbPort, dbUser, dbName, sslMode)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Printf("❌ Could not connect to the DB: %v\n", err)
		panic(err)
	}

	// Configure connection pool for production stability
	db.SetMaxOpenConns(25)                 // Maximum number of open connections
	db.SetMaxIdleConns(5)                  // Maximum number of idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Maximum lifetime of a connection
	db.SetConnMaxIdleTime(1 * time.Minute) // Maximum idle time of a connection

	// Test connection with retry
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		if err := db.Ping(); err != nil {
			fmt.Printf("❌ Could not ping database (attempt %d/%d): %v\n", i+1, maxRetries, err)
			if i == maxRetries-1 {
				panic(err)
			}
			time.Sleep(time.Duration(i+1) * time.Second)
		} else {
			fmt.Println("✅ Database connected successfully")
			break
		}
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

	// CORS configuration (allowlist + credentials)
	allowedOrigins := handlers.AllowedOrigins([]string{
		"http://localhost:5173",
		"http://localhost:3000",
		"http://localhost:5000",
		"https://confettideliadmin.netlify.app",
		"https://confettideli.netlify.app",
	})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	allowedHeaders := handlers.AllowedHeaders([]string{
		"Authorization",
		"Content-Type",
		"X-Requested-With",
		"Accept",
		"Origin",
		"Access-Control-Request-Method",
		"Access-Control-Request-Headers",
	})
	allowCredentials := handlers.AllowCredentials()

	// Serve static files (images)
	r.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))

	// Health check endpoint
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Database connection failed: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	}).Methods("GET")

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

	// Wrap router with CORS
	corsWrapped := handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders, allowCredentials)(r)

	// Start a goroutine to periodically check database health
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := db.Ping(); err != nil {
				fmt.Printf("⚠️ Database health check failed: %v\n", err)
			}
		}
	}()

	fmt.Printf("🚀 Servidor corriendo en http://localhost:%s\n", port)
	http.ListenAndServe(":"+port, corsWrapped)
}

// firstNonEmpty returns the first non-empty string from the provided list.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
