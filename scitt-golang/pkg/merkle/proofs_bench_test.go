package merkle

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
)

// BenchmarkReceiptGeneration_256Entries benchmarks receipt generation for a log with 1 complete tile (256 entries)
// This simulates the realistic scenario where we generate receipts for all entries in a full tile
func BenchmarkReceiptGeneration_256Entries(b *testing.B) {
	store := storage.NewMemoryStorage()

	// Populate log with 256 entries (1 complete tile)
	const treeSize = 256
	for i := int64(0); i < treeSize; i++ {
		leaf := sha256.Sum256([]byte(fmt.Sprintf("entry-%d", i)))
		if err := appendToEntryTileHelper(store, i, leaf[:]); err != nil {
			b.Fatalf("failed to append entry %d: %v", i, err)
		}
	}

	// Benchmark generating receipts for all entries
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Generate receipt for a random entry (entry 128 - middle of the tree)
		// This represents a typical use case
		entryID := int64(128)
		_, err := GenerateInclusionProof(store, entryID, treeSize)
		if err != nil {
			b.Fatalf("failed to generate proof: %v", err)
		}

		// Also compute the root hash (required for receipts)
		_, err = ComputeTreeRoot(store, treeSize)
		if err != nil {
			b.Fatalf("failed to compute root: %v", err)
		}
	}
}

// BenchmarkReceiptGeneration_256Entries_AllEntries benchmarks generating receipts for ALL entries in a 256-entry log
// This is the worst-case scenario that shows the cumulative cost
func BenchmarkReceiptGeneration_256Entries_AllEntries(b *testing.B) {
	store := storage.NewMemoryStorage()

	// Populate log with 256 entries (1 complete tile)
	const treeSize = 256
	for i := int64(0); i < treeSize; i++ {
		leaf := sha256.Sum256([]byte(fmt.Sprintf("entry-%d", i)))
		if err := appendToEntryTileHelper(store, i, leaf[:]); err != nil {
			b.Fatalf("failed to append entry %d: %v", i, err)
		}
	}

	// Benchmark generating receipts for ALL entries
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for entryID := int64(0); entryID < treeSize; entryID++ {
			_, err := GenerateInclusionProof(store, entryID, treeSize)
			if err != nil {
				b.Fatalf("failed to generate proof for entry %d: %v", entryID, err)
			}

			// Compute root once per receipt (as done in GetReceipt)
			_, err = ComputeTreeRoot(store, treeSize)
			if err != nil {
				b.Fatalf("failed to compute root: %v", err)
			}
		}
	}
}

// BenchmarkReceiptGeneration_512Entries benchmarks receipt generation for a log with 2 complete tiles (512 entries)
func BenchmarkReceiptGeneration_512Entries(b *testing.B) {
	store := storage.NewMemoryStorage()

	// Populate log with 512 entries (2 complete tiles)
	const treeSize = 512
	for i := int64(0); i < treeSize; i++ {
		leaf := sha256.Sum256([]byte(fmt.Sprintf("entry-%d", i)))
		if err := appendToEntryTileHelper(store, i, leaf[:]); err != nil {
			b.Fatalf("failed to append entry %d: %v", i, err)
		}
	}

	// Benchmark generating receipts for a middle entry
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Generate receipt for entry 256 (middle of the tree, spanning both tiles)
		entryID := int64(256)
		_, err := GenerateInclusionProof(store, entryID, treeSize)
		if err != nil {
			b.Fatalf("failed to generate proof: %v", err)
		}

		// Also compute the root hash (required for receipts)
		_, err = ComputeTreeRoot(store, treeSize)
		if err != nil {
			b.Fatalf("failed to compute root: %v", err)
		}
	}
}

// BenchmarkReceiptGeneration_512Entries_AllEntries benchmarks generating receipts for ALL entries in a 512-entry log
func BenchmarkReceiptGeneration_512Entries_AllEntries(b *testing.B) {
	store := storage.NewMemoryStorage()

	// Populate log with 512 entries (2 complete tiles)
	const treeSize = 512
	for i := int64(0); i < treeSize; i++ {
		leaf := sha256.Sum256([]byte(fmt.Sprintf("entry-%d", i)))
		if err := appendToEntryTileHelper(store, i, leaf[:]); err != nil {
			b.Fatalf("failed to append entry %d: %v", i, err)
		}
	}

	// Benchmark generating receipts for ALL entries
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for entryID := int64(0); entryID < treeSize; entryID++ {
			_, err := GenerateInclusionProof(store, entryID, treeSize)
			if err != nil {
				b.Fatalf("failed to generate proof for entry %d: %v", entryID, err)
			}

			// Compute root once per receipt (as done in GetReceipt)
			_, err = ComputeTreeRoot(store, treeSize)
			if err != nil {
				b.Fatalf("failed to compute root: %v", err)
			}
		}
	}
}

// BenchmarkComputeTreeRoot_256Entries benchmarks just the root computation for 256 entries
func BenchmarkComputeTreeRoot_256Entries(b *testing.B) {
	store := storage.NewMemoryStorage()

	// Populate log with 256 entries
	const treeSize = 256
	for i := int64(0); i < treeSize; i++ {
		leaf := sha256.Sum256([]byte(fmt.Sprintf("entry-%d", i)))
		if err := appendToEntryTileHelper(store, i, leaf[:]); err != nil {
			b.Fatalf("failed to append entry %d: %v", i, err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ComputeTreeRoot(store, treeSize)
		if err != nil {
			b.Fatalf("failed to compute root: %v", err)
		}
	}
}

// BenchmarkComputeTreeRoot_512Entries benchmarks just the root computation for 512 entries
func BenchmarkComputeTreeRoot_512Entries(b *testing.B) {
	store := storage.NewMemoryStorage()

	// Populate log with 512 entries
	const treeSize = 512
	for i := int64(0); i < treeSize; i++ {
		leaf := sha256.Sum256([]byte(fmt.Sprintf("entry-%d", i)))
		if err := appendToEntryTileHelper(store, i, leaf[:]); err != nil {
			b.Fatalf("failed to append entry %d: %v", i, err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ComputeTreeRoot(store, treeSize)
		if err != nil {
			b.Fatalf("failed to compute root: %v", err)
		}
	}
}

// BenchmarkStorageGet_TileAccess benchmarks raw storage access to understand baseline cost
func BenchmarkStorageGet_TileAccess(b *testing.B) {
	store := storage.NewMemoryStorage()

	// Create a single tile with 256 entries
	const tileSize = 256
	tileData := make([]byte, tileSize*HashSize)
	for i := 0; i < tileSize; i++ {
		leaf := sha256.Sum256([]byte(fmt.Sprintf("entry-%d", i)))
		copy(tileData[i*HashSize:], leaf[:])
	}

	tilePath := EntryTileIndexToPath(0, nil)
	if err := store.Put(tilePath, tileData); err != nil {
		b.Fatalf("failed to put tile: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Get(tilePath)
		if err != nil {
			b.Fatalf("failed to get tile: %v", err)
		}
	}
}

// BenchmarkReceiptGeneration_256Entries_Cached benchmarks with cached storage (1 complete tile)
func BenchmarkReceiptGeneration_256Entries_Cached(b *testing.B) {
	underlying := storage.NewMemoryStorage()
	store := storage.NewCachedStorage(underlying)

	// Populate log with 256 entries (1 complete tile)
	const treeSize = 256
	for i := int64(0); i < treeSize; i++ {
		leaf := sha256.Sum256([]byte(fmt.Sprintf("entry-%d", i)))
		if err := appendToEntryTileHelper(store, i, leaf[:]); err != nil {
			b.Fatalf("failed to append entry %d: %v", i, err)
		}
	}

	// Benchmark generating receipts
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entryID := int64(128)
		_, err := GenerateInclusionProof(store, entryID, treeSize)
		if err != nil {
			b.Fatalf("failed to generate proof: %v", err)
		}

		_, err = ComputeTreeRoot(store, treeSize)
		if err != nil {
			b.Fatalf("failed to compute root: %v", err)
		}
	}
}

// BenchmarkReceiptGeneration_256Entries_AllEntries_Cached benchmarks all entries with cached storage
func BenchmarkReceiptGeneration_256Entries_AllEntries_Cached(b *testing.B) {
	underlying := storage.NewMemoryStorage()
	store := storage.NewCachedStorage(underlying)

	// Populate log with 256 entries (1 complete tile)
	const treeSize = 256
	for i := int64(0); i < treeSize; i++ {
		leaf := sha256.Sum256([]byte(fmt.Sprintf("entry-%d", i)))
		if err := appendToEntryTileHelper(store, i, leaf[:]); err != nil {
			b.Fatalf("failed to append entry %d: %v", i, err)
		}
	}

	// Benchmark generating receipts for ALL entries
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for entryID := int64(0); entryID < treeSize; entryID++ {
			_, err := GenerateInclusionProof(store, entryID, treeSize)
			if err != nil {
				b.Fatalf("failed to generate proof for entry %d: %v", entryID, err)
			}

			_, err = ComputeTreeRoot(store, treeSize)
			if err != nil {
				b.Fatalf("failed to compute root: %v", err)
			}
		}
	}
}

// BenchmarkReceiptGeneration_512Entries_Cached benchmarks with cached storage (2 complete tiles)
func BenchmarkReceiptGeneration_512Entries_Cached(b *testing.B) {
	underlying := storage.NewMemoryStorage()
	store := storage.NewCachedStorage(underlying)

	// Populate log with 512 entries (2 complete tiles)
	const treeSize = 512
	for i := int64(0); i < treeSize; i++ {
		leaf := sha256.Sum256([]byte(fmt.Sprintf("entry-%d", i)))
		if err := appendToEntryTileHelper(store, i, leaf[:]); err != nil {
			b.Fatalf("failed to append entry %d: %v", i, err)
		}
	}

	// Benchmark generating receipts
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entryID := int64(256)
		_, err := GenerateInclusionProof(store, entryID, treeSize)
		if err != nil {
			b.Fatalf("failed to generate proof: %v", err)
		}

		_, err = ComputeTreeRoot(store, treeSize)
		if err != nil {
			b.Fatalf("failed to compute root: %v", err)
		}
	}
}

// BenchmarkReceiptGeneration_512Entries_AllEntries_Cached benchmarks all entries with cached storage
func BenchmarkReceiptGeneration_512Entries_AllEntries_Cached(b *testing.B) {
	underlying := storage.NewMemoryStorage()
	store := storage.NewCachedStorage(underlying)

	// Populate log with 512 entries (2 complete tiles)
	const treeSize = 512
	for i := int64(0); i < treeSize; i++ {
		leaf := sha256.Sum256([]byte(fmt.Sprintf("entry-%d", i)))
		if err := appendToEntryTileHelper(store, i, leaf[:]); err != nil {
			b.Fatalf("failed to append entry %d: %v", i, err)
		}
	}

	// Benchmark generating receipts for ALL entries
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for entryID := int64(0); entryID < treeSize; entryID++ {
			_, err := GenerateInclusionProof(store, entryID, treeSize)
			if err != nil {
				b.Fatalf("failed to generate proof for entry %d: %v", entryID, err)
			}

			_, err = ComputeTreeRoot(store, treeSize)
			if err != nil {
				b.Fatalf("failed to compute root: %v", err)
			}
		}
	}
}

// BenchmarkComputeTreeRoot_256Entries_Cached benchmarks root computation with cached storage
func BenchmarkComputeTreeRoot_256Entries_Cached(b *testing.B) {
	underlying := storage.NewMemoryStorage()
	store := storage.NewCachedStorage(underlying)

	// Populate log with 256 entries
	const treeSize = 256
	for i := int64(0); i < treeSize; i++ {
		leaf := sha256.Sum256([]byte(fmt.Sprintf("entry-%d", i)))
		if err := appendToEntryTileHelper(store, i, leaf[:]); err != nil {
			b.Fatalf("failed to append entry %d: %v", i, err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ComputeTreeRoot(store, treeSize)
		if err != nil {
			b.Fatalf("failed to compute root: %v", err)
		}
	}
}

// BenchmarkComputeTreeRoot_512Entries_Cached benchmarks root computation with cached storage
func BenchmarkComputeTreeRoot_512Entries_Cached(b *testing.B) {
	underlying := storage.NewMemoryStorage()
	store := storage.NewCachedStorage(underlying)

	// Populate log with 512 entries
	const treeSize = 512
	for i := int64(0); i < treeSize; i++ {
		leaf := sha256.Sum256([]byte(fmt.Sprintf("entry-%d", i)))
		if err := appendToEntryTileHelper(store, i, leaf[:]); err != nil {
			b.Fatalf("failed to append entry %d: %v", i, err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ComputeTreeRoot(store, treeSize)
		if err != nil {
			b.Fatalf("failed to compute root: %v", err)
		}
	}
}

// appendToEntryTileHelper is a helper function to append leaves to tiles
// This is used for test setup and matches the logic in service.go
func appendToEntryTileHelper(store storage.Storage, entryID int64, leafHash []byte) error {
	tileIndex := EntryIDToTileIndex(entryID)
	tilePath := EntryTileIndexToPath(tileIndex, nil)

	// Read existing tile (if any)
	existingTile, err := store.Get(tilePath)
	if err != nil {
		return fmt.Errorf("failed to get existing tile: %w", err)
	}

	var currentSize int
	if existingTile != nil {
		currentSize = len(existingTile) / HashSize
	}

	// Append new leaf
	newTile := make([]byte, (currentSize+1)*HashSize)
	if existingTile != nil {
		copy(newTile, existingTile)
	}
	copy(newTile[currentSize*HashSize:], leafHash)

	// Write updated tile
	if err := store.Put(tilePath, newTile); err != nil {
		return fmt.Errorf("failed to put tile: %w", err)
	}

	return nil
}
