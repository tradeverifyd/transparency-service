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
	Leaves            []string           `json:"leaves"`
	Root              string             `json:"root"`
	InclusionProofs   []InclusionProof   `json:"inclusionProofs"`
	ConsistencyProofs []ConsistencyProof `json:"consistencyProofs"`
}

type InclusionProof struct {
	LeafIndex int64    `json:"leafIndex"`
	AuditPath []string `json:"auditPath"`
}

type ConsistencyProof struct {
	OldSize int64    `json:"oldSize"`
	NewSize int64    `json:"newSize"`
	OldRoot string   `json:"oldRoot"`
	NewRoot string   `json:"newRoot"`
	Proof   []string `json:"proof"`
}

func leafHash(data []byte) tlog.Hash {
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(data)
	var hash tlog.Hash
	copy(hash[:], h.Sum(nil))
	return hash
}

// ComputingHashReader computes hashes on-demand using the tlog.StoredHashIndex
type ComputingHashReader struct {
	leafHashes []tlog.Hash
	treeSize   int64
}

func (r *ComputingHashReader) ReadHashes(indexes []int64) ([]tlog.Hash, error) {
	result := make([]tlog.Hash, len(indexes))
	for i, idx := range indexes {
		hash, err := r.computeStoredHash(idx)
		if err != nil {
			return nil, err
		}
		result[i] = hash
	}
	return result, nil
}

func (r *ComputingHashReader) computeStoredHash(index int64) (tlog.Hash, error) {
	// Convert storage index to tree coordinates (level, offset)
	level, n := tlog.StoredHashIndex(index)
	
	// Compute the hash at this level and offset
	return r.computeHashAt(level, n)
}

func (r *ComputingHashReader) computeHashAt(level int32, n int64) (tlog.Hash, error) {
	if level == 0 {
		// Leaf level
		if n < 0 || n >= r.treeSize {
			return tlog.Hash{}, fmt.Errorf("leaf index %d out of range [0,%d)", n, r.treeSize)
		}
		return r.leafHashes[n], nil
	}
	
	// Internal node - compute from children
	leftChild, err := r.computeHashAt(level-1, 2*n)
	if err != nil {
		return tlog.Hash{}, err
	}
	
	rightChild, err := r.computeHashAt(level-1, 2*n+1)
	if err != nil {
		return tlog.Hash{}, err
	}
	
	// Hash the two children
	h := sha256.New()
	h.Write([]byte{0x01})
	h.Write(leftChild[:])
	h.Write(rightChild[:])
	var result tlog.Hash
	copy(result[:], h.Sum(nil))
	return result, nil
}

func main() {
	// Test with tree size 4
	size := 4
	
	// Create leaves
	leafHashes := make([]tlog.Hash, size)
	leaves := make([]string, size)
	for i := 0; i < size; i++ {
		leaf := make([]byte, 32)
		for j := range leaf {
			leaf[j] = byte(i)
		}
		leaves[i] = hex.EncodeToString(leaf)
		leafHashes[i] = leafHash(leaf)
	}
	
	reader := &ComputingHashReader{
		leafHashes: leafHashes,
		treeSize:   int64(size),
	}
	
	// Get root
	root, err := tlog.TreeHash(int64(size), reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error computing root: %v\n", err)
		os.Exit(1)
	}
	
	testCase := TestCase{
		TreeSize: int64(size),
		Leaves:   leaves,
		Root:     hex.EncodeToString(root[:]),
	}
	
	// Generate consistency proofs
	for oldSize := 1; oldSize < size; oldSize++ {
		oldRoot, err := tlog.TreeHash(int64(oldSize), reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error computing old root: %v\n", err)
			os.Exit(1)
		}
		
		proof, err := tlog.ProveTree(int64(size), int64(oldSize), reader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating proof: %v\n", err)
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
	
	// Output JSON
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(testCase); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}
