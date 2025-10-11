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
	ConsistencyProofs []ConsistencyProof `json:"consistencyProofs"`
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

func nodeHash(left, right tlog.Hash) tlog.Hash {
	h := sha256.New()
	h.Write([]byte{0x01})
	h.Write(left[:])
	h.Write(right[:])
	var hash tlog.Hash
	copy(hash[:], h.Sum(nil))
	return hash
}

// FullHashReader computes all hashes using the Crosby-Wallach storage scheme
type FullHashReader struct {
	leafHashes []tlog.Hash
	cache      map[int64]tlog.Hash
}

func NewFullHashReader(leafHashes []tlog.Hash) *FullHashReader {
	return &FullHashReader{
		leafHashes: leafHashes,
		cache:      make(map[int64]tlog.Hash),
	}
}

func (r *FullHashReader) ReadHashes(indexes []int64) ([]tlog.Hash, error) {
	result := make([]tlog.Hash, len(indexes))
	for i, idx := range indexes {
		hash, err := r.getHash(idx)
		if err != nil {
			return nil, err
		}
		result[i] = hash
	}
	return result, nil
}

func (r *FullHashReader) getHash(index int64) (tlog.Hash, error) {
	// Check cache first
	if hash, ok := r.cache[index]; ok {
		return hash, nil
	}

	// Compute the hash for this storage index
	// We need to figure out what tree position this index represents
	// For simplicity, we'll compute all possible hashes we might need
	
	// The tlog library requests specific storage indexes
	// We need to map these back to tree positions and compute the hashes
	
	// For a tree of size n, we need to be able to compute any subtree hash
	// Let's just compute hashes for all possible subtrees and store them
	
	// Actually, let's use a different approach: pre-compute all hashes we'll need
	return tlog.Hash{}, fmt.Errorf("hash index %d not computed", index)
}

// Simple approach: just compute the tree using tlog.TreeHash directly
// and let it build up the hashes it needs

func main() {
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
	
	// Use tlog.TileReader which stores hashes in tile format
	// Actually, let's just use a simpler approach: implement our own proof generation
	// that matches what Go tlog does, based on the RFC 6962 algorithm
	
	// Let's look at what the tlog library SHOULD generate for oldSize=3, newSize=4
	// by manually tracing through the algorithm
	
	fmt.Println("Based on RFC 6962 algorithm for oldSize=3, newSize=4:")
	fmt.Println("Tree 3: hash(hash(h0,h1), h2)")
	fmt.Println("Tree 4: hash(hash(h0,h1), hash(h2,h3))")
	fmt.Println("")
	fmt.Println("The proof should allow reconstructing both roots.")
	fmt.Println("Looking at the structure, we need:")
	fmt.Println("- hash(h0,h1) appears in both trees (common)")
	fmt.Println("- h2 in old tree, hash(h2,h3) in new tree")
	fmt.Println("")
	fmt.Println("According to RFC 6962 consistency proof algorithm:")
	fmt.Println("The proof should be [hash(h0,h1), h3]")
	fmt.Println("Or possibly [h2, h3] depending on the algorithm variant")
	
	// Let's compute these hashes and output them
	h0 := leafHashes[0]
	h1 := leafHashes[1]
	h2 := leafHashes[2]
	h3 := leafHashes[3]
	
	h01 := nodeHash(h0, h1)
	h23 := nodeHash(h2, h3)
	
	root3 := nodeHash(h01, h2)
	root4 := nodeHash(h01, h23)
	
	testCase := TestCase{
		TreeSize: int64(size),
		Leaves:   leaves,
		Root:     hex.EncodeToString(root4[:]),
	}
	
	// For oldSize=3, based on RFC 6962, the proof should be constructed
	// such that we can verify both trees
	// Let me check what makes sense: with proof=[h3], we have:
	// - We're given oldRoot = hash(h01, h2)
	// - We need to compute newRoot = hash(h01, h23)
	// - With just h3, we can't extract h2 or h01 from oldRoot
	// So proof=[h3] CANNOT work!
	
	// The proof must contain BOTH h2 and h3, OR both h01 and h3
	// Let's try proof = [h2, h3]:
	fmt.Println("\n=== Testing proof=[h2, h3] ===")
	proof1 := []tlog.Hash{h2, h3}
	fmt.Printf("proof[0] (h2): %s\n", hex.EncodeToString(h2[:]))
	fmt.Printf("proof[1] (h3): %s\n", hex.EncodeToString(h3[:]))
	
	// Actually, let me read the Go tlog ProveTree source more carefully
	// to understand what it SHOULD generate
}
