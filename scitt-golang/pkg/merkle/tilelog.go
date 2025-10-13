package merkle

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
	"golang.org/x/mod/sumdb/tlog"
)

// TileLog represents a tile-based Merkle tree
// Implements RFC 6962 Merkle tree with C2SP tlog-tiles storage
type TileLog struct {
	storage storage.Storage
	size    int64
}

// TileLogState represents the persistent state of the tree
type TileLogState struct {
	Size int64  `json:"size"`
	Root []byte `json:"root,omitempty"`
}

// NewTileLog creates a new tile log with the given storage
func NewTileLog(storage storage.Storage) *TileLog {
	return &TileLog{
		storage: storage,
		size:    0,
	}
}

// Load loads the tree state from storage
func (tl *TileLog) Load() error {
	stateData, err := tl.storage.Get(".tree-state")
	if err != nil {
		return fmt.Errorf("failed to get tree state: %w", err)
	}

	if stateData == nil {
		// Empty tree - no state file exists yet
		tl.size = 0
		return nil
	}

	var state TileLogState
	if err := json.Unmarshal(stateData, &state); err != nil {
		return fmt.Errorf("failed to unmarshal tree state: %w", err)
	}

	tl.size = state.Size
	return nil
}

// saveState saves the tree state to storage
func (tl *TileLog) saveState() error {
	var root []byte
	if tl.size > 0 {
		rootHash, err := tl.TreeHash(tl.size)
		if err != nil {
			return fmt.Errorf("failed to compute tree hash: %w", err)
		}
		root = rootHash[:]
	}

	state := TileLogState{
		Size: tl.size,
		Root: root,
	}

	stateData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal tree state: %w", err)
	}

	if err := tl.storage.Put(".tree-state", stateData); err != nil {
		return fmt.Errorf("failed to put tree state: %w", err)
	}

	return nil
}

// Append appends a leaf to the tree
// Returns the entry ID of the appended leaf
// The leaf should be the raw record hash (e.g., SHA-256 of statement)
// This function will apply the RFC 6962 leaf prefix (0x00) before storing
func (tl *TileLog) Append(leaf [HashSize]byte) (int64, error) {
	entryID := tl.size

	// Apply RFC 6962 leaf hash prefix (0x00) to the raw record hash
	// This converts the raw hash into a proper Merkle tree leaf hash
	leafHash := hashLeaf(leaf)

	// Store the leaf hash (with RFC 6962 prefix applied) in entry tile
	if err := tl.appendToEntryTile(entryID, leafHash); err != nil {
		return 0, fmt.Errorf("failed to append to entry tile: %w", err)
	}

	// Increment size AFTER appending so TreeHash can read the new leaf
	// TreeHash(n) expects to be able to read leaves 0 through n-1
	// We increment here so the new leaf at entryID is included in tl.size
	tl.size++

	// Persist state (this will compute TreeHash(tl.size) which includes the new leaf)
	if err := tl.saveState(); err != nil {
		// Roll back size increment on failure
		tl.size--
		return 0, fmt.Errorf("failed to save state: %w", err)
	}

	return entryID, nil
}

// Size returns the current tree size (number of leaves)
func (tl *TileLog) Size() int64 {
	return tl.size
}

// Root returns the current tree root hash
func (tl *TileLog) Root() ([HashSize]byte, error) {
	if tl.size == 0 {
		return [HashSize]byte{}, fmt.Errorf("cannot get root of empty tree")
	}

	return tl.TreeHash(tl.size)
}

// GetLeaf retrieves a leaf by entry ID
func (tl *TileLog) GetLeaf(entryID int64) ([HashSize]byte, error) {
	if entryID >= tl.size {
		return [HashSize]byte{}, fmt.Errorf("entry ID %d out of bounds (size: %d)", entryID, tl.size)
	}

	return tl.getLeafDirect(entryID)
}

// getLeafDirect retrieves a leaf by entry ID without bounds checking
func (tl *TileLog) getLeafDirect(entryID int64) ([HashSize]byte, error) {
	tileIndex := EntryIDToTileIndex(entryID)
	tileOffset := EntryIDToTileOffset(entryID)

	tilePath := EntryTileIndexToPath(tileIndex, nil)
	tileData, err := tl.storage.Get(tilePath)
	if err != nil {
		return [HashSize]byte{}, fmt.Errorf("failed to get entry tile: %w", err)
	}

	if tileData == nil {
		return [HashSize]byte{}, fmt.Errorf("entry tile not found: %s", tilePath)
	}

	// Extract the specific hash from the tile
	start := tileOffset * HashSize
	end := start + HashSize

	if end > len(tileData) {
		return [HashSize]byte{}, fmt.Errorf("tile data too short for offset %d", tileOffset)
	}

	var leaf [HashSize]byte
	copy(leaf[:], tileData[start:end])
	return leaf, nil
}

// appendToEntryTile appends a leaf to an entry tile
func (tl *TileLog) appendToEntryTile(entryID int64, leaf [HashSize]byte) error {
	tileIndex := EntryIDToTileIndex(entryID)
	tilePath := EntryTileIndexToPath(tileIndex, nil)

	// Read existing tile (if any)
	existingTile, err := tl.storage.Get(tilePath)
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
	copy(newTile[currentSize*HashSize:], leaf[:])

	// Write updated tile
	if err := tl.storage.Put(tilePath, newTile); err != nil {
		return fmt.Errorf("failed to put tile: %w", err)
	}

	return nil
}

// ReadHashes implements tlog.HashReader interface
// Reads hashes from tree at given indexes
// Note: tlog uses 0-indexed record IDs (0 through n-1 for n records)
// but C2SP tlog-tiles uses 1-indexed storage (records 1 through n)
func (tl *TileLog) ReadHashes(indexes []int64) ([]tlog.Hash, error) {
	hashes := make([]tlog.Hash, len(indexes))

	for i, index := range indexes {
		// Convert from tlog's 0-indexed to our 0-indexed storage
		// tlog requests record at index i, we store it at entry ID i
		leaf, err := tl.getLeafDirect(index)
		if err != nil {
			return nil, fmt.Errorf("failed to get leaf %d: %w", index, err)
		}

		// Convert to tlog.Hash
		var hash tlog.Hash
		copy(hash[:], leaf[:])
		hashes[i] = hash
	}

	return hashes, nil
}

// TreeHash computes the tree hash for a given tree size using tlog
func (tl *TileLog) TreeHash(n int64) (tlog.Hash, error) {
	if n == 0 {
		return tlog.Hash{}, fmt.Errorf("cannot compute hash of empty tree")
	}

	if n > tl.size {
		return tlog.Hash{}, fmt.Errorf("requested size %d exceeds tree size %d", n, tl.size)
	}

	// tlog.TreeHash expects records numbered 0..n-1
	// It will call ReadHashes with those indices
	// We provide those via our getLeafDirect which maps directly to entry IDs
	return tlog.TreeHash(n, tl)
}

// RecordHash computes the hash of a record (leaf) with RFC 6962 prefix
func RecordHash(data []byte) [HashSize]byte {
	h := sha256.New()
	h.Write([]byte{0x00}) // RFC 6962 leaf prefix
	h.Write(data)
	var result [HashSize]byte
	copy(result[:], h.Sum(nil))
	return result
}

// tileHashStorage implements tlog.TileReader for hash tiles
type tileHashStorage struct {
	storage storage.Storage
}

// Height returns the tile height (always 8 for standard tiles)
func (ths *tileHashStorage) Height() int {
	return 8 // 2^8 = 256 hashes per tile
}

// ReadTiles reads tiles from storage
func (ths *tileHashStorage) ReadTiles(tiles []tlog.Tile) ([][]byte, error) {
	data := make([][]byte, len(tiles))

	for i, tile := range tiles {
		// Determine if this is a partial tile
		var path string
		if tile.W != 1<<uint(ths.Height()) {
			// Partial tile
			width := int(tile.W)
			path = TileIndexToPath(tile.L, tile.N, &width)
		} else {
			// Full tile
			path = TileIndexToPath(tile.L, tile.N, nil)
		}

		tileData, err := ths.storage.Get(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get tile %s: %w", path, err)
		}

		data[i] = tileData
	}

	return data, nil
}

// SaveTiles saves tiles to storage
func (ths *tileHashStorage) SaveTiles(tiles []tlog.Tile, data [][]byte) {
	for i, tile := range tiles {
		// Determine if this is a partial tile
		var path string
		if tile.W != 1<<uint(ths.Height()) {
			// Partial tile
			width := int(tile.W)
			path = TileIndexToPath(tile.L, tile.N, &width)
		} else {
			// Full tile
			path = TileIndexToPath(tile.L, tile.N, nil)
		}

		// Ignore errors in SaveTiles (as per tlog.TileReader interface)
		_ = ths.storage.Put(path, data[i])
	}
}
