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
	EntryID       int64  `json:"entryId"`
	StatementHash string `json:"statementHash"`
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
		PayloadHash:            "", // Extract from hash envelope if needed
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
		"entryId":       entryID,
		"statementHash": stmt.StatementHash,
		"treeSize":      stmt.TreeSizeAtRegistration + 1,
		"timestamp":     time.Now().Unix(),
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

	// Compute tree root (simplified - in production would use actual Merkle tree)
	// For now, use a placeholder root
	var rootHash [32]byte
	if treeSize > 0 {
		rootHash = sha256.Sum256([]byte(fmt.Sprintf("root-%d", treeSize)))
	}

	// Create checkpoint
	checkpoint, err := merkle.CreateCheckpoint(
		treeSize,
		rootHash,
		s.privateKey,
		s.config.Origin,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create checkpoint: %w", err)
	}

	// Encode to signed note format
	return merkle.EncodeCheckpoint(checkpoint), nil
}

// GetTransparencyConfiguration returns service configuration
func (s *TransparencyService) GetTransparencyConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"origin": s.config.Origin,
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

// loadPrivateKey loads a private key from PEM file
func loadPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	pemData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	privateKey, err := cose.ImportPrivateKeyFromPEM(string(pemData))
	if err != nil {
		return nil, fmt.Errorf("failed to import private key: %w", err)
	}

	return privateKey, nil
}

// loadPublicKey loads a public key from JWK file
func loadPublicKey(path string) (*ecdsa.PublicKey, error) {
	jwkData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	jwk, err := cose.UnmarshalJWK(jwkData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWK: %w", err)
	}

	publicKey, err := cose.ImportPublicKeyFromJWK(jwk)
	if err != nil {
		return nil, fmt.Errorf("failed to import public key: %w", err)
	}

	return publicKey, nil
}
