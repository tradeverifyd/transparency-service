package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteRepository implements Repository using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// SQLiteOptions holds configuration for opening a SQLite database
type SQLiteOptions struct {
	Path        string
	EnableWAL   bool
	BusyTimeout int // milliseconds
}

// NewSQLiteRepository creates a new SQLite repository
func NewSQLiteRepository(options SQLiteOptions) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", options.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize schema
	if err := initializeSQLiteSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Enable WAL mode if requested
	if options.EnableWAL {
		if err := enableSQLiteWAL(db); err != nil {
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

	return &SQLiteRepository{db: db}, nil
}

// initializeSQLiteSchema creates all tables and indexes
func initializeSQLiteSchema(db *sql.DB) error {
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

	if currentVersion.Valid && currentVersion.String == "2.0.0" {
		// Schema already initialized with new repository pattern
		return nil
	}

	// Statements table: Metadata for registered signed statements
	// Stores only metadata - actual statement blobs are in storage
	// entry_id is explicitly managed by the caller (not AUTOINCREMENT)
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS statements (
			entry_id INTEGER PRIMARY KEY,
			leaf_hash TEXT UNIQUE NOT NULL,

			iss TEXT NOT NULL,
			sub TEXT,
			cty TEXT,
			typ TEXT,

			payload_hash_alg INTEGER NOT NULL,
			payload_hash TEXT NOT NULL,
			preimage_content_type TEXT,
			payload_location TEXT,

			registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			

			entry_tile_key TEXT NOT NULL,
			entry_tile_offset INTEGER NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("failed to create statements table: %w", err)
	}

	// Create indexes for efficient querying
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_statements_leaf_hash ON statements(leaf_hash)",
		"CREATE INDEX IF NOT EXISTS idx_statements_iss ON statements(iss)",
		"CREATE INDEX IF NOT EXISTS idx_statements_sub ON statements(sub)",
		"CREATE INDEX IF NOT EXISTS idx_statements_cty ON statements(cty)",
		"CREATE INDEX IF NOT EXISTS idx_statements_typ ON statements(typ)",
		"CREATE INDEX IF NOT EXISTS idx_statements_registered_at ON statements(registered_at)",
	}

	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
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

	// Mark schema as initialized
	if _, err := db.Exec("INSERT OR REPLACE INTO schema_version (version) VALUES ('2.0.0')"); err != nil {
		return fmt.Errorf("failed to record schema version: %w", err)
	}

	return nil
}

// enableSQLiteWAL enables Write-Ahead Logging mode
func enableSQLiteWAL(db *sql.DB) error {
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

// InsertStatement inserts a new statement metadata
func (r *SQLiteRepository) InsertStatement(ctx context.Context, stmt *StatementMetadata) (int64, error) {
	// Use the EntryID from the provided statement metadata
	// The caller is responsible for managing entry IDs via GetCurrentTreeSize/SetCurrentTreeSize
	query := `
		INSERT INTO statements (
			entry_id, leaf_hash, iss, sub, cty, typ,
			payload_hash_alg, payload_hash,
			preimage_content_type, payload_location,
			 entry_tile_offset
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		stmt.EntryID,
		stmt.LeafHash,
		stmt.Iss,
		stmt.Sub,
		stmt.Cty,
		stmt.Typ,
		stmt.PayloadHashAlg,
		stmt.PayloadHash,
		stmt.PreimageContentType,
		stmt.PayloadLocation,
		stmt.TreeSizeAtRegistration,
		stmt.EntryTileKey,
		stmt.EntryTileOffset,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert statement: %w", err)
	}

	return stmt.EntryID, nil
}

// GetStatementByEntryID retrieves a statement by entry ID
func (r *SQLiteRepository) GetStatementByEntryID(ctx context.Context, entryID int64) (*StatementMetadata, error) {
	query := `
		SELECT entry_id, leaf_hash, iss, sub, cty, typ,
		       payload_hash_alg, payload_hash, preimage_content_type, payload_location,
		       registered_at,  entry_tile_offset
		FROM statements WHERE entry_id = ?
	`

	var stmt StatementMetadata
	var registeredAt string

	err := r.db.QueryRowContext(ctx, query, entryID).Scan(
		&stmt.EntryID,
		&stmt.LeafHash,
		&stmt.Iss,
		&stmt.Sub,
		&stmt.Cty,
		&stmt.Typ,
		&stmt.PayloadHashAlg,
		&stmt.PayloadHash,
		&stmt.PreimageContentType,
		&stmt.PayloadLocation,
		&registeredAt,
		&stmt.TreeSizeAtRegistration,
		&stmt.EntryTileKey,
		&stmt.EntryTileOffset,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("statement with entry ID %d not found", entryID)
		}
		return nil, fmt.Errorf("failed to get statement by entry ID: %w", err)
	}

	// Parse timestamp (SQLite returns RFC3339 format)
	stmt.RegisteredAt, err = time.Parse(time.RFC3339, registeredAt)
	if err != nil {
		// Try alternative format
		stmt.RegisteredAt, err = time.Parse("2006-01-02 15:04:05", registeredAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse registered_at: %w", err)
		}
	}

	return &stmt, nil
}

// GetStatementByLeafHash retrieves a statement by leaf hash
func (r *SQLiteRepository) GetStatementByLeafHash(ctx context.Context, leafHash string) (*StatementMetadata, error) {
	query := `
		SELECT entry_id, leaf_hash, iss, sub, cty, typ,
		       payload_hash_alg, payload_hash, preimage_content_type, payload_location,
		       registered_at,  entry_tile_offset
		FROM statements WHERE leaf_hash = ?
	`

	var stmt StatementMetadata
	var registeredAt string

	err := r.db.QueryRowContext(ctx, query, leafHash).Scan(
		&stmt.EntryID,
		&stmt.LeafHash,
		&stmt.Iss,
		&stmt.Sub,
		&stmt.Cty,
		&stmt.Typ,
		&stmt.PayloadHashAlg,
		&stmt.PayloadHash,
		&stmt.PreimageContentType,
		&stmt.PayloadLocation,
		&registeredAt,
		&stmt.TreeSizeAtRegistration,
		&stmt.EntryTileKey,
		&stmt.EntryTileOffset,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("statement with leaf hash %s not found", leafHash)
		}
		return nil, fmt.Errorf("failed to get statement by leaf hash: %w", err)
	}

	// Parse timestamp (SQLite returns RFC3339 format)
	stmt.RegisteredAt, err = time.Parse(time.RFC3339, registeredAt)
	if err != nil {
		// Try alternative format
		stmt.RegisteredAt, err = time.Parse("2006-01-02 15:04:05", registeredAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse registered_at: %w", err)
		}
	}

	return &stmt, nil
}

// QueryStatements queries statements with filters
func (r *SQLiteRepository) QueryStatements(ctx context.Context, query StatementQuery) ([]*StatementMetadata, error) {
	var conditions []string
	var params []interface{}

	if query.Iss != nil {
		conditions = append(conditions, "iss = ?")
		params = append(params, *query.Iss)
	}

	if query.Sub != nil {
		conditions = append(conditions, "sub = ?")
		params = append(params, *query.Sub)
	}

	if query.Cty != nil {
		conditions = append(conditions, "cty = ?")
		params = append(params, *query.Cty)
	}

	if query.Typ != nil {
		conditions = append(conditions, "typ = ?")
		params = append(params, *query.Typ)
	}

	if query.RegisteredAfter != nil {
		conditions = append(conditions, "registered_at >= ?")
		params = append(params, query.RegisteredAfter.Format("2006-01-02 15:04:05"))
	}

	if query.RegisteredBefore != nil {
		conditions = append(conditions, "registered_at <= ?")
		params = append(params, query.RegisteredBefore.Format("2006-01-02 15:04:05"))
	}

	sqlQuery := `
		SELECT entry_id, leaf_hash, iss, sub, cty, typ,
		       payload_hash_alg, payload_hash, preimage_content_type, payload_location,
		       registered_at,  entry_tile_offset
		FROM statements
	`

	if len(conditions) > 0 {
		sqlQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	sqlQuery += " ORDER BY registered_at DESC"

	// SQLite requires LIMIT when using OFFSET
	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", query.Limit)
		if query.Offset > 0 {
			sqlQuery += fmt.Sprintf(" OFFSET %d", query.Offset)
		}
	} else if query.Offset > 0 {
		// If only offset is specified, use a very large limit
		sqlQuery += fmt.Sprintf(" LIMIT -1 OFFSET %d", query.Offset)
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to query statements: %w", err)
	}
	defer rows.Close()

	return r.scanStatements(rows)
}

// scanStatements is a helper to scan multiple statement rows
func (r *SQLiteRepository) scanStatements(rows *sql.Rows) ([]*StatementMetadata, error) {
	var statements []*StatementMetadata

	for rows.Next() {
		var stmt StatementMetadata
		var registeredAt string

		err := rows.Scan(
			&stmt.EntryID,
			&stmt.LeafHash,
			&stmt.Iss,
			&stmt.Sub,
			&stmt.Cty,
			&stmt.Typ,
			&stmt.PayloadHashAlg,
			&stmt.PayloadHash,
			&stmt.PreimageContentType,
			&stmt.PayloadLocation,
			&registeredAt,
			&stmt.TreeSizeAtRegistration,
			&stmt.EntryTileKey,
			&stmt.EntryTileOffset,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan statement: %w", err)
		}

		// Parse timestamp (SQLite returns RFC3339 format)
		stmt.RegisteredAt, err = time.Parse(time.RFC3339, registeredAt)
		if err != nil {
			// Try alternative format
			stmt.RegisteredAt, err = time.Parse("2006-01-02 15:04:05", registeredAt)
			if err != nil {
				return nil, fmt.Errorf("failed to parse registered_at: %w", err)
			}
		}

		statements = append(statements, &stmt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating statement rows: %w", err)
	}

	return statements, nil
}

// GetCurrentTreeSize returns the current tree size
func (r *SQLiteRepository) GetCurrentTreeSize(ctx context.Context) (int64, error) {
	var treeSize int64
	err := r.db.QueryRowContext(ctx, "SELECT tree_size FROM current_tree_size WHERE id = 1").Scan(&treeSize)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get current tree size: %w", err)
	}
	return treeSize, nil
}

// SetCurrentTreeSize sets the current tree size
func (r *SQLiteRepository) SetCurrentTreeSize(ctx context.Context, size int64) error {
	if size < 0 {
		return fmt.Errorf("tree size cannot be negative: %d", size)
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE current_tree_size
		SET tree_size = ?, last_updated = CURRENT_TIMESTAMP
		WHERE id = 1
	`, size)

	if err != nil {
		return fmt.Errorf("failed to update tree size: %w", err)
	}

	return nil
}

// IncrementTreeSize atomically increments the tree size and returns the new value
func (r *SQLiteRepository) IncrementTreeSize(ctx context.Context) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var treeSize int64
	err = tx.QueryRowContext(ctx, "SELECT tree_size FROM current_tree_size WHERE id = 1").Scan(&treeSize)
	if err != nil {
		return 0, fmt.Errorf("failed to get current tree size: %w", err)
	}

	treeSize++

	_, err = tx.ExecContext(ctx, `
		UPDATE current_tree_size
		SET tree_size = ?, last_updated = CURRENT_TIMESTAMP
		WHERE id = 1
	`, treeSize)
	if err != nil {
		return 0, fmt.Errorf("failed to update tree size: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return treeSize, nil
}

func (r *SQLiteRepository) BeginTx(ctx context.Context) (Transaction, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &sqliteTx{
		tx: tx,
		db: r.db,
	}, nil
}

// Clear removes all data from the database (for development/testing)
func (r *SQLiteRepository) Clear(ctx context.Context) error {
	// Delete all statements
	if _, err := r.db.ExecContext(ctx, "DELETE FROM statements"); err != nil {
		return fmt.Errorf("failed to clear statements: %w", err)
	}

	// Reset tree size to 0
	if _, err := r.db.ExecContext(ctx, "UPDATE current_tree_size SET tree_size = 0, last_updated = CURRENT_TIMESTAMP WHERE id = 1"); err != nil {
		return fmt.Errorf("failed to reset tree size: %w", err)
	}

	return nil
}

// Close closes the database connection
func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

// sqliteTx implements Transaction for SQLite
type sqliteTx struct {
	tx *sql.Tx
	db *sql.DB // Keep reference to underlying db for queries
}

// Commit commits the transaction
func (t *sqliteTx) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *sqliteTx) Rollback() error {
	return t.tx.Rollback()
}

// All repository methods for transaction - delegate to SQLiteRepository but use tx
// For brevity, implementing key methods. In production, implement all Repository interface methods.

func (t *sqliteTx) InsertStatement(ctx context.Context, stmt *StatementMetadata) (int64, error) {
	query := `
		INSERT INTO statements (
			entry_id, leaf_hash, iss, sub, cty, typ,
			payload_hash_alg, payload_hash,
			preimage_content_type, payload_location,
			 entry_tile_offset
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := t.tx.ExecContext(ctx, query,
		stmt.EntryID, stmt.LeafHash, stmt.Iss, stmt.Sub, stmt.Cty, stmt.Typ,
		stmt.PayloadHashAlg, stmt.PayloadHash,
		stmt.PreimageContentType, stmt.PayloadLocation,
		stmt.TreeSizeAtRegistration, stmt.EntryTileKey, stmt.EntryTileOffset,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert statement: %w", err)
	}

	return stmt.EntryID, nil
}

func (t *sqliteTx) GetStatementByEntryID(ctx context.Context, entryID int64) (*StatementMetadata, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

func (t *sqliteTx) GetStatementByLeafHash(ctx context.Context, leafHash string) (*StatementMetadata, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

func (t *sqliteTx) QueryStatements(ctx context.Context, query StatementQuery) ([]*StatementMetadata, error) {
	return nil, fmt.Errorf("not implemented in transaction")
}

func (t *sqliteTx) GetCurrentTreeSize(ctx context.Context) (int64, error) {
	var treeSize int64
	err := t.tx.QueryRowContext(ctx, "SELECT tree_size FROM current_tree_size WHERE id = 1").Scan(&treeSize)
	return treeSize, err
}

func (t *sqliteTx) SetCurrentTreeSize(ctx context.Context, size int64) error {
	_, err := t.tx.ExecContext(ctx, `
		UPDATE current_tree_size SET tree_size = ?, last_updated = CURRENT_TIMESTAMP WHERE id = 1
	`, size)
	return err
}

func (t *sqliteTx) IncrementTreeSize(ctx context.Context) (int64, error) {
	var treeSize int64
	err := t.tx.QueryRowContext(ctx, "SELECT tree_size FROM current_tree_size WHERE id = 1").Scan(&treeSize)
	if err != nil {
		return 0, err
	}

	treeSize++

	_, err = t.tx.ExecContext(ctx, `
		UPDATE current_tree_size SET tree_size = ?, last_updated = CURRENT_TIMESTAMP WHERE id = 1
	`, treeSize)
	if err != nil {
		return 0, err
	}

	return treeSize, nil
}

func (t *sqliteTx) BeginTx(ctx context.Context) (Transaction, error) {
	return nil, fmt.Errorf("nested transactions not supported")
}

func (t *sqliteTx) Clear(ctx context.Context) error {
	// Delete all statements
	if _, err := t.tx.ExecContext(ctx, "DELETE FROM statements"); err != nil {
		return fmt.Errorf("failed to clear statements: %w", err)
	}

	// Reset tree size to 0
	if _, err := t.tx.ExecContext(ctx, "UPDATE current_tree_size SET tree_size = 0, last_updated = CURRENT_TIMESTAMP WHERE id = 1"); err != nil {
		return fmt.Errorf("failed to reset tree size: %w", err)
	}

	return nil
}

func (t *sqliteTx) Close() error {
	return fmt.Errorf("cannot close transaction directly, use Commit or Rollback")
}
