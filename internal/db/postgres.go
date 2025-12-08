package db

import (
	"database/sql"
	"fmt"

	"loveguru/internal/config"
)

func NewDB(cfg *config.DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("pgx://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
