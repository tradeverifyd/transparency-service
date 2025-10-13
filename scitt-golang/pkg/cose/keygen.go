// Package cose provides COSE (RFC 8152) cryptographic operations
// for the transparency service, including ES256 key generation,
// import/export, and COSE_Key conversions.
package cose

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"

	gocose "github.com/veraison/go-cose"
)

// JWK represents a JSON Web Key (RFC 7517) for ES256 keys
type JWK struct {
	Kty string `json:"kty"`           // Key type (always "EC")
	Crv string `json:"crv"`           // Curve (always "P-256")
	X   string `json:"x"`             // X coordinate (base64url)
	Y   string `json:"y"`             // Y coordinate (base64url)
	D   string `json:"d,omitempty"`   // Private key (base64url, optional)
	Kid string `json:"kid,omitempty"` // Key identifier (optional)
	Alg string `json:"alg,omitempty"` // Algorithm (optional, "ES256")
	Use string `json:"use,omitempty"` // Key usage (optional, "sig")
}

// ES256KeyPair holds an ECDSA P-256 key pair
type ES256KeyPair struct {
	Private *ecdsa.PrivateKey
	Public  *ecdsa.PublicKey
}

// GenerateES256KeyPair generates a new ES256 (ECDSA P-256 with SHA-256) key pair
func GenerateES256KeyPair() (*ES256KeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ES256 key pair: %w", err)
	}

	return &ES256KeyPair{
		Private: privateKey,
		Public:  &privateKey.PublicKey,
	}, nil
}

// ExportPublicKeyToJWK exports the public key to JWK format
func ExportPublicKeyToJWK(publicKey *ecdsa.PublicKey) (*JWK, error) {
	if publicKey == nil {
		return nil, errors.New("public key is nil")
	}

	// Ensure we're using P-256 curve
	if publicKey.Curve != elliptic.P256() {
		return nil, errors.New("only P-256 curve is supported")
	}

	// Get coordinates
	xBytes := publicKey.X.Bytes()
	yBytes := publicKey.Y.Bytes()

	// Pad to 32 bytes if necessary (P-256 coordinates are 32 bytes)
	xBytes = padLeft(xBytes, 32)
	yBytes = padLeft(yBytes, 32)

	return &JWK{
		Kty: "EC",
		Crv: "P-256",
		X:   base64URLEncode(xBytes),
		Y:   base64URLEncode(yBytes),
	}, nil
}

// ExportPrivateKeyToPEM exports the private key to PEM format (PKCS#8)
func ExportPrivateKeyToPEM(privateKey *ecdsa.PrivateKey) (string, error) {
	if privateKey == nil {
		return "", errors.New("private key is nil")
	}

	// Marshal private key to PKCS#8 DER format
	derBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Encode to PEM
	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: derBytes,
	}

	return string(pem.EncodeToMemory(pemBlock)), nil
}

// ExportPublicKeyToPEM exports the public key to PEM format (SPKI)
func ExportPublicKeyToPEM(publicKey *ecdsa.PublicKey) (string, error) {
	if publicKey == nil {
		return "", errors.New("public key is nil")
	}

	// Marshal public key to SPKI DER format
	derBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	// Encode to PEM
	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derBytes,
	}

	return string(pem.EncodeToMemory(pemBlock)), nil
}

// ImportPrivateKeyFromPEM imports a private key from PEM format (PKCS#8)
func ImportPrivateKeyFromPEM(pemData string) (*ecdsa.PrivateKey, error) {
	// Decode PEM block
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	if block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("unexpected PEM block type: %s", block.Type)
	}

	// Parse PKCS#8 private key
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKCS#8 private key: %w", err)
	}

	// Ensure it's an ECDSA key
	ecKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("key is not an ECDSA private key")
	}

	// Ensure it's P-256
	if ecKey.Curve != elliptic.P256() {
		return nil, errors.New("only P-256 curve is supported")
	}

	return ecKey, nil
}

// ImportPublicKeyFromJWK imports a public key from JWK format
func ImportPublicKeyFromJWK(jwk *JWK) (*ecdsa.PublicKey, error) {
	if jwk == nil {
		return nil, errors.New("JWK is nil")
	}

	// Validate key type and curve
	if jwk.Kty != "EC" {
		return nil, fmt.Errorf("unsupported key type: %s", jwk.Kty)
	}
	if jwk.Crv != "P-256" {
		return nil, fmt.Errorf("unsupported curve: %s", jwk.Crv)
	}

	// Decode coordinates
	xBytes, err := base64URLDecode(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("failed to decode x coordinate: %w", err)
	}
	yBytes, err := base64URLDecode(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("failed to decode y coordinate: %w", err)
	}

	// Create public key
	x := new(big.Int).SetBytes(xBytes)
	y := new(big.Int).SetBytes(yBytes)

	publicKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}

	// Validate the public key is on the curve
	if !publicKey.Curve.IsOnCurve(x, y) {
		return nil, errors.New("public key point is not on P-256 curve")
	}

	return publicKey, nil
}

// ComputeKeyThumbprint computes the JWK thumbprint (RFC 7638)
// Uses SHA-256 hash of required JWK fields in lexicographic order
func ComputeKeyThumbprint(jwk *JWK) (string, error) {
	if jwk == nil {
		return "", errors.New("JWK is nil")
	}

	// Create canonical JSON with required fields in lexicographic order
	// For EC keys: crv, kty, x, y
	canonical := map[string]string{
		"crv": jwk.Crv,
		"kty": jwk.Kty,
		"x":   jwk.X,
		"y":   jwk.Y,
	}

	// Marshal to JSON (Go maps maintain insertion order for JSON encoding)
	// But we need to ensure lexicographic order, so we construct manually
	jsonStr := fmt.Sprintf(`{"crv":"%s","kty":"%s","x":"%s","y":"%s"}`,
		canonical["crv"], canonical["kty"], canonical["x"], canonical["y"])

	// Compute SHA-256 hash
	hash := sha256.Sum256([]byte(jsonStr))

	// Return base64url-encoded hash
	return base64URLEncode(hash[:]), nil
}

// ComputeCOSEKeyThumbprint computes the COSE Key Thumbprint (RFC 9679)
// Uses SHA-256 hash of deterministic CBOR encoding of required COSE_Key parameters
// Returns hex-encoded thumbprint
func ComputeCOSEKeyThumbprint(publicKey *ecdsa.PublicKey) (string, error) {
	if publicKey == nil {
		return "", errors.New("public key is nil")
	}

	// Ensure we're using P-256 curve
	if publicKey.Curve != elliptic.P256() {
		return "", errors.New("only P-256 curve is supported")
	}

	// Get coordinates
	xBytes := publicKey.X.Bytes()
	yBytes := publicKey.Y.Bytes()

	// Pad to 32 bytes if necessary (P-256 coordinates are 32 bytes)
	xBytes = padLeft(xBytes, 32)
	yBytes = padLeft(yBytes, 32)

	// Create COSE key with only required parameters for thumbprint:
	// For EC2 keys: kty (1), crv (-1), x (-2), y (-3)
	coseKey, err := gocose.NewKeyEC2(gocose.AlgorithmES256, xBytes, yBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create COSE EC2 key: %w", err)
	}

	// Marshal to deterministic CBOR (only required parameters)
	// The go-cose library already produces deterministic CBOR
	cborData, err := coseKey.MarshalCBOR()
	if err != nil {
		return "", fmt.Errorf("failed to marshal COSE key to CBOR: %w", err)
	}

	// Compute SHA-256 hash of the CBOR encoding
	hash := sha256.Sum256(cborData)

	// Return hex-encoded hash (lowercase)
	return fmt.Sprintf("%x", hash), nil
}

// JWKToCOSEKey converts a JWK to COSE_Key format
func JWKToCOSEKey(jwk *JWK) (*gocose.Key, error) {
	if jwk == nil {
		return nil, errors.New("JWK is nil")
	}

	// Decode coordinates
	xBytes, err := base64URLDecode(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("failed to decode x coordinate: %w", err)
	}
	yBytes, err := base64URLDecode(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("failed to decode y coordinate: %w", err)
	}

	// Decode private key if present
	var dBytes []byte
	if jwk.D != "" {
		dBytes, err = base64URLDecode(jwk.D)
		if err != nil {
			return nil, fmt.Errorf("failed to decode d parameter: %w", err)
		}
	}

	// Create COSE key using NewKeyEC2
	coseKey, err := gocose.NewKeyEC2(gocose.AlgorithmES256, xBytes, yBytes, dBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create COSE EC2 key: %w", err)
	}

	// Add optional parameters
	if jwk.Kid != "" {
		coseKey.ID = []byte(jwk.Kid)
	}

	return coseKey, nil
}

// COSEKeyToJWK converts a COSE_Key to JWK format
func COSEKeyToJWK(coseKey *gocose.Key) (*JWK, error) {
	if coseKey == nil {
		return nil, errors.New("COSE key is nil")
	}

	jwk := &JWK{
		Kty: "EC",
		Crv: "P-256",
	}

	// Extract EC2 coordinates using the EC2() method
	_, x, y, d := coseKey.EC2()

	if len(x) == 0 {
		return nil, errors.New("missing x coordinate in COSE key")
	}
	if len(y) == 0 {
		return nil, errors.New("missing y coordinate in COSE key")
	}

	jwk.X = base64URLEncode(x)
	jwk.Y = base64URLEncode(y)

	// Extract optional parameters
	if len(coseKey.ID) > 0 {
		jwk.Kid = string(coseKey.ID)
	}

	if coseKey.Algorithm == gocose.AlgorithmES256 {
		jwk.Alg = "ES256"
	}

	// Extract private key if present
	if len(d) > 0 {
		jwk.D = base64URLEncode(d)
	}

	return jwk, nil
}

// Utility functions

// base64URLEncode encodes bytes to base64url format (RFC 4648)
func base64URLEncode(data []byte) string {
	encoded := base64.RawURLEncoding.EncodeToString(data)
	return encoded
}

// base64URLDecode decodes base64url format to bytes
func base64URLDecode(s string) ([]byte, error) {
	// Handle both padded and unpadded base64url
	s = strings.TrimRight(s, "=")
	decoded, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

// padLeft pads a byte slice to the left with zeros to reach the target length
func padLeft(data []byte, length int) []byte {
	if len(data) >= length {
		return data
	}
	padded := make([]byte, length)
	copy(padded[length-len(data):], data)
	return padded
}

// MarshalJWK marshals a JWK to JSON
func MarshalJWK(jwk *JWK) ([]byte, error) {
	return json.Marshal(jwk)
}

// UnmarshalJWK unmarshals JSON to a JWK
func UnmarshalJWK(data []byte) (*JWK, error) {
	var jwk JWK
	if err := json.Unmarshal(data, &jwk); err != nil {
		return nil, err
	}
	return &jwk, nil
}

// ExportPrivateKeyToCOSECBOR exports a private key as COSE_Key in CBOR format
// The key will be an EC2 key with algorithm ES256
func ExportPrivateKeyToCOSECBOR(privateKey *ecdsa.PrivateKey) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New("private key is nil")
	}

	// Ensure we're using P-256 curve
	if privateKey.Curve != elliptic.P256() {
		return nil, errors.New("only P-256 curve is supported")
	}

	// Get coordinates
	xBytes := privateKey.X.Bytes()
	yBytes := privateKey.Y.Bytes()
	dBytes := privateKey.D.Bytes()

	// Pad to 32 bytes if necessary (P-256 coordinates are 32 bytes)
	xBytes = padLeft(xBytes, 32)
	yBytes = padLeft(yBytes, 32)
	dBytes = padLeft(dBytes, 32)

	// Create COSE key using NewKeyEC2
	coseKey, err := gocose.NewKeyEC2(gocose.AlgorithmES256, xBytes, yBytes, dBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create COSE EC2 key: %w", err)
	}

	// Marshal to CBOR
	cborData, err := coseKey.MarshalCBOR()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal COSE key to CBOR: %w", err)
	}

	return cborData, nil
}

// ExportPublicKeyToCOSECBOR exports a public key as COSE_Key in CBOR format
// The key will be an EC2 key with algorithm ES256
func ExportPublicKeyToCOSECBOR(publicKey *ecdsa.PublicKey) ([]byte, error) {
	if publicKey == nil {
		return nil, errors.New("public key is nil")
	}

	// Ensure we're using P-256 curve
	if publicKey.Curve != elliptic.P256() {
		return nil, errors.New("only P-256 curve is supported")
	}

	// Get coordinates
	xBytes := publicKey.X.Bytes()
	yBytes := publicKey.Y.Bytes()

	// Pad to 32 bytes if necessary (P-256 coordinates are 32 bytes)
	xBytes = padLeft(xBytes, 32)
	yBytes = padLeft(yBytes, 32)

	// Create COSE key using NewKeyEC2 (no d parameter for public key)
	coseKey, err := gocose.NewKeyEC2(gocose.AlgorithmES256, xBytes, yBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create COSE EC2 key: %w", err)
	}

	// Marshal to CBOR
	cborData, err := coseKey.MarshalCBOR()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal COSE key to CBOR: %w", err)
	}

	return cborData, nil
}

// ImportPrivateKeyFromCOSECBOR imports a private key from COSE_Key CBOR format
func ImportPrivateKeyFromCOSECBOR(cborData []byte) (*ecdsa.PrivateKey, error) {
	if len(cborData) == 0 {
		return nil, errors.New("CBOR data is empty")
	}

	// Unmarshal CBOR to COSE key
	coseKey := &gocose.Key{}
	if err := coseKey.UnmarshalCBOR(cborData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CBOR to COSE key: %w", err)
	}

	// Extract EC2 parameters
	_, x, y, d := coseKey.EC2()

	if len(x) == 0 || len(y) == 0 {
		return nil, errors.New("missing EC2 coordinates in COSE key")
	}
	if len(d) == 0 {
		return nil, errors.New("missing private key parameter in COSE key")
	}

	// Verify algorithm
	if coseKey.Algorithm != gocose.AlgorithmES256 {
		return nil, fmt.Errorf("unsupported algorithm: expected ES256, got %v", coseKey.Algorithm)
	}

	// Create private key
	privateKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(x),
			Y:     new(big.Int).SetBytes(y),
		},
		D: new(big.Int).SetBytes(d),
	}

	// Validate the public key is on the curve
	if !privateKey.PublicKey.Curve.IsOnCurve(privateKey.PublicKey.X, privateKey.PublicKey.Y) {
		return nil, errors.New("public key point is not on P-256 curve")
	}

	return privateKey, nil
}

// ImportPublicKeyFromCOSECBOR imports a public key from COSE_Key CBOR format
func ImportPublicKeyFromCOSECBOR(cborData []byte) (*ecdsa.PublicKey, error) {
	if len(cborData) == 0 {
		return nil, errors.New("CBOR data is empty")
	}

	// Unmarshal CBOR to COSE key
	coseKey := &gocose.Key{}
	if err := coseKey.UnmarshalCBOR(cborData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal CBOR to COSE key: %w", err)
	}

	// Extract EC2 parameters
	_, x, y, _ := coseKey.EC2()

	if len(x) == 0 || len(y) == 0 {
		return nil, errors.New("missing EC2 coordinates in COSE key")
	}

	// Verify algorithm
	if coseKey.Algorithm != gocose.AlgorithmES256 {
		return nil, fmt.Errorf("unsupported algorithm: expected ES256, got %v", coseKey.Algorithm)
	}

	// Create public key
	publicKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(x),
		Y:     new(big.Int).SetBytes(y),
	}

	// Validate the public key is on the curve
	if !publicKey.Curve.IsOnCurve(publicKey.X, publicKey.Y) {
		return nil, errors.New("public key point is not on P-256 curve")
	}

	return publicKey, nil
}
