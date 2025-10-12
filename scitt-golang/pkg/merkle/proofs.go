package merkle

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
)

// InclusionProof proves that a leaf is included in a tree of a given size
type InclusionProof struct {
	LeafIndex int64
	TreeSize  int64
	AuditPath [][HashSize]byte // Hashes from leaf to root
}

// ConsistencyProof proves that an older tree is a prefix of a newer tree
type ConsistencyProof struct {
	OldSize int64
	NewSize int64
	Proof   [][HashSize]byte // Hashes proving consistency
}

// GenerateInclusionProof generates an RFC 6962 inclusion proof
// Proves that leaf at leafIndex is included in tree of treeSize
func GenerateInclusionProof(store storage.Storage, leafIndex, treeSize int64) (*InclusionProof, error) {
	if leafIndex >= treeSize {
		return nil, fmt.Errorf("leaf index %d out of bounds for tree size %d", leafIndex, treeSize)
	}

	if treeSize == 0 {
		return nil, fmt.Errorf("cannot generate proof for empty tree")
	}

	if treeSize == 1 {
		// Single entry tree has empty audit path
		return &InclusionProof{
			LeafIndex: leafIndex,
			TreeSize:  treeSize,
			AuditPath: [][HashSize]byte{},
		}, nil
	}

	var auditPath [][HashSize]byte

	// Build audit path by traversing from root to leaf
	index := leafIndex
	size := treeSize
	offset := int64(0)

	for size > 1 {
		k := largestPowerOfTwoLessThan(size)

		if index < k {
			// Leaf is in left subtree, need right subtree hash
			rightSize := size - k
			if rightSize > 0 {
				rightHash, err := computeSubtreeHash(store, offset+k, rightSize)
				if err != nil {
					return nil, fmt.Errorf("failed to compute right subtree hash: %w", err)
				}
				auditPath = append(auditPath, rightHash)
			}
			// Continue in left subtree
			size = k
		} else {
			// Leaf is in right subtree, need left subtree hash
			leftHash, err := computeSubtreeHash(store, offset, k)
			if err != nil {
				return nil, fmt.Errorf("failed to compute left subtree hash: %w", err)
			}
			auditPath = append(auditPath, leftHash)
			// Continue in right subtree
			index = index - k
			offset = offset + k
			size = size - k
		}
	}

	// Reverse to get leaf-to-root order
	for i, j := 0, len(auditPath)-1; i < j; i, j = i+1, j-1 {
		auditPath[i], auditPath[j] = auditPath[j], auditPath[i]
	}

	return &InclusionProof{
		LeafIndex: leafIndex,
		TreeSize:  treeSize,
		AuditPath: auditPath,
	}, nil
}

// VerifyInclusionProof verifies an RFC 6962 inclusion proof
// Note: leaf should be the raw leaf data (will be hashed with 0x00 prefix)
func VerifyInclusionProof(leaf [HashSize]byte, proof *InclusionProof, root [HashSize]byte) bool {
	if proof.TreeSize == 0 {
		return false
	}

	if proof.TreeSize == 1 {
		// Single entry - hash leaf and compare
		leafHash := hashLeaf(leaf)
		return bytes.Equal(leafHash[:], root[:])
	}

	// Start with leaf hash (RFC 6962: 0x00 || leaf)
	currentHash := hashLeaf(leaf)

	// Reconstruct tree traversal to know which subtree we're in at each level
	type treeState struct {
		index int64
		size  int64
	}
	var treeStates []treeState

	// First, recreate the traversal path from root to leaf
	idx := proof.LeafIndex
	sz := proof.TreeSize
	for sz > 1 {
		treeStates = append(treeStates, treeState{index: idx, size: sz})
		k := largestPowerOfTwoLessThan(sz)
		if idx < k {
			sz = k
		} else {
			idx = idx - k
			sz = sz - k
		}
	}

	// Now verify: at each level (bottom to top), combine with the sibling
	for i := len(treeStates) - 1; i >= 0; i-- {
		state := treeStates[i]
		sibling := proof.AuditPath[len(treeStates)-1-i]
		k := largestPowerOfTwoLessThan(state.size)

		if state.index < k {
			// In left subtree, sibling is on right
			currentHash = hashNode(currentHash, sibling)
		} else {
			// In right subtree, sibling is on left
			currentHash = hashNode(sibling, currentHash)
		}
	}

	return bytes.Equal(currentHash[:], root[:])
}

// GenerateConsistencyProof generates an RFC 6962 consistency proof
// Proves that tree of oldSize is a prefix of tree of newSize
func GenerateConsistencyProof(store storage.Storage, oldSize, newSize int64) (*ConsistencyProof, error) {
	if oldSize > newSize {
		return nil, fmt.Errorf("old size %d cannot be greater than new size %d", oldSize, newSize)
	}

	if newSize == 0 {
		return nil, fmt.Errorf("cannot generate proof for empty tree")
	}

	if oldSize == newSize {
		// Same tree - empty proof
		return &ConsistencyProof{
			OldSize: oldSize,
			NewSize: newSize,
			Proof:   [][HashSize]byte{},
		}, nil
	}

	if oldSize == 0 {
		// No old tree - return empty proof
		return &ConsistencyProof{
			OldSize: oldSize,
			NewSize: newSize,
			Proof:   [][HashSize]byte{},
		}, nil
	}

	var proof [][HashSize]byte

	// Generate proof nodes
	if err := consistencyProofHelper(store, oldSize, newSize, 0, newSize, &proof); err != nil {
		return nil, fmt.Errorf("failed to generate consistency proof: %w", err)
	}

	return &ConsistencyProof{
		OldSize: oldSize,
		NewSize: newSize,
		Proof:   proof,
	}, nil
}

// consistencyProofHelper is a recursive helper for consistency proof generation
func consistencyProofHelper(store storage.Storage, oldSize, newSize, lo, hi int64, proof *[][HashSize]byte) error {
	// Validate invariant
	if !(lo < oldSize && oldSize <= hi) {
		return fmt.Errorf("invalid range in consistencyProofHelper: lo=%d, n=%d, hi=%d", lo, oldSize, hi)
	}

	// Base case: reached the exact old tree boundary
	if oldSize == hi {
		if lo == 0 {
			// Root of old tree - nothing to add
			return nil
		}
		// Add the hash of this subtree
		hash, err := computeSubtreeHash(store, lo, hi-lo)
		if err != nil {
			return fmt.Errorf("failed to compute subtree hash: %w", err)
		}
		*proof = append(*proof, hash)
		return nil
	}

	// Find split point
	k := largestPowerOfTwoLessThan(hi - lo)

	if oldSize <= lo+k {
		// Old tree ends in left subtree
		// Recurse left, then add right subtree hash
		if err := consistencyProofHelper(store, oldSize, newSize, lo, lo+k, proof); err != nil {
			return err
		}
		rightHash, err := computeSubtreeHash(store, lo+k, hi-(lo+k))
		if err != nil {
			return fmt.Errorf("failed to compute right subtree hash: %w", err)
		}
		*proof = append(*proof, rightHash)
	} else {
		// Old tree extends into right subtree
		// Recurse right FIRST, then add left subtree hash AT THE END
		leftHash, err := computeSubtreeHash(store, lo, k)
		if err != nil {
			return fmt.Errorf("failed to compute left subtree hash: %w", err)
		}
		if err := consistencyProofHelper(store, oldSize, newSize, lo+k, hi, proof); err != nil {
			return err
		}
		*proof = append(*proof, leftHash)
	}

	return nil
}

// VerifyConsistencyProof verifies an RFC 6962 consistency proof
func VerifyConsistencyProof(proof *ConsistencyProof, oldRoot, newRoot [HashSize]byte) bool {
	if proof.NewSize < 1 || proof.OldSize < 1 || proof.OldSize > proof.NewSize {
		return false
	}

	if proof.OldSize == proof.NewSize {
		// Same tree - roots must match and proof must be empty
		return bytes.Equal(oldRoot[:], newRoot[:]) && len(proof.Proof) == 0
	}

	// Use runTreeProof to compute both old and new roots
	computedOldRoot, computedNewRoot, err := runTreeProof(proof.Proof, 0, proof.NewSize, proof.OldSize, oldRoot)
	if err != nil {
		return false
	}

	return bytes.Equal(computedOldRoot[:], oldRoot[:]) && bytes.Equal(computedNewRoot[:], newRoot[:])
}

// runTreeProof is a recursive tree proof verification
// Returns [oldHash, newHash] computed from the proof
func runTreeProof(p [][HashSize]byte, lo, hi, n int64, oldRoot [HashSize]byte) ([HashSize]byte, [HashSize]byte, error) {
	// Validate range
	if !(lo < n && n <= hi) {
		return [HashSize]byte{}, [HashSize]byte{}, fmt.Errorf("invalid range in runTreeProof: lo=%d, n=%d, hi=%d", lo, n, hi)
	}

	// Reached common ground - both trees are identical up to n
	if n == hi {
		if lo == 0 {
			// Root of old tree
			if len(p) != 0 {
				return [HashSize]byte{}, [HashSize]byte{}, fmt.Errorf("proof too long")
			}
			return oldRoot, oldRoot, nil
		}
		// Subtree root
		if len(p) != 1 {
			return [HashSize]byte{}, [HashSize]byte{}, fmt.Errorf("proof length mismatch: expected 1, got %d", len(p))
		}
		return p[0], p[0], nil
	}

	if len(p) == 0 {
		return [HashSize]byte{}, [HashSize]byte{}, fmt.Errorf("proof too short")
	}

	// Determine subtree size
	k := largestPowerOfTwoLessThan(hi - lo)

	if n <= lo+k {
		// Old tree ends in left subtree
		oh, th, err := runTreeProof(p[:len(p)-1], lo, lo+k, n, oldRoot)
		if err != nil {
			return [HashSize]byte{}, [HashSize]byte{}, err
		}
		// New tree includes right subtree
		newHash := hashNode(th, p[len(p)-1])
		return oh, newHash, nil
	}

	// Old tree spans into right subtree
	oh, th, err := runTreeProof(p[:len(p)-1], lo+k, hi, n, oldRoot)
	if err != nil {
		return [HashSize]byte{}, [HashSize]byte{}, err
	}
	// Both old and new trees include left subtree
	oldHash := hashNode(p[len(p)-1], oh)
	newHash := hashNode(p[len(p)-1], th)
	return oldHash, newHash, nil
}

// ComputeTreeRoot computes the RFC 6962 root hash for a tree of given size
// This is the public API for computing tree roots for proofs
func ComputeTreeRoot(store storage.Storage, treeSize int64) ([HashSize]byte, error) {
	if treeSize == 0 {
		return [HashSize]byte{}, fmt.Errorf("cannot compute root of empty tree")
	}
	return computeSubtreeHash(store, 0, treeSize)
}

// computeSubtreeHash computes hash of a subtree
// Range: [start, start+size)
func computeSubtreeHash(store storage.Storage, start, size int64) ([HashSize]byte, error) {
	if size == 0 {
		return [HashSize]byte{}, fmt.Errorf("cannot compute hash of empty subtree")
	}

	if size == 1 {
		// Single leaf - get from storage and hash with 0x00 prefix
		leaf, err := getLeafFromStorage(store, start)
		if err != nil {
			return [HashSize]byte{}, fmt.Errorf("failed to get leaf: %w", err)
		}
		return hashLeaf(leaf), nil
	}

	// Split subtree and recursively compute
	k := largestPowerOfTwoLessThan(size)

	leftHash, err := computeSubtreeHash(store, start, k)
	if err != nil {
		return [HashSize]byte{}, fmt.Errorf("failed to compute left subtree: %w", err)
	}

	rightSize := size - k
	if rightSize == 0 {
		return leftHash, nil
	}

	rightHash, err := computeSubtreeHash(store, start+k, rightSize)
	if err != nil {
		return [HashSize]byte{}, fmt.Errorf("failed to compute right subtree: %w", err)
	}

	// Combine left and right with 0x01 prefix
	return hashNode(leftHash, rightHash), nil
}

// getLeafFromStorage retrieves a leaf from storage by entry ID
func getLeafFromStorage(store storage.Storage, entryID int64) ([HashSize]byte, error) {
	tileIndex := EntryIDToTileIndex(entryID)
	tileOffset := EntryIDToTileOffset(entryID)

	tilePath := EntryTileIndexToPath(tileIndex, nil)
	tileData, err := store.Get(tilePath)
	if err != nil {
		return [HashSize]byte{}, fmt.Errorf("failed to get tile: %w", err)
	}

	if tileData == nil {
		return [HashSize]byte{}, fmt.Errorf("entry tile not found: %s", tilePath)
	}

	start := tileOffset * HashSize
	end := start + HashSize

	if end > len(tileData) {
		return [HashSize]byte{}, fmt.Errorf("tile data too short")
	}

	var leaf [HashSize]byte
	copy(leaf[:], tileData[start:end])
	return leaf, nil
}

// largestPowerOfTwoLessThan finds largest power of 2 strictly less than n
func largestPowerOfTwoLessThan(n int64) int64 {
	var k int64 = 1
	for k*2 < n {
		k *= 2
	}
	return k
}

// hashLeaf hashes a leaf with RFC 6962 prefix (0x00)
func hashLeaf(leaf [HashSize]byte) [HashSize]byte {
	h := sha256.New()
	h.Write([]byte{0x00}) // RFC 6962 leaf prefix
	h.Write(leaf[:])
	var result [HashSize]byte
	copy(result[:], h.Sum(nil))
	return result
}

// hashNode hashes an internal node with RFC 6962 prefix (0x01)
func hashNode(left, right [HashSize]byte) [HashSize]byte {
	h := sha256.New()
	h.Write([]byte{0x01}) // RFC 6962 node prefix
	h.Write(left[:])
	h.Write(right[:])
	var result [HashSize]byte
	copy(result[:], h.Sum(nil))
	return result
}
