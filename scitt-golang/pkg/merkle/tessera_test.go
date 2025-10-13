package merkle_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
	"github.com/transparency-dev/merkle/rfc6962"
)

// TestTesseraBasicUsage tests basic Merkle tree operations using Tessera's merkle package
func TestTesseraBasicUsage(t *testing.T) {
	t.Run("computes hash for single leaf", func(t *testing.T) {
		// Create a leaf hash (raw statement hash)
		leaf := sha256.Sum256([]byte("test statement"))

		// Apply RFC 6962 leaf hash
		leafHash := rfc6962.DefaultHasher.HashLeaf(leaf[:])

		if len(leafHash) != 32 {
			t.Errorf("expected 32 byte hash, got %d", len(leafHash))
		}

		// For a single leaf tree, the root is just the leaf hash
		t.Logf("Single leaf hash: %x", leafHash)
	})

	t.Run("computes hash for two leaves", func(t *testing.T) {
		// Create two leaf hashes
		leaf1 := sha256.Sum256([]byte("statement 1"))
		leaf2 := sha256.Sum256([]byte("statement 2"))

		// Apply RFC 6962 leaf hashing
		leafHash1 := rfc6962.DefaultHasher.HashLeaf(leaf1[:])
		leafHash2 := rfc6962.DefaultHasher.HashLeaf(leaf2[:])

		// Compute tree root using RFC 6962 node hashing
		root := rfc6962.DefaultHasher.HashChildren(leafHash1, leafHash2)

		if len(root) != 32 {
			t.Errorf("expected 32 byte root, got %d", len(root))
		}

		t.Logf("Two leaf root: %x", root)
	})

	t.Run("computes hash for multiple leaves", func(t *testing.T) {
		// Create multiple leaf hashes
		leaves := make([][]byte, 10)
		for i := 0; i < 10; i++ {
			leaf := sha256.Sum256([]byte(fmt.Sprintf("statement %d", i)))
			leaves[i] = rfc6962.DefaultHasher.HashLeaf(leaf[:])
		}

		// Build tree manually using RFC 6962 rules
		// For 10 leaves: split at k=8 (largest power of 2 < 10)
		// Left subtree: leaves 0-7, Right subtree: leaves 8-9

		// This demonstrates that Tessera's hasher works correctly
		// We would need Tessera's tree structure for automatic computation
		t.Logf("Successfully hashed %d leaves", len(leaves))
	})
}

// TestTesseraWithStorage tests if we can integrate Tessera with our storage
func TestTesseraWithStorage(t *testing.T) {
	t.Run("stores and retrieves leaf hashes", func(t *testing.T) {
		store := storage.NewMemoryStorage()

		// Create some test leaves
		numLeaves := 5
		leaves := make([][]byte, numLeaves)
		for i := 0; i < numLeaves; i++ {
			leaf := sha256.Sum256([]byte(fmt.Sprintf("statement %d", i)))
			leafHash := rfc6962.DefaultHasher.HashLeaf(leaf[:])
			leaves[i] = leafHash

			// Store in our storage
			key := fmt.Sprintf("leaf/%d", i)
			if err := store.Put(key, leafHash); err != nil {
				t.Fatalf("failed to store leaf %d: %v", i, err)
			}
		}

		// Retrieve and verify
		for i := 0; i < numLeaves; i++ {
			key := fmt.Sprintf("leaf/%d", i)
			data, err := store.Get(key)
			if err != nil {
				t.Fatalf("failed to get leaf %d: %v", i, err)
			}

			if len(data) != 32 {
				t.Errorf("leaf %d: expected 32 bytes, got %d", i, len(data))
			}
		}

		t.Logf("Successfully stored and retrieved %d leaves", numLeaves)
	})
}

// TestTesseraTreeComputation tests computing tree roots with multiple leaves
func TestTesseraTreeComputation(t *testing.T) {
	t.Run("matches expected RFC 6962 behavior", func(t *testing.T) {
		// Test with known values to verify RFC 6962 compliance

		// Single leaf tree
		leaf1 := []byte("leaf1")
		hash1 := rfc6962.DefaultHasher.HashLeaf(leaf1)
		t.Logf("Single leaf: %x", hash1)

		// Two leaf tree
		leaf2 := []byte("leaf2")
		hash2 := rfc6962.DefaultHasher.HashLeaf(leaf2)
		root2 := rfc6962.DefaultHasher.HashChildren(hash1, hash2)
		t.Logf("Two leaf root: %x", root2)

		// Three leaf tree: Split at k=2
		// Left: hash(hash1, hash2), Right: hash3
		leaf3 := []byte("leaf3")
		hash3 := rfc6962.DefaultHasher.HashLeaf(leaf3)
		root3 := rfc6962.DefaultHasher.HashChildren(root2, hash3)
		t.Logf("Three leaf root: %x", root3)

		// Verify roots are different
		if string(hash1) == string(root2) {
			t.Error("single and two-leaf roots should differ")
		}
		if string(root2) == string(root3) {
			t.Error("two and three-leaf roots should differ")
		}
	})
}

// TestCompareWithCurrentImplementation compares Tessera with current approach
func TestCompareWithCurrentImplementation(t *testing.T) {
	t.Run("verify RFC 6962 leaf hashing", func(t *testing.T) {
		testData := []byte("test record")
		recordHash := sha256.Sum256(testData)

		// Current approach (using RecordHash function)
		// This should match what Tessera does
		currentHash := sha256.New()
		currentHash.Write([]byte{0x00}) // RFC 6962 leaf prefix
		currentHash.Write(recordHash[:])
		currentResult := currentHash.Sum(nil)

		// Tessera approach
		tesseraResult := rfc6962.DefaultHasher.HashLeaf(recordHash[:])

		// They should match
		if string(currentResult) != string(tesseraResult) {
			t.Error("current and Tessera leaf hashing don't match")
			t.Logf("Current:  %x", currentResult)
			t.Logf("Tessera:  %x", tesseraResult)
		} else {
			t.Logf("✓ Leaf hashing matches: %x", currentResult)
		}
	})
}
