// Package repository provides database abstraction for the transparency service
// This follows the Repository pattern to decouple business logic from data access
package repository

import (
	"context"
	"time"
)

// StatementMetadata represents metadata for a registered signed statement
// Stored in database for efficient querying - actual statement blobs are in storage
type StatementMetadata struct {
	EntryID                int64     // Unique entry ID in the log
	LeafHash               string    // Hex-encoded leaf hash stored in the tile (MTH leaf)
	Iss                    string    // Issuer URL
	Sub                    *string   // Subject (optional)
	Cty                    *string   // Content type (optional)
	Typ                    *string   // Type (optional)
	PayloadHashAlg         int       // Hash algorithm identifier
	PayloadHash            string    // Hex-encoded payload hash
	PreimageContentType    *string   // Preimage content type (optional)
	PayloadLocation        *string   // Payload location (optional)
	RegisteredAt           time.Time // Registration timestamp
	TreeSizeAtRegistration int64     // Tree size when registered
	EntryTileKey           string    // Storage key for entry tile containing this leaf
	EntryTileOffset        int       // Offset within entry tile where this leaf hash is stored
}


// StatementQuery defines filters for querying statement metadata
type StatementQuery struct {
	Iss              *string    // Filter by issuer
	Sub              *string    // Filter by subject
	Cty              *string    // Filter by content type
	Typ              *string    // Filter by type
	RegisteredAfter  *time.Time // Filter by registration date
	RegisteredBefore *time.Time // Filter by registration date
	Limit            int        // Maximum results to return
	Offset           int        // Offset for pagination
}

// Repository defines the interface for transparency service data access
// Implementations support SQLite, MongoDB, PostgreSQL, etc.
type Repository interface {
	// Statement operations
	InsertStatement(ctx context.Context, stmt *StatementMetadata) (int64, error)
	GetStatementByEntryID(ctx context.Context, entryID int64) (*StatementMetadata, error)
	GetStatementByLeafHash(ctx context.Context, leafHash string) (*StatementMetadata, error)
	QueryStatements(ctx context.Context, query StatementQuery) ([]*StatementMetadata, error)

	// Tree size operations
	GetCurrentTreeSize(ctx context.Context) (int64, error)
	SetCurrentTreeSize(ctx context.Context, size int64) error
	IncrementTreeSize(ctx context.Context) (int64, error) // Atomic increment, returns new size

	// Transaction support
	BeginTx(ctx context.Context) (Transaction, error)

	// Data management
	Clear(ctx context.Context) error // Clear all data (for development/testing)

	// Lifecycle
	Close() error
}

// Transaction represents a database transaction
// Supports atomic operations across multiple repository calls
type Transaction interface {
	Repository
	Commit() error
	Rollback() error
}
