// Package database provides SQLite database operations for the transparency service
package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// DatabaseOptions holds configuration for opening a database
type DatabaseOptions struct {
	Path         string
	EnableWAL    bool
	BusyTimeout  int // milliseconds
}

// OpenDatabase opens a SQLite database connection with the specified options
// and initializes the schema if needed
func OpenDatabase(options DatabaseOptions) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", options.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize schema
	if err := initializeSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Enable WAL mode if requested (default: true)
	if options.EnableWAL {
		if err := enableWAL(db); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to enable WAL: %w", err)
		}
	}

	// Set busy timeout
	if options.BusyTimeout > 0 {
		if _, err := db.Exec(fmt.Sprintf("PRAGMA busy_timeout = %d", options.BusyTimeout)); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set busy timeout: %w", err)
		}
	}

	return db, nil
}

// initializeSchema creates all tables, indexes, and initial data
func initializeSchema(db *sql.DB) error {
	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Schema versioning table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create schema_version table: %w", err)
	}

	// Check if schema is already initialized
	var currentVersion sql.NullString
	err := db.QueryRow("SELECT version FROM schema_version ORDER BY applied_at DESC LIMIT 1").Scan(&currentVersion)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check schema version: %w", err)
	}

	if currentVersion.Valid && currentVersion.String == "1.0.0" {
		// Schema already initialized
		return nil
	}

	// Statements table: Metadata for registered signed statements
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS statements (
			entry_id INTEGER PRIMARY KEY AUTOINCREMENT,
			statement_hash TEXT UNIQUE NOT NULL,

			iss TEXT NOT NULL,
			sub TEXT,
			cty TEXT,
			typ TEXT,

			payload_hash_alg INTEGER NOT NULL,
			payload_hash TEXT NOT NULL,
			preimage_content_type TEXT,
			payload_location TEXT,

			registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			tree_size_at_registration INTEGER NOT NULL,

			entry_tile_key TEXT NOT NULL,
			entry_tile_offset INTEGER NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("failed to create statements table: %w", err)
	}

	// Create indexes for efficient querying
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_statements_iss ON statements(iss)",
		"CREATE INDEX IF NOT EXISTS idx_statements_sub ON statements(sub)",
		"CREATE INDEX IF NOT EXISTS idx_statements_cty ON statements(cty)",
		"CREATE INDEX IF NOT EXISTS idx_statements_typ ON statements(typ)",
		"CREATE INDEX IF NOT EXISTS idx_statements_registered_at ON statements(registered_at)",
		"CREATE INDEX IF NOT EXISTS idx_statements_hash ON statements(statement_hash)",
	}

	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Receipts table: Pointers to receipt objects in storage
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS receipts (
			entry_id INTEGER PRIMARY KEY,
			receipt_hash TEXT UNIQUE NOT NULL,
			storage_key TEXT UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

			tree_size INTEGER NOT NULL,
			leaf_index INTEGER NOT NULL,

			FOREIGN KEY (entry_id) REFERENCES statements(entry_id)
		)
	`); err != nil {
		return fmt.Errorf("failed to create receipts table: %w", err)
	}

	// Tiles table: Merkle tree tile metadata
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tiles (
			tile_id INTEGER PRIMARY KEY AUTOINCREMENT,
			level INTEGER NOT NULL,
			tile_index INTEGER NOT NULL,

			storage_key TEXT UNIQUE NOT NULL,

			is_partial BOOLEAN DEFAULT FALSE,
			width INTEGER,

			tile_hash TEXT NOT NULL,

			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

			UNIQUE(level, tile_index)
		)
	`); err != nil {
		return fmt.Errorf("failed to create tiles table: %w", err)
	}

	// Index for tile lookups
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_tiles_level_index ON tiles(level, tile_index)"); err != nil {
		return fmt.Errorf("failed to create tiles index: %w", err)
	}

	// Tree state: Current Merkle tree state
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tree_state (
			tree_size INTEGER PRIMARY KEY,
			root_hash TEXT NOT NULL,
			checkpoint_storage_key TEXT NOT NULL,
			checkpoint_signed_note TEXT NOT NULL,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create tree_state table: %w", err)
	}

	// Current tree size (singleton table)
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS current_tree_size (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			tree_size INTEGER NOT NULL DEFAULT 0,
			last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create current_tree_size table: %w", err)
	}

	// Initialize with tree size 0
	if _, err := db.Exec("INSERT OR IGNORE INTO current_tree_size (id, tree_size) VALUES (1, 0)"); err != nil {
		return fmt.Errorf("failed to initialize current_tree_size: %w", err)
	}

	// Service configuration table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS service_config (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create service_config table: %w", err)
	}

	// Initialize service config
	configDefaults := map[string]string{
		"service_url":           "http://localhost:3000",
		"tile_height":           "8",
		"checkpoint_frequency":  "1000",
		"hash_algorithm":        "-16",  // SHA-256
		"signature_algorithm":   "-7",   // ES256
	}

	stmt, err := db.Prepare("INSERT OR IGNORE INTO service_config (key, value) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare config insert: %w", err)
	}
	defer stmt.Close()

	for key, value := range configDefaults {
		if _, err := stmt.Exec(key, value); err != nil {
			return fmt.Errorf("failed to insert config %s: %w", key, err)
		}
	}

	// Service keys: Transparency service signing keys
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS service_keys (
			kid TEXT PRIMARY KEY,
			public_key_jwk TEXT NOT NULL,
			private_key_pem TEXT NOT NULL,
			algorithm TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			active BOOLEAN DEFAULT TRUE
		)
	`); err != nil {
		return fmt.Errorf("failed to create service_keys table: %w", err)
	}

	// Mark schema as initialized
	if _, err := db.Exec("INSERT INTO schema_version (version) VALUES ('1.0.0')"); err != nil {
		return fmt.Errorf("failed to record schema version: %w", err)
	}

	return nil
}

// enableWAL enables Write-Ahead Logging mode
// Improves concurrent read/write performance
func enableWAL(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 5000",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute %s: %w", pragma, err)
		}
	}

	return nil
}

// CloseDatabase closes the database connection
func CloseDatabase(db *sql.DB) error {
	return db.Close()
}
