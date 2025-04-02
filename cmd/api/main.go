package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	h "github.com/radamesvaz/bakery-app/internal/handlers"
	productsRepository "github.com/radamesvaz/bakery-app/internal/repository/products"

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

	repo := &productsRepository.ProductRepository{DB: db}
	handler := &h.ProductHandler{Repo: repo}

	r := mux.NewRouter()
	r.HandleFunc("/products", handler.GetAllProducts).Methods("GET")
	r.HandleFunc("/products/{id}", handler.GetProductByID).Methods("GET")

	fmt.Println("🚀 Servidor corriendo en http://localhost:8080")
	http.ListenAndServe(":8080", r)
}
