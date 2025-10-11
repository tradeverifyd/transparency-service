package main

import (
	"encoding/hex"
	"fmt"

	"golang.org/x/mod/sumdb/tlog"
)

func hexToHash(s string) tlog.Hash {
	b, _ := hex.DecodeString(s)
	var h tlog.Hash
	copy(h[:], b)
	return h
}

func main() {
	// Test oldSize=3, newSize=4
	// From our test vectors
	oldRoot := hexToHash("ba8d94b7fbcecae7b81c4c80574fe24734a6917bf9c1ecd66ff3e0c34ead4620")
	newRoot := hexToHash("fdea52008cdae79fa8bf806261959e23f5e11681646a2fa2bc9b5e56b32030a2")
	proof := tlog.TreeProof{
		hexToHash("acaa04663a8547a2f70c60cc18f9378796b13c4f9a08f70d6adae662365b30c6"),
	}

	fmt.Printf("Testing oldSize=3, newSize=4\n")
	fmt.Printf("proof length: %d\n", len(proof))

	err := tlog.CheckTree(proof, 4, newRoot, 3, oldRoot)
	if err != nil {
		fmt.Printf("CheckTree FAILED: %v\n", err)
	} else {
		fmt.Printf("CheckTree PASSED!\n")
	}
}
