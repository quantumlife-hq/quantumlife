// Package storage provides persistence for QuantumLife.
package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection
type DB struct {
	conn     *sql.DB
	path     string
	isMemory bool
}

// Config for database initialization
type Config struct {
	Path       string // Path to database file
	InMemory   bool   // Use in-memory database (for testing)
	Passphrase string // Encryption passphrase (for SQLCipher - future)
}

// Open opens or creates a SQLite database
func Open(cfg Config) (*DB, error) {
	var dsn string
	var isMemory bool

	if cfg.InMemory {
		dsn = ":memory:?cache=shared"
		isMemory = true
	} else {
		// Ensure directory exists
		dir := filepath.Dir(cfg.Path)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
		dsn = cfg.Path
		isMemory = false
	}

	// Open connection
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection
	conn.SetMaxOpenConns(1) // SQLite doesn't handle concurrent writes well

	// Enable WAL mode for better concurrency
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable WAL: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return &DB{
		conn:     conn,
		path:     cfg.Path,
		isMemory: isMemory,
	}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying sql.DB for direct access
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// Transaction executes a function within a transaction
func (db *DB) Transaction(fn func(tx *sql.Tx) error) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
