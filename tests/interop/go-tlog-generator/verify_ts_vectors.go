package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/sumdb/tlog"
)

func largestPowerOfTwoLessThan(n int64) int64 {
	k := int64(1)
	for k*2 < n {
		k *= 2
	}
	return k
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

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

func hexToBytes(hexStr string) ([]byte, error) {
	return hex.DecodeString(hexStr)
}

func hexToHash(hexStr string) (tlog.Hash, error) {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return tlog.Hash{}, err
	}
	var hash tlog.Hash
	copy(hash[:], bytes)
	return hash, nil
}

// verifyInclusionProof manually verifies an inclusion proof
func verifyInclusionProof(leaf []byte, leafIndex, treeSize int64, auditPath []tlog.Hash, root tlog.Hash) bool {
	if treeSize == 0 {
		return false
	}

	if treeSize == 1 {
		h := sha256.New()
		h.Write([]byte{0x00})
		h.Write(leaf)
		leafHash := h.Sum(nil)
		return string(leafHash) == string(root[:])
	}

	// Hash the leaf
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(leaf)
	currentHash := h.Sum(nil)

	// Build tree states
	type state struct {
		index int64
		size  int64
	}
	var treeStates []state

	idx := leafIndex
	sz := treeSize
	for sz > 1 {
		treeStates = append(treeStates, state{idx, sz})
		k := largestPowerOfTwoLessThan(sz)
		if idx < k {
			sz = k
		} else {
			idx = idx - k
			sz = sz - k
		}
	}

	// Verify by combining with siblings
	for i := len(treeStates) - 1; i >= 0; i-- {
		st := treeStates[i]
		sibling := auditPath[len(treeStates)-1-i]
		k := largestPowerOfTwoLessThan(st.size)

		h := sha256.New()
		h.Write([]byte{0x01})
		if st.index < k {
			// Left subtree
			h.Write(currentHash)
			h.Write(sibling[:])
		} else {
			// Right subtree
			h.Write(sibling[:])
			h.Write(currentHash)
		}
		currentHash = h.Sum(nil)
	}

	return string(currentHash) == string(root[:])
}

// verifyConsistencyProof manually verifies a consistency proof
func verifyConsistencyProof(proof []tlog.Hash, oldSize, newSize int64, oldRoot, newRoot tlog.Hash) bool {
	if newSize < 1 || oldSize < 1 || oldSize > newSize {
		return false
	}

	if oldSize == newSize {
		return oldRoot == newRoot && len(proof) == 0
	}

	computedOldRoot, computedNewRoot, err := runTreeProof(proof, 0, newSize, oldSize, oldRoot)
	if err != nil {
		return false
	}

	return computedOldRoot == oldRoot && computedNewRoot == newRoot
}

func runTreeProof(p []tlog.Hash, lo, hi, n int64, old tlog.Hash) (tlog.Hash, tlog.Hash, error) {
	if !(lo < n && n <= hi) {
		return tlog.Hash{}, tlog.Hash{}, fmt.Errorf("invalid range")
	}

	if n == hi {
		if lo == 0 {
			if len(p) != 0 {
				return tlog.Hash{}, tlog.Hash{}, fmt.Errorf("proof too long")
			}
			return old, old, nil
		}
		if len(p) != 1 {
			return tlog.Hash{}, tlog.Hash{}, fmt.Errorf("proof length mismatch")
		}
		return p[0], p[0], nil
	}

	if len(p) == 0 {
		return tlog.Hash{}, tlog.Hash{}, fmt.Errorf("proof too short")
	}

	k := largestPowerOfTwoLessThan(hi - lo)

	if n <= lo+k {
		oh, th, err := runTreeProof(p[:len(p)-1], lo, lo+k, n, old)
		if err != nil {
			return tlog.Hash{}, tlog.Hash{}, err
		}
		h := sha256.New()
		h.Write([]byte{0x01})
		h.Write(th[:])
		h.Write(p[len(p)-1][:])
		var newHash tlog.Hash
		copy(newHash[:], h.Sum(nil))
		return oh, newHash, nil
	} else {
		oh, th, err := runTreeProof(p[:len(p)-1], lo+k, hi, n, old)
		if err != nil {
			return tlog.Hash{}, tlog.Hash{}, err
		}

		h1 := sha256.New()
		h1.Write([]byte{0x01})
		h1.Write(p[len(p)-1][:])
		h1.Write(oh[:])
		var oldHash tlog.Hash
		copy(oldHash[:], h1.Sum(nil))

		h2 := sha256.New()
		h2.Write([]byte{0x01})
		h2.Write(p[len(p)-1][:])
		h2.Write(th[:])
		var newHash tlog.Hash
		copy(newHash[:], h2.Sum(nil))

		return oldHash, newHash, nil
	}
}

func main() {
	vectorsDir := "../ts-test-vectors"
	files, err := filepath.Glob(filepath.Join(vectorsDir, "tlog-size-*.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "No test vector files found in %s\n", vectorsDir)
		os.Exit(1)
	}

	totalInclusionTests := 0
	totalConsistencyTests := 0
	passedInclusionTests := 0
	passedConsistencyTests := 0

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", file, err)
			continue
		}

		var testCase TestCase
		if err := json.Unmarshal(data, &testCase); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing JSON from %s: %v\n", file, err)
			continue
		}

		fmt.Printf("\n📋 Testing vectors from %s (tree size %d)\n", filepath.Base(file), testCase.TreeSize)

		// Parse root
		root, err := hexToHash(testCase.Root)
		if err != nil {
			fmt.Printf("  ❌ Error parsing root: %v\n", err)
			continue
		}

		// Test inclusion proofs
		inclusionPassed := 0
		for _, proof := range testCase.InclusionProofs {
			totalInclusionTests++

			leaf, err := hexToBytes(testCase.Leaves[proof.LeafIndex])
			if err != nil {
				fmt.Printf("  ❌ Inclusion proof for leaf %d: error parsing leaf\n", proof.LeafIndex)
				continue
			}

			auditPath := make([]tlog.Hash, len(proof.AuditPath))
			for i, hashHex := range proof.AuditPath {
				h, err := hexToHash(hashHex)
				if err != nil {
					fmt.Printf("  ❌ Inclusion proof for leaf %d: error parsing audit path\n", proof.LeafIndex)
					continue
				}
				auditPath[i] = h
			}

			if verifyInclusionProof(leaf, proof.LeafIndex, testCase.TreeSize, auditPath, root) {
				inclusionPassed++
				passedInclusionTests++
			} else {
				fmt.Printf("  ❌ Inclusion proof FAILED for leaf %d\n", proof.LeafIndex)
			}
		}
		fmt.Printf("  ✅ Inclusion proofs: %d/%d passed\n", inclusionPassed, len(testCase.InclusionProofs))

		// Test consistency proofs
		consistencyPassed := 0
		for _, proof := range testCase.ConsistencyProofs {
			totalConsistencyTests++

			oldRoot, err := hexToHash(proof.OldRoot)
			if err != nil {
				fmt.Printf("  ❌ Consistency proof oldSize=%d: error parsing old root\n", proof.OldSize)
				continue
			}

			newRoot, err := hexToHash(proof.NewRoot)
			if err != nil {
				fmt.Printf("  ❌ Consistency proof oldSize=%d: error parsing new root\n", proof.OldSize)
				continue
			}

			proofHashes := make([]tlog.Hash, len(proof.Proof))
			for i, hashHex := range proof.Proof {
				h, err := hexToHash(hashHex)
				if err != nil {
					fmt.Printf("  ❌ Consistency proof oldSize=%d: error parsing proof\n", proof.OldSize)
					continue
				}
				proofHashes[i] = h
			}

			if verifyConsistencyProof(proofHashes, proof.OldSize, proof.NewSize, oldRoot, newRoot) {
				consistencyPassed++
				passedConsistencyTests++
			} else {
				fmt.Printf("  ❌ Consistency proof FAILED for oldSize=%d, newSize=%d\n", proof.OldSize, proof.NewSize)
			}
		}
		fmt.Printf("  ✅ Consistency proofs: %d/%d passed\n", consistencyPassed, len(testCase.ConsistencyProofs))
	}

	fmt.Printf("\n" + repeat("=", 60) + "\n")
	fmt.Printf("📊 FINAL RESULTS\n")
	fmt.Printf(repeat("=", 60) + "\n")
	fmt.Printf("✅ Inclusion proofs:   %d/%d passed (%.1f%%)\n",
		passedInclusionTests, totalInclusionTests,
		100.0*float64(passedInclusionTests)/float64(totalInclusionTests))
	fmt.Printf("✅ Consistency proofs: %d/%d passed (%.1f%%)\n",
		passedConsistencyTests, totalConsistencyTests,
		100.0*float64(passedConsistencyTests)/float64(totalConsistencyTests))
	fmt.Printf(repeat("=", 60) + "\n")

	if passedInclusionTests == totalInclusionTests && passedConsistencyTests == totalConsistencyTests {
		fmt.Println("\n🎉 SUCCESS: All TypeScript-generated proofs verified by Go!")
		os.Exit(0)
	} else {
		fmt.Println("\n❌ FAILURE: Some proofs failed verification")
		os.Exit(1)
	}
}
