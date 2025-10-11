package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/mod/sumdb/tlog"
)

func leafHash(data []byte) tlog.Hash {
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(data)
	var hash tlog.Hash
	copy(hash[:], h.Sum(nil))
	return hash
}

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
	// Create leaves [0, 1, 2, 3]
	leafHashes := make([]tlog.Hash, 4)
	for i := 0; i < 4; i++ {
		leaf := make([]byte, 32)
		for j := range leaf {
			leaf[j] = byte(i)
		}
		leafHashes[i] = leafHash(leaf)
	}

	reader := &StoredHashReader{leafHashes: leafHashes}

	// Get old root (size 3)
	oldRoot, err := tlog.TreeHash(3, reader)
	if err != nil {
		fmt.Printf("Error computing old root: %v\n", err)
		return
	}

	// Get new root (size 4)
	newRoot, err := tlog.TreeHash(4, reader)
	if err != nil {
		fmt.Printf("Error computing new root: %v\n", err)
		return
	}

	// Generate proof
	proof, err := tlog.ProveTree(4, 3, reader)
	if err != nil {
		fmt.Printf("Error generating proof: %v\n", err)
		return
	}

	fmt.Printf("oldSize=3, newSize=4\n")
	fmt.Printf("oldRoot: %s\n", hex.EncodeToString(oldRoot[:]))
	fmt.Printf("newRoot: %s\n", hex.EncodeToString(newRoot[:]))
	fmt.Printf("proof length: %d\n", len(proof))
	for i, h := range proof {
		fmt.Printf("proof[%d]: %s\n", i, hex.EncodeToString(h[:]))
	}

	// Now manually trace what CheckTree does
	fmt.Printf("\nNow checking the proof...\n")
	err = tlog.CheckTree(proof, 4, oldRoot, 3, newRoot)
	if err != nil {
		fmt.Printf("CheckTree failed: %v\n", err)
	} else {
		fmt.Printf("CheckTree succeeded!\n")
	}
}
