package merkle

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
)

// Checkpoint represents a signed tree head
// It contains a commitment to the current state of the Merkle tree
type Checkpoint struct {
	TreeSize  int64         // Number of entries in the tree
	RootHash  [HashSize]byte // Root hash of the Merkle tree
	Timestamp int64         // Unix timestamp in milliseconds
	Origin    string        // Transparency service URL
	Signature []byte        // ES256 signature
}

// CreateCheckpoint creates a signed checkpoint for the current tree state
func CreateCheckpoint(treeSize int64, rootHash [HashSize]byte, privateKey *ecdsa.PrivateKey, origin string) (*Checkpoint, error) {
	// Validate origin URL
	if _, err := url.Parse(origin); err != nil {
		return nil, fmt.Errorf("invalid origin URL: %w", err)
	}

	timestamp := time.Now().UnixMilli()

	// Create the data to be signed
	dataToSign := encodeCheckpointData(treeSize, rootHash, timestamp, origin)

	// Sign with ES256
	signer, err := cose.NewES256Signer(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	signature, err := signer.Sign(dataToSign)
	if err != nil {
		return nil, fmt.Errorf("failed to sign checkpoint: %w", err)
	}

	return &Checkpoint{
		TreeSize:  treeSize,
		RootHash:  rootHash,
		Timestamp: timestamp,
		Origin:    origin,
		Signature: signature,
	}, nil
}

// VerifyCheckpoint verifies a checkpoint signature
func VerifyCheckpoint(checkpoint *Checkpoint, publicKey *ecdsa.PublicKey) (bool, error) {
	// Reconstruct the signed data
	dataToSign := encodeCheckpointData(
		checkpoint.TreeSize,
		checkpoint.RootHash,
		checkpoint.Timestamp,
		checkpoint.Origin,
	)

	// Verify signature
	verifier, err := cose.NewES256Verifier(publicKey)
	if err != nil {
		return false, fmt.Errorf("failed to create verifier: %w", err)
	}

	valid, err := verifier.Verify(dataToSign, checkpoint.Signature)
	if err != nil {
		return false, fmt.Errorf("failed to verify signature: %w", err)
	}

	return valid, nil
}

// EncodeCheckpoint encodes a checkpoint to signed note format
//
// Format:
//
//	<origin>
//	<tree-size>
//	<root-hash-base64>
//	<timestamp>
//
//	— <origin> <signature-base64>
func EncodeCheckpoint(checkpoint *Checkpoint) string {
	rootHashBase64 := base64.StdEncoding.EncodeToString(checkpoint.RootHash[:])
	signatureBase64 := base64.StdEncoding.EncodeToString(checkpoint.Signature)

	lines := []string{
		checkpoint.Origin,
		fmt.Sprintf("%d", checkpoint.TreeSize),
		rootHashBase64,
		fmt.Sprintf("%d", checkpoint.Timestamp),
		"",
		fmt.Sprintf("— %s %s", checkpoint.Origin, signatureBase64),
	}

	return strings.Join(lines, "\n")
}

// DecodeCheckpoint decodes a checkpoint from signed note format
func DecodeCheckpoint(encoded string) (*Checkpoint, error) {
	lines := strings.Split(strings.TrimSpace(encoded), "\n")

	if len(lines) < 6 {
		return nil, fmt.Errorf("invalid checkpoint format: too few lines (got %d, need at least 6)", len(lines))
	}

	origin := lines[0]

	var treeSize int64
	if _, err := fmt.Sscanf(lines[1], "%d", &treeSize); err != nil {
		return nil, fmt.Errorf("invalid tree size: %w", err)
	}

	rootHashBase64 := lines[2]

	var timestamp int64
	if _, err := fmt.Sscanf(lines[3], "%d", &timestamp); err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}

	// Parse signature line: "— <origin> <signature>"
	signatureLine := lines[5]
	signatureRegex := regexp.MustCompile(`^— .+ (.+)$`)
	matches := signatureRegex.FindStringSubmatch(signatureLine)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid checkpoint format: signature line malformed")
	}
	signatureBase64 := matches[1]

	// Decode base64
	rootHashBytes, err := base64.StdEncoding.DecodeString(rootHashBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid root hash base64: %w", err)
	}

	if len(rootHashBytes) != HashSize {
		return nil, fmt.Errorf("invalid root hash length: %d (expected %d)", len(rootHashBytes), HashSize)
	}

	var rootHash [HashSize]byte
	copy(rootHash[:], rootHashBytes)

	signature, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid signature base64: %w", err)
	}

	return &Checkpoint{
		TreeSize:  treeSize,
		RootHash:  rootHash,
		Timestamp: timestamp,
		Origin:    origin,
		Signature: signature,
	}, nil
}

// encodeCheckpointData encodes checkpoint data for signing
// Binary encoding: tree_size (8 bytes) + root_hash (32 bytes) + timestamp (8 bytes) + origin (variable)
func encodeCheckpointData(treeSize int64, rootHash [HashSize]byte, timestamp int64, origin string) []byte {
	originBytes := []byte(origin)
	buffer := make([]byte, 8+HashSize+8+len(originBytes))

	// Write tree size (64-bit big-endian)
	binary.BigEndian.PutUint64(buffer[0:8], uint64(treeSize))

	// Write root hash
	copy(buffer[8:8+HashSize], rootHash[:])

	// Write timestamp (64-bit big-endian)
	binary.BigEndian.PutUint64(buffer[8+HashSize:8+HashSize+8], uint64(timestamp))

	// Write origin
	copy(buffer[8+HashSize+8:], originBytes)

	return buffer
}
