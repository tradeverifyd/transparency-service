package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func createTempSQLiteRepo(t *testing.T) (*SQLiteRepository, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "sqlite-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")

	repo, err := NewSQLiteRepository(SQLiteOptions{
		Path:        dbPath,
		EnableWAL:   true,
		BusyTimeout: 5000,
	})
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create SQLite repository: %v", err)
	}

	cleanup := func() {
		repo.Close()
		os.RemoveAll(tmpDir)
	}

	return repo, cleanup
}

func TestSQLiteRepository_InsertStatement(t *testing.T) {
	t.Parallel()

	repo, cleanup := createTempSQLiteRepo(t)
	defer cleanup()

	ctx := context.Background()

	issuer := "https://example.com"
	subject := "test-subject"
	contentType := "application/json"

	// Get next entry ID
	nextID, err := repo.IncrementTreeSize(ctx)
	if err != nil {
		t.Fatalf("Failed to increment tree size: %v", err)
	}

	stmt := &StatementMetadata{
		EntryID:                nextID,
		LeafHash:               "abcd1234",
		Iss:                    issuer,
		Sub:                    &subject,
		Cty:                    &contentType,
		PayloadHashAlg:         -16,
		PayloadHash:            "ef567890",
		TreeSizeAtRegistration: 0,
		EntryTileKey:           "tile/x000/x000/000",
		EntryTileOffset:        0,
	}

	entryID, err := repo.InsertStatement(ctx, stmt)
	if err != nil {
		t.Fatalf("Failed to insert statement: %v", err)
	}

	if entryID < 1 {
		t.Errorf("Expected entry ID >= 1, got %d", entryID)
	}

	// Verify it was stored
	retrieved, err := repo.GetStatementByEntryID(ctx, entryID)
	if err != nil {
		t.Fatalf("Failed to retrieve statement: %v", err)
	}

	if retrieved.LeafHash != stmt.LeafHash {
		t.Errorf("Expected leaf hash %s, got %s", stmt.LeafHash, retrieved.LeafHash)
	}
	if retrieved.Iss != stmt.Iss {
		t.Errorf("Expected issuer %s, got %s", stmt.Iss, retrieved.Iss)
	}
}

func TestSQLiteRepository_InsertStatement_DuplicateLeafHash(t *testing.T) {
	t.Parallel()

	repo, cleanup := createTempSQLiteRepo(t)
	defer cleanup()

	ctx := context.Background()

	stmt := &StatementMetadata{
		LeafHash:               "abcd1234",
		Iss:                    "https://example.com",
		PayloadHashAlg:         -16,
		PayloadHash:            "ef567890",
		TreeSizeAtRegistration: 0,
		EntryTileKey:           "tile/x000/x000/000",
		EntryTileOffset:        0,
	}

	_, err := repo.InsertStatement(ctx, stmt)
	if err != nil {
		t.Fatalf("Failed to insert first statement: %v", err)
	}

	// Try to insert duplicate
	_, err = repo.InsertStatement(ctx, stmt)
	if err == nil {
		t.Fatal("Expected error when inserting duplicate leaf hash")
	}
}

func TestSQLiteRepository_GetStatementByLeafHash(t *testing.T) {
	t.Parallel()

	repo, cleanup := createTempSQLiteRepo(t)
	defer cleanup()

	ctx := context.Background()

	leafHash := "abcd1234"
	stmt := &StatementMetadata{
		LeafHash:               leafHash,
		Iss:                    "https://example.com",
		PayloadHashAlg:         -16,
		PayloadHash:            "ef567890",
		TreeSizeAtRegistration: 0,
		EntryTileKey:           "tile/x000/x000/000",
		EntryTileOffset:        0,
	}

	_, err := repo.InsertStatement(ctx, stmt)
	if err != nil {
		t.Fatalf("Failed to insert statement: %v", err)
	}

	// Retrieve by leaf hash
	retrieved, err := repo.GetStatementByLeafHash(ctx, leafHash)
	if err != nil {
		t.Fatalf("Failed to retrieve statement by leaf hash: %v", err)
	}

	if retrieved.LeafHash != leafHash {
		t.Errorf("Expected leaf hash %s, got %s", leafHash, retrieved.LeafHash)
	}
}

func TestSQLiteRepository_GetStatementByLeafHash_NotFound(t *testing.T) {
	t.Parallel()

	repo, cleanup := createTempSQLiteRepo(t)
	defer cleanup()

	ctx := context.Background()

	_, err := repo.GetStatementByLeafHash(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Expected error when retrieving non-existent leaf hash")
	}
}

func TestSQLiteRepository_QueryStatements_ByIssuer(t *testing.T) {
	t.Parallel()

	repo, cleanup := createTempSQLiteRepo(t)
	defer cleanup()

	ctx := context.Background()

	issuer1 := "https://example.com"
	issuer2 := "https://other.com"

	// Insert statements with different issuers
	for i := 0; i < 3; i++ {
		nextID, err := repo.IncrementTreeSize(ctx)
		if err != nil {
			t.Fatalf("Failed to increment tree size: %v", err)
		}
		stmt := &StatementMetadata{
			EntryID:                nextID,
			LeafHash:               string(rune('a' + i)),
			Iss:                    issuer1,
			PayloadHashAlg:         -16,
			PayloadHash:            string(rune('x' + i)),
			TreeSizeAtRegistration: int64(i),
			EntryTileKey:           "tile/x000/x000/000",
			EntryTileOffset:        i,
		}
		_, err = repo.InsertStatement(ctx, stmt)
		if err != nil {
			t.Fatalf("Failed to insert statement: %v", err)
		}
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	nextID, err := repo.IncrementTreeSize(ctx)
	if err != nil {
		t.Fatalf("Failed to increment tree size: %v", err)
	}
	stmt := &StatementMetadata{
		EntryID:                nextID,
		LeafHash:               "d",
		Iss:                    issuer2,
		PayloadHashAlg:         -16,
		PayloadHash:            "y",
		TreeSizeAtRegistration: 3,
		EntryTileKey:           "tile/x000/x000/000",
		EntryTileOffset:        3,
	}
	_, err = repo.InsertStatement(ctx, stmt)
	if err != nil {
		t.Fatalf("Failed to insert statement: %v", err)
	}

	// Query by issuer1
	query := StatementQuery{
		Iss: &issuer1,
	}

	results, err := repo.QueryStatements(ctx, query)
	if err != nil {
		t.Fatalf("Failed to query statements: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for _, result := range results {
		if result.Iss != issuer1 {
			t.Errorf("Expected issuer %s, got %s", issuer1, result.Iss)
		}
	}
}

func TestSQLiteRepository_QueryStatements_WithLimitAndOffset(t *testing.T) {
	t.Parallel()

	repo, cleanup := createTempSQLiteRepo(t)
	defer cleanup()

	ctx := context.Background()

	issuer := "https://example.com"

	// Insert 10 statements
	for i := 0; i < 10; i++ {
		nextID, err := repo.IncrementTreeSize(ctx)
		if err != nil {
			t.Fatalf("Failed to increment tree size: %v", err)
		}
		stmt := &StatementMetadata{
			EntryID:                nextID,
			LeafHash:               string(rune('a' + i)),
			Iss:                    issuer,
			PayloadHashAlg:         -16,
			PayloadHash:            string(rune('x' + i)),
			TreeSizeAtRegistration: int64(i),
			EntryTileKey:           "tile/x000/x000/000",
			EntryTileOffset:        i,
		}
		_, err = repo.InsertStatement(ctx, stmt)
		if err != nil {
			t.Fatalf("Failed to insert statement: %v", err)
		}
		time.Sleep(time.Millisecond) // Ensure different timestamps
	}

	// Query with limit
	query := StatementQuery{
		Limit: 5,
	}

	results, err := repo.QueryStatements(ctx, query)
	if err != nil {
		t.Fatalf("Failed to query statements: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results with limit, got %d", len(results))
	}

	// Query with offset
	query = StatementQuery{
		Offset: 5,
	}

	results, err = repo.QueryStatements(ctx, query)
	if err != nil {
		t.Fatalf("Failed to query statements: %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Expected 5 results with offset, got %d", len(results))
	}

	// Query with limit and offset
	query = StatementQuery{
		Limit:  3,
		Offset: 2,
	}

	results, err = repo.QueryStatements(ctx, query)
	if err != nil {
		t.Fatalf("Failed to query statements: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results with limit and offset, got %d", len(results))
	}
}

func TestSQLiteRepository_TreeSize(t *testing.T) {
	t.Parallel()

	repo, cleanup := createTempSQLiteRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Initial tree size should be 0
	size, err := repo.GetCurrentTreeSize(ctx)
	if err != nil {
		t.Fatalf("Failed to get tree size: %v", err)
	}
	if size != 0 {
		t.Errorf("Expected initial tree size 0, got %d", size)
	}

	// Set tree size
	err = repo.SetCurrentTreeSize(ctx, 42)
	if err != nil {
		t.Fatalf("Failed to set tree size: %v", err)
	}

	size, err = repo.GetCurrentTreeSize(ctx)
	if err != nil {
		t.Fatalf("Failed to get tree size: %v", err)
	}
	if size != 42 {
		t.Errorf("Expected tree size 42, got %d", size)
	}
}

func TestSQLiteRepository_TreeSize_Negative(t *testing.T) {
	t.Parallel()

	repo, cleanup := createTempSQLiteRepo(t)
	defer cleanup()

	ctx := context.Background()

	err := repo.SetCurrentTreeSize(ctx, -1)
	if err == nil {
		t.Fatal("Expected error when setting negative tree size")
	}
}

func TestSQLiteRepository_IncrementTreeSize(t *testing.T) {
	t.Parallel()

	repo, cleanup := createTempSQLiteRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Increment from 0
	newSize, err := repo.IncrementTreeSize(ctx)
	if err != nil {
		t.Fatalf("Failed to increment tree size: %v", err)
	}
	if newSize != 1 {
		t.Errorf("Expected new tree size 1, got %d", newSize)
	}

	// Increment again
	newSize, err = repo.IncrementTreeSize(ctx)
	if err != nil {
		t.Fatalf("Failed to increment tree size: %v", err)
	}
	if newSize != 2 {
		t.Errorf("Expected new tree size 2, got %d", newSize)
	}

	// Verify current size
	size, err := repo.GetCurrentTreeSize(ctx)
	if err != nil {
		t.Fatalf("Failed to get tree size: %v", err)
	}
	if size != 2 {
		t.Errorf("Expected current tree size 2, got %d", size)
	}
}

func TestSQLiteRepository_Transaction(t *testing.T) {
	t.Parallel()

	repo, cleanup := createTempSQLiteRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Begin transaction
	tx, err := repo.BeginTx(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Insert statement in transaction
	stmt := &StatementMetadata{
		LeafHash:               "abcd1234",
		Iss:                    "https://example.com",
		PayloadHashAlg:         -16,
		PayloadHash:            "ef567890",
		TreeSizeAtRegistration: 0,
		EntryTileKey:           "tile/x000/x000/000",
		EntryTileOffset:        0,
	}

	_, err = tx.InsertStatement(ctx, stmt)
	if err != nil {
		t.Fatalf("Failed to insert statement in transaction: %v", err)
	}

	// Rollback
	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	// Verify statement was not inserted
	_, err = repo.GetStatementByLeafHash(ctx, "abcd1234")
	if err == nil {
		t.Fatal("Expected error when retrieving rolled-back statement")
	}

	// Try again with commit
	tx, err = repo.BeginTx(ctx)
	if err != nil {
		t.Fatalf("Failed to begin second transaction: %v", err)
	}

	_, err = tx.InsertStatement(ctx, stmt)
	if err != nil {
		t.Fatalf("Failed to insert statement in second transaction: %v", err)
	}

	// Commit
	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify statement was inserted
	_, err = repo.GetStatementByLeafHash(ctx, "abcd1234")
	if err != nil {
		t.Fatalf("Failed to retrieve committed statement: %v", err)
	}
}

func TestSQLiteRepository_Close(t *testing.T) {
	t.Parallel()

	repo, cleanup := createTempSQLiteRepo(t)
	defer cleanup()

	err := repo.Close()
	if err != nil {
		t.Fatalf("Failed to close repository: %v", err)
	}
}
