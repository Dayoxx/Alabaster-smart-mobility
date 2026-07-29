package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// NewPostgresDB opens a connection pool using DATABASE_URL from the
// environment — never a hardcoded connection string. Set it like:
//
//	export DATABASE_URL="postgres://transport_api:change-me@localhost:5432/transport?sslmode=require"
//
// sslmode=require matters even locally — get in the habit now (T0 checklist:
// TLS on every connection, including the Postgres string itself).
func NewPostgresDB() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging db: %w", err)
	}

	return db, nil
}
