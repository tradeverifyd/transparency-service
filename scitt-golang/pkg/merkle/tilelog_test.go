package merkle_test

import (
	"crypto/sha256"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/merkle"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
)

func TestNewTileLog(t *testing.T) {
	t.Run("creates new tile log", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)

		if tl == nil {
			t.Fatal("expected non-nil tile log")
		}
	})
}

func TestTileLogLoad(t *testing.T) {
	t.Run("loads empty tree", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)

		err := tl.Load()
		if err != nil {
			t.Fatalf("failed to load empty tree: %v", err)
		}

		if tl.Size() != 0 {
			t.Errorf("expected size 0, got %d", tl.Size())
		}
	})

	t.Run("loads persisted state", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)

		err := tl.Load()
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}

		// Append some leaves
		leaf1 := hashData([]byte("leaf1"))
		_, err = tl.Append(leaf1)
		if err != nil {
			t.Fatalf("failed to append leaf1: %v", err)
		}

		leaf2 := hashData([]byte("leaf2"))
		_, err = tl.Append(leaf2)
		if err != nil {
			t.Fatalf("failed to append leaf2: %v", err)
		}

		// Create new tile log with same storage
		tl2 := merkle.NewTileLog(store)
		err = tl2.Load()
		if err != nil {
			t.Fatalf("failed to load persisted state: %v", err)
		}

		if tl2.Size() != 2 {
			t.Errorf("expected size 2, got %d", tl2.Size())
		}
	})
}

func TestTileLogAppend(t *testing.T) {
	t.Run("appends single leaf", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		err := tl.Load()
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}

		leaf := hashData([]byte("test leaf"))
		entryID, err := tl.Append(leaf)
		if err != nil {
			t.Fatalf("failed to append: %v", err)
		}

		if entryID != 0 {
			t.Errorf("expected entry ID 0, got %d", entryID)
		}

		if tl.Size() != 1 {
			t.Errorf("expected size 1, got %d", tl.Size())
		}
	})

	t.Run("appends multiple leaves", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		err := tl.Load()
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}

		numLeaves := 10
		for i := 0; i < numLeaves; i++ {
			leaf := hashData([]byte{byte(i)})
			entryID, err := tl.Append(leaf)
			if err != nil {
				t.Fatalf("failed to append leaf %d: %v", i, err)
			}

			if entryID != int64(i) {
				t.Errorf("leaf %d: expected entry ID %d, got %d", i, i, entryID)
			}
		}

		if tl.Size() != int64(numLeaves) {
			t.Errorf("expected size %d, got %d", numLeaves, tl.Size())
		}
	})

	t.Run("appends across tile boundaries", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		err := tl.Load()
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}

		// Append 257 leaves (crosses first tile boundary at 256)
		numLeaves := 257
		for i := 0; i < numLeaves; i++ {
			leaf := hashData([]byte{byte(i % 256)})
			_, err := tl.Append(leaf)
			if err != nil {
				t.Fatalf("failed to append leaf %d: %v", i, err)
			}
		}

		if tl.Size() != int64(numLeaves) {
			t.Errorf("expected size %d, got %d", numLeaves, tl.Size())
		}
	})
}

func TestTileLogGetLeaf(t *testing.T) {
	t.Run("retrieves appended leaf", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		err := tl.Load()
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}

		originalLeaf := hashData([]byte("test data"))
		entryID, err := tl.Append(originalLeaf)
		if err != nil {
			t.Fatalf("failed to append: %v", err)
		}

		retrievedLeaf, err := tl.GetLeaf(entryID)
		if err != nil {
			t.Fatalf("failed to get leaf: %v", err)
		}

		if retrievedLeaf != originalLeaf {
			t.Error("retrieved leaf does not match original")
		}
	})

	t.Run("retrieves multiple leaves", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		err := tl.Load()
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}

		leaves := make([][32]byte, 5)
		for i := 0; i < 5; i++ {
			leaves[i] = hashData([]byte{byte(i)})
			_, err := tl.Append(leaves[i])
			if err != nil {
				t.Fatalf("failed to append leaf %d: %v", i, err)
			}
		}

		// Retrieve all leaves
		for i := 0; i < 5; i++ {
			retrieved, err := tl.GetLeaf(int64(i))
			if err != nil {
				t.Fatalf("failed to get leaf %d: %v", i, err)
			}

			if retrieved != leaves[i] {
				t.Errorf("leaf %d does not match", i)
			}
		}
	})

	t.Run("rejects out of bounds entry ID", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		err := tl.Load()
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}

		leaf := hashData([]byte("test"))
		_, err = tl.Append(leaf)
		if err != nil {
			t.Fatalf("failed to append: %v", err)
		}

		_, err = tl.GetLeaf(999)
		if err == nil {
			t.Error("expected error for out of bounds entry ID")
		}
	})
}

func TestTileLogRoot(t *testing.T) {
	t.Run("rejects root of empty tree", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		err := tl.Load()
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}

		_, err = tl.Root()
		if err == nil {
			t.Error("expected error for empty tree root")
		}
	})

	t.Run("computes root for single leaf", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		err := tl.Load()
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}

		leaf := hashData([]byte("single leaf"))
		_, err = tl.Append(leaf)
		if err != nil {
			t.Fatalf("failed to append: %v", err)
		}

		root, err := tl.Root()
		if err != nil {
			t.Fatalf("failed to get root: %v", err)
		}

		// Root should be non-zero
		var zero [32]byte
		if root == zero {
			t.Error("root should not be zero")
		}
	})

	t.Run("computes root for multiple leaves", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		err := tl.Load()
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}

		// Append several leaves
		for i := 0; i < 10; i++ {
			leaf := hashData([]byte{byte(i)})
			_, err := tl.Append(leaf)
			if err != nil {
				t.Fatalf("failed to append leaf %d: %v", i, err)
			}
		}

		root, err := tl.Root()
		if err != nil {
			t.Fatalf("failed to get root: %v", err)
		}

		var zero [32]byte
		if root == zero {
			t.Error("root should not be zero")
		}
	})

	t.Run("root changes when leaves are added", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		err := tl.Load()
		if err != nil {
			t.Fatalf("failed to load: %v", err)
		}

		leaf1 := hashData([]byte("leaf1"))
		_, err = tl.Append(leaf1)
		if err != nil {
			t.Fatalf("failed to append leaf1: %v", err)
		}

		root1, err := tl.Root()
		if err != nil {
			t.Fatalf("failed to get root1: %v", err)
		}

		leaf2 := hashData([]byte("leaf2"))
		_, err = tl.Append(leaf2)
		if err != nil {
			t.Fatalf("failed to append leaf2: %v", err)
		}

		root2, err := tl.Root()
		if err != nil {
			t.Fatalf("failed to get root2: %v", err)
		}

		if root1 == root2 {
			t.Error("root should change after appending leaf")
		}
	})
}

func TestRecordHash(t *testing.T) {
	t.Run("computes consistent hash", func(t *testing.T) {
		data := []byte("test data")

		hash1 := merkle.RecordHash(data)
		hash2 := merkle.RecordHash(data)

		if hash1 != hash2 {
			t.Error("record hash should be consistent")
		}
	})

	t.Run("produces different hashes for different data", func(t *testing.T) {
		data1 := []byte("data1")
		data2 := []byte("data2")

		hash1 := merkle.RecordHash(data1)
		hash2 := merkle.RecordHash(data2)

		if hash1 == hash2 {
			t.Error("different data should produce different hashes")
		}
	})

	t.Run("uses RFC 6962 leaf prefix", func(t *testing.T) {
		data := []byte("test")
		hash := merkle.RecordHash(data)

		// Manually compute expected hash with 0x00 prefix
		h := sha256.New()
		h.Write([]byte{0x00})
		h.Write(data)
		expected := h.Sum(nil)

		for i := 0; i < 32; i++ {
			if hash[i] != expected[i] {
				t.Errorf("byte %d: expected %02x, got %02x", i, expected[i], hash[i])
			}
		}
	})
}

func TestTileLogPersistence(t *testing.T) {
	t.Run("persists and restores state", func(t *testing.T) {
		store := storage.NewMemoryStorage()

		// Create first tile log and add data
		tl1 := merkle.NewTileLog(store)
		err := tl1.Load()
		if err != nil {
			t.Fatalf("failed to load tl1: %v", err)
		}

		leaves := make([][32]byte, 3)
		for i := 0; i < 3; i++ {
			leaves[i] = hashData([]byte{byte(i)})
			_, err := tl1.Append(leaves[i])
			if err != nil {
				t.Fatalf("failed to append leaf %d: %v", i, err)
			}
		}

		root1, err := tl1.Root()
		if err != nil {
			t.Fatalf("failed to get root1: %v", err)
		}

		// Create second tile log with same storage
		tl2 := merkle.NewTileLog(store)
		err = tl2.Load()
		if err != nil {
			t.Fatalf("failed to load tl2: %v", err)
		}

		if tl2.Size() != 3 {
			t.Errorf("expected size 3, got %d", tl2.Size())
		}

		root2, err := tl2.Root()
		if err != nil {
			t.Fatalf("failed to get root2: %v", err)
		}

		if root1 != root2 {
			t.Error("roots should match after restore")
		}

		// Verify leaves are accessible
		for i := 0; i < 3; i++ {
			retrieved, err := tl2.GetLeaf(int64(i))
			if err != nil {
				t.Fatalf("failed to get leaf %d: %v", i, err)
			}

			if retrieved != leaves[i] {
				t.Errorf("leaf %d does not match after restore", i)
			}
		}
	})
}

// Helper function to hash data
func hashData(data []byte) [32]byte {
	return sha256.Sum256(data)
}
