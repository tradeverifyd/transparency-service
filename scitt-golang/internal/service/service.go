package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/config"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/merkle"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/repository"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
)

// TransparencyService coordinates all transparency service operations
type TransparencyService struct {
	config                      *config.Config
	repo                        repository.Repository
	storage                     storage.Storage
	privateKey                  *ecdsa.PrivateKey
	publicKey                   *ecdsa.PublicKey
	receiptSigningKeyIdentifier []byte // kid parsed from key file
}

// NewTransparencyService creates a new transparency service instance
func NewTransparencyService(cfg *config.Config) (*TransparencyService, error) {
	// Create repository based on database type
	var repo repository.Repository
	var err error

	ctx := context.Background()

	switch cfg.Database.Type {
	case "sqlite":
		repo, err = repository.NewSQLiteRepository(repository.SQLiteOptions{
			Path:      cfg.Database.Path,
			EnableWAL: cfg.Database.EnableWAL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite repository: %w", err)
		}

	case "mongodb":
		if cfg.Database.MongoDB == nil {
			return nil, fmt.Errorf("MongoDB configuration is required when database type is mongodb")
		}
		repo, err = repository.NewMongoDBRepository(ctx, repository.MongoDBOptions{
			URI:      cfg.Database.MongoDB.URI,
			Database: cfg.Database.MongoDB.Database,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create MongoDB repository: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported database type: %s (supported: sqlite, mongodb)", cfg.Database.Type)
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
	case "azure":
		if cfg.Storage.Azure == nil {
			return nil, fmt.Errorf("Azure storage configuration is required when storage type is azure")
		}
		store, err = storage.NewAzureStorage(ctx, storage.AzureStorageOptions{
			AccountName: cfg.Storage.Azure.AccountName,
			Container:   cfg.Storage.Azure.Container,
			SASURL:      cfg.Storage.Azure.SASURL,
			AccountKey:  cfg.Storage.Azure.AccountKey,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Azure storage: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported storage type: %s (supported: local, memory, azure)", cfg.Storage.Type)
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
		repo:                        repo,
		storage:                     store,
		privateKey:                  privateKey,
		publicKey:                   publicKey,
		receiptSigningKeyIdentifier: receiptSigningKeyIdentifier,
	}, nil
}

// Close closes the service and all resources
func (s *TransparencyService) Close() error {
	if s.repo != nil {
		return s.repo.Close()
	}
	return nil
}

// Reset removes all data from both repository and storage (for development/testing)
// WARNING: This operation is destructive and cannot be undone
func (s *TransparencyService) Reset() error {
	ctx := context.Background()

	// Clear repository (removes all statements and resets tree size to 0)
	if err := s.repo.Clear(ctx); err != nil {
		return fmt.Errorf("failed to clear repository: %w", err)
	}

	// Clear storage (removes all tiles)
	// Different storage implementations have different Clear method signatures
	switch store := s.storage.(type) {
	case interface{ Clear() error }:
		if err := store.Clear(); err != nil {
			return fmt.Errorf("failed to clear storage: %w", err)
		}
	case interface{ Clear() }:
		store.Clear()
	default:
		return fmt.Errorf("storage type does not support Clear operation")
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

	// Extract hash envelope parameters (SCITT statement metadata)
	var payloadHashAlg int
	var preimageContentType, payloadLocation string

	// Label 258: payload-hash-alg
	if alg, ok := headers[cose.HeaderLabelPayloadHashAlg].(int64); ok {
		payloadHashAlg = int(alg)
	} else if alg, ok := headers[cose.HeaderLabelPayloadHashAlg].(int); ok {
		payloadHashAlg = alg
	}

	// Label 259: preimage-content-type
	if cty, ok := headers[cose.HeaderLabelPayloadPreimageContentType].(string); ok {
		preimageContentType = cty
	}

	// Label 260: payload-location
	if loc, ok := headers[cose.HeaderLabelPayloadLocation].(string); ok {
		payloadLocation = loc
	}

	// The payload is the hash itself
	payloadHash := hex.EncodeToString(coseSign1.Payload)

	ctx := context.Background()

	// Get current tree size
	treeSize, err := s.repo.GetCurrentTreeSize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tree size: %w", err)
	}

	// Calculate entry tile coordinates
	entryID := treeSize
	tileIndex := merkle.EntryIDToTileIndex(entryID)
	tileOffset := merkle.EntryIDToTileOffset(entryID)

	// Hash the statement for the Merkle tree
	leafHash := statementHash
	leafHashHex := hex.EncodeToString(leafHash[:])

	// Append to entry tile (tessera-style tile management)
	if err := appendToEntryTile(s.storage, entryID, leafHash[:]); err != nil {
		return nil, fmt.Errorf("failed to append to entry tile: %w", err)
	}

	// Get tile path for database metadata
	tilePath := merkle.EntryTileIndexToPath(tileIndex, nil)

	// Convert strings to pointers for optional fields
	var subPtr, ctyPtr, locPtr *string
	if subject != "" {
		subPtr = &subject
	}
	if preimageContentType != "" {
		ctyPtr = &preimageContentType
	}
	if payloadLocation != "" {
		locPtr = &payloadLocation
	}

	// Insert statement metadata using repository
	stmt := &repository.StatementMetadata{
		EntryID:                entryID,
		LeafHash:               leafHashHex,
		Iss:                    issuer,
		Sub:                    subPtr,
		Cty:                    ctyPtr,
		PayloadHashAlg:         payloadHashAlg,
		PayloadHash:            payloadHash,
		PayloadLocation:        locPtr,
		RegisteredAt:           time.Now().UTC(),
		TreeSizeAtRegistration: treeSize,
		EntryTileKey:           tilePath,
		EntryTileOffset:        int(tileOffset),
	}

	_, err = s.repo.InsertStatement(ctx, stmt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert statement: %w", err)
	}

	// Update tree size
	if err := s.repo.SetCurrentTreeSize(ctx, treeSize+1); err != nil {
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
	ctx := context.Background()

	// Get current tree size
	treeSize, err := s.repo.GetCurrentTreeSize(ctx)
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

	// Build CWT claims with issuer
	cwtClaims := cose.CWTClaimsSet{
		cose.CWTClaimIss: s.config.Issuer, // Issuer URL
	}

	// Build protected headers: kid (4), alg (1), vds (395), CWT claims (15)
	// Use pre-parsed kid from key file (not computed)
	protectedHeaders := cose.ProtectedHeaders{
		cose.HeaderLabelKid:                    s.receiptSigningKeyIdentifier, // kid: parsed from key file
		cose.HeaderLabelAlg:                    int64(-7),                      // alg: ES256
		cose.HeaderLabelVerifiableDataStructure: int64(1),                       // vds: RFC 6962 SHA-256 tree algorithm
		cose.HeaderLabelCWTClaims:              cwtClaims,                      // CWT claims with issuer
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

	// Build COSE Sign1 receipt with detached payload (nil)
	// The payload (Merkle root) can be reconstructed from the inclusion proof
	receipt := &cose.CoseSign1{
		Protected:   protectedBytes,
		Unprotected: unprotectedHeaders,
		Payload:     nil, // Detached - reconstructed from inclusion proof
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
	ctx := context.Background()

	// Get current tree size
	treeSize, err := s.repo.GetCurrentTreeSize(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get tree size: %w", err)
	}

	// Compute tree root
	var rootHash [32]byte
	if treeSize > 0 {
		rootHash, err = merkle.ComputeTreeRoot(s.storage, treeSize)
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

// GetLastReceipt returns a receipt for the last entry in the log
func (s *TransparencyService) GetLastReceipt() ([]byte, error) {
	ctx := context.Background()

	// Get current tree size
	treeSize, err := s.repo.GetCurrentTreeSize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tree size: %w", err)
	}

	// Return empty tree error if no entries
	if treeSize == 0 {
		return nil, fmt.Errorf("log is empty, no entries to checkpoint")
	}

	// Get receipt for the last entry (treeSize - 1)
	lastEntryID := treeSize - 1
	return s.GetReceipt(lastEntryID)
}

// GetTile retrieves a merkle tree tile from storage
// Returns the raw tile data (256 hashes × 32 bytes = 8192 bytes for full tiles)
func (s *TransparencyService) GetTile(level int, index int64, width *int) ([]byte, error) {
	// Generate tile path
	tilePath := merkle.TileIndexToPath(level, index, width)

	// Retrieve from storage
	tileData, err := s.storage.Get(tilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get tile: %w", err)
	}

	if tileData == nil {
		return nil, fmt.Errorf("tile not found: %s", tilePath)
	}

	return tileData, nil
}

// GetEntryTile retrieves an entry tile from storage
// Returns the raw entry tile data (up to 256 hashes × 32 bytes)
func (s *TransparencyService) GetEntryTile(index int64, width *int) ([]byte, error) {
	// Generate entry tile path
	tilePath := merkle.EntryTileIndexToPath(index, width)

	// Retrieve from storage
	tileData, err := s.storage.Get(tilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get entry tile: %w", err)
	}

	if tileData == nil {
		return nil, fmt.Errorf("entry tile not found: %s", tilePath)
	}

	return tileData, nil
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
