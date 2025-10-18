package repository

import (
	"context"
	"testing"
)

func TestMemoryRepository_InsertStatement(t *testing.T) {
	t.Parallel()

	repo := NewMemoryRepository()
	ctx := context.Background()

	issuer := "https://example.com"
	subject := "test-subject"
	contentType := "application/json"

	stmt := &StatementMetadata{
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

	if entryID != 0 {
		t.Errorf("Expected first entry ID to be 0, got %d", entryID)
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

func TestMemoryRepository_InsertStatement_DuplicateLeafHash(t *testing.T) {
	t.Parallel()

	repo := NewMemoryRepository()
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

func TestMemoryRepository_GetStatementByLeafHash(t *testing.T) {
	t.Parallel()

	repo := NewMemoryRepository()
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

func TestMemoryRepository_GetStatementByLeafHash_NotFound(t *testing.T) {
	t.Parallel()

	repo := NewMemoryRepository()
	ctx := context.Background()

	_, err := repo.GetStatementByLeafHash(ctx, "nonexistent")
	if err == nil {
		t.Fatal("Expected error when retrieving non-existent leaf hash")
	}
}

func TestMemoryRepository_QueryStatements_ByIssuer(t *testing.T) {
	t.Parallel()

	repo := NewMemoryRepository()
	ctx := context.Background()

	issuer1 := "https://example.com"
	issuer2 := "https://other.com"

	// Insert statements with different issuers
	for i := 0; i < 3; i++ {
		stmt := &StatementMetadata{
			LeafHash:               string(rune('a' + i)),
			Iss:                    issuer1,
			PayloadHashAlg:         -16,
			PayloadHash:            string(rune('x' + i)),
			TreeSizeAtRegistration: int64(i),
			EntryTileKey:           "tile/x000/x000/000",
			EntryTileOffset:        i,
		}
		_, err := repo.InsertStatement(ctx, stmt)
		if err != nil {
			t.Fatalf("Failed to insert statement: %v", err)
		}
	}

	stmt := &StatementMetadata{
		LeafHash:               "d",
		Iss:                    issuer2,
		PayloadHashAlg:         -16,
		PayloadHash:            "y",
		TreeSizeAtRegistration: 3,
		EntryTileKey:           "tile/x000/x000/000",
		EntryTileOffset:        3,
	}
	_, err := repo.InsertStatement(ctx, stmt)
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

func TestMemoryRepository_QueryStatements_WithLimitAndOffset(t *testing.T) {
	t.Parallel()

	repo := NewMemoryRepository()
	ctx := context.Background()

	issuer := "https://example.com"

	// Insert 10 statements
	for i := 0; i < 10; i++ {
		stmt := &StatementMetadata{
			LeafHash:               string(rune('a' + i)),
			Iss:                    issuer,
			PayloadHashAlg:         -16,
			PayloadHash:            string(rune('x' + i)),
			TreeSizeAtRegistration: int64(i),
			EntryTileKey:           "tile/x000/x000/000",
			EntryTileOffset:        i,
		}
		_, err := repo.InsertStatement(ctx, stmt)
		if err != nil {
			t.Fatalf("Failed to insert statement: %v", err)
		}
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

func TestMemoryRepository_TreeSize(t *testing.T) {
	t.Parallel()

	repo := NewMemoryRepository()
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

func TestMemoryRepository_TreeSize_Negative(t *testing.T) {
	t.Parallel()

	repo := NewMemoryRepository()
	ctx := context.Background()

	err := repo.SetCurrentTreeSize(ctx, -1)
	if err == nil {
		t.Fatal("Expected error when setting negative tree size")
	}
}

func TestMemoryRepository_IncrementTreeSize(t *testing.T) {
	t.Parallel()

	repo := NewMemoryRepository()
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

func TestMemoryRepository_Close(t *testing.T) {
	t.Parallel()

	repo := NewMemoryRepository()
	err := repo.Close()
	if err != nil {
		t.Fatalf("Failed to close repository: %v", err)
	}
}
