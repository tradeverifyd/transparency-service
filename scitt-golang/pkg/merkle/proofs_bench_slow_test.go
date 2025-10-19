package merkle

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
)

// BenchmarkReceiptGeneration_256Entries_SlowStorage simulates realistic storage latency (1ms per read)
func BenchmarkReceiptGeneration_256Entries_SlowStorage(b *testing.B) {
	underlying := storage.NewMemoryStorage()
	store := storage.NewSlowStorage(underlying, 1*time.Millisecond)

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

// BenchmarkReceiptGeneration_256Entries_SlowStorage_Cached uses caching with slow storage
func BenchmarkReceiptGeneration_256Entries_SlowStorage_Cached(b *testing.B) {
	underlying := storage.NewMemoryStorage()
	slow := storage.NewSlowStorage(underlying, 1*time.Millisecond)
	store := storage.NewCachedStorage(slow)

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

// BenchmarkReceiptGeneration_512Entries_SlowStorage simulates realistic storage latency (1ms per read)
func BenchmarkReceiptGeneration_512Entries_SlowStorage(b *testing.B) {
	underlying := storage.NewMemoryStorage()
	store := storage.NewSlowStorage(underlying, 1*time.Millisecond)

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

// BenchmarkReceiptGeneration_512Entries_SlowStorage_Cached uses caching with slow storage
func BenchmarkReceiptGeneration_512Entries_SlowStorage_Cached(b *testing.B) {
	underlying := storage.NewMemoryStorage()
	slow := storage.NewSlowStorage(underlying, 1*time.Millisecond)
	store := storage.NewCachedStorage(slow)

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

// BenchmarkComputeTreeRoot_256Entries_SlowStorage_Comparison compares with and without caching
func BenchmarkComputeTreeRoot_256Entries_SlowStorage_Comparison(b *testing.B) {
	b.Run("Without_Cache", func(b *testing.B) {
		underlying := storage.NewMemoryStorage()
		store := storage.NewSlowStorage(underlying, 1*time.Millisecond)

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
	})

	b.Run("With_Cache", func(b *testing.B) {
		underlying := storage.NewMemoryStorage()
		slow := storage.NewSlowStorage(underlying, 1*time.Millisecond)
		store := storage.NewCachedStorage(slow)

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
	})
}

// BenchmarkComputeTreeRoot_512Entries_SlowStorage_Comparison compares with and without caching
func BenchmarkComputeTreeRoot_512Entries_SlowStorage_Comparison(b *testing.B) {
	b.Run("Without_Cache", func(b *testing.B) {
		underlying := storage.NewMemoryStorage()
		store := storage.NewSlowStorage(underlying, 1*time.Millisecond)

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
	})

	b.Run("With_Cache", func(b *testing.B) {
		underlying := storage.NewMemoryStorage()
		slow := storage.NewSlowStorage(underlying, 1*time.Millisecond)
		store := storage.NewCachedStorage(slow)

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
	})
}
