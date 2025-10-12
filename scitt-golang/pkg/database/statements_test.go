package database_test

import (
	"path/filepath"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/database"
)

func TestInsertStatement(t *testing.T) {
	t.Run("inserts statement and returns entry ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		sub := "subject-1"
		cty := "application/json"

		statement := database.Statement{
			StatementHash:          "hash123",
			Iss:                    "https://issuer.example.com",
			Sub:                    &sub,
			Cty:                    &cty,
			PayloadHashAlg:         -16,
			PayloadHash:            "payload-hash",
			TreeSizeAtRegistration: 1,
			EntryTileKey:           "0/0",
			EntryTileOffset:        0,
		}

		entryID, err := database.InsertStatement(db, statement)
		if err != nil {
			t.Fatalf("failed to insert statement: %v", err)
		}

		if entryID == 0 {
			t.Error("expected non-zero entry ID")
		}

		// Verify statement was inserted
		retrieved, err := database.GetStatementByEntryID(db, entryID)
		if err != nil {
			t.Fatalf("failed to get statement: %v", err)
		}

		if retrieved == nil {
			t.Fatal("statement not found")
		}

		if retrieved.StatementHash != statement.StatementHash {
			t.Errorf("expected hash %s, got %s", statement.StatementHash, retrieved.StatementHash)
		}
	})

	t.Run("inserts multiple statements", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		for i := 0; i < 3; i++ {
			statement := database.Statement{
				StatementHash:          "hash-" + string(rune('0'+i)),
				Iss:                    "https://issuer.example.com",
				PayloadHashAlg:         -16,
				PayloadHash:            "hash",
				TreeSizeAtRegistration: int64(i + 1),
				EntryTileKey:           "0/0",
				EntryTileOffset:        i,
			}

			_, err := database.InsertStatement(db, statement)
			if err != nil {
				t.Fatalf("failed to insert statement %d: %v", i, err)
			}
		}

		// Verify count
		statements, err := database.FindStatementsBy(db, database.StatementQueryFilters{})
		if err != nil {
			t.Fatalf("failed to find statements: %v", err)
		}

		if len(statements) != 3 {
			t.Errorf("expected 3 statements, got %d", len(statements))
		}
	})
}

func TestFindStatementsByIssuer(t *testing.T) {
	t.Run("finds statements by issuer", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		iss1 := "https://issuer1.example.com"
		iss2 := "https://issuer2.example.com"

		// Insert statements with different issuers
		statements := []database.Statement{
			{StatementHash: "hash1", Iss: iss1, PayloadHashAlg: -16, PayloadHash: "ph1", TreeSizeAtRegistration: 1, EntryTileKey: "0/0", EntryTileOffset: 0},
			{StatementHash: "hash2", Iss: iss1, PayloadHashAlg: -16, PayloadHash: "ph2", TreeSizeAtRegistration: 2, EntryTileKey: "0/0", EntryTileOffset: 1},
			{StatementHash: "hash3", Iss: iss2, PayloadHashAlg: -16, PayloadHash: "ph3", TreeSizeAtRegistration: 3, EntryTileKey: "0/0", EntryTileOffset: 2},
		}

		for _, stmt := range statements {
			_, err := database.InsertStatement(db, stmt)
			if err != nil {
				t.Fatalf("failed to insert statement: %v", err)
			}
		}

		// Find by issuer1
		results, err := database.FindStatementsByIssuer(db, iss1)
		if err != nil {
			t.Fatalf("failed to find statements: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 statements for issuer1, got %d", len(results))
		}

		// Find by issuer2
		results, err = database.FindStatementsByIssuer(db, iss2)
		if err != nil {
			t.Fatalf("failed to find statements: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 statement for issuer2, got %d", len(results))
		}
	})
}

func TestFindStatementsBySubject(t *testing.T) {
	t.Run("finds statements by subject", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		sub1 := "subject-1"
		sub2 := "subject-2"

		statements := []database.Statement{
			{StatementHash: "hash1", Iss: "iss", Sub: &sub1, PayloadHashAlg: -16, PayloadHash: "ph1", TreeSizeAtRegistration: 1, EntryTileKey: "0/0", EntryTileOffset: 0},
			{StatementHash: "hash2", Iss: "iss", Sub: &sub1, PayloadHashAlg: -16, PayloadHash: "ph2", TreeSizeAtRegistration: 2, EntryTileKey: "0/0", EntryTileOffset: 1},
			{StatementHash: "hash3", Iss: "iss", Sub: &sub2, PayloadHashAlg: -16, PayloadHash: "ph3", TreeSizeAtRegistration: 3, EntryTileKey: "0/0", EntryTileOffset: 2},
		}

		for _, stmt := range statements {
			_, err := database.InsertStatement(db, stmt)
			if err != nil {
				t.Fatalf("failed to insert statement: %v", err)
			}
		}

		results, err := database.FindStatementsBySubject(db, sub1)
		if err != nil {
			t.Fatalf("failed to find statements: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 statements for subject-1, got %d", len(results))
		}
	})
}

func TestFindStatementsByContentType(t *testing.T) {
	t.Run("finds statements by content type", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		cty1 := "application/json"
		cty2 := "application/xml"

		statements := []database.Statement{
			{StatementHash: "hash1", Iss: "iss", Cty: &cty1, PayloadHashAlg: -16, PayloadHash: "ph1", TreeSizeAtRegistration: 1, EntryTileKey: "0/0", EntryTileOffset: 0},
			{StatementHash: "hash2", Iss: "iss", Cty: &cty1, PayloadHashAlg: -16, PayloadHash: "ph2", TreeSizeAtRegistration: 2, EntryTileKey: "0/0", EntryTileOffset: 1},
			{StatementHash: "hash3", Iss: "iss", Cty: &cty2, PayloadHashAlg: -16, PayloadHash: "ph3", TreeSizeAtRegistration: 3, EntryTileKey: "0/0", EntryTileOffset: 2},
		}

		for _, stmt := range statements {
			_, err := database.InsertStatement(db, stmt)
			if err != nil {
				t.Fatalf("failed to insert statement: %v", err)
			}
		}

		results, err := database.FindStatementsByContentType(db, cty1)
		if err != nil {
			t.Fatalf("failed to find statements: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 statements for application/json, got %d", len(results))
		}
	})
}

func TestFindStatementsByType(t *testing.T) {
	t.Run("finds statements by type", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		typ1 := "type-a"
		typ2 := "type-b"

		statements := []database.Statement{
			{StatementHash: "hash1", Iss: "iss", Typ: &typ1, PayloadHashAlg: -16, PayloadHash: "ph1", TreeSizeAtRegistration: 1, EntryTileKey: "0/0", EntryTileOffset: 0},
			{StatementHash: "hash2", Iss: "iss", Typ: &typ2, PayloadHashAlg: -16, PayloadHash: "ph2", TreeSizeAtRegistration: 2, EntryTileKey: "0/0", EntryTileOffset: 1},
		}

		for _, stmt := range statements {
			_, err := database.InsertStatement(db, stmt)
			if err != nil {
				t.Fatalf("failed to insert statement: %v", err)
			}
		}

		results, err := database.FindStatementsByType(db, typ1)
		if err != nil {
			t.Fatalf("failed to find statements: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 statement for type-a, got %d", len(results))
		}

		if results[0].StatementHash != "hash1" {
			t.Errorf("expected hash1, got %s", results[0].StatementHash)
		}
	})
}

func TestFindStatementsBy(t *testing.T) {
	t.Run("returns all statements with no filters", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		for i := 0; i < 3; i++ {
			statement := database.Statement{
				StatementHash:          "hash-" + string(rune('0'+i)),
				Iss:                    "iss",
				PayloadHashAlg:         -16,
				PayloadHash:            "hash",
				TreeSizeAtRegistration: int64(i + 1),
				EntryTileKey:           "0/0",
				EntryTileOffset:        i,
			}
			_, err := database.InsertStatement(db, statement)
			if err != nil {
				t.Fatalf("failed to insert statement: %v", err)
			}
		}

		results, err := database.FindStatementsBy(db, database.StatementQueryFilters{})
		if err != nil {
			t.Fatalf("failed to find statements: %v", err)
		}

		if len(results) != 3 {
			t.Errorf("expected 3 statements, got %d", len(results))
		}
	})

	t.Run("filters by multiple criteria", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		iss1 := "https://issuer1.example.com"
		iss2 := "https://issuer2.example.com"
		sub1 := "subject-1"
		cty1 := "application/json"

		statements := []database.Statement{
			{StatementHash: "hash1", Iss: iss1, Sub: &sub1, Cty: &cty1, PayloadHashAlg: -16, PayloadHash: "ph1", TreeSizeAtRegistration: 1, EntryTileKey: "0/0", EntryTileOffset: 0},
			{StatementHash: "hash2", Iss: iss1, Sub: &sub1, PayloadHashAlg: -16, PayloadHash: "ph2", TreeSizeAtRegistration: 2, EntryTileKey: "0/0", EntryTileOffset: 1},
			{StatementHash: "hash3", Iss: iss2, PayloadHashAlg: -16, PayloadHash: "ph3", TreeSizeAtRegistration: 3, EntryTileKey: "0/0", EntryTileOffset: 2},
		}

		for _, stmt := range statements {
			_, err := database.InsertStatement(db, stmt)
			if err != nil {
				t.Fatalf("failed to insert statement: %v", err)
			}
		}

		// Filter by issuer and subject
		results, err := database.FindStatementsBy(db, database.StatementQueryFilters{
			Iss: &iss1,
			Sub: &sub1,
		})
		if err != nil {
			t.Fatalf("failed to find statements: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 statements, got %d", len(results))
		}

		// Filter by issuer, subject, and content type
		results, err = database.FindStatementsBy(db, database.StatementQueryFilters{
			Iss: &iss1,
			Sub: &sub1,
			Cty: &cty1,
		})
		if err != nil {
			t.Fatalf("failed to find statements: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 statement, got %d", len(results))
		}

		if results[0].StatementHash != "hash1" {
			t.Errorf("expected hash1, got %s", results[0].StatementHash)
		}
	})
}

func TestGetStatementByEntryID(t *testing.T) {
	t.Run("returns nil for non-existent entry ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		statement, err := database.GetStatementByEntryID(db, 999)
		if err != nil {
			t.Fatalf("failed to get statement: %v", err)
		}

		if statement != nil {
			t.Error("expected nil for non-existent entry ID")
		}
	})

	t.Run("retrieves statement by entry ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		statement := database.Statement{
			StatementHash:          "unique-hash",
			Iss:                    "https://issuer.example.com",
			PayloadHashAlg:         -16,
			PayloadHash:            "payload-hash",
			TreeSizeAtRegistration: 1,
			EntryTileKey:           "0/0",
			EntryTileOffset:        0,
		}

		entryID, err := database.InsertStatement(db, statement)
		if err != nil {
			t.Fatalf("failed to insert statement: %v", err)
		}

		retrieved, err := database.GetStatementByEntryID(db, entryID)
		if err != nil {
			t.Fatalf("failed to get statement: %v", err)
		}

		if retrieved == nil {
			t.Fatal("statement not found")
		}

		if retrieved.StatementHash != statement.StatementHash {
			t.Errorf("expected hash %s, got %s", statement.StatementHash, retrieved.StatementHash)
		}
	})
}

func TestGetStatementByHash(t *testing.T) {
	t.Run("returns nil for non-existent hash", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		statement, err := database.GetStatementByHash(db, "non-existent")
		if err != nil {
			t.Fatalf("failed to get statement: %v", err)
		}

		if statement != nil {
			t.Error("expected nil for non-existent hash")
		}
	})

	t.Run("retrieves statement by hash", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		statement := database.Statement{
			StatementHash:          "findable-hash",
			Iss:                    "https://issuer.example.com",
			PayloadHashAlg:         -16,
			PayloadHash:            "payload-hash",
			TreeSizeAtRegistration: 1,
			EntryTileKey:           "0/0",
			EntryTileOffset:        0,
		}

		_, err = database.InsertStatement(db, statement)
		if err != nil {
			t.Fatalf("failed to insert statement: %v", err)
		}

		retrieved, err := database.GetStatementByHash(db, "findable-hash")
		if err != nil {
			t.Fatalf("failed to get statement: %v", err)
		}

		if retrieved == nil {
			t.Fatal("statement not found")
		}

		if retrieved.StatementHash != statement.StatementHash {
			t.Errorf("expected hash %s, got %s", statement.StatementHash, retrieved.StatementHash)
		}
	})
}

func TestSaveAndGetStatementBlob(t *testing.T) {
	t.Run("saves and retrieves statement blob", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		entryID := "entry-123"
		statementBytes := []byte("COSE Sign1 bytes here")
		leafHash := []byte{0x01, 0x02, 0x03, 0x04}
		leafIndex := int64(42)

		err = database.SaveStatement(db, entryID, statementBytes, leafHash, leafIndex)
		if err != nil {
			t.Fatalf("failed to save statement: %v", err)
		}

		retrieved, err := database.GetStatementBlob(db, entryID)
		if err != nil {
			t.Fatalf("failed to get statement blob: %v", err)
		}

		if retrieved == nil {
			t.Fatal("statement blob not found")
		}

		if string(retrieved) != string(statementBytes) {
			t.Errorf("expected %s, got %s", string(statementBytes), string(retrieved))
		}
	})

	t.Run("returns nil for non-existent entry ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		// First save a statement to create the table
		err = database.SaveStatement(db, "existing", []byte("data"), []byte{0x01}, 1)
		if err != nil {
			t.Fatalf("failed to save initial statement: %v", err)
		}

		// Now try to get non-existent entry
		retrieved, err := database.GetStatementBlob(db, "non-existent")
		if err != nil {
			t.Fatalf("failed to get statement blob: %v", err)
		}

		if retrieved != nil {
			t.Error("expected nil for non-existent entry ID")
		}
	})
}
