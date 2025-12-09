package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"

	"loveguru/internal/config"
)

func NewDB(cfg *config.DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Log successful connection
	log.Printf("Successfully connected to database: %s:%d/%s", cfg.Host, cfg.Port, cfg.DBName)

	// Test database version
	var version string
	if err := db.QueryRow("SELECT version()").Scan(&version); err != nil {
		log.Printf("Warning: Failed to get database version: %v", err)
	} else {
		log.Printf("Database version: %s", version)
	}

	return db, nil
}
