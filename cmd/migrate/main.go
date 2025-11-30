package main

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/radamesvaz/bakery-app/internal/logger"
)

func main() {
	// Load .env first (if it exists)
	err := godotenv.Load()
	if err != nil {
		// .env file is optional, especially in production
	}

	// Get log level from environment, default to "info" if not set
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logger.Init(logLevel)

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
		logger.Error().
			Str("db_user", dbUser).
			Str("db_host", dbHost).
			Str("db_port", dbPort).
			Str("db_name", dbName).
			Msg("Missing required database environment variables")
		logger.Fatal("Missing required database environment variables")
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
		logger.Logger.Fatal().Err(err).Msg("Could not connect to database")
	}
	defer db.Close()

	// Test the connection
	if err := db.Ping(); err != nil {
		logger.Logger.Fatal().Err(err).Msg("Could not ping database")
	}

	// Create migrate instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logger.Logger.Fatal().Err(err).Msg("Could not create migrate driver")
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		logger.Logger.Fatal().Err(err).Msg("Could not create migrate instance")
	}

	// Run migrations
	logger.Info().Msg("Running database migrations...")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		// If database is dirty, force version and try again
		if strings.Contains(err.Error(), "Dirty database version") {
			// Extract version number from error message
			re := regexp.MustCompile(`Dirty database version (\d+)`)
			matches := re.FindStringSubmatch(err.Error())
			if len(matches) > 1 {
				version := matches[1]
				logger.Warn().
					Str("version", version).
					Msg("Database is dirty at version, cleaning and resetting...")

				// Force version to 1 (first migration)
				if forceErr := m.Force(1); forceErr != nil {
					logger.Logger.Fatal().Err(forceErr).Msg("Could not force version to 1")
				}

				// Drop all tables to start fresh
				logger.Info().Msg("Dropping all tables to start fresh...")
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
						logger.Warn().Err(execErr).
							Str("sql", dropSQL).
							Msg("Could not execute drop statement")
					}
				}

				logger.Info().Msg("Running migrations from scratch...")
				if retryErr := m.Up(); retryErr != nil && retryErr != migrate.ErrNoChange {
					logger.Logger.Fatal().Err(retryErr).Msg("Could not run migrations after reset")
				}
			} else {
				logger.Logger.Fatal().Err(err).Msg("Could not parse dirty version")
			}
		} else if strings.Contains(err.Error(), "no migration found for version 0") {
			logger.Warn().Msg("Migration failed due to missing version 0 down migration. Cleaning database and starting fresh...")

			// Drop all tables and custom types first
			logger.Info().Msg("Dropping all tables to start fresh...")
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
					logger.Warn().Err(execErr).
						Str("sql", dropSQL).
						Msg("Could not execute drop statement")
				}
			}

			// Force version to 0 (no migrations applied)
			if forceErr := m.Force(0); forceErr != nil {
				logger.Logger.Fatal().Err(forceErr).Msg("Could not force version to 0 after specific error")
			}
			logger.Info().Msg("Running all migrations from scratch...")
			if retryErr := m.Up(); retryErr != nil && retryErr != migrate.ErrNoChange {
				logger.Logger.Fatal().Err(retryErr).Msg("Could not run migrations after force (specific error)")
			}
		} else {
			logger.Logger.Fatal().Err(err).Msg("Could not run migrations")
		}
	}

	logger.Info().Msg("Database migrations completed successfully!")
	logger.Info().Msg("All migrations applied correctly!")
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
