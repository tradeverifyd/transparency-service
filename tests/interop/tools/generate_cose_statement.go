package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
)

// StatementPayload represents a SCITT statement payload
type StatementPayload struct {
	Type    string `json:"type"`
	Issuer  string `json:"issuer"`
	Subject string `json:"subject"`
	Data    string `json:"data"`
	Size    string `json:"size"`
}

func main() {
	// Parse command line arguments
	size := flag.String("size", "small", "Payload size (small, medium, large)")
	output := flag.String("output", "", "Output file path (default: fixtures/statements/{size}.cose)")
	flag.Parse()

	// Determine output path
	outputPath := *output
	if outputPath == "" {
		fixturesDir := filepath.Join("..", "fixtures", "statements")
		os.MkdirAll(fixturesDir, 0755)
		outputPath = filepath.Join(fixturesDir, fmt.Sprintf("%s.cose", *size))
	}

	// Generate test keypair
	fmt.Println("Generating ES256 keypair...")
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate key: %v\n", err)
		os.Exit(1)
	}

	// Create payload
	fmt.Printf("Creating %s payload...\n", *size)
	payload := createPayload(*size)
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal payload: %v\n", err)
		os.Exit(1)
	}

	// Create COSE Sign1 message
	fmt.Println("Signing with COSE Sign1...")
	coseBytes, err := signCOSE(payloadBytes, privateKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to sign: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	fmt.Printf("Writing to %s...\n", outputPath)
	if err := os.WriteFile(outputPath, coseBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully generated COSE Sign1 statement (%d bytes)\n", len(coseBytes))
	fmt.Printf("  Output: %s\n", outputPath)
	fmt.Printf("  Payload size: %s\n", *size)
	fmt.Printf("  Payload length: %d bytes\n", len(payloadBytes))
}

// createPayload creates a test payload of the specified size
func createPayload(size string) StatementPayload {
	basePayload := StatementPayload{
		Type:    "application/vnd.scitt.statement+json",
		Issuer:  "https://example.com/test-issuer",
		Subject: "test-subject-" + size,
		Size:    size,
	}

	// Adjust data field based on size
	switch size {
	case "small":
		basePayload.Data = "This is a small test payload for SCITT statement testing."
	case "medium":
		// Create medium payload (~1KB)
		basePayload.Data = generateData(1000)
	case "large":
		// Create large payload (~10KB)
		basePayload.Data = generateData(10000)
	default:
		basePayload.Data = "Default test payload"
	}

	return basePayload
}

// generateData creates a data string of approximately the specified length
func generateData(targetLength int) string {
	const chunk = "This is test data for SCITT statement validation. "
	result := ""
	for len(result) < targetLength {
		result += chunk
	}
	return result[:targetLength]
}

// signCOSE creates a COSE Sign1 message with the given payload using the internal COSE package
func signCOSE(payload []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	// Create ES256 signer
	signer, err := cose.NewES256Signer(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	// Create protected headers with algorithm identifier
	protectedHeaders := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
		Alg: cose.AlgorithmES256,
		Cty: "application/json",
	})

	// Create COSE Sign1 structure
	coseSign1, err := cose.CreateCoseSign1(protectedHeaders, payload, signer, cose.CoseSign1Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create COSE Sign1: %w", err)
	}

	// Encode to CBOR
	coseBytes, err := cose.EncodeCoseSign1(coseSign1)
	if err != nil {
		return nil, fmt.Errorf("failed to encode COSE Sign1: %w", err)
	}

	return coseBytes, nil
}
