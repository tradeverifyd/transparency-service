package service

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/tradeverifyd/transparency-service/scitt-golang/internal/config"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/database"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/merkle"
	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/storage"
)

// TransparencyService coordinates all transparency service operations
type TransparencyService struct {
	config     *config.Config
	db         *sql.DB
	storage    storage.Storage
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
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

	return &TransparencyService{
		config:     cfg,
		db:         db,
		storage:    store,
		privateKey: privateKey,
		publicKey:  publicKey,
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
	EntryID       int64  `json:"entry_id"`
	StatementHash string `json:"statement_hash"`
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
	tilePath := merkle.TileIndexToPath(0, tileIndex, nil)

	// Hash the statement for the Merkle tree
	leafHash := statementHash

	// Store entry tile
	entryTileKey := fmt.Sprintf("tile/entries/%d", entryID)
	if err := s.storage.Put(entryTileKey, leafHash[:]); err != nil {
		return nil, fmt.Errorf("failed to store entry tile: %w", err)
	}

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

	insertedID, err := database.InsertStatement(s.db, stmt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert statement: %w", err)
	}

	// Update tree size
	if err := database.SetCurrentTreeSize(s.db, treeSize+1); err != nil {
		return nil, fmt.Errorf("failed to update tree size: %w", err)
	}

	// Generate receipt (simplified for now)
	// In production, this would:
	// 1. Compute Merkle inclusion proof
	// 2. Generate signed checkpoint
	// 3. Store receipt in object storage
	// 4. Update receipt metadata in database

	return &RegisterStatementResponse{
		EntryID:       insertedID,
		StatementHash: statementHashHex,
	}, nil
}

// GetReceipt retrieves a receipt for a registered statement
func (s *TransparencyService) GetReceipt(entryID int64) ([]byte, error) {
	// Query statement by entry ID
	stmt, err := database.FindStatementByEntryID(s.db, entryID)
	if err != nil {
		return nil, fmt.Errorf("statement not found: %w", err)
	}

	// In production, this would retrieve the full receipt from object storage
	// For now, return a placeholder indicating the statement is registered
	receipt := map[string]interface{}{
		"entry_id":       entryID,
		"statement_hash": stmt.StatementHash,
		"tree_size":      stmt.TreeSizeAtRegistration + 1,
		"timestamp":      time.Now().Unix(),
	}

	// Convert to JSON (in production would be CBOR)
	return []byte(fmt.Sprintf("%+v", receipt)), nil
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
