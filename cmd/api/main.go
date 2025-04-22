package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	h "github.com/radamesvaz/bakery-app/internal/handlers"
	"github.com/radamesvaz/bakery-app/internal/handlers/auth"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	productsRepository "github.com/radamesvaz/bakery-app/internal/repository/products"
	"github.com/radamesvaz/bakery-app/internal/repository/user"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
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

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("Could not connect to the DB")
		panic(err)
	}
	defer db.Close()

	productRepo := &productsRepository.ProductRepository{DB: db}
	productHandler := &h.ProductHandler{Repo: productRepo}

	userRepo := user.UserRepository{DB: db}
	authService := authService.AuthService{}
	authHandler := &auth.LoginHandler{
		UserRepo:    userRepo,
		AuthService: authService,
	}

	r := mux.NewRouter()
	r.HandleFunc("/products", productHandler.GetAllProducts).Methods("GET")
	r.HandleFunc("/products/{id}", productHandler.GetProductByID).Methods("GET")
	// Auth endpoints
	r.HandleFunc("/login", authHandler.Login).Methods("POST")

	// Test middleware endpoint
	auth := r.PathPrefix("/auth").Subrouter()
	auth.Use(middleware.AuthMiddleware)
	auth.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Token válido, acceso permitido")
	}).Methods("GET")

	auth.HandleFunc("/products/{id}", productHandler.UpdateProduct).Methods("PUT")

	fmt.Println("🚀 Servidor corriendo en http://localhost:8080")
	http.ListenAndServe(":8080", r)
}
