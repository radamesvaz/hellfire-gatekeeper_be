package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func main() {
	// Get database connection details from environment variables
	dbUser := os.Getenv("MYSQL_USER")
	dbPassword := os.Getenv("MYSQL_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("MYSQL_DATABASE")

	if dbUser == "" || dbPassword == "" || dbHost == "" || dbPort == "" || dbName == "" {
		log.Fatal("Missing required database environment variables")
	}

	// Create database connection string
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Open database connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	defer db.Close()

	// Test the connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Could not ping database: %v", err)
	}

	// Create migrate instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("Could not create migrate driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		log.Fatalf("Could not create migrate instance: %v", err)
	}

	// Run migrations
	fmt.Println("🔄 Running database migrations...")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Could not run migrations: %v", err)
	}

	fmt.Println("✅ Database migrations completed successfully!")
}
