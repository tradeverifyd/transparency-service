package main

import (
	"crypto/sha256"
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

type DebugHashReader struct {
	leafHashes []tlog.Hash
	requests   []int64
}

func (r *DebugHashReader) ReadHashes(indexes []int64) ([]tlog.Hash, error) {
	fmt.Printf("ReadHashes called with indexes: %v\n", indexes)
	r.requests = append(r.requests, indexes...)
	return nil, fmt.Errorf("intentional error to see what's requested")
}

func main() {
	// Create 4 leaves
	leafHashes := make([]tlog.Hash, 4)
	for i := 0; i < 4; i++ {
		leaf := make([]byte, 32)
		for j := range leaf {
			leaf[j] = byte(i)
		}
		leafHashes[i] = leafHash(leaf)
	}

	reader := &DebugHashReader{leafHashes: leafHashes}

	fmt.Println("Attempting to generate proof for oldSize=3, newSize=4...")
	_, err := tlog.ProveTree(4, 3, reader)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Printf("\nAll requested indexes: %v\n", reader.requests)
}
