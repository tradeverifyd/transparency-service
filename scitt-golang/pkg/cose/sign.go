package cose

import (
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

// COSE Header Label constants (RFC 9052)
const (
	HeaderLabelAlg            = 1  // Algorithm identifier
	HeaderLabelCrit           = 2  // Critical headers
	HeaderLabelContentType    = 3  // Content type
	HeaderLabelKid            = 4  // Key identifier
	HeaderLabelIV             = 5  // Initialization vector
	HeaderLabelPartialIV      = 6  // Partial initialization vector
	HeaderLabelCounterSig     = 7  // Counter signature
	HeaderLabelCWTClaims      = 15 // CWT Claims Set (RFC 9597)
	HeaderLabelTyp            = 16 // Type (media type of content)
	HeaderLabelIss            = 391 // Issuer (SCITT specific)
	HeaderLabelSub            = 392 // Subject (SCITT specific)
	HeaderLabelPayloadHashAlg = 258 // Hash algorithm for payload
	HeaderLabelPayloadPreimageContentType = 259 // Content type of original payload
	HeaderLabelPayloadLocation            = 260 // Location of original payload
	HeaderLabelVerifiableDataStructure    = 395 // VDS algorithm
	HeaderLabelVerifiableDataProof        = 396 // VDP (inclusion/consistency proof)
	HeaderLabelReceipts                   = 394 // Array of receipts
)

// COSE Algorithm Identifiers (RFC 9053)
const (
	AlgorithmES256 = -7  // ECDSA w/ SHA-256
	AlgorithmES384 = -35 // ECDSA w/ SHA-384
	AlgorithmES512 = -36 // ECDSA w/ SHA-512
	AlgorithmEdDSA = -8  // EdDSA
)

// CWT Claim Keys (RFC 8392)
const (
	CWTClaimIss   = 1  // Issuer
	CWTClaimSub   = 2  // Subject
	CWTClaimAud   = 3  // Audience
	CWTClaimExp   = 4  // Expiration Time
	CWTClaimNbf   = 5  // Not Before
	CWTClaimIat   = 6  // Issued At
	CWTClaimCti   = 7  // CWT ID
	CWTClaimCnf   = 8  // Confirmation
	CWTClaimScope = 9  // Scope
	CWTClaimNonce = 10 // Nonce
)

// ProtectedHeaders represents COSE protected headers (encoded as CBOR map)
type ProtectedHeaders map[interface{}]interface{}

// CWTClaimsSet represents CWT claims (used in CWT Claims header)
type CWTClaimsSet map[interface{}]interface{}

// CWTClaimsOptions holds options for creating CWT claims
type CWTClaimsOptions struct {
	Iss   string
	Sub   string
	Aud   string
	Exp   int64
	Nbf   int64
	Iat   int64
	Cti   []byte
	Scope string
	Nonce []byte
}

// CreateCWTClaims creates a CWT claims set from options
func CreateCWTClaims(opts CWTClaimsOptions) CWTClaimsSet {
	claims := make(CWTClaimsSet)

	if opts.Iss != "" {
		claims[CWTClaimIss] = opts.Iss
	}
	if opts.Sub != "" {
		claims[CWTClaimSub] = opts.Sub
	}
	if opts.Aud != "" {
		claims[CWTClaimAud] = opts.Aud
	}
	if opts.Exp != 0 {
		claims[CWTClaimExp] = opts.Exp
	}
	if opts.Nbf != 0 {
		claims[CWTClaimNbf] = opts.Nbf
	}
	if opts.Iat != 0 {
		claims[CWTClaimIat] = opts.Iat
	}
	if len(opts.Cti) > 0 {
		claims[CWTClaimCti] = opts.Cti
	}
	if opts.Scope != "" {
		claims[CWTClaimScope] = opts.Scope
	}
	if len(opts.Nonce) > 0 {
		claims[CWTClaimNonce] = opts.Nonce
	}

	return claims
}

// ProtectedHeadersOptions holds options for creating protected headers
type ProtectedHeadersOptions struct {
	Alg       int
	Kid       interface{} // string or []byte
	Cty       string
	Typ       string
	CWTClaims CWTClaimsSet
}

// CreateProtectedHeaders creates a protected headers map from options
//
// Note: Per RFC 9597, issuer (iss) and subject (sub) claims should be
// placed inside the CWT Claims header (label 15) using CreateCWTClaims(),
// not as separate SCITT headers (labels 391/392).
func CreateProtectedHeaders(opts ProtectedHeadersOptions) ProtectedHeaders {
	headers := make(ProtectedHeaders)

	headers[HeaderLabelAlg] = opts.Alg

	if opts.Kid != nil {
		headers[HeaderLabelKid] = opts.Kid
	}
	if opts.Cty != "" {
		headers[HeaderLabelContentType] = opts.Cty
	}
	if opts.Typ != "" {
		headers[HeaderLabelTyp] = opts.Typ
	}
	if len(opts.CWTClaims) > 0 {
		headers[HeaderLabelCWTClaims] = opts.CWTClaims
	}

	return headers
}

// CoseSign1 represents a COSE Sign1 structure (RFC 9052)
type CoseSign1 struct {
	Protected   []byte                 // CBOR-encoded protected headers
	Unprotected map[interface{}]interface{} // Unprotected headers
	Payload     []byte                 // Payload (nil for detached mode)
	Signature   []byte                 // Signature bytes
}

// CoseSign1Options holds options for creating COSE Sign1
type CoseSign1Options struct {
	Detached bool // If true, payload is nil in structure
}

// CreateCoseSign1 creates and signs a COSE Sign1 structure
//
// This function:
// 1. Encodes the protected headers to CBOR
// 2. Constructs the Sig_structure for signing
// 3. Signs the Sig_structure using the provided signer
// 4. Returns the complete COSE Sign1 structure
//
// Parameters:
//   - protectedHeaders: Map of protected header labels to values
//   - payload: Data to sign
//   - signer: Signer implementation (e.g., ES256Signer)
//   - options: Additional options (e.g., detached mode)
//
// Returns:
//   - CoseSign1 structure with signature
func CreateCoseSign1(
	protectedHeaders ProtectedHeaders,
	payload []byte,
	signer Signer,
	options CoseSign1Options,
) (*CoseSign1, error) {
	// Encode protected headers to CBOR
	protectedEncoded, err := cbor.Marshal(protectedHeaders)
	if err != nil {
		return nil, fmt.Errorf("failed to encode protected headers: %w", err)
	}

	// Create Sig_structure for signing
	// Sig_structure = [
	//   context = "Signature1",
	//   body_protected,
	//   external_aad = h'',
	//   payload
	// ]
	sigStructure := []interface{}{
		"Signature1",
		protectedEncoded,
		[]byte{}, // empty external AAD
		payload,
	}

	toBeSigned, err := cbor.Marshal(sigStructure)
	if err != nil {
		return nil, fmt.Errorf("failed to encode Sig_structure: %w", err)
	}

	// Sign using the provided signer
	signature, err := signer.Sign(toBeSigned)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	// Determine payload (nil for detached mode)
	resultPayload := payload
	if options.Detached {
		resultPayload = nil
	}

	return &CoseSign1{
		Protected:   protectedEncoded,
		Unprotected: make(map[interface{}]interface{}),
		Payload:     resultPayload,
		Signature:   signature,
	}, nil
}

// VerifyCoseSign1 verifies a COSE Sign1 signature
//
// This function:
// 1. Reconstructs the Sig_structure from the COSE Sign1
// 2. Verifies the signature using the provided verifier
//
// Parameters:
//   - coseSign1: COSE Sign1 structure to verify
//   - verifier: Verifier implementation (e.g., ES256Verifier)
//   - externalPayload: External payload (for detached mode)
//
// Returns:
//   - true if signature is valid, false otherwise
func VerifyCoseSign1(
	coseSign1 *CoseSign1,
	verifier Verifier,
	externalPayload []byte,
) (bool, error) {
	// Determine payload to verify
	payload := coseSign1.Payload
	if payload == nil {
		if externalPayload == nil {
			return false, fmt.Errorf("no payload available for verification (detached mode requires externalPayload)")
		}
		payload = externalPayload
	}

	// Reconstruct Sig_structure
	sigStructure := []interface{}{
		"Signature1",
		coseSign1.Protected,
		[]byte{}, // empty external AAD
		payload,
	}

	toBeSigned, err := cbor.Marshal(sigStructure)
	if err != nil {
		return false, fmt.Errorf("failed to encode Sig_structure: %w", err)
	}

	// Verify using the provided verifier
	return verifier.Verify(toBeSigned, coseSign1.Signature)
}

// GetProtectedHeaders decodes and returns the protected headers from COSE Sign1
func GetProtectedHeaders(coseSign1 *CoseSign1) (ProtectedHeaders, error) {
	var headers ProtectedHeaders
	if err := cbor.Unmarshal(coseSign1.Protected, &headers); err != nil {
		return nil, fmt.Errorf("failed to decode protected headers: %w", err)
	}
	return headers, nil
}

// EncodeCoseSign1 encodes a COSE Sign1 structure to CBOR bytes
//
// COSE_Sign1 = [
//   protected: bstr,
//   unprotected: {},
//   payload: bstr / nil,
//   signature: bstr
// ]
func EncodeCoseSign1(coseSign1 *CoseSign1) ([]byte, error) {
	coseArray := []interface{}{
		coseSign1.Protected,
		coseSign1.Unprotected,
		coseSign1.Payload,
		coseSign1.Signature,
	}

	encoded, err := cbor.Marshal(coseArray)
	if err != nil {
		return nil, fmt.Errorf("failed to encode COSE Sign1: %w", err)
	}

	return encoded, nil
}

// DecodeCoseSign1 decodes CBOR bytes to a COSE Sign1 structure
//
// Handles both tagged (tag 18) and untagged COSE_Sign1 structures.
// RFC 9052 defines tag 18 for COSE_Sign1.
func DecodeCoseSign1(encoded []byte) (*CoseSign1, error) {
	// Try to decode as array first
	var coseArray []interface{}
	if err := cbor.Unmarshal(encoded, &coseArray); err != nil {
		return nil, fmt.Errorf("failed to decode COSE Sign1: %w", err)
	}

	if len(coseArray) != 4 {
		return nil, fmt.Errorf("invalid COSE Sign1 structure: expected 4 elements, got %d", len(coseArray))
	}

	// Extract components
	protected, ok := coseArray[0].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid protected headers: expected bytes")
	}

	unprotected, ok := coseArray[1].(map[interface{}]interface{})
	if !ok {
		// Try to handle empty map
		if coseArray[1] == nil {
			unprotected = make(map[interface{}]interface{})
		} else {
			return nil, fmt.Errorf("invalid unprotected headers: expected map")
		}
	}

	// Payload can be nil (detached mode) or bytes
	var payload []byte
	if coseArray[2] != nil {
		payload, ok = coseArray[2].([]byte)
		if !ok {
			return nil, fmt.Errorf("invalid payload: expected bytes or nil")
		}
	}

	signature, ok := coseArray[3].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid signature: expected bytes")
	}

	return &CoseSign1{
		Protected:   protected,
		Unprotected: unprotected,
		Payload:     payload,
		Signature:   signature,
	}, nil
}
