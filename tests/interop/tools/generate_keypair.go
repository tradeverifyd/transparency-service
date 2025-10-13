package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// KeypairFixture represents a test keypair in the format expected by test-fixtures.schema.json
type KeypairFixture struct {
	FixtureID   string          `json:"fixture_id"`
	FixtureType string          `json:"fixture_type"`
	Version     string          `json:"version"`
	Keypair     KeypairData     `json:"keypair"`
	Description string          `json:"description"`
	Tags        []string        `json:"tags"`
	CreatedAt   string          `json:"created_at"`
	CreatedBy   string          `json:"created_by"`
}

// KeypairData contains the actual key material
type KeypairData struct {
	PrivateKeyPEM  string `json:"private_key_pem"`
	PublicKeyJWK   string `json:"public_key_jwk"`
	Algorithm      string `json:"algorithm"`
	JWKThumbprint  string `json:"jwk_thumbprint"`
	KeyID          string `json:"key_id,omitempty"`
}

// JWK represents a JSON Web Key (RFC 7517)
type JWK struct {
	Kty string `json:"kty"` // Key Type
	Crv string `json:"crv"` // Curve
	X   string `json:"x"`   // X coordinate (base64url)
	Y   string `json:"y"`   // Y coordinate (base64url)
}

func main() {
	fixtureID := flag.String("id", "keypair_test", "Fixture ID")
	outputDir := flag.String("output", "../fixtures/keys", "Output directory")
	description := flag.String("desc", "Test keypair for ES256", "Description")
	flag.Parse()

	// Generate ECDSA P-256 keypair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate keypair: %v", err)
	}

	// Export private key as PEM
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		log.Fatalf("Failed to marshal private key: %v", err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// Export public key as JWK
	publicKey := &privateKey.PublicKey
	xBytes := publicKey.X.Bytes()
	yBytes := publicKey.Y.Bytes()

	// Pad to 32 bytes for P-256
	xPadded := make([]byte, 32)
	yPadded := make([]byte, 32)
	copy(xPadded[32-len(xBytes):], xBytes)
	copy(yPadded[32-len(yBytes):], yBytes)

	jwk := JWK{
		Kty: "EC",
		Crv: "P-256",
		X:   base64URLEncode(xPadded),
		Y:   base64URLEncode(yPadded),
	}

	jwkJSON, err := json.Marshal(jwk)
	if err != nil {
		log.Fatalf("Failed to marshal JWK: %v", err)
	}

	// Compute JWK thumbprint (RFC 7638)
	// Canonical JSON: {"crv":"P-256","kty":"EC","x":"...","y":"..."}
	// Thumbprint is base64url-encoded SHA-256 hash (NOT hex)
	canonicalJWK := fmt.Sprintf(`{"crv":"P-256","kty":"EC","x":"%s","y":"%s"}`, jwk.X, jwk.Y)
	hash := sha256.Sum256([]byte(canonicalJWK))
	thumbprint := base64URLEncode(hash[:])

	// Create fixture
	fixture := KeypairFixture{
		FixtureID:   *fixtureID,
		FixtureType: "keypair",
		Version:     "1.0",
		Keypair: KeypairData{
			PrivateKeyPEM: string(privateKeyPEM),
			PublicKeyJWK:  string(jwkJSON),
			Algorithm:     "ES256",
			JWKThumbprint: thumbprint,
			KeyID:         *fixtureID,
		},
		Description: *description,
		Tags:        []string{"crypto", "positive"},
		CreatedAt:   "2025-10-12T00:00:00Z",
		CreatedBy:   "generate_keypair.go",
	}

	// Write fixture to file
	outputPath := filepath.Join(*outputDir, fmt.Sprintf("%s.json", *fixtureID))
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	fixtureJSON, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal fixture: %v", err)
	}

	if err := os.WriteFile(outputPath, fixtureJSON, 0644); err != nil {
		log.Fatalf("Failed to write fixture: %v", err)
	}

	fmt.Printf("Generated keypair fixture: %s\n", outputPath)
	fmt.Printf("JWK Thumbprint: %s\n", thumbprint)
}

// base64URLEncode encodes bytes as base64url (RFC 4648)
func base64URLEncode(data []byte) string {
	const base64URL = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	result := ""

	// Standard base64 encoding
	for i := 0; i < len(data); i += 3 {
		chunk := uint32(data[i]) << 16
		if i+1 < len(data) {
			chunk |= uint32(data[i+1]) << 8
		}
		if i+2 < len(data) {
			chunk |= uint32(data[i+2])
		}

		result += string(base64URL[(chunk>>18)&0x3F])
		result += string(base64URL[(chunk>>12)&0x3F])
		if i+1 < len(data) {
			result += string(base64URL[(chunk>>6)&0x3F])
		}
		if i+2 < len(data) {
			result += string(base64URL[chunk&0x3F])
		}
	}

	// Remove padding
	return result
}
