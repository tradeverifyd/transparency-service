package cose_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
)

func TestCreateHashEnvelope(t *testing.T) {
	t.Run("creates hash envelope with default algorithm", func(t *testing.T) {
		data := []byte("Hello, World!")
		opts := cose.HashEnvelopeOptions{}

		envelope, err := cose.CreateHashEnvelope(data, opts)
		if err != nil {
			t.Fatalf("failed to create hash envelope: %v", err)
		}

		if envelope == nil {
			t.Fatal("envelope is nil")
		}
		if len(envelope.PayloadHash) == 0 {
			t.Error("payload hash is empty")
		}
		if envelope.PayloadHashAlg != cose.HashAlgorithmSHA256 {
			t.Errorf("expected SHA-256 algorithm (%d), got %d", cose.HashAlgorithmSHA256, envelope.PayloadHashAlg)
		}
	})

	t.Run("creates hash envelope with specified algorithm", func(t *testing.T) {
		data := []byte("Test data")
		opts := cose.HashEnvelopeOptions{
			HashAlgorithm: cose.HashAlgorithmSHA384,
		}

		envelope, err := cose.CreateHashEnvelope(data, opts)
		if err != nil {
			t.Fatalf("failed to create hash envelope: %v", err)
		}

		if envelope.PayloadHashAlg != cose.HashAlgorithmSHA384 {
			t.Errorf("expected SHA-384 algorithm, got %d", envelope.PayloadHashAlg)
		}
	})

	t.Run("creates hash envelope with metadata", func(t *testing.T) {
		data := []byte("Test data")
		opts := cose.HashEnvelopeOptions{
			ContentType: "application/json",
			Location:    "https://example.com/artifact.json",
		}

		envelope, err := cose.CreateHashEnvelope(data, opts)
		if err != nil {
			t.Fatalf("failed to create hash envelope: %v", err)
		}

		if envelope.PreimageContentType != opts.ContentType {
			t.Errorf("expected content type %s, got %s", opts.ContentType, envelope.PreimageContentType)
		}
		if envelope.PayloadLocation != opts.Location {
			t.Errorf("expected location %s, got %s", opts.Location, envelope.PayloadLocation)
		}
	})

	t.Run("produces consistent hashes", func(t *testing.T) {
		data := []byte("Consistent data")
		opts := cose.HashEnvelopeOptions{}

		envelope1, err := cose.CreateHashEnvelope(data, opts)
		if err != nil {
			t.Fatalf("failed to create envelope 1: %v", err)
		}

		envelope2, err := cose.CreateHashEnvelope(data, opts)
		if err != nil {
			t.Fatalf("failed to create envelope 2: %v", err)
		}

		if !bytes.Equal(envelope1.PayloadHash, envelope2.PayloadHash) {
			t.Error("hashes should be consistent for same data")
		}
	})

	t.Run("produces different hashes for different data", func(t *testing.T) {
		data1 := []byte("Data 1")
		data2 := []byte("Data 2")
		opts := cose.HashEnvelopeOptions{}

		envelope1, err := cose.CreateHashEnvelope(data1, opts)
		if err != nil {
			t.Fatalf("failed to create envelope 1: %v", err)
		}

		envelope2, err := cose.CreateHashEnvelope(data2, opts)
		if err != nil {
			t.Fatalf("failed to create envelope 2: %v", err)
		}

		if bytes.Equal(envelope1.PayloadHash, envelope2.PayloadHash) {
			t.Error("hashes should be different for different data")
		}
	})
}

func TestHashData(t *testing.T) {
	t.Run("hashes data with SHA-256", func(t *testing.T) {
		data := []byte("test")
		hash, err := cose.HashData(data, cose.HashAlgorithmSHA256)
		if err != nil {
			t.Fatalf("failed to hash data: %v", err)
		}

		if len(hash) != 32 {
			t.Errorf("expected 32 bytes for SHA-256, got %d", len(hash))
		}
	})

	t.Run("hashes data with SHA-384", func(t *testing.T) {
		data := []byte("test")
		hash, err := cose.HashData(data, cose.HashAlgorithmSHA384)
		if err != nil {
			t.Fatalf("failed to hash data: %v", err)
		}

		if len(hash) != 48 {
			t.Errorf("expected 48 bytes for SHA-384, got %d", len(hash))
		}
	})

	t.Run("hashes data with SHA-512", func(t *testing.T) {
		data := []byte("test")
		hash, err := cose.HashData(data, cose.HashAlgorithmSHA512)
		if err != nil {
			t.Fatalf("failed to hash data: %v", err)
		}

		if len(hash) != 64 {
			t.Errorf("expected 64 bytes for SHA-512, got %d", len(hash))
		}
	})

	t.Run("rejects unsupported algorithm", func(t *testing.T) {
		data := []byte("test")
		_, err := cose.HashData(data, 9999)
		if err == nil {
			t.Error("expected error for unsupported algorithm")
		}
	})
}

func TestStreamHashFromFile(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "cose-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("hashes file content", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tmpDir, "test.txt")
		testData := []byte("File content for hashing")
		if err := os.WriteFile(testFile, testData, 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		// Hash file
		fileHash, err := cose.StreamHashFromFile(testFile, cose.HashAlgorithmSHA256)
		if err != nil {
			t.Fatalf("failed to hash file: %v", err)
		}

		// Hash data directly
		dataHash, err := cose.HashData(testData, cose.HashAlgorithmSHA256)
		if err != nil {
			t.Fatalf("failed to hash data: %v", err)
		}

		// Hashes should match
		if !bytes.Equal(fileHash, dataHash) {
			t.Error("file hash and data hash should match")
		}
	})

	t.Run("handles large files", func(t *testing.T) {
		// Create large test file (1MB)
		testFile := filepath.Join(tmpDir, "large.bin")
		largeData := make([]byte, 1024*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}
		if err := os.WriteFile(testFile, largeData, 0644); err != nil {
			t.Fatalf("failed to write large file: %v", err)
		}

		// Hash file
		fileHash, err := cose.StreamHashFromFile(testFile, cose.HashAlgorithmSHA256)
		if err != nil {
			t.Fatalf("failed to hash large file: %v", err)
		}

		if len(fileHash) != 32 {
			t.Errorf("expected 32 bytes, got %d", len(fileHash))
		}
	})

	t.Run("rejects non-existent file", func(t *testing.T) {
		_, err := cose.StreamHashFromFile("/non/existent/file.txt", cose.HashAlgorithmSHA256)
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})
}

func TestValidateHashEnvelope(t *testing.T) {
	t.Run("validates correct hash", func(t *testing.T) {
		data := []byte("Test data")
		opts := cose.HashEnvelopeOptions{}

		envelope, err := cose.CreateHashEnvelope(data, opts)
		if err != nil {
			t.Fatalf("failed to create envelope: %v", err)
		}

		valid, err := cose.ValidateHashEnvelope(envelope, data)
		if err != nil {
			t.Fatalf("validation failed: %v", err)
		}

		if !valid {
			t.Error("hash should be valid")
		}
	})

	t.Run("rejects incorrect data", func(t *testing.T) {
		originalData := []byte("Original data")
		tamperedData := []byte("Tampered data")
		opts := cose.HashEnvelopeOptions{}

		envelope, err := cose.CreateHashEnvelope(originalData, opts)
		if err != nil {
			t.Fatalf("failed to create envelope: %v", err)
		}

		valid, err := cose.ValidateHashEnvelope(envelope, tamperedData)
		if err != nil {
			t.Fatalf("validation failed: %v", err)
		}

		if valid {
			t.Error("tampered data should not validate")
		}
	})
}

func TestSignHashEnvelope(t *testing.T) {
	// Generate test key pair
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	signer, err := cose.NewES256Signer(keyPair.Private)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	t.Run("signs hash envelope", func(t *testing.T) {
		artifact := []byte("Artifact to sign")
		opts := cose.HashEnvelopeOptions{
			ContentType:   "application/octet-stream",
			Location:      "https://example.com/artifact.bin",
			HashAlgorithm: cose.HashAlgorithmSHA256,
		}

		coseSign1, err := cose.SignHashEnvelope(artifact, opts, signer, nil, false)
		if err != nil {
			t.Fatalf("failed to sign hash envelope: %v", err)
		}

		if coseSign1 == nil {
			t.Fatal("COSE Sign1 is nil")
		}
		if len(coseSign1.Signature) == 0 {
			t.Error("signature is empty")
		}
		if len(coseSign1.Payload) == 0 {
			t.Error("payload (hash) is empty")
		}
	})

	t.Run("signs with CWT claims", func(t *testing.T) {
		artifact := []byte("Artifact with claims")
		opts := cose.HashEnvelopeOptions{}
		cwtClaims := cose.CreateCWTClaims(cose.CWTClaimsOptions{
			Iss: "https://issuer.example.com",
			Sub: "artifact-123",
		})

		coseSign1, err := cose.SignHashEnvelope(artifact, opts, signer, cwtClaims, false)
		if err != nil {
			t.Fatalf("failed to sign with claims: %v", err)
		}

		// Verify claims are in protected headers
		headers, err := cose.GetProtectedHeaders(coseSign1)
		if err != nil {
			t.Fatalf("failed to get headers: %v", err)
		}

		if _, ok := headers[uint64(cose.HeaderLabelCWTClaims)]; !ok {
			t.Error("CWT claims should be in protected headers")
		}
	})

	t.Run("creates detached signature", func(t *testing.T) {
		artifact := []byte("Detached artifact")
		opts := cose.HashEnvelopeOptions{}

		coseSign1, err := cose.SignHashEnvelope(artifact, opts, signer, nil, true)
		if err != nil {
			t.Fatalf("failed to create detached signature: %v", err)
		}

		if coseSign1.Payload != nil {
			t.Error("detached signature should have nil payload")
		}
	})
}

func TestVerifyHashEnvelope(t *testing.T) {
	// Generate test key pair
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	signer, err := cose.NewES256Signer(keyPair.Private)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	verifier, err := cose.NewES256Verifier(keyPair.Public)
	if err != nil {
		t.Fatalf("failed to create verifier: %v", err)
	}

	t.Run("verifies valid hash envelope", func(t *testing.T) {
		artifact := []byte("Valid artifact")
		opts := cose.HashEnvelopeOptions{}

		coseSign1, err := cose.SignHashEnvelope(artifact, opts, signer, nil, false)
		if err != nil {
			t.Fatalf("failed to sign: %v", err)
		}

		result, err := cose.VerifyHashEnvelope(coseSign1, artifact, verifier)
		if err != nil {
			t.Fatalf("verification failed: %v", err)
		}

		if !result.SignatureValid {
			t.Error("signature should be valid")
		}
		if !result.HashValid {
			t.Error("hash should be valid")
		}
	})

	t.Run("detects tampered artifact", func(t *testing.T) {
		originalArtifact := []byte("Original artifact")
		tamperedArtifact := []byte("Tampered artifact")
		opts := cose.HashEnvelopeOptions{}

		coseSign1, err := cose.SignHashEnvelope(originalArtifact, opts, signer, nil, false)
		if err != nil {
			t.Fatalf("failed to sign: %v", err)
		}

		result, err := cose.VerifyHashEnvelope(coseSign1, tamperedArtifact, verifier)
		if err != nil {
			t.Fatalf("verification failed: %v", err)
		}

		if !result.SignatureValid {
			t.Error("signature should still be valid")
		}
		if result.HashValid {
			t.Error("hash should NOT be valid for tampered artifact")
		}
	})

	t.Run("detects tampered signature", func(t *testing.T) {
		artifact := []byte("Test artifact")
		opts := cose.HashEnvelopeOptions{}

		coseSign1, err := cose.SignHashEnvelope(artifact, opts, signer, nil, false)
		if err != nil {
			t.Fatalf("failed to sign: %v", err)
		}

		// Tamper with signature
		coseSign1.Signature[0] ^= 0xFF

		result, err := cose.VerifyHashEnvelope(coseSign1, artifact, verifier)
		if err != nil {
			t.Fatalf("verification failed: %v", err)
		}

		if result.SignatureValid {
			t.Error("tampered signature should NOT be valid")
		}
	})
}

func TestExtractHashEnvelopeParams(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	signer, err := cose.NewES256Signer(keyPair.Private)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	t.Run("extracts parameters from COSE Sign1", func(t *testing.T) {
		artifact := []byte("Test artifact")
		opts := cose.HashEnvelopeOptions{
			ContentType:   "text/plain",
			Location:      "https://example.com/file.txt",
			HashAlgorithm: cose.HashAlgorithmSHA256,
		}

		coseSign1, err := cose.SignHashEnvelope(artifact, opts, signer, nil, false)
		if err != nil {
			t.Fatalf("failed to sign: %v", err)
		}

		params, err := cose.ExtractHashEnvelopeParams(coseSign1)
		if err != nil {
			t.Fatalf("failed to extract params: %v", err)
		}

		if params.PayloadHashAlg != opts.HashAlgorithm {
			t.Errorf("expected hash alg %d, got %d", opts.HashAlgorithm, params.PayloadHashAlg)
		}
		if params.PreimageContentType != opts.ContentType {
			t.Errorf("expected content type %s, got %s", opts.ContentType, params.PreimageContentType)
		}
		if params.PayloadLocation != opts.Location {
			t.Errorf("expected location %s, got %s", opts.Location, params.PayloadLocation)
		}
	})

	t.Run("handles missing optional parameters", func(t *testing.T) {
		artifact := []byte("Test artifact")
		opts := cose.HashEnvelopeOptions{
			HashAlgorithm: cose.HashAlgorithmSHA256,
		}

		coseSign1, err := cose.SignHashEnvelope(artifact, opts, signer, nil, false)
		if err != nil {
			t.Fatalf("failed to sign: %v", err)
		}

		params, err := cose.ExtractHashEnvelopeParams(coseSign1)
		if err != nil {
			t.Fatalf("failed to extract params: %v", err)
		}

		if params.PreimageContentType != "" {
			t.Error("content type should be empty")
		}
		if params.PayloadLocation != "" {
			t.Error("location should be empty")
		}
	})
}
