package cose_test

import (
	"crypto/elliptic"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tradeverifyd/transparency-service/scitt-golang/pkg/cose"
)

func TestGenerateES256KeyPair(t *testing.T) {
	t.Run("generates valid key pair", func(t *testing.T) {
		keyPair, err := cose.GenerateES256KeyPair()
		if err != nil {
			t.Fatalf("failed to generate key pair: %v", err)
		}

		if keyPair == nil {
			t.Fatal("key pair is nil")
		}
		if keyPair.Private == nil {
			t.Fatal("private key is nil")
		}
		if keyPair.Public == nil {
			t.Fatal("public key is nil")
		}
	})

	t.Run("generates P-256 curve", func(t *testing.T) {
		keyPair, err := cose.GenerateES256KeyPair()
		if err != nil {
			t.Fatalf("failed to generate key pair: %v", err)
		}

		if keyPair.Private.Curve != elliptic.P256() {
			t.Errorf("expected P-256 curve, got %v", keyPair.Private.Curve)
		}
		if keyPair.Public.Curve != elliptic.P256() {
			t.Errorf("expected P-256 curve, got %v", keyPair.Public.Curve)
		}
	})

	t.Run("generates different keys each time", func(t *testing.T) {
		keyPair1, err := cose.GenerateES256KeyPair()
		if err != nil {
			t.Fatalf("failed to generate key pair 1: %v", err)
		}

		keyPair2, err := cose.GenerateES256KeyPair()
		if err != nil {
			t.Fatalf("failed to generate key pair 2: %v", err)
		}

		// Check that private keys are different
		if keyPair1.Private.D.Cmp(keyPair2.Private.D) == 0 {
			t.Error("generated identical private keys")
		}
	})
}

func TestExportPublicKeyToJWK(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	t.Run("exports valid JWK", func(t *testing.T) {
		jwk, err := cose.ExportPublicKeyToJWK(keyPair.Public)
		if err != nil {
			t.Fatalf("failed to export to JWK: %v", err)
		}

		if jwk.Kty != "EC" {
			t.Errorf("expected kty=EC, got %s", jwk.Kty)
		}
		if jwk.Crv != "P-256" {
			t.Errorf("expected crv=P-256, got %s", jwk.Crv)
		}
		if jwk.X == "" {
			t.Error("x coordinate is empty")
		}
		if jwk.Y == "" {
			t.Error("y coordinate is empty")
		}
		if jwk.D != "" {
			t.Error("d parameter should not be present in public key")
		}
	})

	t.Run("exports base64url-encoded coordinates", func(t *testing.T) {
		jwk, err := cose.ExportPublicKeyToJWK(keyPair.Public)
		if err != nil {
			t.Fatalf("failed to export to JWK: %v", err)
		}

		// Base64url should not contain + or /
		if strings.Contains(jwk.X, "+") || strings.Contains(jwk.X, "/") {
			t.Error("x coordinate is not base64url-encoded")
		}
		if strings.Contains(jwk.Y, "+") || strings.Contains(jwk.Y, "/") {
			t.Error("y coordinate is not base64url-encoded")
		}
	})

	t.Run("rejects nil public key", func(t *testing.T) {
		_, err := cose.ExportPublicKeyToJWK(nil)
		if err == nil {
			t.Error("expected error for nil public key")
		}
	})
}

func TestExportPrivateKeyToPEM(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	t.Run("exports valid PEM", func(t *testing.T) {
		pem, err := cose.ExportPrivateKeyToPEM(keyPair.Private)
		if err != nil {
			t.Fatalf("failed to export to PEM: %v", err)
		}

		if !strings.Contains(pem, "BEGIN PRIVATE KEY") {
			t.Error("PEM does not contain BEGIN PRIVATE KEY header")
		}
		if !strings.Contains(pem, "END PRIVATE KEY") {
			t.Error("PEM does not contain END PRIVATE KEY footer")
		}
	})

	t.Run("rejects nil private key", func(t *testing.T) {
		_, err := cose.ExportPrivateKeyToPEM(nil)
		if err == nil {
			t.Error("expected error for nil private key")
		}
	})
}

func TestExportPublicKeyToPEM(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	t.Run("exports valid PEM", func(t *testing.T) {
		pem, err := cose.ExportPublicKeyToPEM(keyPair.Public)
		if err != nil {
			t.Fatalf("failed to export to PEM: %v", err)
		}

		if !strings.Contains(pem, "BEGIN PUBLIC KEY") {
			t.Error("PEM does not contain BEGIN PUBLIC KEY header")
		}
		if !strings.Contains(pem, "END PUBLIC KEY") {
			t.Error("PEM does not contain END PUBLIC KEY footer")
		}
	})

	t.Run("rejects nil public key", func(t *testing.T) {
		_, err := cose.ExportPublicKeyToPEM(nil)
		if err == nil {
			t.Error("expected error for nil public key")
		}
	})
}

func TestImportPrivateKeyFromPEM(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	pem, err := cose.ExportPrivateKeyToPEM(keyPair.Private)
	if err != nil {
		t.Fatalf("failed to export to PEM: %v", err)
	}

	t.Run("imports valid PEM", func(t *testing.T) {
		imported, err := cose.ImportPrivateKeyFromPEM(pem)
		if err != nil {
			t.Fatalf("failed to import from PEM: %v", err)
		}

		if imported.Curve != elliptic.P256() {
			t.Errorf("expected P-256 curve, got %v", imported.Curve)
		}

		// Check that the private key matches
		if imported.D.Cmp(keyPair.Private.D) != 0 {
			t.Error("imported private key does not match original")
		}
	})

	t.Run("rejects invalid PEM", func(t *testing.T) {
		_, err := cose.ImportPrivateKeyFromPEM("invalid pem data")
		if err == nil {
			t.Error("expected error for invalid PEM")
		}
	})

	t.Run("rejects empty PEM", func(t *testing.T) {
		_, err := cose.ImportPrivateKeyFromPEM("")
		if err == nil {
			t.Error("expected error for empty PEM")
		}
	})
}

func TestImportPublicKeyFromJWK(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	jwk, err := cose.ExportPublicKeyToJWK(keyPair.Public)
	if err != nil {
		t.Fatalf("failed to export to JWK: %v", err)
	}

	t.Run("imports valid JWK", func(t *testing.T) {
		imported, err := cose.ImportPublicKeyFromJWK(jwk)
		if err != nil {
			t.Fatalf("failed to import from JWK: %v", err)
		}

		if imported.Curve != elliptic.P256() {
			t.Errorf("expected P-256 curve, got %v", imported.Curve)
		}

		// Check that coordinates match
		if imported.X.Cmp(keyPair.Public.X) != 0 {
			t.Error("imported X coordinate does not match original")
		}
		if imported.Y.Cmp(keyPair.Public.Y) != 0 {
			t.Error("imported Y coordinate does not match original")
		}
	})

	t.Run("rejects nil JWK", func(t *testing.T) {
		_, err := cose.ImportPublicKeyFromJWK(nil)
		if err == nil {
			t.Error("expected error for nil JWK")
		}
	})

	t.Run("rejects invalid key type", func(t *testing.T) {
		invalidJWK := &cose.JWK{
			Kty: "RSA",
			Crv: "P-256",
			X:   jwk.X,
			Y:   jwk.Y,
		}
		_, err := cose.ImportPublicKeyFromJWK(invalidJWK)
		if err == nil {
			t.Error("expected error for invalid key type")
		}
	})

	t.Run("rejects invalid curve", func(t *testing.T) {
		invalidJWK := &cose.JWK{
			Kty: "EC",
			Crv: "P-384",
			X:   jwk.X,
			Y:   jwk.Y,
		}
		_, err := cose.ImportPublicKeyFromJWK(invalidJWK)
		if err == nil {
			t.Error("expected error for invalid curve")
		}
	})
}

func TestComputeKeyThumbprint(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	jwk, err := cose.ExportPublicKeyToJWK(keyPair.Public)
	if err != nil {
		t.Fatalf("failed to export to JWK: %v", err)
	}

	t.Run("computes thumbprint", func(t *testing.T) {
		thumbprint, err := cose.ComputeKeyThumbprint(jwk)
		if err != nil {
			t.Fatalf("failed to compute thumbprint: %v", err)
		}

		if thumbprint == "" {
			t.Error("thumbprint is empty")
		}

		// Base64url should not contain + or /
		if strings.Contains(thumbprint, "+") || strings.Contains(thumbprint, "/") {
			t.Error("thumbprint is not base64url-encoded")
		}
	})

	t.Run("produces consistent thumbprints", func(t *testing.T) {
		thumbprint1, err := cose.ComputeKeyThumbprint(jwk)
		if err != nil {
			t.Fatalf("failed to compute thumbprint 1: %v", err)
		}

		thumbprint2, err := cose.ComputeKeyThumbprint(jwk)
		if err != nil {
			t.Fatalf("failed to compute thumbprint 2: %v", err)
		}

		if thumbprint1 != thumbprint2 {
			t.Error("thumbprints are not consistent")
		}
	})

	t.Run("rejects nil JWK", func(t *testing.T) {
		_, err := cose.ComputeKeyThumbprint(nil)
		if err == nil {
			t.Error("expected error for nil JWK")
		}
	})
}

func TestComputeCOSEKeyThumbprint(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	t.Run("computes COSE key thumbprint", func(t *testing.T) {
		thumbprint, err := cose.ComputeCOSEKeyThumbprint(keyPair.Public)
		if err != nil {
			t.Fatalf("failed to compute COSE key thumbprint: %v", err)
		}

		if thumbprint == "" {
			t.Error("thumbprint is empty")
		}

		// Should be 64 hex characters (SHA-256 = 32 bytes = 64 hex chars)
		if len(thumbprint) != 64 {
			t.Errorf("expected thumbprint length 64, got %d", len(thumbprint))
		}

		// Should only contain hex characters (0-9, a-f)
		for _, c := range thumbprint {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("thumbprint contains non-hex character: %c", c)
			}
		}
	})

	t.Run("produces consistent COSE key thumbprints", func(t *testing.T) {
		thumbprint1, err := cose.ComputeCOSEKeyThumbprint(keyPair.Public)
		if err != nil {
			t.Fatalf("failed to compute thumbprint 1: %v", err)
		}

		thumbprint2, err := cose.ComputeCOSEKeyThumbprint(keyPair.Public)
		if err != nil {
			t.Fatalf("failed to compute thumbprint 2: %v", err)
		}

		if thumbprint1 != thumbprint2 {
			t.Error("thumbprints are not consistent")
		}
	})

	t.Run("produces different thumbprints for different keys", func(t *testing.T) {
		keyPair2, err := cose.GenerateES256KeyPair()
		if err != nil {
			t.Fatalf("failed to generate second key pair: %v", err)
		}

		thumbprint1, err := cose.ComputeCOSEKeyThumbprint(keyPair.Public)
		if err != nil {
			t.Fatalf("failed to compute thumbprint 1: %v", err)
		}

		thumbprint2, err := cose.ComputeCOSEKeyThumbprint(keyPair2.Public)
		if err != nil {
			t.Fatalf("failed to compute thumbprint 2: %v", err)
		}

		if thumbprint1 == thumbprint2 {
			t.Error("different keys produced identical thumbprints")
		}
	})

	t.Run("rejects nil public key", func(t *testing.T) {
		_, err := cose.ComputeCOSEKeyThumbprint(nil)
		if err == nil {
			t.Error("expected error for nil public key")
		}
	})
}

func TestJWKToCOSEKey(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	jwk, err := cose.ExportPublicKeyToJWK(keyPair.Public)
	if err != nil {
		t.Fatalf("failed to export to JWK: %v", err)
	}
	jwk.Kid = "test-key-1"
	jwk.Alg = "ES256"

	t.Run("converts to COSE key", func(t *testing.T) {
		coseKey, err := cose.JWKToCOSEKey(jwk)
		if err != nil {
			t.Fatalf("failed to convert to COSE key: %v", err)
		}

		if coseKey == nil {
			t.Fatal("COSE key is nil")
		}
	})

	t.Run("rejects nil JWK", func(t *testing.T) {
		_, err := cose.JWKToCOSEKey(nil)
		if err == nil {
			t.Error("expected error for nil JWK")
		}
	})
}

func TestCOSEKeyToJWK(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	jwk, err := cose.ExportPublicKeyToJWK(keyPair.Public)
	if err != nil {
		t.Fatalf("failed to export to JWK: %v", err)
	}
	jwk.Kid = "test-key-2"
	jwk.Alg = "ES256"

	coseKey, err := cose.JWKToCOSEKey(jwk)
	if err != nil {
		t.Fatalf("failed to convert to COSE key: %v", err)
	}

	t.Run("converts from COSE key", func(t *testing.T) {
		converted, err := cose.COSEKeyToJWK(coseKey)
		if err != nil {
			t.Fatalf("failed to convert from COSE key: %v", err)
		}

		if converted.Kty != "EC" {
			t.Errorf("expected kty=EC, got %s", converted.Kty)
		}
		if converted.Crv != "P-256" {
			t.Errorf("expected crv=P-256, got %s", converted.Crv)
		}
		if converted.Kid != jwk.Kid {
			t.Errorf("expected kid=%s, got %s", jwk.Kid, converted.Kid)
		}
		if converted.Alg != jwk.Alg {
			t.Errorf("expected alg=%s, got %s", jwk.Alg, converted.Alg)
		}
	})

	t.Run("rejects nil COSE key", func(t *testing.T) {
		_, err := cose.COSEKeyToJWK(nil)
		if err == nil {
			t.Error("expected error for nil COSE key")
		}
	})
}

func TestRoundTrip(t *testing.T) {
	t.Run("JWK -> COSE -> JWK", func(t *testing.T) {
		keyPair, err := cose.GenerateES256KeyPair()
		if err != nil {
			t.Fatalf("failed to generate key pair: %v", err)
		}

		original, err := cose.ExportPublicKeyToJWK(keyPair.Public)
		if err != nil {
			t.Fatalf("failed to export to JWK: %v", err)
		}
		original.Kid = "test-key-3"
		original.Alg = "ES256"

		// JWK -> COSE
		coseKey, err := cose.JWKToCOSEKey(original)
		if err != nil {
			t.Fatalf("failed to convert to COSE key: %v", err)
		}

		// COSE -> JWK
		converted, err := cose.COSEKeyToJWK(coseKey)
		if err != nil {
			t.Fatalf("failed to convert from COSE key: %v", err)
		}

		// Compare
		if original.Kty != converted.Kty {
			t.Errorf("kty mismatch: %s != %s", original.Kty, converted.Kty)
		}
		if original.Crv != converted.Crv {
			t.Errorf("crv mismatch: %s != %s", original.Crv, converted.Crv)
		}
		if original.X != converted.X {
			t.Errorf("x mismatch: %s != %s", original.X, converted.X)
		}
		if original.Y != converted.Y {
			t.Errorf("y mismatch: %s != %s", original.Y, converted.Y)
		}
		if original.Kid != converted.Kid {
			t.Errorf("kid mismatch: %s != %s", original.Kid, converted.Kid)
		}
		if original.Alg != converted.Alg {
			t.Errorf("alg mismatch: %s != %s", original.Alg, converted.Alg)
		}
	})

	t.Run("PEM -> PrivateKey -> PEM", func(t *testing.T) {
		keyPair, err := cose.GenerateES256KeyPair()
		if err != nil {
			t.Fatalf("failed to generate key pair: %v", err)
		}

		// Export to PEM
		originalPEM, err := cose.ExportPrivateKeyToPEM(keyPair.Private)
		if err != nil {
			t.Fatalf("failed to export to PEM: %v", err)
		}

		// Import from PEM
		imported, err := cose.ImportPrivateKeyFromPEM(originalPEM)
		if err != nil {
			t.Fatalf("failed to import from PEM: %v", err)
		}

		// Export again
		convertedPEM, err := cose.ExportPrivateKeyToPEM(imported)
		if err != nil {
			t.Fatalf("failed to export imported key to PEM: %v", err)
		}

		// Compare (PEM format should be identical)
		if originalPEM != convertedPEM {
			t.Error("PEM formats do not match after round-trip")
		}
	})
}

func TestMarshalUnmarshalJWK(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	jwk, err := cose.ExportPublicKeyToJWK(keyPair.Public)
	if err != nil {
		t.Fatalf("failed to export to JWK: %v", err)
	}
	jwk.Kid = "test-key-4"
	jwk.Alg = "ES256"
	jwk.Use = "sig"

	t.Run("marshals to JSON", func(t *testing.T) {
		data, err := cose.MarshalJWK(jwk)
		if err != nil {
			t.Fatalf("failed to marshal JWK: %v", err)
		}

		// Validate JSON structure
		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("marshaled data is not valid JSON: %v", err)
		}

		if parsed["kty"] != "EC" {
			t.Error("kty not present in JSON")
		}
	})

	t.Run("unmarshals from JSON", func(t *testing.T) {
		data, err := cose.MarshalJWK(jwk)
		if err != nil {
			t.Fatalf("failed to marshal JWK: %v", err)
		}

		unmarshaled, err := cose.UnmarshalJWK(data)
		if err != nil {
			t.Fatalf("failed to unmarshal JWK: %v", err)
		}

		if unmarshaled.Kty != jwk.Kty {
			t.Errorf("kty mismatch: %s != %s", unmarshaled.Kty, jwk.Kty)
		}
		if unmarshaled.Crv != jwk.Crv {
			t.Errorf("crv mismatch: %s != %s", unmarshaled.Crv, jwk.Crv)
		}
		if unmarshaled.Kid != jwk.Kid {
			t.Errorf("kid mismatch: %s != %s", unmarshaled.Kid, jwk.Kid)
		}
	})
}

func TestExportPrivateKeyToCOSECBOR(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	t.Run("exports valid COSE CBOR", func(t *testing.T) {
		cborData, err := cose.ExportPrivateKeyToCOSECBOR(keyPair.Private)
		if err != nil {
			t.Fatalf("failed to export to COSE CBOR: %v", err)
		}

		if len(cborData) == 0 {
			t.Error("CBOR data is empty")
		}

		// CBOR data should start with a map (0xa1-0xbf for definite length maps)
		if cborData[0] < 0xa0 || cborData[0] > 0xbf {
			t.Errorf("CBOR data does not start with a map: 0x%02x", cborData[0])
		}
	})

	t.Run("rejects nil private key", func(t *testing.T) {
		_, err := cose.ExportPrivateKeyToCOSECBOR(nil)
		if err == nil {
			t.Error("expected error for nil private key")
		}
	})

	t.Run("round-trip private key", func(t *testing.T) {
		// Export to CBOR
		cborData, err := cose.ExportPrivateKeyToCOSECBOR(keyPair.Private)
		if err != nil {
			t.Fatalf("failed to export to COSE CBOR: %v", err)
		}

		// Import from CBOR
		imported, err := cose.ImportPrivateKeyFromCOSECBOR(cborData)
		if err != nil {
			t.Fatalf("failed to import from COSE CBOR: %v", err)
		}

		// Verify the key matches
		if imported.D.Cmp(keyPair.Private.D) != 0 {
			t.Error("imported private key D does not match original")
		}
		if imported.X.Cmp(keyPair.Private.X) != 0 {
			t.Error("imported public key X does not match original")
		}
		if imported.Y.Cmp(keyPair.Private.Y) != 0 {
			t.Error("imported public key Y does not match original")
		}
	})
}

func TestExportPublicKeyToCOSECBOR(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	t.Run("exports valid COSE CBOR", func(t *testing.T) {
		cborData, err := cose.ExportPublicKeyToCOSECBOR(keyPair.Public)
		if err != nil {
			t.Fatalf("failed to export to COSE CBOR: %v", err)
		}

		if len(cborData) == 0 {
			t.Error("CBOR data is empty")
		}

		// CBOR data should start with a map
		if cborData[0] < 0xa0 || cborData[0] > 0xbf {
			t.Errorf("CBOR data does not start with a map: 0x%02x", cborData[0])
		}
	})

	t.Run("rejects nil public key", func(t *testing.T) {
		_, err := cose.ExportPublicKeyToCOSECBOR(nil)
		if err == nil {
			t.Error("expected error for nil public key")
		}
	})

	t.Run("round-trip public key", func(t *testing.T) {
		// Export to CBOR
		cborData, err := cose.ExportPublicKeyToCOSECBOR(keyPair.Public)
		if err != nil {
			t.Fatalf("failed to export to COSE CBOR: %v", err)
		}

		// Import from CBOR
		imported, err := cose.ImportPublicKeyFromCOSECBOR(cborData)
		if err != nil {
			t.Fatalf("failed to import from COSE CBOR: %v", err)
		}

		// Verify the key matches
		if imported.X.Cmp(keyPair.Public.X) != 0 {
			t.Error("imported public key X does not match original")
		}
		if imported.Y.Cmp(keyPair.Public.Y) != 0 {
			t.Error("imported public key Y does not match original")
		}
	})
}

func TestImportPrivateKeyFromCOSECBOR(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	cborData, err := cose.ExportPrivateKeyToCOSECBOR(keyPair.Private)
	if err != nil {
		t.Fatalf("failed to export to COSE CBOR: %v", err)
	}

	t.Run("imports valid COSE CBOR", func(t *testing.T) {
		imported, err := cose.ImportPrivateKeyFromCOSECBOR(cborData)
		if err != nil {
			t.Fatalf("failed to import from COSE CBOR: %v", err)
		}

		if imported.Curve != elliptic.P256() {
			t.Errorf("expected P-256 curve, got %v", imported.Curve)
		}
	})

	t.Run("rejects empty CBOR data", func(t *testing.T) {
		_, err := cose.ImportPrivateKeyFromCOSECBOR([]byte{})
		if err == nil {
			t.Error("expected error for empty CBOR data")
		}
	})

	t.Run("rejects invalid CBOR data", func(t *testing.T) {
		_, err := cose.ImportPrivateKeyFromCOSECBOR([]byte{0xff, 0xff, 0xff})
		if err == nil {
			t.Error("expected error for invalid CBOR data")
		}
	})
}

func TestImportPublicKeyFromCOSECBOR(t *testing.T) {
	keyPair, err := cose.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	cborData, err := cose.ExportPublicKeyToCOSECBOR(keyPair.Public)
	if err != nil {
		t.Fatalf("failed to export to COSE CBOR: %v", err)
	}

	t.Run("imports valid COSE CBOR", func(t *testing.T) {
		imported, err := cose.ImportPublicKeyFromCOSECBOR(cborData)
		if err != nil {
			t.Fatalf("failed to import from COSE CBOR: %v", err)
		}

		if imported.Curve != elliptic.P256() {
			t.Errorf("expected P-256 curve, got %v", imported.Curve)
		}
	})

	t.Run("rejects empty CBOR data", func(t *testing.T) {
		_, err := cose.ImportPublicKeyFromCOSECBOR([]byte{})
		if err == nil {
			t.Error("expected error for empty CBOR data")
		}
	})

	t.Run("rejects invalid CBOR data", func(t *testing.T) {
		_, err := cose.ImportPublicKeyFromCOSECBOR([]byte{0xff, 0xff, 0xff})
		if err == nil {
			t.Error("expected error for invalid CBOR data")
		}
	})
}
