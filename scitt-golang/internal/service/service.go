package service

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/fxamacker/cbor/v2"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/config"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/database"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/merkle"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
)

// TransparencyService coordinates all transparency service operations
type TransparencyService struct {
	config                      *config.Config
	db                          *sql.DB
	storage                     storage.Storage
	privateKey                  *ecdsa.PrivateKey
	publicKey                   *ecdsa.PublicKey
	receiptSigningKeyIdentifier []byte // kid parsed from key file
}

// NewTransparencyService creates a new transparency service instance
func NewTransparencyService(cfg *config.Config) (*TransparencyService, error) {
	// Open database
	db, err := database.OpenDatabase(database.DatabaseOptions{
		Path:      cfg.Database.Path,
		EnableWAL: cfg.Database.EnableWAL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize storage
	var store storage.Storage
	switch cfg.Storage.Type {
	case "local":
		store, err = storage.NewLocalStorage(cfg.Storage.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize local storage: %w", err)
		}
	case "memory":
		store = storage.NewMemoryStorage()
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}

	// Load private key
	privateKey, err := loadPrivateKey(cfg.Keys.Private)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	// Load public key
	publicKey, err := loadPublicKey(cfg.Keys.Public)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	// Parse kid from public key file (not computed)
	publicKeyData, err := os.ReadFile(cfg.Keys.Public)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file for kid extraction: %w", err)
	}

	receiptSigningKeyIdentifier, err := cose.GetKidFromCOSEKey(publicKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract kid from public key: %w", err)
	}

	return &TransparencyService{
		config:                      cfg,
		db:                          db,
		storage:                     store,
		privateKey:                  privateKey,
		publicKey:                   publicKey,
		receiptSigningKeyIdentifier: receiptSigningKeyIdentifier,
	}, nil
}

// Close closes the service and all resources
func (s *TransparencyService) Close() error {
	if s.db != nil {
		return database.CloseDatabase(s.db)
	}
	return nil
}

// RegisterStatementRequest represents a statement registration request
type RegisterStatementRequest struct {
	Statement []byte // CBOR-encoded COSE Sign1
}

// RegisterStatementResponse represents a statement registration response
type RegisterStatementResponse struct {
	EntryID       int64  // Entry ID in the log
	StatementHash string // Hex-encoded statement hash
	Receipt       []byte // CBOR-encoded COSE receipt
}

// RegisterStatement registers a new statement in the transparency log
func (s *TransparencyService) RegisterStatement(req *RegisterStatementRequest) (*RegisterStatementResponse, error) {
	// Decode COSE Sign1
	coseSign1, err := cose.DecodeCoseSign1(req.Statement)
	if err != nil {
		return nil, fmt.Errorf("invalid COSE Sign1 structure: %w", err)
	}

	// Verify signature (basic validation - in production would also verify issuer key)
	// For now, we'll skip verification and focus on registration logic

	// Compute statement hash
	statementHash := sha256.Sum256(req.Statement)
	statementHashHex := hex.EncodeToString(statementHash[:])

	// Get protected headers to extract metadata
	headers, err := cose.GetProtectedHeaders(coseSign1)
	if err != nil {
		return nil, fmt.Errorf("failed to get protected headers: %w", err)
	}

	// Extract issuer and subject from CWT claims if present
	var issuer, subject string
	if cwtClaims, ok := headers[cose.HeaderLabelCWTClaims].(map[interface{}]interface{}); ok {
		if iss, ok := cwtClaims[cose.CWTClaimIss].(string); ok {
			issuer = iss
		}
		if sub, ok := cwtClaims[cose.CWTClaimSub].(string); ok {
			subject = sub
		}
	}

	// Get content type
	var contentType string
	if cty, ok := headers[cose.HeaderLabelContentType].(string); ok {
		contentType = cty
	}

	// Get current tree size
	treeSize, err := database.GetCurrentTreeSize(s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to get tree size: %w", err)
	}

	// Calculate entry tile coordinates
	entryID := treeSize
	tileIndex := merkle.EntryIDToTileIndex(entryID)
	tileOffset := merkle.EntryIDToTileOffset(entryID)

	// Hash the statement for the Merkle tree
	leafHash := statementHash

	// Append to entry tile (tessera-style tile management)
	if err := appendToEntryTile(s.storage, entryID, leafHash[:]); err != nil {
		return nil, fmt.Errorf("failed to append to entry tile: %w", err)
	}

	// Get tile path for database metadata
	tilePath := merkle.EntryTileIndexToPath(tileIndex, nil)

	// Convert strings to pointers for optional fields
	var subPtr, ctyPtr *string
	if subject != "" {
		subPtr = &subject
	}
	if contentType != "" {
		ctyPtr = &contentType
	}

	// Insert statement metadata
	stmt := database.Statement{
		StatementHash:          statementHashHex,
		Iss:                    issuer,
		Sub:                    subPtr,
		Cty:                    ctyPtr,
		PayloadHashAlg:         -16, // SHA-256
		PayloadHash:            "",  // Extract from hash envelope if needed
		TreeSizeAtRegistration: treeSize,
		EntryTileKey:           tilePath,
		EntryTileOffset:        int(tileOffset),
	}

	_, err = database.InsertStatement(s.db, stmt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert statement: %w", err)
	}

	// Update tree size
	if err := database.SetCurrentTreeSize(s.db, treeSize+1); err != nil {
		return nil, fmt.Errorf("failed to update tree size: %w", err)
	}

	// Generate receipt using the entryID (which is treeSize before increment)
	receipt, err := s.GetReceipt(entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate receipt: %w", err)
	}

	return &RegisterStatementResponse{
		EntryID:       entryID,
		StatementHash: statementHashHex,
		Receipt:       receipt,
	}, nil
}

// GetReceipt retrieves a receipt for a registered statement
// Implements draft-ietf-cose-merkle-tree-proofs with inclusion proof and signed tree head
// The receipt is computed dynamically from the current tree state
func (s *TransparencyService) GetReceipt(entryID int64) ([]byte, error) {
	// Get current tree size
	treeSize, err := database.GetCurrentTreeSize(s.db)
	if err != nil {
		return nil, fmt.Errorf("failed to get tree size: %w", err)
	}

	// Verify entry ID is valid (within tree bounds)
	if entryID >= treeSize {
		return nil, fmt.Errorf("entry ID %d not found in tree of size %d", entryID, treeSize)
	}

	// Compute Merkle root using tessera library
	rootHash, err := merkle.ComputeTreeRoot(s.storage, treeSize)
	if err != nil {
		return nil, fmt.Errorf("failed to compute merkle root: %w", err)
	}

	// Generate inclusion proof using tessera library
	inclusionProof, err := merkle.GenerateInclusionProof(s.storage, entryID, treeSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate inclusion proof: %w", err)
	}

	// Build protected headers: kid (4), alg (1), vds (395)
	// Use pre-parsed kid from key file (not computed)
	protectedHeaders := cose.ProtectedHeaders{
		cose.HeaderLabelKid:                    s.receiptSigningKeyIdentifier, // kid: parsed from key file
		cose.HeaderLabelAlg:                    int64(-7),                      // alg: ES256
		cose.HeaderLabelVerifiableDataStructure: int64(1),                       // vds: RFC 6962 SHA-256 tree algorithm
	}

	// Encode protected headers using cbor
	protectedBytes, err := cbor.Marshal(protectedHeaders)
	if err != nil {
		return nil, fmt.Errorf("failed to encode protected headers: %w", err)
	}

	// Build inclusion-path as array of hashes (initialize as empty array, not nil)
	inclusionPath := make([]interface{}, 0, len(inclusionProof.AuditPath))
	for _, hash := range inclusionProof.AuditPath {
		inclusionPath = append(inclusionPath, hash[:])
	}

	// Build inclusion proof array as [tree-size, leaf-index, inclusion-path]
	inclusionProofArray := []interface{}{
		treeSize,                 // tree size
		inclusionProof.LeafIndex, // leaf index
		inclusionPath,            // inclusion-path: array of hashes
	}

	// CBOR encode the entire inclusion proof array
	inclusionProofCBOR, err := cbor.Marshal(inclusionProofArray)
	if err != nil {
		return nil, fmt.Errorf("failed to encode inclusion proof: %w", err)
	}

	// Build unprotected headers with CBOR-encoded inclusion proof
	// Label 396: verifiable-data-proofs contains a map with key -1 for inclusion proofs
	unprotectedHeaders := map[interface{}]interface{}{
		cose.HeaderLabelVerifiableDataProof: map[interface{}]interface{}{ // 396: verifiable-data-proofs
			int64(-1): inclusionProofCBOR, // -1: CBOR-encoded inclusion proof
		},
	}

	// Payload is the Merkle tree root hash
	payload := rootHash[:]

	// Create Sig_structure for signing (same structure as CreateCoseSign1)
	sigStructure := []interface{}{
		"Signature1",
		protectedBytes,
		[]byte{}, // empty external AAD
		payload,
	}

	toBeSigned, err := cbor.Marshal(sigStructure)
	if err != nil {
		return nil, fmt.Errorf("failed to encode Sig_structure: %w", err)
	}

	// Sign using ES256 signer
	signer, err := cose.NewES256Signer(s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	signature, err := signer.Sign(toBeSigned)
	if err != nil {
		return nil, fmt.Errorf("failed to sign receipt: %w", err)
	}

	// Build COSE Sign1 receipt
	receipt := &cose.CoseSign1{
		Protected:   protectedBytes,
		Unprotected: unprotectedHeaders,
		Payload:     payload,
		Signature:   signature,
	}

	// Encode as CBOR with COSE_Sign1 tag (18)
	receiptBytes, err := cose.EncodeCoseSign1(receipt)
	if err != nil {
		return nil, fmt.Errorf("failed to encode receipt: %w", err)
	}

	return receiptBytes, nil
}

// GetCheckpoint returns the current signed tree head
func (s *TransparencyService) GetCheckpoint() (string, error) {
	// Get current tree size
	treeSize, err := database.GetCurrentTreeSize(s.db)
	if err != nil {
		return "", fmt.Errorf("failed to get tree size: %w", err)
	}

	// Compute tree root
	var rootHash [32]byte
	if treeSize > 0 {
		rootHash, err = s.computeMerkleRoot(treeSize)
		if err != nil {
			return "", fmt.Errorf("failed to compute merkle root: %w", err)
		}
	}

	// Create checkpoint
	checkpoint, err := merkle.CreateCheckpoint(
		treeSize,
		rootHash,
		s.privateKey,
		s.config.Issuer,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create checkpoint: %w", err)
	}

	// Encode to signed note format
	return merkle.EncodeCheckpoint(checkpoint), nil
}

// GetSCITTConfiguration returns service configuration
func (s *TransparencyService) GetSCITTConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"issuer": s.config.Issuer,
		"supported_algorithms": []string{
			"ES256",
		},
		"supported_hash_algorithms": []string{
			"SHA-256",
		},
		"registration_policy": map[string]interface{}{
			"type": "open",
		},
	}
}

// GetSCITTKeys returns service verification keys as COSE Key Set (CBOR)
func (s *TransparencyService) GetSCITTKeys() ([]byte, error) {
	// Export public key as COSE Key Set (array of COSE_Keys) in CBOR format
	// This follows RFC 9052 Section 7 and SCRAPI specification
	cborData, err := cose.ExportCOSEKeySetToCBOR([]*ecdsa.PublicKey{s.publicKey})
	if err != nil {
		return nil, fmt.Errorf("failed to export COSE key set: %w", err)
	}

	return cborData, nil
}

// loadPrivateKey loads a private key from PEM or CBOR file
// Supports both .pem and .cbor file extensions
func loadPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	// Try CBOR format first (if file extension is .cbor)
	if len(path) > 5 && path[len(path)-5:] == ".cbor" {
		privateKey, err := cose.ImportPrivateKeyFromCOSECBOR(keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to import CBOR private key: %w", err)
		}
		return privateKey, nil
	}

	// Fall back to PEM format
	privateKey, err := cose.ImportPrivateKeyFromPEM(string(keyData))
	if err != nil {
		return nil, fmt.Errorf("failed to import PEM private key: %w", err)
	}

	return privateKey, nil
}

// loadPublicKey loads a public key from JWK or CBOR file
// Supports both .jwk/.json and .cbor file extensions
func loadPublicKey(path string) (*ecdsa.PublicKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	// Try CBOR format first (if file extension is .cbor)
	if len(path) > 5 && path[len(path)-5:] == ".cbor" {
		publicKey, err := cose.ImportPublicKeyFromCOSECBOR(keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to import CBOR public key: %w", err)
		}
		return publicKey, nil
	}

	// Fall back to JWK format
	jwk, err := cose.UnmarshalJWK(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWK: %w", err)
	}

	publicKey, err := cose.ImportPublicKeyFromJWK(jwk)
	if err != nil {
		return nil, fmt.Errorf("failed to import JWK public key: %w", err)
	}

	return publicKey, nil
}

// computeMerkleRoot computes the RFC 6962 merkle tree root from statement hashes
func (s *TransparencyService) computeMerkleRoot(treeSize int64) ([32]byte, error) {
	if treeSize == 0 {
		return [32]byte{}, fmt.Errorf("cannot compute root of empty tree")
	}

	// Query all statement hashes in order
	rows, err := s.db.Query(`
		SELECT statement_hash
		FROM statements
		WHERE entry_id <= ?
		ORDER BY entry_id ASC
	`, treeSize)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to query statement hashes: %w", err)
	}
	defer rows.Close()

	// Collect leaf hashes
	var leafHashes [][32]byte
	for rows.Next() {
		var hashHex string
		if err := rows.Scan(&hashHex); err != nil {
			return [32]byte{}, fmt.Errorf("failed to scan statement hash: %w", err)
		}

		// Decode hex hash
		hashBytes, err := hex.DecodeString(hashHex)
		if err != nil {
			return [32]byte{}, fmt.Errorf("invalid hash hex: %w", err)
		}

		if len(hashBytes) != 32 {
			return [32]byte{}, fmt.Errorf("invalid hash length: %d", len(hashBytes))
		}

		var hash [32]byte
		copy(hash[:], hashBytes)
		leafHashes = append(leafHashes, hash)
	}

	if err := rows.Err(); err != nil {
		return [32]byte{}, fmt.Errorf("error iterating rows: %w", err)
	}

	if int64(len(leafHashes)) != treeSize {
		return [32]byte{}, fmt.Errorf("expected %d leaves, got %d", treeSize, len(leafHashes))
	}

	// Compute merkle root using RFC 6962 algorithm
	if len(leafHashes) == 1 {
		// Single leaf: MTH({d[0]}) = SHA-256(0x00 || d[0])
		return hashLeaf(leafHashes[0]), nil
	}

	return computeSubtreeHash(leafHashes), nil
}

// computeSubtreeHash computes the RFC 6962 merkle tree hash of a subtree
// RFC 6962: MTH(D[n]) = SHA-256(0x01 || MTH(D[0:k]) || MTH(D[k:n]))
// where k is the largest power of 2 less than n
func computeSubtreeHash(leaves [][32]byte) [32]byte {
	n := len(leaves)

	if n == 0 {
		panic("cannot compute hash of empty subtree")
	}

	if n == 1 {
		// Single leaf
		return hashLeaf(leaves[0])
	}

	// Find k: largest power of 2 less than n
	k := largestPowerOfTwoLessThan(n)

	// Split into left and right subtrees
	left := leaves[:k]
	right := leaves[k:]

	leftHash := computeSubtreeHash(left)

	if len(right) == 0 {
		return leftHash
	}

	rightHash := computeSubtreeHash(right)

	// Hash the two subtrees together with node prefix
	return hashNode(leftHash, rightHash)
}

// hashLeaf hashes a leaf with RFC 6962 prefix (0x00)
func hashLeaf(leaf [32]byte) [32]byte {
	h := sha256.New()
	h.Write([]byte{0x00}) // RFC 6962 leaf prefix
	h.Write(leaf[:])
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

// hashNode hashes two child nodes with RFC 6962 prefix (0x01)
func hashNode(left, right [32]byte) [32]byte {
	h := sha256.New()
	h.Write([]byte{0x01}) // RFC 6962 node prefix
	h.Write(left[:])
	h.Write(right[:])
	var result [32]byte
	copy(result[:], h.Sum(nil))
	return result
}

// largestPowerOfTwoLessThan finds the largest power of 2 strictly less than n
func largestPowerOfTwoLessThan(n int) int {
	k := 1
	for k*2 < n {
		k *= 2
	}
	return k
}

// appendToEntryTile appends a leaf to an entry tile (tessera-style)
// This matches the tile storage format expected by the merkle proof library
func appendToEntryTile(store storage.Storage, entryID int64, leafHash []byte) error {
	tileIndex := merkle.EntryIDToTileIndex(entryID)
	tilePath := merkle.EntryTileIndexToPath(tileIndex, nil)

	// Read existing tile (if any)
	existingTile, err := store.Get(tilePath)
	if err != nil {
		return fmt.Errorf("failed to get existing tile: %w", err)
	}

	var currentSize int
	if existingTile != nil {
		currentSize = len(existingTile) / 32 // 32 bytes per hash
	}

	// Append new leaf
	newTile := make([]byte, (currentSize+1)*32)
	if existingTile != nil {
		copy(newTile, existingTile)
	}
	copy(newTile[currentSize*32:], leafHash)

	// Write updated tile
	if err := store.Put(tilePath, newTile); err != nil {
		return fmt.Errorf("failed to put tile: %w", err)
	}

	return nil
}
