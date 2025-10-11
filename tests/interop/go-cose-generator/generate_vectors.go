package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	_ "crypto/sha256" // Initialize SHA-256 for ECDSA
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/veraison/go-cose"
)

type TestVector struct {
	Payload           string `json:"payload"`           // hex encoded
	ProtectedHeaders  map[string]interface{} `json:"protectedHeaders"`
	CoseSign1Bytes    string `json:"coseSign1Bytes"`    // hex encoded COSE_Sign1
	PublicKeyX        string `json:"publicKeyX"`        // hex encoded
	PublicKeyY        string `json:"publicKeyY"`        // hex encoded
	Description       string `json:"description"`
}

func main() {
	vectors := []TestVector{}

	// Test 1: Simple message
	vector1, err := generateVector(
		[]byte("Hello, COSE!"),
		map[string]interface{}{
			"alg": int64(-7), // ES256
			"iss": "https://example.com/issuer",
			"kid": "key-1",
		},
		"Simple message with iss and kid",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating vector 1: %v\n", err)
		os.Exit(1)
	}
	vectors = append(vectors, vector1)

	// Test 2: Empty payload
	vector2, err := generateVector(
		[]byte{},
		map[string]interface{}{
			"alg": int64(-7),
			"iss": "https://example.com/issuer",
		},
		"Empty payload",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating vector 2: %v\n", err)
		os.Exit(1)
	}
	vectors = append(vectors, vector2)

	// Test 3: Large payload
	largePayload := make([]byte, 1024)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}
	vector3, err := generateVector(
		largePayload,
		map[string]interface{}{
			"alg": int64(-7),
			"iss": "https://issuer.example",
			"kid": "large-key",
			"sub": "urn:example:subject",
		},
		"Large payload (1KB) with subject",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating vector 3: %v\n", err)
		os.Exit(1)
	}
	vectors = append(vectors, vector3)

	// Test 4: With content type
	vector4, err := generateVector(
		[]byte(`{"data": "value"}`),
		map[string]interface{}{
			"alg": int64(-7),
			"iss": "https://example.com/issuer",
			"kid": "key-json",
			"cty": "application/json",
		},
		"JSON payload with content type",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating vector 4: %v\n", err)
		os.Exit(1)
	}
	vectors = append(vectors, vector4)

	// Test 5: Minimal headers (alg only)
	vector5, err := generateVector(
		[]byte("Minimal headers"),
		map[string]interface{}{
			"alg": int64(-7),
		},
		"Minimal protected headers (alg only)",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating vector 5: %v\n", err)
		os.Exit(1)
	}
	vectors = append(vectors, vector5)

	// Write vectors to file
	outputPath := "../cose-vectors/go-cose-vectors.json"
	os.MkdirAll("../cose-vectors", 0755)

	file, err := os.Create(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(vectors); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %d test vectors\n", len(vectors))
	fmt.Printf("Output: %s\n", outputPath)
}

func generateVector(payload []byte, protectedHeaders map[string]interface{}, description string) (TestVector, error) {
	// Generate ES256 key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return TestVector{}, fmt.Errorf("failed to generate key: %w", err)
	}

	// Create signer
	signer, err := cose.NewSigner(cose.AlgorithmES256, privateKey)
	if err != nil {
		return TestVector{}, fmt.Errorf("failed to create signer: %w", err)
	}

	// Build protected headers
	headers := cose.Headers{
		Protected: cose.ProtectedHeader{},
	}

	// Set headers from map
	for key, value := range protectedHeaders {
		switch key {
		case "alg":
			headers.Protected.SetAlgorithm(cose.Algorithm(value.(int64)))
		default:
			// For custom headers like iss, kid, sub, cty
			headers.Protected[key] = value
		}
	}

	// Create COSE_Sign1
	sign1 := &cose.Sign1Message{
		Headers: headers,
		Payload: payload,
	}

	// Sign the message
	err = sign1.Sign(rand.Reader, nil, signer)
	if err != nil {
		return TestVector{}, fmt.Errorf("failed to sign: %w", err)
	}

	// Marshal to CBOR
	coseBytes, err := sign1.MarshalCBOR()
	if err != nil {
		return TestVector{}, fmt.Errorf("failed to marshal: %w", err)
	}

	// Extract public key coordinates
	publicKey := privateKey.PublicKey
	xBytes := publicKey.X.Bytes()
	yBytes := publicKey.Y.Bytes()

	// Pad to 32 bytes if needed
	xPadded := make([]byte, 32)
	yPadded := make([]byte, 32)
	copy(xPadded[32-len(xBytes):], xBytes)
	copy(yPadded[32-len(yBytes):], yBytes)

	return TestVector{
		Payload:          hex.EncodeToString(payload),
		ProtectedHeaders: protectedHeaders,
		CoseSign1Bytes:   hex.EncodeToString(coseBytes),
		PublicKeyX:       hex.EncodeToString(xPadded),
		PublicKeyY:       hex.EncodeToString(yPadded),
		Description:      description,
	}, nil
}
