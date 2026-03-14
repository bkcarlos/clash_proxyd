package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var embeddedSchema string

// DB holds the database connection
type DB struct {
	*sql.DB
}

// NewDB creates a new database connection
func NewDB(path string, foreignKeys bool) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection parameters
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	// Enable foreign keys if requested
	if foreignKeys {
		if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
			return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
		}
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	wrapped := &DB{db}

	// Auto-initialize schema and seed defaults on every open.
	// All DDL statements use CREATE IF NOT EXISTS and INSERT OR IGNORE,
	// so this is safe to run against an existing database.
	if err := wrapped.InitSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return wrapped, nil
}

// InitSchema applies the embedded schema SQL to the database.
// Safe to call on an existing database (uses CREATE IF NOT EXISTS / INSERT OR IGNORE).
func (db *DB) InitSchema() error {
	if _, err := db.Exec(embeddedSchema); err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
