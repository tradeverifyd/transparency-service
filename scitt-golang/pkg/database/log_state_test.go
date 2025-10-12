package database_test

import (
	"path/filepath"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/database"
)

func TestGetCurrentTreeSize(t *testing.T) {
	t.Run("returns initial tree size of 0", func(t *testing.T) {
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

		size, err := database.GetCurrentTreeSize(db)
		if err != nil {
			t.Fatalf("failed to get tree size: %v", err)
		}

		if size != 0 {
			t.Errorf("expected tree size 0, got %d", size)
		}
	})

	t.Run("returns updated tree size", func(t *testing.T) {
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

		// Update tree size
		err = database.UpdateTreeSize(db, 100)
		if err != nil {
			t.Fatalf("failed to update tree size: %v", err)
		}

		// Get tree size
		size, err := database.GetCurrentTreeSize(db)
		if err != nil {
			t.Fatalf("failed to get tree size: %v", err)
		}

		if size != 100 {
			t.Errorf("expected tree size 100, got %d", size)
		}
	})
}

func TestUpdateTreeSize(t *testing.T) {
	t.Run("updates tree size", func(t *testing.T) {
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

		err = database.UpdateTreeSize(db, 42)
		if err != nil {
			t.Fatalf("failed to update tree size: %v", err)
		}

		size, err := database.GetCurrentTreeSize(db)
		if err != nil {
			t.Fatalf("failed to get tree size: %v", err)
		}

		if size != 42 {
			t.Errorf("expected tree size 42, got %d", size)
		}
	})

	t.Run("updates tree size multiple times", func(t *testing.T) {
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

		sizes := []int64{10, 20, 30, 40}

		for _, expectedSize := range sizes {
			err = database.UpdateTreeSize(db, expectedSize)
			if err != nil {
				t.Fatalf("failed to update tree size: %v", err)
			}

			size, err := database.GetCurrentTreeSize(db)
			if err != nil {
				t.Fatalf("failed to get tree size: %v", err)
			}

			if size != expectedSize {
				t.Errorf("expected tree size %d, got %d", expectedSize, size)
			}
		}
	})
}

func TestRecordTreeState(t *testing.T) {
	t.Run("records tree state", func(t *testing.T) {
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

		state := database.TreeState{
			TreeSize:             100,
			RootHash:             "abc123",
			CheckpointStorageKey: "checkpoints/100",
			CheckpointSignedNote: "signed note content",
		}

		err = database.RecordTreeState(db, state)
		if err != nil {
			t.Fatalf("failed to record tree state: %v", err)
		}

		// Verify state was recorded
		retrieved, err := database.GetTreeState(db, 100)
		if err != nil {
			t.Fatalf("failed to get tree state: %v", err)
		}

		if retrieved == nil {
			t.Fatal("tree state not found")
		}

		if retrieved.TreeSize != state.TreeSize {
			t.Errorf("expected tree size %d, got %d", state.TreeSize, retrieved.TreeSize)
		}
		if retrieved.RootHash != state.RootHash {
			t.Errorf("expected root hash %s, got %s", state.RootHash, retrieved.RootHash)
		}
	})

	t.Run("records multiple tree states", func(t *testing.T) {
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

		states := []database.TreeState{
			{TreeSize: 100, RootHash: "hash1", CheckpointStorageKey: "cp1", CheckpointSignedNote: "note1"},
			{TreeSize: 200, RootHash: "hash2", CheckpointStorageKey: "cp2", CheckpointSignedNote: "note2"},
			{TreeSize: 300, RootHash: "hash3", CheckpointStorageKey: "cp3", CheckpointSignedNote: "note3"},
		}

		for _, state := range states {
			err = database.RecordTreeState(db, state)
			if err != nil {
				t.Fatalf("failed to record tree state: %v", err)
			}
		}

		// Verify all states were recorded
		for _, state := range states {
			retrieved, err := database.GetTreeState(db, state.TreeSize)
			if err != nil {
				t.Fatalf("failed to get tree state: %v", err)
			}

			if retrieved == nil {
				t.Errorf("tree state %d not found", state.TreeSize)
				continue
			}

			if retrieved.RootHash != state.RootHash {
				t.Errorf("tree size %d: expected root hash %s, got %s", state.TreeSize, state.RootHash, retrieved.RootHash)
			}
		}
	})
}

func TestGetTreeState(t *testing.T) {
	t.Run("returns nil for non-existent tree state", func(t *testing.T) {
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

		state, err := database.GetTreeState(db, 999)
		if err != nil {
			t.Fatalf("failed to get tree state: %v", err)
		}

		if state != nil {
			t.Error("expected nil for non-existent tree state")
		}
	})

	t.Run("retrieves existing tree state", func(t *testing.T) {
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

		original := database.TreeState{
			TreeSize:             500,
			RootHash:             "xyz789",
			CheckpointStorageKey: "checkpoints/500",
			CheckpointSignedNote: "checkpoint note",
		}

		err = database.RecordTreeState(db, original)
		if err != nil {
			t.Fatalf("failed to record tree state: %v", err)
		}

		retrieved, err := database.GetTreeState(db, 500)
		if err != nil {
			t.Fatalf("failed to get tree state: %v", err)
		}

		if retrieved == nil {
			t.Fatal("tree state not found")
		}

		if retrieved.TreeSize != original.TreeSize {
			t.Errorf("expected tree size %d, got %d", original.TreeSize, retrieved.TreeSize)
		}
		if retrieved.RootHash != original.RootHash {
			t.Errorf("expected root hash %s, got %s", original.RootHash, retrieved.RootHash)
		}
		if retrieved.CheckpointStorageKey != original.CheckpointStorageKey {
			t.Errorf("expected checkpoint key %s, got %s", original.CheckpointStorageKey, retrieved.CheckpointStorageKey)
		}
		if retrieved.CheckpointSignedNote != original.CheckpointSignedNote {
			t.Errorf("expected signed note %s, got %s", original.CheckpointSignedNote, retrieved.CheckpointSignedNote)
		}
	})
}

func TestGetTreeStateHistory(t *testing.T) {
	t.Run("returns empty history for new database", func(t *testing.T) {
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

		history, err := database.GetTreeStateHistory(db, 0)
		if err != nil {
			t.Fatalf("failed to get tree state history: %v", err)
		}

		if len(history) != 0 {
			t.Errorf("expected empty history, got %d states", len(history))
		}
	})

	t.Run("returns history in descending order", func(t *testing.T) {
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

		// Record states
		states := []database.TreeState{
			{TreeSize: 100, RootHash: "hash1", CheckpointStorageKey: "cp1", CheckpointSignedNote: "note1"},
			{TreeSize: 200, RootHash: "hash2", CheckpointStorageKey: "cp2", CheckpointSignedNote: "note2"},
			{TreeSize: 300, RootHash: "hash3", CheckpointStorageKey: "cp3", CheckpointSignedNote: "note3"},
		}

		for _, state := range states {
			err = database.RecordTreeState(db, state)
			if err != nil {
				t.Fatalf("failed to record tree state: %v", err)
			}
		}

		// Get history
		history, err := database.GetTreeStateHistory(db, 0)
		if err != nil {
			t.Fatalf("failed to get tree state history: %v", err)
		}

		if len(history) != 3 {
			t.Fatalf("expected 3 states, got %d", len(history))
		}

		// Verify descending order
		if history[0].TreeSize != 300 {
			t.Errorf("expected first tree size 300, got %d", history[0].TreeSize)
		}
		if history[1].TreeSize != 200 {
			t.Errorf("expected second tree size 200, got %d", history[1].TreeSize)
		}
		if history[2].TreeSize != 100 {
			t.Errorf("expected third tree size 100, got %d", history[2].TreeSize)
		}
	})

	t.Run("respects limit parameter", func(t *testing.T) {
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

		// Record 5 states
		for i := 1; i <= 5; i++ {
			state := database.TreeState{
				TreeSize:             int64(i * 100),
				RootHash:             "hash",
				CheckpointStorageKey: "cp",
				CheckpointSignedNote: "note",
			}
			err = database.RecordTreeState(db, state)
			if err != nil {
				t.Fatalf("failed to record tree state: %v", err)
			}
		}

		// Get limited history
		history, err := database.GetTreeStateHistory(db, 2)
		if err != nil {
			t.Fatalf("failed to get tree state history: %v", err)
		}

		if len(history) != 2 {
			t.Errorf("expected 2 states with limit, got %d", len(history))
		}

		// Should get the 2 most recent
		if history[0].TreeSize != 500 {
			t.Errorf("expected first tree size 500, got %d", history[0].TreeSize)
		}
		if history[1].TreeSize != 400 {
			t.Errorf("expected second tree size 400, got %d", history[1].TreeSize)
		}
	})
}

func TestGetLatestCheckpoint(t *testing.T) {
	t.Run("returns nil when no checkpoints exist", func(t *testing.T) {
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

		checkpoint, err := database.GetLatestCheckpoint(db)
		if err != nil {
			t.Fatalf("failed to get latest checkpoint: %v", err)
		}

		if checkpoint != nil {
			t.Error("expected nil checkpoint for empty database")
		}
	})

	t.Run("returns most recent checkpoint", func(t *testing.T) {
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

		// Record checkpoints
		states := []database.TreeState{
			{TreeSize: 100, RootHash: "hash1", CheckpointStorageKey: "cp1", CheckpointSignedNote: "note1"},
			{TreeSize: 200, RootHash: "hash2", CheckpointStorageKey: "cp2", CheckpointSignedNote: "note2"},
			{TreeSize: 300, RootHash: "hash3", CheckpointStorageKey: "cp3", CheckpointSignedNote: "note3"},
		}

		for _, state := range states {
			err = database.RecordTreeState(db, state)
			if err != nil {
				t.Fatalf("failed to record tree state: %v", err)
			}
		}

		// Get latest
		latest, err := database.GetLatestCheckpoint(db)
		if err != nil {
			t.Fatalf("failed to get latest checkpoint: %v", err)
		}

		if latest == nil {
			t.Fatal("latest checkpoint not found")
		}

		if latest.TreeSize != 300 {
			t.Errorf("expected latest tree size 300, got %d", latest.TreeSize)
		}
		if latest.RootHash != "hash3" {
			t.Errorf("expected latest root hash hash3, got %s", latest.RootHash)
		}
	})
}
