package merkle

import (
	"encoding/json"
	"fmt"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
	"github.com/transparency-dev/merkle/compact"
	"github.com/transparency-dev/merkle/rfc6962"
)

// TileLog represents a tile-based Merkle tree
// Uses Tessera's compact range for efficient RFC 6962 Merkle tree operations
type TileLog struct {
	storage storage.Storage
	size    int64
	rf      *compact.RangeFactory
	cr      *compact.Range
}

// TileLogState represents the persistent state of the tree
type TileLogState struct {
	Size   int64    `json:"size"`
	Root   []byte   `json:"root,omitempty"`
	Hashes [][]byte `json:"hashes,omitempty"` // Compact range hashes
}

// NewTileLog creates a new tile log with the given storage
func NewTileLog(storage storage.Storage) *TileLog {
	rf := &compact.RangeFactory{
		Hash: rfc6962.DefaultHasher.HashChildren,
	}
	return &TileLog{
		storage: storage,
		size:    0,
		rf:      rf,
		cr:      rf.NewEmptyRange(0),
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
		tl.cr = tl.rf.NewEmptyRange(0)
		return nil
	}

	var state TileLogState
	if err := json.Unmarshal(stateData, &state); err != nil {
		return fmt.Errorf("failed to unmarshal tree state: %w", err)
	}

	tl.size = state.Size

	// Restore compact range from saved hashes
	if state.Size > 0 && len(state.Hashes) > 0 {
		cr, err := tl.rf.NewRange(0, uint64(state.Size), state.Hashes)
		if err != nil {
			return fmt.Errorf("failed to restore compact range: %w", err)
		}
		tl.cr = cr
	} else {
		tl.cr = tl.rf.NewEmptyRange(0)
	}

	return nil
}

// saveState saves the tree state to storage
func (tl *TileLog) saveState() error {
	var root []byte
	var hashes [][]byte

	if tl.size > 0 {
		// Get root hash from compact range
		rootHash, err := tl.cr.GetRootHash(nil)
		if err != nil {
			return fmt.Errorf("failed to compute tree hash: %w", err)
		}
		root = rootHash

		// Save compact range hashes for restoration
		hashes = tl.cr.Hashes()
	}

	state := TileLogState{
		Size:   tl.size,
		Root:   root,
		Hashes: hashes,
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
// This function will apply the RFC 6962 leaf prefix (0x00) for tree computation
func (tl *TileLog) Append(leaf [HashSize]byte) (int64, error) {
	entryID := tl.size

	// Store the RAW leaf hash in entry tile (without RFC 6962 prefix)
	// This preserves the original hash for retrieval
	if err := tl.appendToEntryTile(entryID, leaf[:]); err != nil {
		return 0, fmt.Errorf("failed to append to entry tile: %w", err)
	}

	// Apply RFC 6962 leaf hash prefix (0x00) for tree computation only
	// The compact range uses this for computing Merkle tree roots
	leafHash := rfc6962.DefaultHasher.HashLeaf(leaf[:])

	// Append to compact range for efficient tree computation
	if err := tl.cr.Append(leafHash, nil); err != nil {
		return 0, fmt.Errorf("failed to append to compact range: %w", err)
	}

	// Increment size after successful append
	tl.size++

	// Persist state
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

	root, err := tl.cr.GetRootHash(nil)
	if err != nil {
		return [HashSize]byte{}, fmt.Errorf("failed to get root hash: %w", err)
	}

	var result [HashSize]byte
	copy(result[:], root)
	return result, nil
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
func (tl *TileLog) appendToEntryTile(entryID int64, leafHash []byte) error {
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
	copy(newTile[currentSize*HashSize:], leafHash)

	// Write updated tile
	if err := tl.storage.Put(tilePath, newTile); err != nil {
		return fmt.Errorf("failed to put tile: %w", err)
	}

	return nil
}

// RecordHash computes the hash of a record (leaf) with RFC 6962 prefix
// This is a convenience function that wraps Tessera's HashLeaf
func RecordHash(data []byte) [HashSize]byte {
	leafHash := rfc6962.DefaultHasher.HashLeaf(data)
	var result [HashSize]byte
	copy(result[:], leafHash)
	return result
}
