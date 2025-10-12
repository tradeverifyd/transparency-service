// Package cose provides COSE (RFC 8152/9052) cryptographic operations
package cose

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
)

// Signer interface for creating signatures
// This abstraction allows for HSM integration in the future
type Signer interface {
	Sign(data []byte) ([]byte, error)
}

// Verifier interface for validating signatures
type Verifier interface {
	Verify(data []byte, signature []byte) (bool, error)
}

// ES256Signer implements the Signer interface using ECDSA P-256 + SHA-256
type ES256Signer struct {
	privateKey *ecdsa.PrivateKey
}

// NewES256Signer creates a new ES256 signer from a private key
func NewES256Signer(privateKey *ecdsa.PrivateKey) (*ES256Signer, error) {
	if privateKey == nil {
		return nil, fmt.Errorf("private key is nil")
	}
	return &ES256Signer{privateKey: privateKey}, nil
}

// Sign signs the data using ECDSA P-256 with SHA-256
// Returns the signature in IEEE P1363 format (r || s)
func (s *ES256Signer) Sign(data []byte) ([]byte, error) {
	// Hash the data with SHA-256
	hash := crypto.SHA256
	hashed, err := hashData(data, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to hash data: %w", err)
	}

	// Sign using ECDSA
	r, sigS, err := ecdsa.Sign(rand.Reader, s.privateKey, hashed)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	// Convert to IEEE P1363 format (r || s)
	// For P-256, both r and s are 32 bytes
	signature := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := sigS.Bytes()

	// Pad to 32 bytes if necessary
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):64], sBytes)

	return signature, nil
}

// ES256Verifier implements the Verifier interface using ECDSA P-256 + SHA-256
type ES256Verifier struct {
	publicKey *ecdsa.PublicKey
}

// NewES256Verifier creates a new ES256 verifier from a public key
func NewES256Verifier(publicKey *ecdsa.PublicKey) (*ES256Verifier, error) {
	if publicKey == nil {
		return nil, fmt.Errorf("public key is nil")
	}
	return &ES256Verifier{publicKey: publicKey}, nil
}

// Verify verifies the signature using ECDSA P-256 with SHA-256
// Expects signature in IEEE P1363 format (r || s)
func (v *ES256Verifier) Verify(data []byte, signature []byte) (bool, error) {
	// Verify signature length (64 bytes for P-256)
	if len(signature) != 64 {
		return false, fmt.Errorf("invalid signature length: expected 64 bytes, got %d", len(signature))
	}

	// Hash the data with SHA-256
	hash := crypto.SHA256
	hashed, err := hashData(data, hash)
	if err != nil {
		return false, fmt.Errorf("failed to hash data: %w", err)
	}

	// Parse signature from IEEE P1363 format
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])

	// Verify using ECDSA
	valid := ecdsa.Verify(v.publicKey, hashed, r, s)
	return valid, nil
}

// hashData hashes data using the specified hash algorithm
func hashData(data []byte, hashAlg crypto.Hash) ([]byte, error) {
	if !hashAlg.Available() {
		return nil, fmt.Errorf("hash algorithm not available")
	}
	h := hashAlg.New()
	h.Write(data)
	return h.Sum(nil), nil
}
