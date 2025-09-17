package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

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
		// If database is dirty, force version and try again
		if strings.Contains(err.Error(), "Dirty database version") {
			// Extract version number from error message
			re := regexp.MustCompile(`Dirty database version (\d+)`)
			matches := re.FindStringSubmatch(err.Error())
			if len(matches) > 1 {
				version := matches[1]
				fmt.Printf("⚠️  Database is dirty at version %s, forcing version...\n", version)
				if forceErr := m.Force(1); forceErr != nil {
					log.Fatalf("Could not force version: %v", forceErr)
				}
				fmt.Println("🔄 Retrying migrations...")
				if retryErr := m.Up(); retryErr != nil && retryErr != migrate.ErrNoChange {
					log.Fatalf("Could not run migrations after force: %v", retryErr)
				}
			} else {
				log.Fatalf("Could not parse dirty version: %v", err)
			}
		} else {
			log.Fatalf("Could not run migrations: %v", err)
		}
	}

	fmt.Println("✅ Database migrations completed successfully!")
}
