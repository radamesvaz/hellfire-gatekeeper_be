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
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env if present (non-fatal if missing)
	_ = godotenv.Load()

	// Get database connection details from environment variables
	databaseURL := os.Getenv("DATABASE_URL")
	dbUser := firstNonEmpty(
		os.Getenv("POSTGRES_USER"),
		os.Getenv("PGUSER"),
		os.Getenv("DB_USER"),
		os.Getenv("MYSQL_USER"),
	)

	dbPassword := firstNonEmpty(
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("PGPASSWORD"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("MYSQL_PASSWORD"),
	)

	dbHost := firstNonEmpty(
		os.Getenv("DB_HOST"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("PGHOST"),
		"localhost",
	)

	dbPort := firstNonEmpty(
		os.Getenv("DB_PORT"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("PGPORT"),
		"5432",
	)

	dbName := firstNonEmpty(
		os.Getenv("POSTGRES_DB"),
		os.Getenv("PGDATABASE"),
		os.Getenv("DB_NAME"),
		os.Getenv("MYSQL_DATABASE"),
	)

	if databaseURL == "" && (dbUser == "" || dbPassword == "" || dbHost == "" || dbPort == "" || dbName == "") {
		fmt.Printf("dbUser: %s", dbUser)
		fmt.Println("dbPassword", dbPassword)
		fmt.Println("dbHost", dbHost)
		fmt.Println("dbPort", dbPort)
		fmt.Println("dbName", dbName)
		log.Fatal("Missing required database environment variables")
	}

	// Create database connection string
	var dsn string
	if databaseURL != "" {
		dsn = databaseURL
	} else {
		sslMode := "require"
		lowerHost := strings.ToLower(dbHost)
		if lowerHost == "localhost" || lowerHost == "127.0.0.1" || lowerHost == "::1" {
			sslMode = "disable"
		}
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			dbHost, dbPort, dbUser, dbPassword, dbName, sslMode)
	}

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
				fmt.Printf("⚠️  Database is dirty at version %s, cleaning and resetting...\n", version)

				// Force version to 1 (first migration)
				if forceErr := m.Force(1); forceErr != nil {
					log.Fatalf("Could not force version to 1: %v", forceErr)
				}

				// Drop all tables to start fresh
				fmt.Println("🗑️  Dropping all tables to start fresh...")
				dropTables := []string{
					"DROP TABLE IF EXISTS order_items CASCADE;",
					"DROP TABLE IF EXISTS orders_history CASCADE;",
					"DROP TABLE IF EXISTS orders CASCADE;",
					"DROP TABLE IF EXISTS products_history CASCADE;",
					"DROP TABLE IF EXISTS products CASCADE;",
					"DROP TABLE IF EXISTS users CASCADE;",
					"DROP TABLE IF EXISTS roles CASCADE;",
					"DROP TABLE IF EXISTS schema_migrations CASCADE;",
					"DROP TYPE IF EXISTS order_status CASCADE;",
					"DROP TYPE IF EXISTS history_action CASCADE;",
				}

				for _, dropSQL := range dropTables {
					if _, execErr := db.Exec(dropSQL); execErr != nil {
						fmt.Printf("⚠️  Warning: Could not execute %s: %v\n", dropSQL, execErr)
					}
				}

				fmt.Println("🔄 Running migrations from scratch...")
				if retryErr := m.Up(); retryErr != nil && retryErr != migrate.ErrNoChange {
					log.Fatalf("Could not run migrations after reset: %v", retryErr)
				}
			} else {
				log.Fatalf("Could not parse dirty version: %v", err)
			}
		} else if strings.Contains(err.Error(), "no migration found for version 0") {
			fmt.Println("⚠️  Migration failed due to missing version 0 down migration. Cleaning database and starting fresh...")

			// Drop all tables and custom types first
			fmt.Println("🗑️  Dropping all tables to start fresh...")
			dropTables := []string{
				"DROP TABLE IF EXISTS order_items CASCADE;",
				"DROP TABLE IF EXISTS orders_history CASCADE;",
				"DROP TABLE IF EXISTS orders CASCADE;",
				"DROP TABLE IF EXISTS products_history CASCADE;",
				"DROP TABLE IF EXISTS products CASCADE;",
				"DROP TABLE IF EXISTS users CASCADE;",
				"DROP TABLE IF EXISTS roles CASCADE;",
				"DROP TABLE IF EXISTS schema_migrations CASCADE;",
				"DROP TYPE IF EXISTS order_status CASCADE;",
				"DROP TYPE IF EXISTS history_action CASCADE;",
			}

			for _, dropSQL := range dropTables {
				if _, execErr := db.Exec(dropSQL); execErr != nil {
					fmt.Printf("⚠️  Warning: Could not execute %s: %v\n", dropSQL, execErr)
				}
			}

			// Force version to 0 (no migrations applied)
			if forceErr := m.Force(0); forceErr != nil {
				log.Fatalf("Could not force version to 0 after specific error: %v", forceErr)
			}
			fmt.Println("🔄 Running all migrations from scratch...")
			if retryErr := m.Up(); retryErr != nil && retryErr != migrate.ErrNoChange {
				log.Fatalf("Could not run migrations after force (specific error): %v", retryErr)
			}
		} else {
			log.Fatalf("Could not run migrations: %v", err)
		}
	}

	fmt.Println("✅ Database migrations completed successfully!")
	fmt.Println("🎉 All migrations applied correctly!")
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
