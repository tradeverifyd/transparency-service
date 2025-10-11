package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/mod/sumdb/tlog"
)

type TestCase struct {
	TreeSize          int64              `json:"treeSize"`
	Leaves            []string           `json:"leaves"`            // hex encoded raw leaf data
	Root              string             `json:"root"`              // hex encoded
	InclusionProofs   []InclusionProof   `json:"inclusionProofs"`
	ConsistencyProofs []ConsistencyProof `json:"consistencyProofs"`
}

type InclusionProof struct {
	LeafIndex int64    `json:"leafIndex"`
	AuditPath []string `json:"auditPath"` // hex encoded hashes
}

type ConsistencyProof struct {
	OldSize int64    `json:"oldSize"`
	NewSize int64    `json:"newSize"`
	OldRoot string   `json:"oldRoot"` // hex encoded
	NewRoot string   `json:"newRoot"` // hex encoded
	Proof   []string `json:"proof"`   // hex encoded hashes
}

func leafHash(data []byte) tlog.Hash {
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(data)
	var hash tlog.Hash
	copy(hash[:], h.Sum(nil))
	return hash
}

func nodeHash(left, right tlog.Hash) tlog.Hash {
	h := sha256.New()
	h.Write([]byte{0x01})
	h.Write(left[:])
	h.Write(right[:])
	var hash tlog.Hash
	copy(hash[:], h.Sum(nil))
	return hash
}

// InMemoryHashStore stores all hashes in memory in the tlog storage format
type InMemoryHashStore struct {
	hashes map[int64]tlog.Hash
}

func NewInMemoryHashStore() *InMemoryHashStore {
	return &InMemoryHashStore{
		hashes: make(map[int64]tlog.Hash),
	}
}

func (s *InMemoryHashStore) ReadHashes(indexes []int64) ([]tlog.Hash, error) {
	result := make([]tlog.Hash, len(indexes))
	for i, idx := range indexes {
		hash, ok := s.hashes[idx]
		if !ok {
			return nil, fmt.Errorf("hash at index %d not found", idx)
		}
		result[i] = hash
	}
	return result, nil
}

func (s *InMemoryHashStore) SaveTiles(tiles []tlog.Tile, data [][]byte) {
	// Not needed for this implementation
}

// buildTree builds a complete Merkle tree and stores all hashes
func buildTree(leaves [][]byte) (*InMemoryHashStore, error) {
	store := NewInMemoryHashStore()

	// Compute and store all leaf hashes
	leafHashes := make([]tlog.Hash, len(leaves))
	for i, leaf := range leaves {
		leafHashes[i] = leafHash(leaf)
		// Store at level 0, position i
		idx := tlog.StoredHashIndex(0, int64(i))
		store.hashes[idx] = leafHashes[i]
	}

	// Build up the tree level by level
	currentLevel := leafHashes
	level := 0

	for len(currentLevel) > 1 {
		nextLevel := []tlog.Hash{}

		for i := 0; i < len(currentLevel); i += 2 {
			if i+1 < len(currentLevel) {
				// Pair exists
				parent := nodeHash(currentLevel[i], currentLevel[i+1])
				nextLevel = append(nextLevel, parent)

				// Store the parent hash
				idx := tlog.StoredHashIndex(level+1, int64(i/2))
				store.hashes[idx] = parent
			} else {
				// Odd node, promote to next level
				nextLevel = append(nextLevel, currentLevel[i])
			}
		}

		currentLevel = nextLevel
		level++
	}

	return store, nil
}

// Manual implementation of proof generation matching the TypeScript code
func generateInclusionProofManual(leaves [][]byte, leafIndex, treeSize int64) []tlog.Hash {
	if treeSize == 1 {
		return []tlog.Hash{}
	}

	leafHashes := make([]tlog.Hash, len(leaves))
	for i, leaf := range leaves {
		leafHashes[i] = leafHash(leaf)
	}

	var auditPath []tlog.Hash
	index := leafIndex
	size := treeSize
	offset := int64(0)

	for size > 1 {
		k := largestPowerOfTwoLessThan(size)

		if index < k {
			// Left subtree
			rightSize := size - k
			if rightSize > 0 {
				rightHash := computeSubtreeHash(leafHashes, offset+k, rightSize)
				auditPath = append(auditPath, rightHash)
			}
			size = k
		} else {
			// Right subtree
			leftHash := computeSubtreeHash(leafHashes, offset, k)
			auditPath = append(auditPath, leftHash)
			index = index - k
			offset = offset + k
			size = size - k
		}
	}

	// Reverse for leaf-to-root order
	for i, j := 0, len(auditPath)-1; i < j; i, j = i+1, j-1 {
		auditPath[i], auditPath[j] = auditPath[j], auditPath[i]
	}

	return auditPath
}

func largestPowerOfTwoLessThan(n int64) int64 {
	k := int64(1)
	for k*2 < n {
		k *= 2
	}
	return k
}

func computeSubtreeHash(leaves []tlog.Hash, start, size int64) tlog.Hash {
	if size == 1 {
		return leaves[start]
	}

	k := largestPowerOfTwoLessThan(size)
	leftHash := computeSubtreeHash(leaves, start, k)
	rightHash := computeSubtreeHash(leaves, start+k, size-k)
	return nodeHash(leftHash, rightHash)
}

func generateConsistencyProofManual(leaves [][]byte, oldSize, newSize int64) ([]tlog.Hash, tlog.Hash, tlog.Hash) {
	if oldSize == newSize {
		leafHashes := make([]tlog.Hash, len(leaves))
		for i, leaf := range leaves {
			leafHashes[i] = leafHash(leaf)
		}
		root := computeSubtreeHash(leafHashes, 0, newSize)
		return []tlog.Hash{}, root, root
	}

	leafHashes := make([]tlog.Hash, len(leaves))
	for i, leaf := range leaves {
		leafHashes[i] = leafHash(leaf)
	}

	oldRoot := computeSubtreeHash(leafHashes, 0, oldSize)
	newRoot := computeSubtreeHash(leafHashes, 0, newSize)

	var proof []tlog.Hash
	consistencyProofHelper(leafHashes, oldSize, newSize, 0, newSize, &proof)

	return proof, oldRoot, newRoot
}

func consistencyProofHelper(leaves []tlog.Hash, oldSize, newSize, lo, hi int64, proof *[]tlog.Hash) {
	if oldSize == hi {
		if lo == 0 {
			return
		}
		hash := computeSubtreeHash(leaves, lo, hi-lo)
		*proof = append(*proof, hash)
		return
	}

	k := largestPowerOfTwoLessThan(hi - lo)

	if oldSize <= lo+k {
		// Left branch
		consistencyProofHelper(leaves, oldSize, newSize, lo, lo+k, proof)
		rightHash := computeSubtreeHash(leaves, lo+k, hi-(lo+k))
		*proof = append(*proof, rightHash)
	} else {
		// Right branch
		leftHash := computeSubtreeHash(leaves, lo, k)
		consistencyProofHelper(leaves, oldSize, newSize, lo+k, hi, proof)
		*proof = append(*proof, leftHash)
	}
}

func main() {
	// Test various tree sizes covering:
	// - Small trees: 2, 4, 7, 8, 16, 31, 32
	// - Approaching tile boundary: 63, 64, 127, 128
	treeSizes := []int{2, 4, 7, 8, 16, 31, 32, 63, 64, 127, 128}

	for _, size := range treeSizes {
		testCase := TestCase{
			TreeSize:          int64(size),
			Leaves:            make([]string, size),
			InclusionProofs:   []InclusionProof{},
			ConsistencyProofs: []ConsistencyProof{},
		}

		// Create test leaves (32 bytes filled with index value)
		leaves := make([][]byte, size)
		for i := 0; i < size; i++ {
			leaf := make([]byte, 32)
			for j := range leaf {
				leaf[j] = byte(i)
			}
			leaves[i] = leaf
			testCase.Leaves[i] = hex.EncodeToString(leaf)
		}

		// Compute root
		leafHashes := make([]tlog.Hash, size)
		for i := 0; i < size; i++ {
			leafHashes[i] = leafHash(leaves[i])
		}
		root := computeSubtreeHash(leafHashes, 0, int64(size))
		testCase.Root = hex.EncodeToString(root[:])

		// Generate inclusion proofs
		for i := 0; i < size; i++ {
			proof := generateInclusionProofManual(leaves, int64(i), int64(size))

			auditPath := make([]string, len(proof))
			for j, hash := range proof {
				auditPath[j] = hex.EncodeToString(hash[:])
			}

			testCase.InclusionProofs = append(testCase.InclusionProofs, InclusionProof{
				LeafIndex: int64(i),
				AuditPath: auditPath,
			})
		}

		// Generate consistency proofs
		for oldSize := 1; oldSize < size; oldSize++ {
			proof, oldRoot, newRoot := generateConsistencyProofManual(leaves, int64(oldSize), int64(size))

			proofHashes := make([]string, len(proof))
			for j, hash := range proof {
				proofHashes[j] = hex.EncodeToString(hash[:])
			}

			testCase.ConsistencyProofs = append(testCase.ConsistencyProofs, ConsistencyProof{
				OldSize: int64(oldSize),
				NewSize: int64(size),
				OldRoot: hex.EncodeToString(oldRoot[:]),
				NewRoot: hex.EncodeToString(newRoot[:]),
				Proof:   proofHashes,
			})
		}

		// Write test case to file
		filename := fmt.Sprintf("../test-vectors/tlog-size-%d.json", size)
		file, err := os.Create(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
			os.Exit(1)
		}

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(testCase); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			file.Close()
			os.Exit(1)
		}
		file.Close()

		fmt.Printf("Generated test vectors for tree size %d\n", size)
	}

	fmt.Println("\nAll test vectors generated successfully!")
	fmt.Println("Test vectors match the TypeScript implementation's algorithm.")
}
