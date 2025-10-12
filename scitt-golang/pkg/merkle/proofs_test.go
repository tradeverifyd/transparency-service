package merkle_test

import (
	"bytes"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/merkle"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
)

// TestGenerateInclusionProof tests inclusion proof generation
func TestGenerateInclusionProof(t *testing.T) {
	t.Run("rejects empty tree", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		_, err := merkle.GenerateInclusionProof(store, 0, 0)
		if err == nil {
			t.Error("expected error for empty tree")
		}
	})

	t.Run("rejects leaf index out of bounds", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf := hashData([]byte("test"))
		_, _ = tl.Append(leaf)

		_, err := merkle.GenerateInclusionProof(store, 5, 1)
		if err == nil {
			t.Error("expected error for out of bounds leaf index")
		}
	})

	t.Run("generates proof for single entry tree", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf := hashData([]byte("single leaf"))
		_, _ = tl.Append(leaf)

		proof, err := merkle.GenerateInclusionProof(store, 0, 1)
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}

		if proof.LeafIndex != 0 {
			t.Errorf("expected leaf index 0, got %d", proof.LeafIndex)
		}
		if proof.TreeSize != 1 {
			t.Errorf("expected tree size 1, got %d", proof.TreeSize)
		}
		if len(proof.AuditPath) != 0 {
			t.Errorf("expected empty audit path, got %d hashes", len(proof.AuditPath))
		}
	})

	t.Run("generates proof for first leaf in two-entry tree", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf1 := hashData([]byte("leaf1"))
		leaf2 := hashData([]byte("leaf2"))
		_, _ = tl.Append(leaf1)
		_, _ = tl.Append(leaf2)

		proof, err := merkle.GenerateInclusionProof(store, 0, 2)
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}

		if proof.LeafIndex != 0 {
			t.Errorf("expected leaf index 0, got %d", proof.LeafIndex)
		}
		if proof.TreeSize != 2 {
			t.Errorf("expected tree size 2, got %d", proof.TreeSize)
		}
		if len(proof.AuditPath) != 1 {
			t.Errorf("expected audit path of length 1, got %d", len(proof.AuditPath))
		}
	})

	t.Run("generates proof for second leaf in two-entry tree", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf1 := hashData([]byte("leaf1"))
		leaf2 := hashData([]byte("leaf2"))
		_, _ = tl.Append(leaf1)
		_, _ = tl.Append(leaf2)

		proof, err := merkle.GenerateInclusionProof(store, 1, 2)
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}

		if proof.LeafIndex != 1 {
			t.Errorf("expected leaf index 1, got %d", proof.LeafIndex)
		}
		if len(proof.AuditPath) != 1 {
			t.Errorf("expected audit path of length 1, got %d", len(proof.AuditPath))
		}
	})

	t.Run("generates proof for larger tree", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		// Build tree with 7 leaves
		for i := 0; i < 7; i++ {
			leaf := hashData([]byte{byte(i)})
			_, _ = tl.Append(leaf)
		}

		// Generate proof for leaf 3
		proof, err := merkle.GenerateInclusionProof(store, 3, 7)
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}

		if proof.LeafIndex != 3 {
			t.Errorf("expected leaf index 3, got %d", proof.LeafIndex)
		}
		if proof.TreeSize != 7 {
			t.Errorf("expected tree size 7, got %d", proof.TreeSize)
		}
		if len(proof.AuditPath) == 0 {
			t.Error("expected non-empty audit path")
		}
	})
}

// TestVerifyInclusionProof tests inclusion proof verification
func TestVerifyInclusionProof(t *testing.T) {
	t.Run("rejects proof for empty tree", func(t *testing.T) {
		leaf := hashData([]byte("test"))
		root := hashData([]byte("root"))

		proof := &merkle.InclusionProof{
			LeafIndex: 0,
			TreeSize:  0,
			AuditPath: [][32]byte{},
		}

		valid := merkle.VerifyInclusionProof(leaf, proof, root)
		if valid {
			t.Error("should reject proof for empty tree")
		}
	})

	t.Run("verifies proof for single entry tree", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf := hashData([]byte("single leaf"))
		_, _ = tl.Append(leaf)

		root, _ := merkle.ComputeTreeRoot(store, 1)

		proof := &merkle.InclusionProof{
			LeafIndex: 0,
			TreeSize:  1,
			AuditPath: [][32]byte{},
		}

		valid := merkle.VerifyInclusionProof(leaf, proof, root)
		if !valid {
			t.Error("should verify proof for single entry tree")
		}
	})

	t.Run("verifies proof for two-entry tree", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf1 := hashData([]byte("leaf1"))
		leaf2 := hashData([]byte("leaf2"))
		_, _ = tl.Append(leaf1)
		_, _ = tl.Append(leaf2)

		root, _ := merkle.ComputeTreeRoot(store, 2)

		// Generate and verify proof for leaf 0
		proof, _ := merkle.GenerateInclusionProof(store, 0, 2)
		valid := merkle.VerifyInclusionProof(leaf1, proof, root)
		if !valid {
			t.Error("should verify proof for leaf 0")
		}

		// Generate and verify proof for leaf 1
		proof2, _ := merkle.GenerateInclusionProof(store, 1, 2)
		valid2 := merkle.VerifyInclusionProof(leaf2, proof2, root)
		if !valid2 {
			t.Error("should verify proof for leaf 1")
		}
	})

	t.Run("verifies proof for larger tree", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		// Build tree with 10 leaves
		leaves := make([][32]byte, 10)
		for i := 0; i < 10; i++ {
			leaves[i] = hashData([]byte{byte(i)})
			_, _ = tl.Append(leaves[i])
		}

		root, _ := merkle.ComputeTreeRoot(store, 10)

		// Verify proofs for all leaves
		for i := 0; i < 10; i++ {
			proof, err := merkle.GenerateInclusionProof(store, int64(i), 10)
			if err != nil {
				t.Fatalf("failed to generate proof for leaf %d: %v", i, err)
			}

			valid := merkle.VerifyInclusionProof(leaves[i], proof, root)
			if !valid {
				t.Errorf("should verify proof for leaf %d", i)
			}
		}
	})

	t.Run("rejects proof with wrong root", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf := hashData([]byte("leaf"))
		_, _ = tl.Append(leaf)

		proof, _ := merkle.GenerateInclusionProof(store, 0, 1)

		wrongRoot := hashData([]byte("wrong root"))
		valid := merkle.VerifyInclusionProof(leaf, proof, wrongRoot)
		if valid {
			t.Error("should reject proof with wrong root")
		}
	})

	t.Run("rejects proof with tampered leaf", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf1 := hashData([]byte("leaf1"))
		leaf2 := hashData([]byte("leaf2"))
		_, _ = tl.Append(leaf1)
		_, _ = tl.Append(leaf2)

		root, _ := merkle.ComputeTreeRoot(store, tl.Size())
		proof, _ := merkle.GenerateInclusionProof(store, 0, 2)

		// Try to verify with different leaf
		tamperedLeaf := hashData([]byte("tampered"))
		valid := merkle.VerifyInclusionProof(tamperedLeaf, proof, root)
		if valid {
			t.Error("should reject proof with tampered leaf")
		}
	})
}

// TestGenerateConsistencyProof tests consistency proof generation
func TestGenerateConsistencyProof(t *testing.T) {
	t.Run("rejects new size zero", func(t *testing.T) {
		store := storage.NewMemoryStorage()

		_, err := merkle.GenerateConsistencyProof(store, 0, 0)
		if err == nil {
			t.Error("expected error for new size zero")
		}
	})

	t.Run("rejects old size greater than new size", func(t *testing.T) {
		store := storage.NewMemoryStorage()

		_, err := merkle.GenerateConsistencyProof(store, 5, 3)
		if err == nil {
			t.Error("expected error for old size > new size")
		}
	})

	t.Run("generates empty proof for equal sizes", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf := hashData([]byte("test"))
		_, _ = tl.Append(leaf)

		proof, err := merkle.GenerateConsistencyProof(store, 1, 1)
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}

		if proof.OldSize != 1 {
			t.Errorf("expected old size 1, got %d", proof.OldSize)
		}
		if proof.NewSize != 1 {
			t.Errorf("expected new size 1, got %d", proof.NewSize)
		}
		if len(proof.Proof) != 0 {
			t.Errorf("expected empty proof, got %d hashes", len(proof.Proof))
		}
	})

	t.Run("generates empty proof for old size zero", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf := hashData([]byte("test"))
		_, _ = tl.Append(leaf)

		proof, err := merkle.GenerateConsistencyProof(store, 0, 1)
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}

		if len(proof.Proof) != 0 {
			t.Errorf("expected empty proof for old size 0, got %d hashes", len(proof.Proof))
		}
	})

	t.Run("generates proof from size 1 to 2", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf1 := hashData([]byte("leaf1"))
		leaf2 := hashData([]byte("leaf2"))
		_, _ = tl.Append(leaf1)
		_, _ = tl.Append(leaf2)

		proof, err := merkle.GenerateConsistencyProof(store, 1, 2)
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}

		if proof.OldSize != 1 {
			t.Errorf("expected old size 1, got %d", proof.OldSize)
		}
		if proof.NewSize != 2 {
			t.Errorf("expected new size 2, got %d", proof.NewSize)
		}
		if len(proof.Proof) == 0 {
			t.Error("expected non-empty proof")
		}
	})

	t.Run("generates proof for larger tree growth", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		// Build tree with 10 leaves
		for i := 0; i < 10; i++ {
			leaf := hashData([]byte{byte(i)})
			_, _ = tl.Append(leaf)
		}

		proof, err := merkle.GenerateConsistencyProof(store, 5, 10)
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}

		if proof.OldSize != 5 {
			t.Errorf("expected old size 5, got %d", proof.OldSize)
		}
		if proof.NewSize != 10 {
			t.Errorf("expected new size 10, got %d", proof.NewSize)
		}
		if len(proof.Proof) == 0 {
			t.Error("expected non-empty proof")
		}
	})
}

// TestVerifyConsistencyProof tests consistency proof verification
func TestVerifyConsistencyProof(t *testing.T) {
	t.Run("rejects invalid sizes", func(t *testing.T) {
		root := hashData([]byte("root"))

		// Old size > new size
		proof := &merkle.ConsistencyProof{
			OldSize: 5,
			NewSize: 3,
			Proof:   [][32]byte{},
		}
		valid := merkle.VerifyConsistencyProof(proof, root, root)
		if valid {
			t.Error("should reject old size > new size")
		}

		// Zero size
		proof2 := &merkle.ConsistencyProof{
			OldSize: 0,
			NewSize: 0,
			Proof:   [][32]byte{},
		}
		valid2 := merkle.VerifyConsistencyProof(proof2, root, root)
		if valid2 {
			t.Error("should reject zero sizes")
		}
	})

	t.Run("verifies proof for equal sizes", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf := hashData([]byte("test"))
		_, _ = tl.Append(leaf)

		root, _ := merkle.ComputeTreeRoot(store, tl.Size())

		proof := &merkle.ConsistencyProof{
			OldSize: 1,
			NewSize: 1,
			Proof:   [][32]byte{},
		}

		valid := merkle.VerifyConsistencyProof(proof, root, root)
		if !valid {
			t.Error("should verify proof for equal sizes")
		}
	})

	t.Run("verifies proof from size 1 to 2", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf1 := hashData([]byte("leaf1"))
		_, _ = tl.Append(leaf1)
		root1, _ := merkle.ComputeTreeRoot(store, 1)

		leaf2 := hashData([]byte("leaf2"))
		_, _ = tl.Append(leaf2)
		root2, _ := merkle.ComputeTreeRoot(store, 2)

		proof, _ := merkle.GenerateConsistencyProof(store, 1, 2)

		valid := merkle.VerifyConsistencyProof(proof, root1, root2)
		if !valid {
			t.Error("should verify proof from size 1 to 2")
		}
	})

	t.Run("verifies proof for larger tree growth", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		// Build tree with 5 leaves
		for i := 0; i < 5; i++ {
			leaf := hashData([]byte{byte(i)})
			_, _ = tl.Append(leaf)
		}
		root5, _ := merkle.ComputeTreeRoot(store, 5)

		// Add 5 more leaves
		for i := 5; i < 10; i++ {
			leaf := hashData([]byte{byte(i)})
			_, _ = tl.Append(leaf)
		}
		root10, _ := merkle.ComputeTreeRoot(store, 10)

		proof, _ := merkle.GenerateConsistencyProof(store, 5, 10)

		valid := merkle.VerifyConsistencyProof(proof, root5, root10)
		if !valid {
			t.Error("should verify proof from size 5 to 10")
		}
	})

	t.Run("rejects proof with wrong old root", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf1 := hashData([]byte("leaf1"))
		_, _ = tl.Append(leaf1)

		leaf2 := hashData([]byte("leaf2"))
		_, _ = tl.Append(leaf2)
		root2, _ := merkle.ComputeTreeRoot(store, 2)

		proof, _ := merkle.GenerateConsistencyProof(store, 1, 2)

		wrongRoot := hashData([]byte("wrong"))
		valid := merkle.VerifyConsistencyProof(proof, wrongRoot, root2)
		if valid {
			t.Error("should reject proof with wrong old root")
		}
	})

	t.Run("rejects proof with wrong new root", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		leaf1 := hashData([]byte("leaf1"))
		_, _ = tl.Append(leaf1)
		root1, _ := merkle.ComputeTreeRoot(store, 1)

		leaf2 := hashData([]byte("leaf2"))
		_, _ = tl.Append(leaf2)

		proof, _ := merkle.GenerateConsistencyProof(store, 1, 2)

		wrongRoot := hashData([]byte("wrong"))
		valid := merkle.VerifyConsistencyProof(proof, root1, wrongRoot)
		if valid {
			t.Error("should reject proof with wrong new root")
		}
	})

	t.Run("rejects equal sizes with different roots", func(t *testing.T) {
		root1 := hashData([]byte("root1"))
		root2 := hashData([]byte("root2"))

		proof := &merkle.ConsistencyProof{
			OldSize: 1,
			NewSize: 1,
			Proof:   [][32]byte{},
		}

		valid := merkle.VerifyConsistencyProof(proof, root1, root2)
		if valid {
			t.Error("should reject equal sizes with different roots")
		}
	})
}

// TestConsistencyProofRoundTrip tests round-trip consistency proofs
func TestConsistencyProofRoundTrip(t *testing.T) {
	t.Run("verifies all intermediate tree states", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		// Track roots at each size
		roots := make([][32]byte, 16)

		// Build tree with 15 leaves
		for i := 0; i < 15; i++ {
			leaf := hashData([]byte{byte(i)})
			_, _ = tl.Append(leaf)
			roots[i+1], _ = merkle.ComputeTreeRoot(store, int64(i+1))
		}

		// Test consistency between all pairs of sizes
		for oldSize := 1; oldSize <= 15; oldSize++ {
			for newSize := oldSize; newSize <= 15; newSize++ {
				proof, err := merkle.GenerateConsistencyProof(store, int64(oldSize), int64(newSize))
				if err != nil {
					t.Fatalf("failed to generate proof (%d -> %d): %v", oldSize, newSize, err)
				}

				valid := merkle.VerifyConsistencyProof(proof, roots[oldSize], roots[newSize])
				if !valid {
					t.Errorf("failed to verify proof (%d -> %d)", oldSize, newSize)
				}
			}
		}
	})
}

// TestInclusionProofEdgeCases tests edge cases for inclusion proofs
func TestInclusionProofEdgeCases(t *testing.T) {
	t.Run("verifies proofs at power-of-two boundaries", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		// Build tree with 16 leaves (power of 2)
		leaves := make([][32]byte, 16)
		for i := 0; i < 16; i++ {
			leaves[i] = hashData([]byte{byte(i)})
			_, _ = tl.Append(leaves[i])
		}

		root, _ := merkle.ComputeTreeRoot(store, tl.Size())

		// Verify proofs for all leaves
		for i := 0; i < 16; i++ {
			proof, err := merkle.GenerateInclusionProof(store, int64(i), 16)
			if err != nil {
				t.Fatalf("failed to generate proof for leaf %d: %v", i, err)
			}

			valid := merkle.VerifyInclusionProof(leaves[i], proof, root)
			if !valid {
				t.Errorf("failed to verify proof for leaf %d", i)
			}
		}
	})

	t.Run("verifies proofs just after power-of-two boundary", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		// Build tree with 17 leaves (just after power of 2)
		leaves := make([][32]byte, 17)
		for i := 0; i < 17; i++ {
			leaves[i] = hashData([]byte{byte(i)})
			_, _ = tl.Append(leaves[i])
		}

		root, _ := merkle.ComputeTreeRoot(store, tl.Size())

		// Verify proofs for all leaves
		for i := 0; i < 17; i++ {
			proof, err := merkle.GenerateInclusionProof(store, int64(i), 17)
			if err != nil {
				t.Fatalf("failed to generate proof for leaf %d: %v", i, err)
			}

			valid := merkle.VerifyInclusionProof(leaves[i], proof, root)
			if !valid {
				t.Errorf("failed to verify proof for leaf %d", i)
			}
		}
	})
}

// TestHashingFunctions tests the RFC 6962 hashing functions
func TestHashingFunctions(t *testing.T) {
	t.Run("hashLeaf uses 0x00 prefix", func(t *testing.T) {
		leaf := hashData([]byte("test"))

		// Use private function through proof verification
		// This is an indirect test since hashLeaf is not exported
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		_, _ = tl.Append(leaf)
		root, _ := merkle.ComputeTreeRoot(store, tl.Size())

		// Single leaf root should be hashLeaf(leaf)
		// Verify by checking proof verification
		proof := &merkle.InclusionProof{
			LeafIndex: 0,
			TreeSize:  1,
			AuditPath: [][32]byte{},
		}

		valid := merkle.VerifyInclusionProof(leaf, proof, root)
		if !valid {
			t.Error("leaf hashing does not match expected RFC 6962 behavior")
		}
	})

	t.Run("consistent hashing across operations", func(t *testing.T) {
		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		// Add same leaves twice in different sessions
		leaves := make([][32]byte, 5)
		for i := 0; i < 5; i++ {
			leaves[i] = hashData([]byte{byte(i)})
			_, _ = tl.Append(leaves[i])
		}
		root1, _ := merkle.ComputeTreeRoot(store, 5)

		// Create new tile log with same storage
		tl2 := merkle.NewTileLog(store)
		_ = tl2.Load()
		root2, _ := merkle.ComputeTreeRoot(store, 5)

		if !bytes.Equal(root1[:], root2[:]) {
			t.Error("roots should be identical for same leaves")
		}
	})
}
