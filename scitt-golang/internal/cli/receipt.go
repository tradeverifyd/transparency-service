package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/fxamacker/cbor/v2"
	"github.com/spf13/cobra"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/merkle"
)

// NewReceiptCommand creates the receipt command
func NewReceiptCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "receipt",
		Short: "Manage SCITT receipts",
		Long: `Manage SCITT receipts including verification and inspection.

Receipts are cryptographic proofs that a statement has been registered
in a transparency log. They contain:
  - Inclusion proof (proves statement is in the Merkle tree)
  - Checkpoint (signed tree head)
  - Statement metadata

Subcommands:
  verify  - Verify a receipt
  info    - Display receipt information`,
	}

	cmd.AddCommand(NewReceiptVerifyCommand())
	cmd.AddCommand(NewReceiptInfoCommand())

	return cmd
}

type receiptVerifyOptions struct {
	receipt   string
	statement string
	publicKey string // Deprecated - will fetch from issuer
}

// NewReceiptVerifyCommand creates the receipt verify command
func NewReceiptVerifyCommand() *cobra.Command {
	opts := &receiptVerifyOptions{}

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify a SCITT receipt",
		Long: `Verify a SCITT receipt's cryptographic proofs.

This command:
  1. Decodes the receipt and extracts the issuer from CWT claims
  2. Fetches the service's COSE keys from the issuer's well-known endpoint
  3. Selects the verification key matching the kid in the receipt
  4. Reconstructs the Merkle root from the inclusion proof and statement hash
  5. Verifies the COSE signature on the receipt

Example:
  scitt receipt verify --receipt receipt.cbor --statement statement.cbor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReceiptVerify(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.receipt, "receipt", "r", "", "receipt file (required)")
	cmd.Flags().StringVarP(&opts.statement, "statement", "s", "", "statement file (required)")

	cmd.MarkFlagRequired("receipt")
	cmd.MarkFlagRequired("statement")

	return cmd
}

func runReceiptVerify(opts *receiptVerifyOptions) error {
	// Read receipt
	receiptData, err := os.ReadFile(opts.receipt)
	if err != nil {
		return fmt.Errorf("failed to read receipt file: %w", err)
	}

	// Read statement
	statementData, err := os.ReadFile(opts.statement)
	if err != nil {
		return fmt.Errorf("failed to read statement file: %w", err)
	}

	if verbose {
		fmt.Printf("Receipt:   %s (%d bytes)\n", opts.receipt, len(receiptData))
		fmt.Printf("Statement: %s (%d bytes)\n", opts.statement, len(statementData))
	}

	// 1. Decode receipt CBOR
	receipt, err := cose.DecodeCoseSign1(receiptData)
	if err != nil {
		return fmt.Errorf("failed to decode receipt: %w", err)
	}

	// 2. Get protected headers from receipt
	headers, err := cose.GetProtectedHeaders(receipt)
	if err != nil {
		return fmt.Errorf("failed to get protected headers: %w", err)
	}

	// 3. Extract issuer URL from CWT claims
	// Note: header keys might be int64 or uint64 depending on CBOR decoder
	var cwtClaims map[interface{}]interface{}
	var ok bool

	// Try both int64 and uint64 keys for header label 15 (CWT claims)
	cwtClaims, ok = headers[int64(cose.HeaderLabelCWTClaims)].(map[interface{}]interface{})
	if !ok {
		cwtClaims, ok = headers[uint64(cose.HeaderLabelCWTClaims)].(map[interface{}]interface{})
		if !ok {
			return fmt.Errorf("CWT claims not found in receipt protected headers")
		}
	}

	// Try both int64 and uint64 keys for claim 1 (iss)
	var issuer string
	issuer, ok = cwtClaims[int64(cose.CWTClaimIss)].(string)
	if !ok {
		issuer, ok = cwtClaims[uint64(cose.CWTClaimIss)].(string)
		if !ok {
			return fmt.Errorf("issuer (iss) not found in CWT claims")
		}
	}

	if verbose {
		fmt.Printf("Issuer:    %s\n", issuer)
	}

	// 4. Fetch SCITT keys from issuer's well-known endpoint
	keysURL := issuer + "/.well-known/scitt-keys"
	if verbose {
		fmt.Printf("Fetching keys from: %s\n", keysURL)
	}

	resp, err := http.Get(keysURL)
	if err != nil {
		return fmt.Errorf("failed to fetch SCITT keys from %s: %w", keysURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch SCITT keys: HTTP %d", resp.StatusCode)
	}

	keysData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read SCITT keys response: %w", err)
	}

	// 5. Decode COSE Key Set
	var keySetArray []interface{}
	if err := cbor.Unmarshal(keysData, &keySetArray); err != nil {
		return fmt.Errorf("failed to decode COSE Key Set: %w", err)
	}

	// 6. Extract kid from receipt
	// Try both int64 and uint64 keys for header label 4 (kid)
	var kidFromReceipt []byte
	kidFromReceipt, ok = headers[int64(cose.HeaderLabelKid)].([]byte)
	if !ok {
		kidFromReceipt, ok = headers[uint64(cose.HeaderLabelKid)].([]byte)
		if !ok {
			return fmt.Errorf("kid not found in receipt protected headers")
		}
	}

	if verbose {
		fmt.Printf("Kid (from receipt): %x\n", kidFromReceipt)
	}

	// 7. Find matching key in key set
	var matchingKeyData []byte
	for _, keyInterface := range keySetArray {
		keyBytes, err := cbor.Marshal(keyInterface)
		if err != nil {
			continue
		}

		// Extract kid from this key
		keyKid, err := cose.GetKidFromCOSEKey(keyBytes)
		if err != nil {
			continue
		}

		// Compare kids
		if bytes.Equal(keyKid, kidFromReceipt) {
			matchingKeyData = keyBytes
			break
		}
	}

	if matchingKeyData == nil {
		return fmt.Errorf("no key found matching kid %x", kidFromReceipt)
	}

	if verbose {
		fmt.Println("Found matching verification key")
	}

	// 8. Import public key from COSE key
	publicKey, err := cose.ImportPublicKeyFromCOSECBOR(matchingKeyData)
	if err != nil {
		return fmt.Errorf("failed to import public key: %w", err)
	}

	// 9. Extract inclusion proof from unprotected headers
	// Try both int64 and uint64 keys for header label 396 (VDP)
	var vdpHeader interface{}
	vdpHeader, ok = receipt.Unprotected[int64(cose.HeaderLabelVerifiableDataProof)]
	if !ok {
		vdpHeader, ok = receipt.Unprotected[uint64(cose.HeaderLabelVerifiableDataProof)]
		if !ok {
			return fmt.Errorf("verifiable data proof not found in unprotected headers")
		}
	}

	vdpMap, ok := vdpHeader.(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("verifiable data proof is not a map")
	}

	// Try both int64 and uint64 keys for inclusion proof (-1)
	var inclusionProofCBOR []byte
	inclusionProofCBOR, ok = vdpMap[int64(-1)].([]byte)
	if !ok {
		inclusionProofCBOR, ok = vdpMap[uint64(18446744073709551615)].([]byte) // -1 as uint64
		if !ok {
			return fmt.Errorf("inclusion proof not found in verifiable data proof")
		}
	}

	// 10. Decode inclusion proof [tree-size, leaf-index, inclusion-path]
	var inclusionProofArray []interface{}
	if err := cbor.Unmarshal(inclusionProofCBOR, &inclusionProofArray); err != nil {
		return fmt.Errorf("failed to decode inclusion proof: %w", err)
	}

	if len(inclusionProofArray) != 3 {
		return fmt.Errorf("invalid inclusion proof structure: expected 3 elements, got %d", len(inclusionProofArray))
	}

	treeSize, ok := inclusionProofArray[0].(int64)
	if !ok {
		// Try uint64
		if ts, ok := inclusionProofArray[0].(uint64); ok {
			treeSize = int64(ts)
		} else {
			return fmt.Errorf("tree size is not an integer")
		}
	}

	leafIndex, ok := inclusionProofArray[1].(int64)
	if !ok {
		// Try uint64
		if li, ok := inclusionProofArray[1].(uint64); ok {
			leafIndex = int64(li)
		} else {
			return fmt.Errorf("leaf index is not an integer")
		}
	}

	inclusionPathInterface, ok := inclusionProofArray[2].([]interface{})
	if !ok {
		return fmt.Errorf("inclusion path is not an array")
	}

	// Convert inclusion path to [][32]byte
	var auditPath [][32]byte
	for i, hashInterface := range inclusionPathInterface {
		hashBytes, ok := hashInterface.([]byte)
		if !ok {
			return fmt.Errorf("hash at index %d is not bytes", i)
		}
		if len(hashBytes) != 32 {
			return fmt.Errorf("hash at index %d has invalid length: %d", i, len(hashBytes))
		}
		var hash [32]byte
		copy(hash[:], hashBytes)
		auditPath = append(auditPath, hash)
	}

	if verbose {
		fmt.Printf("Tree size: %d\n", treeSize)
		fmt.Printf("Leaf index: %d\n", leafIndex)
		fmt.Printf("Audit path length: %d\n", len(auditPath))
	}

	// 11. Compute entry (leaf hash) from statement CBOR
	// The entry is SHA-256 hash of the complete statement
	leafHash := sha256.Sum256(statementData)

	if verbose {
		fmt.Printf("Entry (leaf hash): %x\n", leafHash)
	}

	// 12. Reconstruct Merkle root from inclusion proof using tessera/merkle library
	// Create InclusionProof structure
	inclusionProof := &merkle.InclusionProof{
		LeafIndex: leafIndex,
		TreeSize:  treeSize,
		AuditPath: auditPath,
	}

	// Reconstruct the Merkle root from the leaf hash and inclusion proof
	reconstructedRoot := merkle.ReconstructRootFromInclusionProof(leafHash, inclusionProof)

	if verbose {
		fmt.Printf("Reconstructed Merkle root: %x\n", reconstructedRoot)
	}

	// 13. Verify COSE signature on receipt using reconstructed root as external payload
	// Since the receipt has detached payload, we must provide the Merkle root as external payload
	verifier, err := cose.NewES256Verifier(publicKey)
	if err != nil {
		return fmt.Errorf("failed to create verifier: %w", err)
	}

	valid, err := cose.VerifyCoseSign1(receipt, verifier, reconstructedRoot[:])
	if err != nil {
		return fmt.Errorf("failed to verify receipt signature: %w", err)
	}

	if !valid {
		return fmt.Errorf("receipt signature is invalid")
	}

	if verbose {
		fmt.Println("✓ Receipt signature verified")
		fmt.Println("✓ Inclusion proof verified")
	}

	fmt.Println("\n✓ Receipt verification successful")
	fmt.Printf("  Statement: %s\n", opts.statement)
	fmt.Printf("  Receipt: %s\n", opts.receipt)
	fmt.Printf("  Issuer: %s\n", issuer)
	fmt.Printf("  Tree size: %d\n", treeSize)
	fmt.Printf("  Leaf index: %d\n", leafIndex)

	return nil
}

type receiptInfoOptions struct {
	receipt string
}

// NewReceiptInfoCommand creates the receipt info command
func NewReceiptInfoCommand() *cobra.Command {
	opts := &receiptInfoOptions{}

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Display receipt information",
		Long: `Display information about a SCITT receipt.

Example:
  scitt receipt info --receipt receipt.cbor`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReceiptInfo(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.receipt, "receipt", "r", "", "receipt file (required)")

	cmd.MarkFlagRequired("receipt")

	return cmd
}

func runReceiptInfo(opts *receiptInfoOptions) error {
	// Read receipt
	receiptData, err := os.ReadFile(opts.receipt)
	if err != nil {
		return fmt.Errorf("failed to read receipt file: %w", err)
	}

	// TODO: Implement receipt parsing and display
	// This requires:
	// 1. Decode receipt CBOR structure
	// 2. Parse checkpoint, inclusion proof, metadata
	// 3. Display in human-readable format

	fmt.Printf("Receipt Information:\n")
	fmt.Printf("  File: %s\n", opts.receipt)
	fmt.Printf("  Size: %d bytes\n", len(receiptData))
	fmt.Printf("  Hash: %s\n", hex.EncodeToString(receiptData[:32]))
	fmt.Println("\nNote: Detailed receipt parsing not yet implemented")
	fmt.Println("This will be completed as part of integration testing (T027)")

	return nil
}
