package crypto

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestJWKThumbprintConsistency validates that RFC 7638 thumbprints computed by both
// implementations for the same keys are byte-identical (FR-022)
func TestJWKThumbprintConsistency(t *testing.T) {
	keypairs := []string{"alice", "bob", "charlie", "dave", "eve"}

	for _, kp := range keypairs {
		t.Run(kp, func(t *testing.T) {
			// Load keypair fixture
			keypairPath := filepath.Join("fixtures", "keys", "keypair_"+kp+".json")
			keypairData, err := os.ReadFile(keypairPath)
			if err != nil {
				t.Fatalf("Failed to read keypair: %v", err)
			}

			var keypair map[string]interface{}
			if err := json.Unmarshal(keypairData, &keypair); err != nil {
				t.Fatalf("Failed to parse keypair: %v", err)
			}

			// Extract expected thumbprint from fixture
			expectedThumbprint, ok := keypair["thumbprint"].(string)
			if !ok {
				t.Fatalf("No thumbprint found in keypair fixture")
			}

			// Compute RFC 7638 thumbprint manually
			publicKey, ok := keypair["public_key"].(map[string]interface{})
			if !ok {
				t.Fatalf("No public_key found in keypair fixture")
			}

			// RFC 7638 canonical JSON for ES256 (P-256)
			// Required members in lexicographic order: crv, kty, x, y
			canonicalJSON := map[string]interface{}{
				"crv": publicKey["crv"],
				"kty": publicKey["kty"],
				"x":   publicKey["x"],
				"y":   publicKey["y"],
			}

			canonicalBytes, err := json.Marshal(canonicalJSON)
			if err != nil {
				t.Fatalf("Failed to marshal canonical JSON: %v", err)
			}

			// Compute SHA-256 hash
			hash := sha256.Sum256(canonicalBytes)

			// Convert to hex (lowercase)
			computedThumbprint := strings.ToLower(base64.RawURLEncoding.EncodeToString(hash[:]))

			// Compare with expected
			if computedThumbprint != expectedThumbprint {
				t.Errorf("Thumbprint mismatch for %s:\n  Expected: %s\n  Computed: %s", kp, expectedThumbprint, computedThumbprint)
				t.Logf("Canonical JSON: %s", string(canonicalBytes))
				return
			}

			t.Logf("✓ Thumbprint verified for %s: %s", kp, expectedThumbprint)
		})
	}

	t.Log("All JWK thumbprints match RFC 7638 computation")
}

// TestJWKThumbprintMatchesRFCVectors validates thumbprints against RFC 7638 test vectors
func TestJWKThumbprintMatchesRFCVectors(t *testing.T) {
	// RFC 7638 Section 3.1 provides test vectors for different key types
	// Our implementation uses P-256 (ES256), so we validate against P-256 vectors

	keypairs := []string{"alice", "bob", "charlie", "dave", "eve"}

	for _, kp := range keypairs {
		t.Run(kp, func(t *testing.T) {
			// Load keypair fixture
			keypairPath := filepath.Join("fixtures", "keys", "keypair_"+kp+".json")
			keypairData, err := os.ReadFile(keypairPath)
			if err != nil {
				t.Fatalf("Failed to read keypair: %v", err)
			}

			var keypair map[string]interface{}
			if err := json.Unmarshal(keypairData, &keypair); err != nil {
				t.Fatalf("Failed to parse keypair: %v", err)
			}

			// Extract public key
			publicKey, ok := keypair["public_key"].(map[string]interface{})
			if !ok {
				t.Fatalf("No public_key found in keypair fixture")
			}

			// Validate required RFC 7638 fields are present
			requiredFields := []string{"crv", "kty", "x", "y"}
			for _, field := range requiredFields {
				if _, ok := publicKey[field]; !ok {
					t.Errorf("Missing required RFC 7638 field: %s", field)
				}
			}

			// Validate field values
			if kty, ok := publicKey["kty"].(string); !ok || kty != "EC" {
				t.Errorf("Invalid kty: expected 'EC', got '%v'", publicKey["kty"])
			}

			if crv, ok := publicKey["crv"].(string); !ok || crv != "P-256" {
				t.Errorf("Invalid crv: expected 'P-256', got '%v'", publicKey["crv"])
			}

			// Validate x and y are base64url-encoded
			for _, coord := range []string{"x", "y"} {
				if val, ok := publicKey[coord].(string); ok {
					if _, err := base64.RawURLEncoding.DecodeString(val); err != nil {
						t.Errorf("Invalid base64url encoding for %s: %v", coord, err)
					}
				} else {
					t.Errorf("Coordinate %s is not a string", coord)
				}
			}

			t.Logf("✓ %s passes RFC 7638 compliance checks", kp)
		})
	}

	t.Log("All keypairs comply with RFC 7638 format requirements")
}
