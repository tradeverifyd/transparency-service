package merkle

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
)

// TestAnalyzeTileUsage analyzes which tiles are accessed for different tree sizes
func TestAnalyzeTileUsage(t *testing.T) {
	testCases := []struct {
		name     string
		treeSize int64
	}{
		{"256 entries", 256},
		{"512 entries", 512},
		{"1024 entries", 1024},
		{"10000 entries", 10000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create tracking storage
			underlying := storage.NewMemoryStorage()
			tracker := &trackingStorage{
				Storage: underlying,
				gets:    make(map[string]int),
			}

			// Populate log
			for i := int64(0); i < tc.treeSize; i++ {
				leaf := sha256.Sum256([]byte(fmt.Sprintf("entry-%d", i)))
				if err := appendToEntryTileHelper(tracker, i, leaf[:]); err != nil {
					t.Fatalf("failed to append entry %d: %v", i, err)
				}
			}

			// Clear tracking (we only care about reads, not writes)
			tracker.gets = make(map[string]int)

			// Generate a single receipt (middle of tree)
			entryID := tc.treeSize / 2
			_, err := GenerateInclusionProof(tracker, entryID, tc.treeSize)
			if err != nil {
				t.Fatalf("failed to generate proof: %v", err)
			}

			_, err = ComputeTreeRoot(tracker, tc.treeSize)
			if err != nil {
				t.Fatalf("failed to compute root: %v", err)
			}

			// Analyze tile access
			t.Logf("\n=== Tile Access for %s (single receipt) ===", tc.name)
			t.Logf("Total unique tiles accessed: %d", len(tracker.gets))

			// Categorize by tile type
			entryTiles := 0
			partialEntryTiles := 0
			level0Tiles := 0
			level1Tiles := 0
			otherTiles := 0

			for path, count := range tracker.gets {
				switch {
				case contains(path, "tile/entries/") && contains(path, ".p/"):
					partialEntryTiles++
					t.Logf("  Partial entry tile: %s (accessed %d times)", path, count)
				case contains(path, "tile/entries/"):
					entryTiles++
					if count > 1 {
						t.Logf("  Full entry tile: %s (accessed %d times)", path, count)
					}
				case contains(path, "tile/0/"):
					level0Tiles++
				case contains(path, "tile/1/"):
					level1Tiles++
				default:
					otherTiles++
				}
			}

			t.Logf("\nTile breakdown:")
			t.Logf("  Entry tiles (full): %d", entryTiles)
			t.Logf("  Entry tiles (partial): %d", partialEntryTiles)
			t.Logf("  Level 0 tiles: %d", level0Tiles)
			t.Logf("  Level 1 tiles: %d", level1Tiles)
			t.Logf("  Other tiles: %d", otherTiles)

			// Calculate storage size
			t.Logf("\nStorage requirements:")
			t.Logf("  Entry tiles: %d × 8KB = %d KB", entryTiles+partialEntryTiles,
				(entryTiles+partialEntryTiles)*8)
			t.Logf("  Expected cache size for full tiles: ~%d KB", entryTiles*8)
		})
	}
}

// trackingStorage wraps storage to track Get operations
type trackingStorage struct {
	storage.Storage
	gets map[string]int
}

func (t *trackingStorage) Get(key string) ([]byte, error) {
	t.gets[key]++
	return t.Storage.Get(key)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
		 (len(s) > len(substr) && s[:len(substr)] == substr) ||
		 indexOfSubstring(s, substr) >= 0)
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
