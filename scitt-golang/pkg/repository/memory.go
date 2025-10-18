package repository

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryRepository is an in-memory implementation of Repository
// Useful for testing and development
type MemoryRepository struct {
	mu              sync.RWMutex
	statements      map[int64]*StatementMetadata // keyed by EntryID
	currentTreeSize int64
	nextEntryID     int64
}

// NewMemoryRepository creates a new in-memory repository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		statements:      make(map[int64]*StatementMetadata),
		currentTreeSize: 0,
		nextEntryID:     0,
	}
}

// InsertStatement inserts a new statement metadata
func (r *MemoryRepository) InsertStatement(ctx context.Context, stmt *StatementMetadata) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate leaf hash
	for _, existing := range r.statements {
		if existing.LeafHash == stmt.LeafHash {
			return 0, fmt.Errorf("statement with leaf hash %s already exists", stmt.LeafHash)
		}
	}

	// Use the EntryID from the provided statement metadata
	// The caller is responsible for managing entry IDs via GetCurrentTreeSize/SetCurrentTreeSize
	entryID := stmt.EntryID

	// Create copy with timestamp
	newStmt := *stmt
	if newStmt.RegisteredAt.IsZero() {
		newStmt.RegisteredAt = time.Now()
	}

	// Store
	r.statements[entryID] = &newStmt

	return entryID, nil
}

// GetStatementByEntryID retrieves a statement by entry ID
func (r *MemoryRepository) GetStatementByEntryID(ctx context.Context, entryID int64) (*StatementMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stmt, exists := r.statements[entryID]
	if !exists {
		return nil, fmt.Errorf("statement with entry ID %d not found", entryID)
	}

	// Return copy
	result := *stmt
	return &result, nil
}

// GetStatementByLeafHash retrieves a statement by leaf hash
func (r *MemoryRepository) GetStatementByLeafHash(ctx context.Context, leafHash string) (*StatementMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, stmt := range r.statements {
		if stmt.LeafHash == leafHash {
			// Return copy
			result := *stmt
			return &result, nil
		}
	}

	return nil, fmt.Errorf("statement with leaf hash %s not found", leafHash)
}

// QueryStatements queries statements with filters
func (r *MemoryRepository) QueryStatements(ctx context.Context, query StatementQuery) ([]*StatementMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make([]*StatementMetadata, 0)

	for _, stmt := range r.statements {
		// Apply filters
		if query.Iss != nil && stmt.Iss != *query.Iss {
			continue
		}
		if query.Sub != nil && (stmt.Sub == nil || *stmt.Sub != *query.Sub) {
			continue
		}
		if query.Cty != nil && (stmt.Cty == nil || *stmt.Cty != *query.Cty) {
			continue
		}
		if query.Typ != nil && (stmt.Typ == nil || *stmt.Typ != *query.Typ) {
			continue
		}
		if query.RegisteredAfter != nil && stmt.RegisteredAt.Before(*query.RegisteredAfter) {
			continue
		}
		if query.RegisteredBefore != nil && stmt.RegisteredAt.After(*query.RegisteredBefore) {
			continue
		}

		// Add to results
		result := *stmt
		results = append(results, &result)
	}

	// Apply limit and offset
	if query.Offset > 0 {
		if query.Offset >= len(results) {
			return []*StatementMetadata{}, nil
		}
		results = results[query.Offset:]
	}

	if query.Limit > 0 && query.Limit < len(results) {
		results = results[:query.Limit]
	}

	return results, nil
}

// GetCurrentTreeSize returns the current tree size
func (r *MemoryRepository) GetCurrentTreeSize(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.currentTreeSize, nil
}

// SetCurrentTreeSize sets the current tree size
func (r *MemoryRepository) SetCurrentTreeSize(ctx context.Context, size int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if size < 0 {
		return fmt.Errorf("tree size cannot be negative: %d", size)
	}

	r.currentTreeSize = size
	return nil
}

// IncrementTreeSize atomically increments the tree size and returns the new value
func (r *MemoryRepository) IncrementTreeSize(ctx context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.currentTreeSize++
	return r.currentTreeSize, nil
}

// BeginTx begins a transaction (not supported in memory implementation)
func (r *MemoryRepository) BeginTx(ctx context.Context) (Transaction, error) {
	return nil, fmt.Errorf("transactions not supported in memory repository")
}

// Clear removes all data from the repository (for development/testing)
func (r *MemoryRepository) Clear(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.statements = make(map[int64]*StatementMetadata)
	r.currentTreeSize = 0
	r.nextEntryID = 0

	return nil
}

// Close closes the repository
func (r *MemoryRepository) Close() error {
	return nil
}
