// Package merkle provides Merkle tree operations for the transparency service
package merkle

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// C2SP Constants
const (
	TileSize       = 256 // Hashes per tile
	HashSize       = 32  // SHA-256 hash size in bytes
	FullTileBytes  = TileSize * HashSize // 8192 bytes
)

// ParsedTilePath represents components of a parsed tile path
type ParsedTilePath struct {
	Level     int
	Index     int64
	IsPartial bool
	Width     int
}

// ParsedEntryTilePath represents components of a parsed entry tile path
type ParsedEntryTilePath struct {
	Index     int64
	IsPartial bool
	Width     int
}

// TileIndexToPath generates a tile path from level and index
// Format: tile/<L>/<N>[.p/<W>]
//
// For indices >= 256, uses x-prefixed segments:
// Index 1234067 → tile/0/x001/x234/067
func TileIndexToPath(level int, index int64, width *int) string {
	if width != nil && (*width < 1 || *width > 255) {
		panic(fmt.Sprintf("invalid partial tile width: %d. Must be between 1 and 255", *width))
	}

	indexPath := formatIndexPath(index)
	basePath := fmt.Sprintf("tile/%d/%s", level, indexPath)

	if width != nil {
		return fmt.Sprintf("%s.p/%d", basePath, *width)
	}

	return basePath
}

// ParseTilePath parses a tile path into components
func ParseTilePath(path string) (*ParsedTilePath, error) {
	// Match format: tile/<level>/<segments...>[.p/<width>]
	if strings.Contains(path, ".p/") {
		parts := strings.SplitN(path, ".p/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid partial tile path format: %s", path)
		}

		basePath := parts[0]
		widthStr := parts[1]

		level, indexPath, err := parseTileBasePath(basePath)
		if err != nil {
			return nil, err
		}

		index, err := parseIndexPath(indexPath)
		if err != nil {
			return nil, err
		}

		width, err := strconv.Atoi(widthStr)
		if err != nil {
			return nil, fmt.Errorf("invalid width: %w", err)
		}

		return &ParsedTilePath{
			Level:     level,
			Index:     index,
			IsPartial: true,
			Width:     width,
		}, nil
	}

	level, indexPath, err := parseTileBasePath(path)
	if err != nil {
		return nil, err
	}

	index, err := parseIndexPath(indexPath)
	if err != nil {
		return nil, err
	}

	return &ParsedTilePath{
		Level:     level,
		Index:     index,
		IsPartial: false,
	}, nil
}

// EntryTileIndexToPath generates an entry tile path
// Format: tile/entries/<N>[.p/<W>]
func EntryTileIndexToPath(index int64, width *int) string {
	if width != nil && (*width < 1 || *width > 255) {
		panic(fmt.Sprintf("invalid partial tile width: %d. Must be between 1 and 255", *width))
	}

	indexPath := formatIndexPath(index)
	basePath := fmt.Sprintf("tile/entries/%s", indexPath)

	if width != nil {
		return fmt.Sprintf("%s.p/%d", basePath, *width)
	}

	return basePath
}

// ParseEntryTilePath parses an entry tile path into components
func ParseEntryTilePath(path string) (*ParsedEntryTilePath, error) {
	if strings.Contains(path, ".p/") {
		parts := strings.SplitN(path, ".p/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid partial entry tile path format: %s", path)
		}

		basePath := parts[0]
		widthStr := parts[1]

		if !strings.HasPrefix(basePath, "tile/entries/") {
			return nil, fmt.Errorf("invalid entry tile path format: %s", path)
		}

		indexPath := strings.TrimPrefix(basePath, "tile/entries/")
		index, err := parseIndexPath(indexPath)
		if err != nil {
			return nil, err
		}

		width, err := strconv.Atoi(widthStr)
		if err != nil {
			return nil, fmt.Errorf("invalid width: %w", err)
		}

		return &ParsedEntryTilePath{
			Index:     index,
			IsPartial: true,
			Width:     width,
		}, nil
	}

	if !strings.HasPrefix(path, "tile/entries/") {
		return nil, fmt.Errorf("invalid entry tile path format: %s", path)
	}

	indexPath := strings.TrimPrefix(path, "tile/entries/")
	index, err := parseIndexPath(indexPath)
	if err != nil {
		return nil, err
	}

	return &ParsedEntryTilePath{
		Index:     index,
		IsPartial: false,
	}, nil
}

// EntryIDToTileIndex calculates tile index from entry ID
// Entry ID 0-255 → tile 0, 256-511 → tile 1, etc.
func EntryIDToTileIndex(entryID int64) int64 {
	return entryID / TileSize
}

// EntryIDToTileOffset calculates tile offset from entry ID
// Entry ID modulo tile size
func EntryIDToTileOffset(entryID int64) int {
	return int(entryID % TileSize)
}

// TileCoordinatesToEntryID calculates entry ID from tile coordinates
// Reverse of EntryIDToTileIndex/EntryIDToTileOffset
func TileCoordinatesToEntryID(tileIndex int64, tileOffset int) int64 {
	return tileIndex*TileSize + int64(tileOffset)
}

// formatIndexPath formats index as path segments
// C2SP tlog-tiles uses a hybrid approach:
// - Index 0-255: Simple 3-digit format "042"
// - Index 256-65535: Base-256 encoding "x001/000"
// - Index >= 65536: Decimal digit grouping "x001/x234/067"
func formatIndexPath(index int64) string {
	if index < 256 {
		// Simple 3-digit format with leading zeros
		return fmt.Sprintf("%03d", index)
	}

	if index < 65536 {
		// Base-256 encoding for indices 256-65535
		var digits []int64
		remaining := index

		for remaining >= 256 {
			digits = append([]int64{remaining % 256}, digits...)
			remaining = remaining / 256
		}
		digits = append([]int64{remaining}, digits...)

		segments := make([]string, len(digits))
		for i, digit := range digits {
			if i < len(digits)-1 {
				segments[i] = fmt.Sprintf("x%03d", digit)
			} else {
				segments[i] = fmt.Sprintf("%03d", digit)
			}
		}

		return strings.Join(segments, "/")
	}

	// For indices >= 65536, use decimal digit grouping
	indexStr := strconv.FormatInt(index, 10)
	paddedLength := int(math.Ceil(float64(len(indexStr))/3.0)) * 3
	paddedIndex := fmt.Sprintf("%0*s", paddedLength, indexStr)

	var segments []string
	for i := 0; i < len(paddedIndex); i += 3 {
		segments = append(segments, paddedIndex[i:i+3])
	}

	// Add x prefix to all but the last segment
	for i := 0; i < len(segments)-1; i++ {
		segments[i] = "x" + segments[i]
	}

	return strings.Join(segments, "/")
}

// parseIndexPath parses index path segments into index number
// Reverse of formatIndexPath
func parseIndexPath(indexPath string) (int64, error) {
	segments := strings.Split(indexPath, "/")

	if len(segments) == 1 {
		// Simple 3-digit format (0-255)
		return strconv.ParseInt(segments[0], 10, 64)
	}

	// Strip x prefixes and parse values
	values := make([]int64, len(segments))
	for i, s := range segments {
		if strings.HasPrefix(s, "x") {
			val, err := strconv.ParseInt(s[1:], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid segment %s: %w", s, err)
			}
			values[i] = val
		} else {
			val, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid segment %s: %w", s, err)
			}
			values[i] = val
		}
	}

	// Try base-256 first (for indices 256-65535)
	var base256Result int64
	for _, val := range values {
		base256Result = base256Result*256 + val
	}

	// Check if this could be base-256 encoding
	// Base-256 is only valid if all segments are < 256 and result < 65536
	allValid := true
	for _, val := range values {
		if val >= 256 {
			allValid = false
			break
		}
	}

	if allValid && base256Result < 65536 {
		return base256Result, nil
	}

	// Otherwise, it's decimal grouping (e.g., "x001/x234/067" → 1234067)
	// Concatenate all segments as decimal digits
	var concatenated strings.Builder
	for _, val := range values {
		concatenated.WriteString(fmt.Sprintf("%03d", val))
	}

	return strconv.ParseInt(concatenated.String(), 10, 64)
}

// parseTileBasePath extracts level and index path from tile base path
func parseTileBasePath(path string) (level int, indexPath string, err error) {
	if !strings.HasPrefix(path, "tile/") {
		return 0, "", fmt.Errorf("invalid tile path format: %s", path)
	}

	parts := strings.SplitN(strings.TrimPrefix(path, "tile/"), "/", 2)
	if len(parts) < 2 {
		return 0, "", fmt.Errorf("invalid tile path format: %s", path)
	}

	level, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid level: %w", err)
	}

	return level, parts[1], nil
}
