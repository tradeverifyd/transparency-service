package cose_test

import (
	"bytes"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
)

func TestCreateCWTClaims(t *testing.T) {
	t.Run("creates claims with all fields", func(t *testing.T) {
		opts := cose.CWTClaimsOptions{
			Iss:   "https://example.com",
			Sub:   "user@example.com",
			Aud:   "https://audience.com",
			Exp:   1234567890,
			Nbf:   1234567800,
			Iat:   1234567800,
			Cti:   []byte("unique-id"),
			Scope: "read write",
			Nonce: []byte("random-nonce"),
		}

		claims := cose.CreateCWTClaims(opts)

		if claims[cose.CWTClaimIss] != opts.Iss {
			t.Errorf("expected iss=%s, got %v", opts.Iss, claims[cose.CWTClaimIss])
		}
		if claims[cose.CWTClaimSub] != opts.Sub {
			t.Errorf("expected sub=%s, got %v", opts.Sub, claims[cose.CWTClaimSub])
		}
	})

	t.Run("creates claims with optional fields", func(t *testing.T) {
		opts := cose.CWTClaimsOptions{
			Iss: "https://example.com",
		}

		claims := cose.CreateCWTClaims(opts)

		if claims[cose.CWTClaimIss] != opts.Iss {
			t.Errorf("expected iss=%s, got %v", opts.Iss, claims[cose.CWTClaimIss])
		}
		if _, exists := claims[cose.CWTClaimSub]; exists {
			t.Error("sub should not be present")
		}
	})
}

func TestCreateProtectedHeaders(t *testing.T) {
	t.Run("creates headers with required fields", func(t *testing.T) {
		opts := cose.ProtectedHeadersOptions{
			Alg: cose.AlgorithmES256,
			Kid: "test-key-1",
		}

		headers := cose.CreateProtectedHeaders(opts)

		if headers[cose.HeaderLabelAlg] != cose.AlgorithmES256 {
			t.Errorf("expected alg=%d, got %v", cose.AlgorithmES256, headers[cose.HeaderLabelAlg])
		}
		if headers[cose.HeaderLabelKid] != "test-key-1" {
			t.Errorf("expected kid=test-key-1, got %v", headers[cose.HeaderLabelKid])
		}
	})

	t.Run("creates headers with CWT claims", func(t *testing.T) {
		cwtClaims := cose.CreateCWTClaims(cose.CWTClaimsOptions{
			Iss: "https://example.com",
			Sub: "user@example.com",
		})

		opts := cose.ProtectedHeadersOptions{
			Alg:       cose.AlgorithmES256,
			CWTClaims: cwtClaims,
		}

		headers := cose.CreateProtectedHeaders(opts)

		if _, exists := headers[cose.HeaderLabelCWTClaims]; !exists {
			t.Error("CWT claims should be present")
		}
	})
}

func TestCreateCoseSign1(t *testing.T) {
	// Generate test key pair
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	signer, err := cose.NewES256Signer(keyPair.Private)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	t.Run("creates valid COSE Sign1", func(t *testing.T) {
		headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
			Alg: cose.AlgorithmES256,
			Kid: "test-key-1",
		})

		payload := []byte("Hello, World!")

		coseSign1, err := cose.CreateCoseSign1(headers, payload, signer, cose.CoseSign1Options{})
		if err != nil {
			t.Fatalf("failed to create COSE Sign1: %v", err)
		}

		if coseSign1 == nil {
			t.Fatal("COSE Sign1 is nil")
		}
		if len(coseSign1.Protected) == 0 {
			t.Error("protected headers are empty")
		}
		if len(coseSign1.Signature) == 0 {
			t.Error("signature is empty")
		}
		if !bytes.Equal(coseSign1.Payload, payload) {
			t.Error("payload does not match")
		}
	})

	t.Run("creates detached COSE Sign1", func(t *testing.T) {
		headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
			Alg: cose.AlgorithmES256,
		})

		payload := []byte("Detached payload")

		coseSign1, err := cose.CreateCoseSign1(
			headers,
			payload,
			signer,
			cose.CoseSign1Options{Detached: true},
		)
		if err != nil {
			t.Fatalf("failed to create detached COSE Sign1: %v", err)
		}

		if coseSign1.Payload != nil {
			t.Error("payload should be nil for detached mode")
		}
	})

	t.Run("signature is different for different payloads", func(t *testing.T) {
		headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
			Alg: cose.AlgorithmES256,
		})

		payload1 := []byte("Payload 1")
		payload2 := []byte("Payload 2")

		coseSign1_1, err := cose.CreateCoseSign1(headers, payload1, signer, cose.CoseSign1Options{})
		if err != nil {
			t.Fatalf("failed to create COSE Sign1 (1): %v", err)
		}

		coseSign1_2, err := cose.CreateCoseSign1(headers, payload2, signer, cose.CoseSign1Options{})
		if err != nil {
			t.Fatalf("failed to create COSE Sign1 (2): %v", err)
		}

		if bytes.Equal(coseSign1_1.Signature, coseSign1_2.Signature) {
			t.Error("signatures should be different for different payloads")
		}
	})
}

func TestVerifyCoseSign1(t *testing.T) {
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

	t.Run("verifies valid signature", func(t *testing.T) {
		headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
			Alg: cose.AlgorithmES256,
		})

		payload := []byte("Test message")

		coseSign1, err := cose.CreateCoseSign1(headers, payload, signer, cose.CoseSign1Options{})
		if err != nil {
			t.Fatalf("failed to create COSE Sign1: %v", err)
		}

		valid, err := cose.VerifyCoseSign1(coseSign1, verifier, nil)
		if err != nil {
			t.Fatalf("verification failed: %v", err)
		}

		if !valid {
			t.Error("signature should be valid")
		}
	})

	t.Run("verifies detached signature", func(t *testing.T) {
		headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
			Alg: cose.AlgorithmES256,
		})

		payload := []byte("Detached message")

		coseSign1, err := cose.CreateCoseSign1(
			headers,
			payload,
			signer,
			cose.CoseSign1Options{Detached: true},
		)
		if err != nil {
			t.Fatalf("failed to create COSE Sign1: %v", err)
		}

		valid, err := cose.VerifyCoseSign1(coseSign1, verifier, payload)
		if err != nil {
			t.Fatalf("verification failed: %v", err)
		}

		if !valid {
			t.Error("detached signature should be valid")
		}
	})

	t.Run("rejects tampered payload", func(t *testing.T) {
		headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
			Alg: cose.AlgorithmES256,
		})

		payload := []byte("Original message")

		coseSign1, err := cose.CreateCoseSign1(headers, payload, signer, cose.CoseSign1Options{})
		if err != nil {
			t.Fatalf("failed to create COSE Sign1: %v", err)
		}

		// Tamper with payload
		coseSign1.Payload = []byte("Tampered message")

		valid, err := cose.VerifyCoseSign1(coseSign1, verifier, nil)
		if err != nil {
			t.Fatalf("verification failed: %v", err)
		}

		if valid {
			t.Error("tampered payload should not verify")
		}
	})

	t.Run("rejects wrong external payload", func(t *testing.T) {
		headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
			Alg: cose.AlgorithmES256,
		})

		payload := []byte("Original payload")

		coseSign1, err := cose.CreateCoseSign1(
			headers,
			payload,
			signer,
			cose.CoseSign1Options{Detached: true},
		)
		if err != nil {
			t.Fatalf("failed to create COSE Sign1: %v", err)
		}

		wrongPayload := []byte("Wrong payload")
		valid, err := cose.VerifyCoseSign1(coseSign1, verifier, wrongPayload)
		if err != nil {
			t.Fatalf("verification failed: %v", err)
		}

		if valid {
			t.Error("wrong external payload should not verify")
		}
	})

	t.Run("rejects missing external payload", func(t *testing.T) {
		headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
			Alg: cose.AlgorithmES256,
		})

		payload := []byte("Detached payload")

		coseSign1, err := cose.CreateCoseSign1(
			headers,
			payload,
			signer,
			cose.CoseSign1Options{Detached: true},
		)
		if err != nil {
			t.Fatalf("failed to create COSE Sign1: %v", err)
		}

		_, err = cose.VerifyCoseSign1(coseSign1, verifier, nil)
		if err == nil {
			t.Error("should error on missing external payload")
		}
	})
}

func TestGetProtectedHeaders(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	signer, err := cose.NewES256Signer(keyPair.Private)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	t.Run("retrieves protected headers", func(t *testing.T) {
		headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
			Alg: cose.AlgorithmES256,
			Kid: "test-key-1",
			Cty: "application/json",
		})

		payload := []byte("Test")

		coseSign1, err := cose.CreateCoseSign1(headers, payload, signer, cose.CoseSign1Options{})
		if err != nil {
			t.Fatalf("failed to create COSE Sign1: %v", err)
		}

		retrievedHeaders, err := cose.GetProtectedHeaders(coseSign1)
		if err != nil {
			t.Fatalf("failed to get protected headers: %v", err)
		}

		// CBOR uses uint64 for positive integer keys
		algKey := uint64(cose.HeaderLabelAlg)
		if algVal, ok := retrievedHeaders[algKey]; ok {
			if int(algVal.(int64)) != cose.AlgorithmES256 {
				t.Errorf("expected alg=%d, got %v", cose.AlgorithmES256, algVal)
			}
		} else {
			t.Errorf("alg header not found, available keys: %v", retrievedHeaders)
		}

		kidKey := uint64(cose.HeaderLabelKid)
		if kidVal, ok := retrievedHeaders[kidKey]; ok {
			if kidVal.(string) != "test-key-1" {
				t.Errorf("expected kid=test-key-1, got %v", kidVal)
			}
		} else {
			t.Error("kid header not found")
		}
	})
}

func TestEncodeDecode(t *testing.T) {
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

	t.Run("encode and decode COSE Sign1", func(t *testing.T) {
		headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
			Alg: cose.AlgorithmES256,
			Kid: "test-key-1",
		})

		payload := []byte("Test message for encoding")

		coseSign1, err := cose.CreateCoseSign1(headers, payload, signer, cose.CoseSign1Options{})
		if err != nil {
			t.Fatalf("failed to create COSE Sign1: %v", err)
		}

		// Encode
		encoded, err := cose.EncodeCoseSign1(coseSign1)
		if err != nil {
			t.Fatalf("failed to encode COSE Sign1: %v", err)
		}

		if len(encoded) == 0 {
			t.Error("encoded data is empty")
		}

		// Decode
		decoded, err := cose.DecodeCoseSign1(encoded)
		if err != nil {
			t.Fatalf("failed to decode COSE Sign1: %v", err)
		}

		// Verify decoded signature
		valid, err := cose.VerifyCoseSign1(decoded, verifier, nil)
		if err != nil {
			t.Fatalf("verification failed: %v", err)
		}

		if !valid {
			t.Error("decoded signature should be valid")
		}

		// Check payload
		if !bytes.Equal(decoded.Payload, payload) {
			t.Error("decoded payload does not match original")
		}
	})

	t.Run("round-trip preserves structure", func(t *testing.T) {
		headers := cose.CreateProtectedHeaders(cose.ProtectedHeadersOptions{
			Alg: cose.AlgorithmES256,
		})

		payload := []byte("Round-trip test")

		original, err := cose.CreateCoseSign1(headers, payload, signer, cose.CoseSign1Options{})
		if err != nil {
			t.Fatalf("failed to create COSE Sign1: %v", err)
		}

		encoded, err := cose.EncodeCoseSign1(original)
		if err != nil {
			t.Fatalf("failed to encode: %v", err)
		}

		decoded, err := cose.DecodeCoseSign1(encoded)
		if err != nil {
			t.Fatalf("failed to decode: %v", err)
		}

		if !bytes.Equal(original.Protected, decoded.Protected) {
			t.Error("protected headers do not match after round-trip")
		}
		if !bytes.Equal(original.Payload, decoded.Payload) {
			t.Error("payload does not match after round-trip")
		}
		if !bytes.Equal(original.Signature, decoded.Signature) {
			t.Error("signature does not match after round-trip")
		}
	})
}
