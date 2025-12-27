// Package storage provides persistence for QuantumLife.
package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate runs all pending migrations
func (db *DB) Migrate() error {
	// Create migrations table if not exists
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS _migrations (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := db.getAppliedMigrations()
	if err != nil {
		return err
	}

	// Get available migrations
	migrations, err := db.getAvailableMigrations()
	if err != nil {
		return err
	}

	// Apply pending migrations
	for _, m := range migrations {
		if _, ok := applied[m.name]; ok {
			continue // Already applied
		}

		if err := db.applyMigration(m); err != nil {
			return fmt.Errorf("migration %s failed: %w", m.name, err)
		}

		fmt.Printf("Applied migration: %s\n", m.name)
	}

	return nil
}

type migration struct {
	name    string
	content string
}

func (db *DB) getAppliedMigrations() (map[string]bool, error) {
	rows, err := db.conn.Query("SELECT name FROM _migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}

	return applied, rows.Err()
}

func (db *DB) getAvailableMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations: %w", err)
	}

	var migrations []migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := fs.ReadFile(migrationsFS, "migrations/"+entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, migration{
			name:    entry.Name(),
			content: string(content),
		})
	}

	// Sort by name (which starts with number)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].name < migrations[j].name
	})

	return migrations, nil
}

func (db *DB) applyMigration(m migration) error {
	return db.Transaction(func(tx *sql.Tx) error {
		// Execute migration
		if _, err := tx.Exec(m.content); err != nil {
			return err
		}

		// Record migration
		_, err := tx.Exec("INSERT INTO _migrations (name) VALUES (?)", m.name)
		return err
	})
}
