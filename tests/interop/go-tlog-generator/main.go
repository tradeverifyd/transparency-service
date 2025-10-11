package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/mod/sumdb/tlog"
)

// TestCase represents a test case for interoperability
type TestCase struct {
	TreeSize          int64              `json:"treeSize"`
	Leaves            []string           `json:"leaves"`            // hex encoded
	Root              string             `json:"root"`              // hex encoded
	InclusionProofs   []InclusionProof   `json:"inclusionProofs"`
	ConsistencyProofs []ConsistencyProof `json:"consistencyProofs"`
}

// InclusionProof represents an inclusion proof test case
type InclusionProof struct {
	LeafIndex int64    `json:"leafIndex"`
	AuditPath []string `json:"auditPath"` // hex encoded hashes
}

// ConsistencyProof represents a consistency proof test case
type ConsistencyProof struct {
	OldSize int64    `json:"oldSize"`
	NewSize int64    `json:"newSize"`
	OldRoot string   `json:"oldRoot"` // hex encoded
	NewRoot string   `json:"newRoot"` // hex encoded
	Proof   []string `json:"proof"`   // hex encoded hashes
}

// leafHash computes RFC 6962 leaf hash (0x00 || data)
func leafHash(data []byte) tlog.Hash {
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(data)
	var hash tlog.Hash
	copy(hash[:], h.Sum(nil))
	return hash
}

// nodeHash computes RFC 6962 node hash (0x01 || left || right)
func nodeHash(left, right tlog.Hash) tlog.Hash {
	h := sha256.New()
	h.Write([]byte{0x01})
	h.Write(left[:])
	h.Write(right[:])
	var hash tlog.Hash
	copy(hash[:], h.Sum(nil))
	return hash
}

// largestPowerOfTwoLessThan finds the largest power of 2 strictly less than n
func largestPowerOfTwoLessThan(n int64) int64 {
	k := int64(1)
	for k*2 < n {
		k *= 2
	}
	return k
}

// computeSubtreeHash computes the MTH for a subtree [start, start+size)
func computeSubtreeHash(leaves []tlog.Hash, start, size int64) tlog.Hash {
	if size == 1 {
		return leaves[start]
	}

	k := largestPowerOfTwoLessThan(size)
	leftHash := computeSubtreeHash(leaves, start, k)
	rightHash := computeSubtreeHash(leaves, start+k, size-k)
	return nodeHash(leftHash, rightHash)
}

// generateInclusionProof generates an RFC 6962 inclusion proof
func generateInclusionProof(leaves []tlog.Hash, leafIndex, treeSize int64) []tlog.Hash {
	var auditPath []tlog.Hash

	index := leafIndex
	size := treeSize
	offset := int64(0)

	for size > 1 {
		k := largestPowerOfTwoLessThan(size)

		if index < k {
			// Leaf in left subtree, need right subtree hash
			rightSize := size - k
			if rightSize > 0 {
				rightHash := computeSubtreeHash(leaves, offset+k, rightSize)
				auditPath = append(auditPath, rightHash)
			}
			size = k
		} else {
			// Leaf in right subtree, need left subtree hash
			leftHash := computeSubtreeHash(leaves, offset, k)
			auditPath = append(auditPath, leftHash)
			index = index - k
			offset = offset + k
			size = size - k
		}
	}

	// Reverse to get leaf-to-root order
	for i, j := 0, len(auditPath)-1; i < j; i, j = i+1, j-1 {
		auditPath[i], auditPath[j] = auditPath[j], auditPath[i]
	}

	return auditPath
}

// StoredHashReader provides access to stored hashes for tlog
type StoredHashReader struct {
	leafHashes []tlog.Hash
}

func (r *StoredHashReader) ReadHashes(indexes []int64) ([]tlog.Hash, error) {
	result := make([]tlog.Hash, len(indexes))
	for i, idx := range indexes {
		if idx < 0 || idx >= int64(len(r.leafHashes)) {
			return nil, fmt.Errorf("index %d out of bounds", idx)
		}
		result[i] = r.leafHashes[idx]
	}
	return result, nil
}

func main() {
	// Test multiple tree sizes including edge cases
	treeSizes := []int{2, 4, 7, 8, 16}

	for _, size := range treeSizes {
		testCase := TestCase{
			TreeSize:        int64(size),
			Leaves:          make([]string, size),
			InclusionProofs: []InclusionProof{},
		}

		// Create test leaves (32 bytes filled with index value)
		leafHashes := make([]tlog.Hash, size)
		for i := 0; i < size; i++ {
			leaf := make([]byte, 32)
			for j := range leaf {
				leaf[j] = byte(i)
			}
			testCase.Leaves[i] = hex.EncodeToString(leaf)
			leafHashes[i] = leafHash(leaf)
		}

		// Compute tree root
		root := computeSubtreeHash(leafHashes, 0, int64(size))
		testCase.Root = hex.EncodeToString(root[:])

		// Generate inclusion proofs for all entries
		for i := 0; i < size; i++ {
			proof := generateInclusionProof(leafHashes, int64(i), int64(size))

			auditPath := make([]string, len(proof))
			for j, hash := range proof {
				auditPath[j] = hex.EncodeToString(hash[:])
			}

			testCase.InclusionProofs = append(testCase.InclusionProofs, InclusionProof{
				LeafIndex: int64(i),
				AuditPath: auditPath,
			})
		}

		// Generate consistency proofs for growing tree using actual tlog library
		// Test consistency from each smaller size to current size
		reader := &StoredHashReader{leafHashes: leafHashes}
		for oldSize := 1; oldSize < size; oldSize++ {
			oldRoot, err := tlog.TreeHash(int64(oldSize), reader)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error computing old tree hash: %v\n", err)
				os.Exit(1)
			}

			proof, err := tlog.ProveTree(int64(size), int64(oldSize), reader)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating consistency proof: %v\n", err)
				os.Exit(1)
			}

			proofHashes := make([]string, len(proof))
			for j, hash := range proof {
				proofHashes[j] = hex.EncodeToString(hash[:])
			}

			testCase.ConsistencyProofs = append(testCase.ConsistencyProofs, ConsistencyProof{
				OldSize: int64(oldSize),
				NewSize: int64(size),
				OldRoot: hex.EncodeToString(oldRoot[:]),
				NewRoot: hex.EncodeToString(root[:]),
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

	fmt.Println("All test vectors generated successfully")
}
