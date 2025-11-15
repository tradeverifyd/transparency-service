package cose_test

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/merkle"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
)

const (
	demoReceiptDir = "../../demo/receipt-samples"
)

// TestReceiptOfInclusion tests generation and validation of an inclusion receipt
func TestReceiptOfInclusion(t *testing.T) {
	// Setup: Create a merkle tree with several entries
	store := storage.NewMemoryStorage()
	tl := merkle.NewTileLog(store)
	_ = tl.Load()

	// Add 5 statements to the tree
	statements := make([][]byte, 5)
	statementHashes := make([][32]byte, 5)
	for i := 0; i < 5; i++ {
		// Create a simple statement (in practice, this would be a COSE Sign1)
		statements[i] = []byte("statement-" + string(rune('0'+i)))
		statementHashes[i] = sha256.Sum256(statements[i])
		_, err := tl.Append(statementHashes[i])
		if err != nil {
			t.Fatalf("failed to append statement %d: %v", i, err)
		}
	}

	// Generate inclusion proof for statement at index 3
	leafIndex := int64(3)
	treeSize := int64(5)
	inclusionProof, err := merkle.GenerateInclusionProof(store, leafIndex, treeSize)
	if err != nil {
		t.Fatalf("failed to generate inclusion proof: %v", err)
	}

	// Compute the tree root
	root, err := merkle.ComputeTreeRoot(store, treeSize)
	if err != nil {
		t.Fatalf("failed to compute tree root: %v", err)
	}

	// Create a key pair for signing the receipt
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	signer, err := cose.NewES256Signer(keyPair.Private)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	verifier, err := cose.NewES256Verifier(keyPair.Public)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	// Create CWT claims for the receipt
	cwtClaims := cose.CreateCWTClaims(cose.CWTClaimsOptions{
		Iss: "https://transparency-service.example.com",
	})

	// VDS value: 1 = RFC 9162 SHA-256 merkle tree
	vdsValue := 1

	// Create protected headers for the receipt
	protectedHeaders := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
		Alg:       cose.AlgorithmES256,
		Kid:       []byte("test-key-1"),
		VDS:       &vdsValue,
		CWTClaims: cwtClaims,
	})

	// Create the receipt (COSE Sign1 with detached payload = merkle root)
	receipt, err := cose.CreateCoseSign1(
		protectedHeaders,
		root[:],
		signer,
		cose.CoseSign1Options{Detached: true},
	)
	if err != nil {
		t.Fatalf("failed to create receipt: %v", err)
	}

	// Add inclusion proof to unprotected headers
	// Convert AuditPath to []interface{} for CBOR encoding
	var auditPathInterface []interface{}
	for _, hash := range inclusionProof.AuditPath {
		auditPathInterface = append(auditPathInterface, hash[:])
	}

	// Create inclusion proof array: [tree-size, leaf-index, audit-path]
	inclusionProofArray := []interface{}{
		inclusionProof.TreeSize,
		inclusionProof.LeafIndex,
		auditPathInterface,
	}

	// Encode the entire proof array as a single CBOR bytestring
	inclusionProofCBOR, _ := cbor.Marshal(inclusionProofArray)

	// Add to unprotected headers using VDP (Verifiable Data Proof) label
	// Store as array containing the CBOR-encoded proof
	receipt.Unprotected[cose.HeaderLabelVerifiableDataProof] = map[interface{}]interface{}{
		int64(-1): []interface{}{inclusionProofCBOR}, // -1 is the label for inclusion proof
	}

	// Encode the receipt to CBOR
	receiptCBOR, err := cose.EncodeCoseSign1(receipt)
	if err != nil {
		t.Fatalf("failed to encode receipt: %v", err)
	}

	// Save to demo directory
	if err := os.MkdirAll(demoReceiptDir, 0755); err != nil {
		t.Logf("warning: failed to create demo directory: %v", err)
	} else {
		receiptPath := filepath.Join(demoReceiptDir, "inclusion-receipt.cbor")
		if err := os.WriteFile(receiptPath, receiptCBOR, 0644); err != nil {
			t.Logf("warning: failed to save receipt: %v", err)
		} else {
			t.Logf("✓ Saved inclusion receipt to: %s", receiptPath)
		}

		// Also save the statement that this receipt is for
		statementPath := filepath.Join(demoReceiptDir, "inclusion-statement.cbor")
		if err := os.WriteFile(statementPath, statements[3], 0644); err != nil {
			t.Logf("warning: failed to save statement: %v", err)
		} else {
			t.Logf("✓ Saved statement to: %s", statementPath)
		}
	}

	// Validation: Verify the receipt
	t.Run("validate inclusion receipt", func(t *testing.T) {
		// 1. Decode the receipt
		decodedReceipt, err := cose.DecodeCoseSign1(receiptCBOR)
		if err != nil {
			t.Fatalf("failed to decode receipt: %v", err)
		}

		// 2. Extract inclusion proof from unprotected headers
		// Try both int64 and uint64 keys for VDP header (396)
		var vdpHeader interface{}
		var ok bool
		vdpHeader, ok = decodedReceipt.Unprotected[int64(cose.HeaderLabelVerifiableDataProof)]
		if !ok {
			vdpHeader, ok = decodedReceipt.Unprotected[uint64(cose.HeaderLabelVerifiableDataProof)]
			if !ok {
				t.Fatalf("VDP header not found in receipt. Available keys: %v", decodedReceipt.Unprotected)
			}
		}

		vdpMap, ok := vdpHeader.(map[interface{}]interface{})
		if !ok {
			t.Fatal("VDP header is not a map")
		}

		// 3. Extract inclusion proof (stored as array of CBOR bytestrings)
		inclusionProofs, ok := vdpMap[int64(-1)].([]interface{})
		if !ok || len(inclusionProofs) == 0 {
			t.Fatal("inclusion proof not found in VDP or not an array")
		}

		// Extract first proof from the array (it's a CBOR-encoded bytestring)
		inclusionProofBytes, ok := inclusionProofs[0].([]byte)
		if !ok {
			t.Fatal("inclusion proof is not a bytestring")
		}

		// Decode the entire proof array from CBOR
		var inclusionProofArray []interface{}
		if err := cbor.Unmarshal(inclusionProofBytes, &inclusionProofArray); err != nil {
			t.Fatalf("failed to decode inclusion proof: %v", err)
		}

		if len(inclusionProofArray) != 3 {
			t.Fatalf("invalid inclusion proof structure: expected 3 elements, got %d", len(inclusionProofArray))
		}

		// Extract tree size (handle both int64 and uint64)
		var extractedTreeSize int64
		switch v := inclusionProofArray[0].(type) {
		case int64:
			extractedTreeSize = v
		case uint64:
			extractedTreeSize = int64(v)
		default:
			t.Fatalf("unexpected type for tree size: %T", inclusionProofArray[0])
		}

		// Extract leaf index (handle both int64 and uint64)
		var extractedLeafIndex int64
		switch v := inclusionProofArray[1].(type) {
		case int64:
			extractedLeafIndex = v
		case uint64:
			extractedLeafIndex = int64(v)
		default:
			t.Fatalf("unexpected type for leaf index: %T", inclusionProofArray[1])
		}

		extractedAuditPath := inclusionProofArray[2].([]interface{})

		// Convert audit path back to [][32]byte
		var auditPath [][32]byte
		for _, hashInterface := range extractedAuditPath {
			hashBytes := hashInterface.([]byte)
			var hash [32]byte
			copy(hash[:], hashBytes)
			auditPath = append(auditPath, hash)
		}

		extractedProof := &merkle.InclusionProof{
			LeafIndex: extractedLeafIndex,
			TreeSize:  extractedTreeSize,
			AuditPath: auditPath,
		}

		// 4. Reconstruct root from statement hash and inclusion proof
		reconstructedRoot := merkle.ReconstructRootFromInclusionProof(statementHashes[3], extractedProof)

		// 5. Verify COSE signature on receipt using reconstructed root as external payload
		valid, err := cose.VerifyCoseSign1(decodedReceipt, verifier, reconstructedRoot[:])
		if err != nil {
			t.Fatalf("failed to verify receipt: %v", err)
		}

		if !valid {
			t.Error("receipt signature is invalid")
		}

		// 6. Verify that reconstructed root matches expected root
		if reconstructedRoot != root {
			t.Error("reconstructed root does not match expected root")
		}

		t.Log("✓ Inclusion receipt validated successfully")
		t.Logf("  Tree size: %d", extractedTreeSize)
		t.Logf("  Leaf index: %d", extractedLeafIndex)
		t.Logf("  Audit path length: %d", len(auditPath))
	})
}

// TestReceiptOfConsistency tests generation and validation of a consistency receipt
func TestReceiptOfConsistency(t *testing.T) {
	// Setup: Create a merkle tree and grow it
	store := storage.NewMemoryStorage()
	tl := merkle.NewTileLog(store)
	_ = tl.Load()

	// Add 3 statements initially (old tree)
	for i := 0; i < 3; i++ {
		statement := []byte("statement-" + string(rune('0'+i)))
		statementHash := sha256.Sum256(statement)
		_, err := tl.Append(statementHash)
		if err != nil {
			t.Fatalf("failed to append statement %d: %v", i, err)
		}
	}

	// Compute old tree root
	oldSize := int64(3)
	oldRoot, err := merkle.ComputeTreeRoot(store, oldSize)
	if err != nil {
		t.Fatalf("failed to compute old tree root: %v", err)
	}

	// Add 2 more statements (new tree)
	for i := 3; i < 5; i++ {
		statement := []byte("statement-" + string(rune('0'+i)))
		statementHash := sha256.Sum256(statement)
		_, err := tl.Append(statementHash)
		if err != nil {
			t.Fatalf("failed to append statement %d: %v", i, err)
		}
	}

	// Compute new tree root
	newSize := int64(5)
	newRoot, err := merkle.ComputeTreeRoot(store, newSize)
	if err != nil {
		t.Fatalf("failed to compute new tree root: %v", err)
	}

	// Generate consistency proof
	consistencyProof, err := merkle.GenerateConsistencyProof(store, oldSize, newSize)
	if err != nil {
		t.Fatalf("failed to generate consistency proof: %v", err)
	}

	// Create a key pair for signing the receipt
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	signer, err := cose.NewES256Signer(keyPair.Private)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	verifier, err := cose.NewES256Verifier(keyPair.Public)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	// Create CWT claims for the receipt
	cwtClaims := cose.CreateCWTClaims(cose.CWTClaimsOptions{
		Iss: "https://transparency-service.example.com",
	})

	// VDS value: 1 = RFC 9162 SHA-256 merkle tree
	vdsValue := 1

	// Create protected headers for the receipt
	protectedHeaders := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
		Alg:       cose.AlgorithmES256,
		Kid:       []byte("test-key-1"),
		VDS:       &vdsValue,
		CWTClaims: cwtClaims,
	})

	// Create the receipt (COSE Sign1 with detached payload = new merkle root)
	receipt, err := cose.CreateCoseSign1(
		protectedHeaders,
		newRoot[:],
		signer,
		cose.CoseSign1Options{Detached: true},
	)
	if err != nil {
		t.Fatalf("failed to create receipt: %v", err)
	}

	// Add consistency proof to unprotected headers
	// Convert Proof to []interface{} for CBOR encoding
	var proofInterface []interface{}
	for _, hash := range consistencyProof.Proof {
		proofInterface = append(proofInterface, hash[:])
	}

	// Create consistency proof array: [old-size, new-size, proof-hashes]
	consistencyProofArray := []interface{}{
		consistencyProof.OldSize,
		consistencyProof.NewSize,
		proofInterface,
	}

	// Encode the entire proof array as a single CBOR bytestring
	consistencyProofCBOR, _ := cbor.Marshal(consistencyProofArray)

	// Add to unprotected headers using VDP (Verifiable Data Proof) label
	// -2 is a custom label for consistency proof (not in standard)
	// Store as array containing the CBOR-encoded proof
	receipt.Unprotected[cose.HeaderLabelVerifiableDataProof] = map[interface{}]interface{}{
		int64(-2): []interface{}{consistencyProofCBOR},
	}

	// Encode the receipt to CBOR
	receiptCBOR, err := cose.EncodeCoseSign1(receipt)
	if err != nil {
		t.Fatalf("failed to encode receipt: %v", err)
	}

	// Save to demo directory
	if err := os.MkdirAll(demoReceiptDir, 0755); err != nil {
		t.Logf("warning: failed to create demo directory: %v", err)
	} else {
		receiptPath := filepath.Join(demoReceiptDir, "consistency-receipt.cbor")
		if err := os.WriteFile(receiptPath, receiptCBOR, 0644); err != nil {
			t.Logf("warning: failed to save receipt: %v", err)
		} else {
			t.Logf("✓ Saved consistency receipt to: %s", receiptPath)
		}
	}

	// Validation: Verify the receipt
	t.Run("validate consistency receipt", func(t *testing.T) {
		// 1. Decode the receipt
		decodedReceipt, err := cose.DecodeCoseSign1(receiptCBOR)
		if err != nil {
			t.Fatalf("failed to decode receipt: %v", err)
		}

		// 2. Extract consistency proof from unprotected headers
		// Try both int64 and uint64 keys for VDP header (396)
		var vdpHeader interface{}
		var ok bool
		vdpHeader, ok = decodedReceipt.Unprotected[int64(cose.HeaderLabelVerifiableDataProof)]
		if !ok {
			vdpHeader, ok = decodedReceipt.Unprotected[uint64(cose.HeaderLabelVerifiableDataProof)]
			if !ok {
				t.Fatalf("VDP header not found in receipt. Available keys: %v", decodedReceipt.Unprotected)
			}
		}

		vdpMap, ok := vdpHeader.(map[interface{}]interface{})
		if !ok {
			t.Fatal("VDP header is not a map")
		}

		// 3. Extract consistency proof (stored as array of CBOR bytestrings)
		consistencyProofs, ok := vdpMap[int64(-2)].([]interface{})
		if !ok || len(consistencyProofs) == 0 {
			t.Fatal("consistency proof not found in VDP or not an array")
		}

		// Extract first proof from the array (it's a CBOR-encoded bytestring)
		consistencyProofBytes, ok := consistencyProofs[0].([]byte)
		if !ok {
			t.Fatal("consistency proof is not a bytestring")
		}

		// Decode the entire proof array from CBOR
		var consistencyProofArray []interface{}
		if err := cbor.Unmarshal(consistencyProofBytes, &consistencyProofArray); err != nil {
			t.Fatalf("failed to decode consistency proof: %v", err)
		}

		if len(consistencyProofArray) != 3 {
			t.Fatalf("invalid consistency proof structure: expected 3 elements, got %d", len(consistencyProofArray))
		}

		// Extract old size (handle both int64 and uint64)
		var extractedOldSize int64
		switch v := consistencyProofArray[0].(type) {
		case int64:
			extractedOldSize = v
		case uint64:
			extractedOldSize = int64(v)
		default:
			t.Fatalf("unexpected type for old size: %T", consistencyProofArray[0])
		}

		// Extract new size (handle both int64 and uint64)
		var extractedNewSize int64
		switch v := consistencyProofArray[1].(type) {
		case int64:
			extractedNewSize = v
		case uint64:
			extractedNewSize = int64(v)
		default:
			t.Fatalf("unexpected type for new size: %T", consistencyProofArray[1])
		}

		extractedProofHashes := consistencyProofArray[2].([]interface{})

		// Convert proof hashes back to [][32]byte
		var proofHashes [][32]byte
		for _, hashInterface := range extractedProofHashes {
			hashBytes := hashInterface.([]byte)
			var hash [32]byte
			copy(hash[:], hashBytes)
			proofHashes = append(proofHashes, hash)
		}

		extractedProof := &merkle.ConsistencyProof{
			OldSize: extractedOldSize,
			NewSize: extractedNewSize,
			Proof:   proofHashes,
		}

		// 4. Verify consistency proof
		valid := merkle.VerifyConsistencyProof(extractedProof, oldRoot, newRoot)
		if !valid {
			t.Error("consistency proof verification failed")
		}

		// 5. Verify COSE signature on receipt using new root as external payload
		validSig, err := cose.VerifyCoseSign1(decodedReceipt, verifier, newRoot[:])
		if err != nil {
			t.Fatalf("failed to verify receipt signature: %v", err)
		}

		if !validSig {
			t.Error("receipt signature is invalid")
		}

		t.Log("✓ Consistency receipt validated successfully")
		t.Logf("  Old tree size: %d", extractedOldSize)
		t.Logf("  New tree size: %d", extractedNewSize)
		t.Logf("  Proof length: %d", len(proofHashes))
	})
}

// TestReceiptRoundTrip tests full end-to-end receipt generation and validation
func TestReceiptRoundTrip(t *testing.T) {
	t.Run("inclusion receipt round-trip", func(t *testing.T) {
		// This test verifies that a receipt can be:
		// 1. Generated with an inclusion proof
		// 2. Encoded to CBOR
		// 3. Decoded from CBOR
		// 4. Validated successfully

		store := storage.NewMemoryStorage()
		tl := merkle.NewTileLog(store)
		_ = tl.Load()

		// Add entries
		for i := 0; i < 10; i++ {
			hash := sha256.Sum256([]byte{byte(i)})
			_, _ = tl.Append(hash)
		}

		// Generate proof for entry 5
		proof, _ := merkle.GenerateInclusionProof(store, 5, 10)
		root, _ := merkle.ComputeTreeRoot(store, 10)

		// Create signed receipt
		keyPair, _ := cose.GenerateES256KeyPair()
		signer, _ := cose.NewES256Signer(keyPair.Private)
		verifier, _ := cose.NewES256Verifier(keyPair.Public)

		cwtClaims := cose.CreateCWTClaims(cose.CWTClaimsOptions{
			Iss: "https://test.example.com",
		})

		headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
			Alg:       cose.AlgorithmES256,
			Kid:       []byte("test-key"),
			CWTClaims: cwtClaims,
		})

		receipt, _ := cose.CreateCoseSign1(headers, root[:], signer, cose.CoseSign1Options{Detached: true})

		// Add proof to unprotected headers
		var auditPathInterface []interface{}
		for _, hash := range proof.AuditPath {
			auditPathInterface = append(auditPathInterface, hash[:])
		}

		inclusionProofArray := []interface{}{proof.TreeSize, proof.LeafIndex, auditPathInterface}
		inclusionProofCBOR, _ := cbor.Marshal(inclusionProofArray)

		receipt.Unprotected[cose.HeaderLabelVerifiableDataProof] = map[interface{}]interface{}{
			int64(-1): inclusionProofCBOR,
		}

		// Encode and decode
		encoded, err := cose.EncodeCoseSign1(receipt)
		if err != nil {
			t.Fatalf("failed to encode: %v", err)
		}

		decoded, err := cose.DecodeCoseSign1(encoded)
		if err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		// Verify signature
		valid, err := cose.VerifyCoseSign1(decoded, verifier, root[:])
		if err != nil {
			t.Fatalf("verification failed: %v", err)
		}

		if !valid {
			t.Error("signature should be valid after round-trip")
		}

		t.Log("✓ Inclusion receipt round-trip successful")
	})
}
