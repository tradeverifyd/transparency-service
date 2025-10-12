package merkle_test

import (
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/merkle"
)

func TestTileIndexToPath(t *testing.T) {
	tests := []struct {
		name     string
		level    int
		index    int64
		width    *int
		expected string
	}{
		{"simple index", 0, 42, nil, "tile/0/042"},
		{"index 0", 0, 0, nil, "tile/0/000"},
		{"index 255", 0, 255, nil, "tile/0/255"},
		{"index 256 base-256", 0, 256, nil, "tile/0/x001/000"},
		{"index 1234 base-256", 0, 1234, nil, "tile/0/x004/210"},
		{"index 65536 decimal", 0, 65536, nil, "tile/0/x065/536"},
		{"index 1234067 decimal", 0, 1234067, nil, "tile/0/x001/x234/067"},
		{"partial tile", 0, 42, intPtr(128), "tile/0/042.p/128"},
		{"level 1", 1, 10, nil, "tile/1/010"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := merkle.TileIndexToPath(tt.level, tt.index, tt.width)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParseTilePath(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		expectedLevel int
		expectedIndex int64
		expectedWidth *int
	}{
		{"simple", "tile/0/042", 0, 42, nil},
		{"index 0", "tile/0/000", 0, 0, nil},
		{"base-256", "tile/0/x004/210", 0, 1234, nil},
		{"decimal grouping", "tile/0/x001/x234/067", 0, 1234067, nil},
		{"partial tile", "tile/0/042.p/128", 0, 42, intPtr(128)},
		{"level 1", "tile/1/010", 1, 10, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := merkle.ParseTilePath(tt.path)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			if parsed.Level != tt.expectedLevel {
				t.Errorf("expected level %d, got %d", tt.expectedLevel, parsed.Level)
			}
			if parsed.Index != tt.expectedIndex {
				t.Errorf("expected index %d, got %d", tt.expectedIndex, parsed.Index)
			}

			if tt.expectedWidth == nil {
				if parsed.IsPartial {
					t.Error("expected full tile, got partial")
				}
			} else {
				if !parsed.IsPartial {
					t.Error("expected partial tile, got full")
				}
				if parsed.Width != *tt.expectedWidth {
					t.Errorf("expected width %d, got %d", *tt.expectedWidth, parsed.Width)
				}
			}
		})
	}
}

func TestEntryTileIndexToPath(t *testing.T) {
	tests := []struct {
		name     string
		index    int64
		width    *int
		expected string
	}{
		{"simple index", 0, nil, "tile/entries/000"},
		{"index 42", 42, nil, "tile/entries/042"},
		{"index 256", 256, nil, "tile/entries/x001/000"},
		{"partial tile", 10, intPtr(200), "tile/entries/010.p/200"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := merkle.EntryTileIndexToPath(tt.index, tt.width)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestParseEntryTilePath(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		expectedIndex int64
		expectedWidth *int
	}{
		{"simple", "tile/entries/000", 0, nil},
		{"index 42", "tile/entries/042", 42, nil},
		{"base-256", "tile/entries/x001/000", 256, nil},
		{"partial", "tile/entries/010.p/200", 10, intPtr(200)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := merkle.ParseEntryTilePath(tt.path)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			if parsed.Index != tt.expectedIndex {
				t.Errorf("expected index %d, got %d", tt.expectedIndex, parsed.Index)
			}

			if tt.expectedWidth == nil {
				if parsed.IsPartial {
					t.Error("expected full tile, got partial")
				}
			} else {
				if !parsed.IsPartial {
					t.Error("expected partial tile, got full")
				}
				if parsed.Width != *tt.expectedWidth {
					t.Errorf("expected width %d, got %d", *tt.expectedWidth, parsed.Width)
				}
			}
		})
	}
}

func TestEntryIDToTileIndex(t *testing.T) {
	tests := []struct {
		entryID       int64
		expectedIndex int64
	}{
		{0, 0},
		{1, 0},
		{255, 0},
		{256, 1},
		{257, 1},
		{511, 1},
		{512, 2},
		{1000, 3},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			index := merkle.EntryIDToTileIndex(tt.entryID)
			if index != tt.expectedIndex {
				t.Errorf("entry ID %d: expected tile index %d, got %d", tt.entryID, tt.expectedIndex, index)
			}
		})
	}
}

func TestEntryIDToTileOffset(t *testing.T) {
	tests := []struct {
		entryID        int64
		expectedOffset int
	}{
		{0, 0},
		{1, 1},
		{255, 255},
		{256, 0},
		{257, 1},
		{511, 255},
		{512, 0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			offset := merkle.EntryIDToTileOffset(tt.entryID)
			if offset != tt.expectedOffset {
				t.Errorf("entry ID %d: expected offset %d, got %d", tt.entryID, tt.expectedOffset, offset)
			}
		})
	}
}

func TestTileCoordinatesToEntryID(t *testing.T) {
	tests := []struct {
		tileIndex      int64
		tileOffset     int
		expectedEntryID int64
	}{
		{0, 0, 0},
		{0, 1, 1},
		{0, 255, 255},
		{1, 0, 256},
		{1, 1, 257},
		{2, 0, 512},
		{3, 232, 1000},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			entryID := merkle.TileCoordinatesToEntryID(tt.tileIndex, tt.tileOffset)
			if entryID != tt.expectedEntryID {
				t.Errorf("tile %d offset %d: expected entry ID %d, got %d",
					tt.tileIndex, tt.tileOffset, tt.expectedEntryID, entryID)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	t.Run("tile path round trip", func(t *testing.T) {
		testIndices := []int64{0, 1, 42, 255, 256, 1234, 65536, 1234067}

		for _, index := range testIndices {
			path := merkle.TileIndexToPath(0, index, nil)
			parsed, err := merkle.ParseTilePath(path)
			if err != nil {
				t.Fatalf("failed to parse path %s: %v", path, err)
			}

			if parsed.Index != index {
				t.Errorf("index %d: round trip failed, got %d", index, parsed.Index)
			}
		}
	})

	t.Run("entry coordinates round trip", func(t *testing.T) {
		testEntryIDs := []int64{0, 1, 255, 256, 512, 1000, 10000}

		for _, entryID := range testEntryIDs {
			tileIndex := merkle.EntryIDToTileIndex(entryID)
			tileOffset := merkle.EntryIDToTileOffset(entryID)
			roundTrip := merkle.TileCoordinatesToEntryID(tileIndex, tileOffset)

			if roundTrip != entryID {
				t.Errorf("entry ID %d: round trip failed, got %d", entryID, roundTrip)
			}
		}
	})
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
