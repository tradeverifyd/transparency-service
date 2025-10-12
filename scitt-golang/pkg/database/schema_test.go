package database_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/database"
)

func TestOpenDatabase(t *testing.T) {
	t.Run("creates new database with schema", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: true,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		// Verify database file was created
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("database file was not created")
		}

		// Verify schema version
		var version string
		err = db.QueryRow("SELECT version FROM schema_version").Scan(&version)
		if err != nil {
			t.Fatalf("failed to query schema version: %v", err)
		}

		if version != "1.0.0" {
			t.Errorf("expected schema version 1.0.0, got %s", version)
		}
	})

	t.Run("opens existing database without reinitializing", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		// Create database
		db1, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: true,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}

		// Insert test data
		_, err = db1.Exec("INSERT INTO current_tree_size (id, tree_size) VALUES (1, 42) ON CONFLICT(id) DO UPDATE SET tree_size = 42")
		if err != nil {
			t.Fatalf("failed to insert test data: %v", err)
		}

		database.CloseDatabase(db1)

		// Reopen database
		db2, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: true,
		})
		if err != nil {
			t.Fatalf("failed to reopen database: %v", err)
		}
		defer database.CloseDatabase(db2)

		// Verify data persisted
		var treeSize int64
		err = db2.QueryRow("SELECT tree_size FROM current_tree_size WHERE id = 1").Scan(&treeSize)
		if err != nil {
			t.Fatalf("failed to query tree size: %v", err)
		}

		if treeSize != 42 {
			t.Errorf("expected tree size 42, got %d", treeSize)
		}
	})

	t.Run("creates all required tables", func(t *testing.T) {
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

		requiredTables := []string{
			"schema_version",
			"statements",
			"receipts",
			"tiles",
			"tree_state",
			"current_tree_size",
			"service_config",
			"service_keys",
		}

		for _, table := range requiredTables {
			var name string
			err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
			if err != nil {
				t.Errorf("table %s not found: %v", table, err)
			}
		}
	})

	t.Run("initializes service config defaults", func(t *testing.T) {
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

		expectedConfigs := map[string]string{
			"service_url":          "http://localhost:3000",
			"tile_height":          "8",
			"checkpoint_frequency": "1000",
			"hash_algorithm":       "-16",
			"signature_algorithm":  "-7",
		}

		for key, expectedValue := range expectedConfigs {
			var value string
			err := db.QueryRow("SELECT value FROM service_config WHERE key = ?", key).Scan(&value)
			if err != nil {
				t.Errorf("config key %s not found: %v", key, err)
				continue
			}

			if value != expectedValue {
				t.Errorf("config %s: expected %s, got %s", key, expectedValue, value)
			}
		}
	})

	t.Run("initializes current tree size to 0", func(t *testing.T) {
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

		var treeSize int64
		err = db.QueryRow("SELECT tree_size FROM current_tree_size WHERE id = 1").Scan(&treeSize)
		if err != nil {
			t.Fatalf("failed to query tree size: %v", err)
		}

		if treeSize != 0 {
			t.Errorf("expected initial tree size 0, got %d", treeSize)
		}
	})

	t.Run("sets busy timeout when specified", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:        dbPath,
			EnableWAL:   false,
			BusyTimeout: 10000,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}
		defer database.CloseDatabase(db)

		// Verify busy timeout is set (query will succeed if pragma worked)
		var timeout int
		err = db.QueryRow("PRAGMA busy_timeout").Scan(&timeout)
		if err != nil {
			t.Fatalf("failed to query busy timeout: %v", err)
		}

		if timeout != 10000 {
			t.Errorf("expected busy timeout 10000, got %d", timeout)
		}
	})
}

func TestCloseDatabase(t *testing.T) {
	t.Run("closes database connection", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := database.OpenDatabase(database.DatabaseOptions{
			Path:      dbPath,
			EnableWAL: false,
		})
		if err != nil {
			t.Fatalf("failed to open database: %v", err)
		}

		err = database.CloseDatabase(db)
		if err != nil {
			t.Errorf("failed to close database: %v", err)
		}

		// Verify database is closed (queries should fail)
		var version string
		err = db.QueryRow("SELECT version FROM schema_version").Scan(&version)
		if err == nil {
			t.Error("expected error after closing database, but query succeeded")
		}
	})
}
