package database

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
)

// Statement represents metadata for a registered signed statement
type Statement struct {
	EntryID                 int64   `json:"entry_id,omitempty"`
	StatementHash           string  `json:"statement_hash"`
	Iss                     string  `json:"iss"`
	Sub                     *string `json:"sub"`
	Cty                     *string `json:"cty"`
	Typ                     *string `json:"typ"`
	PayloadHashAlg          int     `json:"payload_hash_alg"`
	PayloadHash             string  `json:"payload_hash"`
	PreimageContentType     *string `json:"preimage_content_type"`
	PayloadLocation         *string `json:"payload_location"`
	RegisteredAt            string  `json:"registered_at,omitempty"`
	TreeSizeAtRegistration  int64   `json:"tree_size_at_registration"`
	EntryTileKey            string  `json:"entry_tile_key"`
	EntryTileOffset         int     `json:"entry_tile_offset"`
}

// StatementQueryFilters holds filters for querying statements
type StatementQueryFilters struct {
	Iss              *string
	Sub              *string
	Cty              *string
	Typ              *string
	RegisteredAfter  *string
	RegisteredBefore *string
}

// InsertStatement inserts a new statement into the database
// Returns the auto-generated entry ID
func InsertStatement(db *sql.DB, statement Statement) (int64, error) {
	stmt, err := db.Prepare(`
		INSERT INTO statements (
			statement_hash, iss, sub, cty, typ,
			payload_hash_alg, payload_hash,
			preimage_content_type, payload_location,
			tree_size_at_registration, entry_tile_key, entry_tile_offset
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		statement.StatementHash,
		statement.Iss,
		statement.Sub,
		statement.Cty,
		statement.Typ,
		statement.PayloadHashAlg,
		statement.PayloadHash,
		statement.PreimageContentType,
		statement.PayloadLocation,
		statement.TreeSizeAtRegistration,
		statement.EntryTileKey,
		statement.EntryTileOffset,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert statement: %w", err)
	}

	entryID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	return entryID, nil
}

// FindStatementsByIssuer finds all statements by issuer URL
func FindStatementsByIssuer(db *sql.DB, iss string) ([]Statement, error) {
	rows, err := db.Query(`
		SELECT entry_id, statement_hash, iss, sub, cty, typ,
		       payload_hash_alg, payload_hash, preimage_content_type, payload_location,
		       registered_at, tree_size_at_registration, entry_tile_key, entry_tile_offset
		FROM statements WHERE iss = ? ORDER BY registered_at DESC
	`, iss)
	if err != nil {
		return nil, fmt.Errorf("failed to query statements by issuer: %w", err)
	}
	defer rows.Close()

	return scanStatements(rows)
}

// FindStatementsBySubject finds all statements by subject
func FindStatementsBySubject(db *sql.DB, sub string) ([]Statement, error) {
	rows, err := db.Query(`
		SELECT entry_id, statement_hash, iss, sub, cty, typ,
		       payload_hash_alg, payload_hash, preimage_content_type, payload_location,
		       registered_at, tree_size_at_registration, entry_tile_key, entry_tile_offset
		FROM statements WHERE sub = ? ORDER BY registered_at DESC
	`, sub)
	if err != nil {
		return nil, fmt.Errorf("failed to query statements by subject: %w", err)
	}
	defer rows.Close()

	return scanStatements(rows)
}

// FindStatementsByContentType finds all statements by content type
func FindStatementsByContentType(db *sql.DB, cty string) ([]Statement, error) {
	rows, err := db.Query(`
		SELECT entry_id, statement_hash, iss, sub, cty, typ,
		       payload_hash_alg, payload_hash, preimage_content_type, payload_location,
		       registered_at, tree_size_at_registration, entry_tile_key, entry_tile_offset
		FROM statements WHERE cty = ? ORDER BY registered_at DESC
	`, cty)
	if err != nil {
		return nil, fmt.Errorf("failed to query statements by content type: %w", err)
	}
	defer rows.Close()

	return scanStatements(rows)
}

// FindStatementsByType finds all statements by type
func FindStatementsByType(db *sql.DB, typ string) ([]Statement, error) {
	rows, err := db.Query(`
		SELECT entry_id, statement_hash, iss, sub, cty, typ,
		       payload_hash_alg, payload_hash, preimage_content_type, payload_location,
		       registered_at, tree_size_at_registration, entry_tile_key, entry_tile_offset
		FROM statements WHERE typ = ? ORDER BY registered_at DESC
	`, typ)
	if err != nil {
		return nil, fmt.Errorf("failed to query statements by type: %w", err)
	}
	defer rows.Close()

	return scanStatements(rows)
}

// FindStatementsByDateRange finds statements within a date range
func FindStatementsByDateRange(db *sql.DB, startDate, endDate string) ([]Statement, error) {
	rows, err := db.Query(`
		SELECT entry_id, statement_hash, iss, sub, cty, typ,
		       payload_hash_alg, payload_hash, preimage_content_type, payload_location,
		       registered_at, tree_size_at_registration, entry_tile_key, entry_tile_offset
		FROM statements
		WHERE registered_at BETWEEN ? AND ?
		ORDER BY registered_at DESC
	`, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query statements by date range: %w", err)
	}
	defer rows.Close()

	return scanStatements(rows)
}

// FindStatementsBy finds statements using combined filters
func FindStatementsBy(db *sql.DB, filters StatementQueryFilters) ([]Statement, error) {
	var conditions []string
	var params []interface{}

	if filters.Iss != nil {
		conditions = append(conditions, "iss = ?")
		params = append(params, *filters.Iss)
	}

	if filters.Sub != nil {
		conditions = append(conditions, "sub = ?")
		params = append(params, *filters.Sub)
	}

	if filters.Cty != nil {
		conditions = append(conditions, "cty = ?")
		params = append(params, *filters.Cty)
	}

	if filters.Typ != nil {
		conditions = append(conditions, "typ = ?")
		params = append(params, *filters.Typ)
	}

	if filters.RegisteredAfter != nil {
		conditions = append(conditions, "registered_at >= ?")
		params = append(params, *filters.RegisteredAfter)
	}

	if filters.RegisteredBefore != nil {
		conditions = append(conditions, "registered_at <= ?")
		params = append(params, *filters.RegisteredBefore)
	}

	query := `
		SELECT entry_id, statement_hash, iss, sub, cty, typ,
		       payload_hash_alg, payload_hash, preimage_content_type, payload_location,
		       registered_at, tree_size_at_registration, entry_tile_key, entry_tile_offset
		FROM statements
	`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY registered_at DESC"

	rows, err := db.Query(query, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to query statements with filters: %w", err)
	}
	defer rows.Close()

	return scanStatements(rows)
}

// GetStatementByEntryID retrieves a statement by its entry ID
func GetStatementByEntryID(db *sql.DB, entryID int64) (*Statement, error) {
	var stmt Statement
	err := db.QueryRow(`
		SELECT entry_id, statement_hash, iss, sub, cty, typ,
		       payload_hash_alg, payload_hash, preimage_content_type, payload_location,
		       registered_at, tree_size_at_registration, entry_tile_key, entry_tile_offset
		FROM statements WHERE entry_id = ?
	`, entryID).Scan(
		&stmt.EntryID,
		&stmt.StatementHash,
		&stmt.Iss,
		&stmt.Sub,
		&stmt.Cty,
		&stmt.Typ,
		&stmt.PayloadHashAlg,
		&stmt.PayloadHash,
		&stmt.PreimageContentType,
		&stmt.PayloadLocation,
		&stmt.RegisteredAt,
		&stmt.TreeSizeAtRegistration,
		&stmt.EntryTileKey,
		&stmt.EntryTileOffset,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get statement by entry ID: %w", err)
	}

	return &stmt, nil
}

// GetStatementByHash retrieves a statement by its hash
func GetStatementByHash(db *sql.DB, hash string) (*Statement, error) {
	var stmt Statement
	err := db.QueryRow(`
		SELECT entry_id, statement_hash, iss, sub, cty, typ,
		       payload_hash_alg, payload_hash, preimage_content_type, payload_location,
		       registered_at, tree_size_at_registration, entry_tile_key, entry_tile_offset
		FROM statements WHERE statement_hash = ?
	`, hash).Scan(
		&stmt.EntryID,
		&stmt.StatementHash,
		&stmt.Iss,
		&stmt.Sub,
		&stmt.Cty,
		&stmt.Typ,
		&stmt.PayloadHashAlg,
		&stmt.PayloadHash,
		&stmt.PreimageContentType,
		&stmt.PayloadLocation,
		&stmt.RegisteredAt,
		&stmt.TreeSizeAtRegistration,
		&stmt.EntryTileKey,
		&stmt.EntryTileOffset,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get statement by hash: %w", err)
	}

	return &stmt, nil
}

// SaveStatement stores the raw COSE Sign1 bytes in the database
func SaveStatement(db *sql.DB, entryID string, statementBytes []byte, leafHash []byte, leafIndex int64) error {
	// Create statement_blobs table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS statement_blobs (
			entry_id TEXT PRIMARY KEY,
			data BLOB NOT NULL,
			leaf_hash TEXT NOT NULL,
			leaf_index INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create statement_blobs table: %w", err)
	}

	// Convert leaf hash to base64
	leafHashBase64 := base64.StdEncoding.EncodeToString(leafHash)

	stmt, err := db.Prepare(`
		INSERT INTO statement_blobs (entry_id, data, leaf_hash, leaf_index)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare save statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(entryID, statementBytes, leafHashBase64, leafIndex)
	if err != nil {
		return fmt.Errorf("failed to save statement: %w", err)
	}

	return nil
}

// GetStatementBlob retrieves the raw COSE Sign1 bytes by entry ID
func GetStatementBlob(db *sql.DB, entryID string) ([]byte, error) {
	var data []byte
	err := db.QueryRow(`
		SELECT data FROM statement_blobs WHERE entry_id = ?
	`, entryID).Scan(&data)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get statement blob: %w", err)
	}

	return data, nil
}

// scanStatements is a helper function to scan multiple statement rows
func scanStatements(rows *sql.Rows) ([]Statement, error) {
	var statements []Statement

	for rows.Next() {
		var stmt Statement
		err := rows.Scan(
			&stmt.EntryID,
			&stmt.StatementHash,
			&stmt.Iss,
			&stmt.Sub,
			&stmt.Cty,
			&stmt.Typ,
			&stmt.PayloadHashAlg,
			&stmt.PayloadHash,
			&stmt.PreimageContentType,
			&stmt.PayloadLocation,
			&stmt.RegisteredAt,
			&stmt.TreeSizeAtRegistration,
			&stmt.EntryTileKey,
			&stmt.EntryTileOffset,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan statement: %w", err)
		}
		statements = append(statements, stmt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating statement rows: %w", err)
	}

	return statements, nil
}

// FindStatementByEntryID finds a statement by entry ID
func FindStatementByEntryID(db *sql.DB, entryID int64) (*Statement, error) {
	var stmt Statement
	err := db.QueryRow(`
		SELECT entry_id, statement_hash, iss, sub, cty, typ,
			   payload_hash_alg, payload_hash,
			   preimage_content_type, payload_location,
			   registered_at, tree_size_at_registration,
			   entry_tile_key, entry_tile_offset
		FROM statements
		WHERE entry_id = ?
	`, entryID).Scan(
		&stmt.EntryID,
		&stmt.StatementHash,
		&stmt.Iss,
		&stmt.Sub,
		&stmt.Cty,
		&stmt.Typ,
		&stmt.PayloadHashAlg,
		&stmt.PayloadHash,
		&stmt.PreimageContentType,
		&stmt.PayloadLocation,
		&stmt.RegisteredAt,
		&stmt.TreeSizeAtRegistration,
		&stmt.EntryTileKey,
		&stmt.EntryTileOffset,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("statement not found")
		}
		return nil, fmt.Errorf("failed to query statement: %w", err)
	}

	return &stmt, nil
}
