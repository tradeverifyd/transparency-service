package merkle_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/transparency-dev/merkle/compact"
	"github.com/transparency-dev/merkle/rfc6962"
)

// TestTesseraCompactRange tests using Tessera's compact range for efficient tree computation
func TestTesseraCompactRange(t *testing.T) {
	t.Run("builds tree with compact range", func(t *testing.T) {
		// Create a range factory with RFC 6962 hash function
		rf := &compact.RangeFactory{
			Hash: rfc6962.DefaultHasher.HashChildren,
		}
		// Create a compact range (efficient tree representation)
		cr := rf.NewEmptyRange(0)

		// Add leaves one by one
		leaves := [][]byte{}
		for i := 0; i < 5; i++ {
			// Create leaf hash
			data := []byte(fmt.Sprintf("statement %d", i))
			recordHash := sha256.Sum256(data)
			leafHash := rfc6962.DefaultHasher.HashLeaf(recordHash[:])
			leaves = append(leaves, leafHash)

			// Append to compact range
			if err := cr.Append(leafHash, nil); err != nil {
				t.Fatalf("failed to append leaf %d: %v", i, err)
			}
		}

		// Get the root hash
		root, err := cr.GetRootHash(nil)
		if err != nil {
			t.Fatalf("failed to get root hash: %v", err)
		}

		t.Logf("Tree with %d leaves, root: %x", len(leaves), root)

		// Verify we can get the root for different tree sizes
		for size := int64(1); size <= int64(len(leaves)); size++ {
			// Create a new range for this size
			crPartial := rf.NewEmptyRange(0)
			for i := int64(0); i < size; i++ {
				if err := crPartial.Append(leaves[i], nil); err != nil {
					t.Fatalf("failed to append leaf %d: %v", i, err)
				}
			}
			partialRoot, err := crPartial.GetRootHash(nil)
			if err != nil {
				t.Fatalf("failed to get root for size %d: %v", size, err)
			}
			t.Logf("  Size %d root: %x", size, partialRoot)
		}
	})

	t.Run("compact range handles single leaf", func(t *testing.T) {
		rf := &compact.RangeFactory{
			Hash: rfc6962.DefaultHasher.HashChildren,
		}
		cr := rf.NewEmptyRange(0)

		// Add single leaf
		data := []byte("single statement")
		recordHash := sha256.Sum256(data)
		leafHash := rfc6962.DefaultHasher.HashLeaf(recordHash[:])

		if err := cr.Append(leafHash, nil); err != nil {
			t.Fatalf("failed to append leaf: %v", err)
		}

		root, err := cr.GetRootHash(nil)
		if err != nil {
			t.Fatalf("failed to get root: %v", err)
		}

		// For a single leaf, root should equal the leaf hash
		if string(root) != string(leafHash) {
			t.Error("single leaf root should equal leaf hash")
			t.Logf("Leaf: %x", leafHash)
			t.Logf("Root: %x", root)
		} else {
			t.Logf("✓ Single leaf root matches: %x", root)
		}
	})

	t.Run("compact range handles power of two sizes", func(t *testing.T) {
		rf := &compact.RangeFactory{
			Hash: rfc6962.DefaultHasher.HashChildren,
		}
		// Test with 2, 4, 8, 16 leaves
		for _, size := range []int{2, 4, 8, 16} {
			cr := rf.NewEmptyRange(0)

			for i := 0; i < size; i++ {
				data := []byte(fmt.Sprintf("leaf %d", i))
				recordHash := sha256.Sum256(data)
				leafHash := rfc6962.DefaultHasher.HashLeaf(recordHash[:])

				if err := cr.Append(leafHash, nil); err != nil {
					t.Fatalf("size %d: failed to append leaf %d: %v", size, i, err)
				}
			}

			root, err := cr.GetRootHash(nil)
			if err != nil {
				t.Fatalf("size %d: failed to get root: %v", size, err)
			}

			t.Logf("Size %d (2^%d) root: %x", size, log2(size), root)
		}
	})

	t.Run("compact range handles non-power of two sizes", func(t *testing.T) {
		rf := &compact.RangeFactory{
			Hash: rfc6962.DefaultHasher.HashChildren,
		}
		// Test with 3, 5, 7, 9 leaves
		for _, size := range []int{3, 5, 7, 9} {
			cr := rf.NewEmptyRange(0)

			for i := 0; i < size; i++ {
				data := []byte(fmt.Sprintf("leaf %d", i))
				recordHash := sha256.Sum256(data)
				leafHash := rfc6962.DefaultHasher.HashLeaf(recordHash[:])

				if err := cr.Append(leafHash, nil); err != nil {
					t.Fatalf("size %d: failed to append leaf %d: %v", size, i, err)
				}
			}

			root, err := cr.GetRootHash(nil)
			if err != nil {
				t.Fatalf("size %d: failed to get root: %v", size, err)
			}

			t.Logf("Size %d root: %x", size, root)
		}
	})
}

// TestCompactRangeVsManualComputation compares compact range with manual tree computation
func TestCompactRangeVsManualComputation(t *testing.T) {
	t.Run("two leaves", func(t *testing.T) {
		rf := &compact.RangeFactory{
			Hash: rfc6962.DefaultHasher.HashChildren,
		}

		// Manual computation
		leaf1Data := []byte("leaf1")
		leaf2Data := []byte("leaf2")
		hash1 := sha256.Sum256(leaf1Data)
		hash2 := sha256.Sum256(leaf2Data)
		leafHash1 := rfc6962.DefaultHasher.HashLeaf(hash1[:])
		leafHash2 := rfc6962.DefaultHasher.HashLeaf(hash2[:])
		manualRoot := rfc6962.DefaultHasher.HashChildren(leafHash1, leafHash2)

		// Compact range computation
		cr := rf.NewEmptyRange(0)
		cr.Append(leafHash1, nil)
		cr.Append(leafHash2, nil)
		compactRoot, _ := cr.GetRootHash(nil)

		if string(manualRoot) != string(compactRoot) {
			t.Error("manual and compact roots differ for 2 leaves")
			t.Logf("Manual:  %x", manualRoot)
			t.Logf("Compact: %x", compactRoot)
		} else {
			t.Logf("✓ Roots match for 2 leaves: %x", compactRoot)
		}
	})

	t.Run("three leaves", func(t *testing.T) {
		rf := &compact.RangeFactory{
			Hash: rfc6962.DefaultHasher.HashChildren,
		}

		// Manual computation (split at k=2)
		leaves := [][]byte{}
		for i := 0; i < 3; i++ {
			data := []byte(fmt.Sprintf("leaf%d", i))
			hash := sha256.Sum256(data)
			leafHash := rfc6962.DefaultHasher.HashLeaf(hash[:])
			leaves = append(leaves, leafHash)
		}

		// Left subtree: leaves 0-1
		leftRoot := rfc6962.DefaultHasher.HashChildren(leaves[0], leaves[1])
		// Right subtree: leaf 2
		rightRoot := leaves[2]
		// Tree root
		manualRoot := rfc6962.DefaultHasher.HashChildren(leftRoot, rightRoot)

		// Compact range computation
		cr := rf.NewEmptyRange(0)
		for _, leaf := range leaves {
			cr.Append(leaf, nil)
		}
		compactRoot, _ := cr.GetRootHash(nil)

		if string(manualRoot) != string(compactRoot) {
			t.Error("manual and compact roots differ for 3 leaves")
			t.Logf("Manual:  %x", manualRoot)
			t.Logf("Compact: %x", compactRoot)
		} else {
			t.Logf("✓ Roots match for 3 leaves: %x", compactRoot)
		}
	})
}

func log2(n int) int {
	count := 0
	for n > 1 {
		n >>= 1
		count++
	}
	return count
}
