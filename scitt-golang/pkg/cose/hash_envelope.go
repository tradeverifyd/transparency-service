package cose

import (
	"bytes"
	"crypto"
	"fmt"
	"io"
	"os"
)

// Hash Algorithm Constants (for COSE Hash Envelope)
const (
	HashAlgorithmSHA256 = -16 // SHA-256
	HashAlgorithmSHA384 = -43 // SHA-384
	HashAlgorithmSHA512 = -44 // SHA-512
)

// HashEnvelope represents a COSE hash envelope structure
// Used for signing large files by signing their hash instead of the full content
type HashEnvelope struct {
	PayloadHash            []byte // Hash of the payload
	PayloadHashAlg         int    // Hash algorithm identifier
	PreimageContentType    string // Content type of original payload (optional)
	PayloadLocation        string // Location of original payload (optional)
}

// HashEnvelopeOptions holds options for creating hash envelopes
type HashEnvelopeOptions struct {
	ContentType   string // Content type of the artifact
	Location      string // Location/URL of the artifact
	HashAlgorithm int    // Hash algorithm to use (default: SHA-256)
}

// HashEnvelopeVerificationResult holds the result of hash envelope verification
type HashEnvelopeVerificationResult struct {
	SignatureValid bool // Whether the COSE signature is valid
	HashValid      bool // Whether the hash matches the artifact
}

// CreateHashEnvelope creates a hash envelope from data
func CreateHashEnvelope(data []byte, options HashEnvelopeOptions) (*HashEnvelope, error) {
	hashAlgorithm := options.HashAlgorithm
	if hashAlgorithm == 0 {
		hashAlgorithm = HashAlgorithmSHA256
	}

	hash, err := HashData(data, hashAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("failed to hash data: %w", err)
	}

	return &HashEnvelope{
		PayloadHash:         hash,
		PayloadHashAlg:      hashAlgorithm,
		PreimageContentType: options.ContentType,
		PayloadLocation:     options.Location,
	}, nil
}

// HashData hashes data using the specified COSE hash algorithm
func HashData(data []byte, algorithm int) ([]byte, error) {
	hashAlg, err := getCryptoHashAlgorithm(algorithm)
	if err != nil {
		return nil, err
	}

	if !hashAlg.Available() {
		return nil, fmt.Errorf("hash algorithm not available")
	}

	h := hashAlg.New()
	h.Write(data)
	return h.Sum(nil), nil
}

// StreamHashFromFile computes hash of a file using streaming I/O
// Efficient for large files
func StreamHashFromFile(filePath string, algorithm int) ([]byte, error) {
	hashAlg, err := getCryptoHashAlgorithm(algorithm)
	if err != nil {
		return nil, err
	}

	if !hashAlg.Available() {
		return nil, fmt.Errorf("hash algorithm not available")
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create hash
	h := hashAlg.New()

	// Copy file to hash (streaming)
	if _, err := io.Copy(h, file); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return h.Sum(nil), nil
}

// ValidateHashEnvelope validates that the hash envelope matches the provided data
func ValidateHashEnvelope(envelope *HashEnvelope, data []byte) (bool, error) {
	computedHash, err := HashData(data, envelope.PayloadHashAlg)
	if err != nil {
		return false, fmt.Errorf("failed to compute hash: %w", err)
	}

	return bytes.Equal(computedHash, envelope.PayloadHash), nil
}

// SignHashEnvelope creates a COSE Sign1 structure with hash envelope
//
// This function:
// 1. Computes the hash of the artifact
// 2. Creates protected headers with hash envelope labels (258-260)
// 3. Signs the hash (not the full artifact)
//
// Parameters:
//   - artifact: The original data to hash and sign
//   - options: Hash envelope options (content type, location, hash algorithm)
//   - signer: Signer implementation (e.g., ES256Signer)
//   - cwtClaims: Optional CWT claims (iss, sub, etc.) per RFC 9597
//   - detached: Whether to create detached signature
//
// Returns:
//   - COSE Sign1 structure with hash as payload
func SignHashEnvelope(
	artifact []byte,
	options HashEnvelopeOptions,
	signer Signer,
	cwtClaims CWTClaimsSet,
	detached bool,
) (*CoseSign1, error) {
	// Create hash envelope
	envelope, err := CreateHashEnvelope(artifact, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create hash envelope: %w", err)
	}

	// Build protected headers with hash envelope labels
	headers := make(ProtectedHeaders)

	// Set algorithm (ES256 for now)
	headers[HeaderLabelAlg] = AlgorithmES256

	// Set hash envelope labels (258-260)
	headers[HeaderLabelPayloadHashAlg] = envelope.PayloadHashAlg

	if envelope.PreimageContentType != "" {
		headers[HeaderLabelPayloadPreimageContentType] = envelope.PreimageContentType
	}

	if envelope.PayloadLocation != "" {
		headers[HeaderLabelPayloadLocation] = envelope.PayloadLocation
	}

	// Add CWT claims if provided
	if len(cwtClaims) > 0 {
		headers[HeaderLabelCWTClaims] = cwtClaims
	}

	// Sign with hash as payload
	return CreateCoseSign1(
		headers,
		envelope.PayloadHash,
		signer,
		CoseSign1Options{Detached: detached},
	)
}

// VerifyHashEnvelope verifies a hash envelope COSE Sign1 structure
//
// This function verifies both:
// 1. The COSE signature is valid
// 2. The hash in the payload matches the provided artifact
//
// Parameters:
//   - coseSign1: COSE Sign1 structure with hash envelope
//   - artifact: Original artifact to verify against
//   - verifier: Verifier implementation (e.g., ES256Verifier)
//
// Returns:
//   - Verification result with signature and hash validity
func VerifyHashEnvelope(
	coseSign1 *CoseSign1,
	artifact []byte,
	verifier Verifier,
) (*HashEnvelopeVerificationResult, error) {
	// Extract hash envelope parameters
	params, err := ExtractHashEnvelopeParams(coseSign1)
	if err != nil {
		return &HashEnvelopeVerificationResult{
			SignatureValid: false,
			HashValid:      false,
		}, fmt.Errorf("failed to extract hash envelope params: %w", err)
	}

	// Verify signature
	signatureValid, err := VerifyCoseSign1(coseSign1, verifier, nil)
	if err != nil {
		signatureValid = false
	}

	// Verify hash
	computedHash, err := HashData(artifact, params.PayloadHashAlg)
	if err != nil {
		return &HashEnvelopeVerificationResult{
			SignatureValid: signatureValid,
			HashValid:      false,
		}, fmt.Errorf("failed to compute hash: %w", err)
	}

	expectedHash := coseSign1.Payload
	hashValid := expectedHash != nil && bytes.Equal(computedHash, expectedHash)

	return &HashEnvelopeVerificationResult{
		SignatureValid: signatureValid,
		HashValid:      hashValid,
	}, nil
}

// ExtractHashEnvelopeParams extracts hash envelope parameters from COSE Sign1 protected headers
func ExtractHashEnvelopeParams(coseSign1 *CoseSign1) (*HashEnvelope, error) {
	headers, err := GetProtectedHeaders(coseSign1)
	if err != nil {
		return nil, fmt.Errorf("failed to get protected headers: %w", err)
	}

	// Extract payload_hash_alg (label 258) - required
	payloadHashAlgKey := uint64(HeaderLabelPayloadHashAlg)
	payloadHashAlgVal, ok := headers[payloadHashAlgKey]
	if !ok {
		return nil, fmt.Errorf("missing payload_hash_alg (label 258) in protected headers")
	}

	payloadHashAlg, ok := payloadHashAlgVal.(int64)
	if !ok {
		return nil, fmt.Errorf("invalid payload_hash_alg type: expected int64")
	}

	// Check payload exists
	if coseSign1.Payload == nil {
		return nil, fmt.Errorf("missing payload (hash) in COSE Sign1")
	}

	// Extract optional parameters
	var preimageContentType, payloadLocation string

	if val, ok := headers[uint64(HeaderLabelPayloadPreimageContentType)]; ok {
		if str, ok := val.(string); ok {
			preimageContentType = str
		}
	}

	if val, ok := headers[uint64(HeaderLabelPayloadLocation)]; ok {
		if str, ok := val.(string); ok {
			payloadLocation = str
		}
	}

	return &HashEnvelope{
		PayloadHash:         coseSign1.Payload,
		PayloadHashAlg:      int(payloadHashAlg),
		PreimageContentType: preimageContentType,
		PayloadLocation:     payloadLocation,
	}, nil
}

// getCryptoHashAlgorithm converts COSE hash algorithm to crypto.Hash
func getCryptoHashAlgorithm(algorithm int) (crypto.Hash, error) {
	switch algorithm {
	case HashAlgorithmSHA256:
		return crypto.SHA256, nil
	case HashAlgorithmSHA384:
		return crypto.SHA384, nil
	case HashAlgorithmSHA512:
		return crypto.SHA512, nil
	default:
		return 0, fmt.Errorf("unsupported hash algorithm: %d", algorithm)
	}
}
