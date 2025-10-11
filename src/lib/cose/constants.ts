/**
 * COSE Constants
 * 
 * Defines algorithm identifiers and header labels from:
 * - RFC 9052 (COSE)
 * - RFC 9053 (COSE Algorithms)
 * - draft-ietf-scitt-architecture (SCITT extensions)
 */

/**
 * COSE Algorithm Identifiers (RFC 9053)
 */
export const CoseAlgorithm = {
  // ECDSA
  ES256: -7,   // ECDSA w/ SHA-256
  ES384: -35,  // ECDSA w/ SHA-384
  ES512: -36,  // ECDSA w/ SHA-512
  
  // EdDSA
  EdDSA: -8,   // EdDSA
  
  // RSASSA-PSS
  PS256: -37,  // RSASSA-PSS w/ SHA-256
  PS384: -38,  // RSASSA-PSS w/ SHA-384
  PS512: -39,  // RSASSA-PSS w/ SHA-512
  
  // HMAC
  HMAC_256_256: 5,  // HMAC w/ SHA-256 truncated to 256 bits
  HMAC_384_384: 6,  // HMAC w/ SHA-384 truncated to 384 bits
  HMAC_512_512: 7,  // HMAC w/ SHA-512 truncated to 512 bits
} as const;

/**
 * COSE Header Labels (RFC 9052)
 */
export const CoseHeaderLabel = {
  // Common Headers (RFC 9052)
  ALG: 1,       // Algorithm identifier
  CRIT: 2,      // Critical headers
  CTY: 3,       // Content type
  KID: 4,       // Key identifier
  IV: 5,        // Initialization vector
  PARTIAL_IV: 6, // Partial initialization vector
  COUNTER_SIG: 7, // Counter signature
  
  // Additional Common Headers
  CWT_CLAIMS: 15, // CWT Claims Set (RFC 9597)
  TYP: 16,      // Type (media type of content)

  // SCITT-Specific Headers (draft-ietf-scitt-architecture)
  // Note: When using RFC 9597, use CWT claims (label 15) with CWTClaim.ISS/SUB instead
  ISS: 391,     // Issuer (URL) - SCITT specific
  SUB: 392,     // Subject - SCITT specific
  
  // COSE Hash Envelope Headers (draft-ietf-cose-hash-envelope)
  PAYLOAD_HASH_ALG: 258,           // Hash algorithm for payload
  PAYLOAD_PREIMAGE_CONTENT_TYPE: 259, // Content type of original payload
  PAYLOAD_LOCATION: 260,           // Location of original payload
  
  // COSE Merkle Tree Proofs Headers (draft-ietf-cose-merkle-tree-proofs)
  VERIFIABLE_DATA_STRUCTURE: 395, // VDS algorithm
  VERIFIABLE_DATA_PROOF: 396,     // VDP (inclusion/consistency proof)
  RECEIPTS: 394,                   // Array of receipts
} as const;

/**
 * COSE Key Type (kty) Values (RFC 9052)
 */
export const CoseKeyType = {
  OKP: 1,   // Octet Key Pair (Ed25519, Ed448)
  EC2: 2,   // Elliptic Curve Keys w/ x- and y-coordinate
  RSA: 3,   // RSA Key
  SYMMETRIC: 4, // Symmetric Keys
} as const;

/**
 * COSE Key Common Parameters (RFC 9052)
 */
export const CoseKeyParameter = {
  KTY: 1,   // Key Type
  KID: 2,   // Key ID
  ALG: 3,   // Algorithm
  KEY_OPS: 4, // Key Operations
  BASE_IV: 5, // Base IV
} as const;

/**
 * COSE EC2 Key Parameters (RFC 9052)
 */
export const CoseEC2KeyParameter = {
  CRV: -1,  // EC Curve identifier
  X: -2,    // x-coordinate
  Y: -3,    // y-coordinate (optional for compressed points)
  D: -4,    // Private key
} as const;

/**
 * COSE Elliptic Curves (RFC 9052)
 */
export const CoseEllipticCurve = {
  P_256: 1,   // NIST P-256 (secp256r1)
  P_384: 2,   // NIST P-384 (secp384r1)
  P_521: 3,   // NIST P-521 (secp521r1)
  X25519: 4,  // X25519 (ECDH only)
  X448: 5,    // X448 (ECDH only)
  Ed25519: 6, // Ed25519 (EdDSA only)
  Ed448: 7,   // Ed448 (EdDSA only)
} as const;

/**
 * Hash Algorithm Identifiers (for COSE Hash Envelope)
 */
export const CoseHashAlgorithm = {
  SHA_256: -16,  // SHA-256
  SHA_384: -43,  // SHA-384
  SHA_512: -44,  // SHA-512
  SHA_1: -14,    // SHA-1 (deprecated, for legacy only)
} as const;

/**
 * COSE Message Tags (RFC 9052)
 */
export const CoseTag = {
  COSE_SIGN: 98,      // COSE_Sign
  COSE_SIGN1: 18,     // COSE_Sign1
  COSE_ENCRYPT: 96,   // COSE_Encrypt
  COSE_ENCRYPT0: 16,  // COSE_Encrypt0
  COSE_MAC: 97,       // COSE_Mac
  COSE_MAC0: 17,      // COSE_Mac0
} as const;

/**
 * Verifiable Data Structure Algorithms (draft-ietf-cose-merkle-tree-proofs)
 */
export const CoseVDS = {
  RFC9162_SHA256: 1,  // RFC 9162 (Certificate Transparency) with SHA-256
} as const;

/**
 * CWT (CBOR Web Token) Claim Keys (RFC 8392 and IANA CWT Claims registry)
 * Used in CWT Claims header (label 15) per RFC 9597
 */
export const CWTClaim = {
  // Core Claims (RFC 8392)
  ISS: 1,       // Issuer
  SUB: 2,       // Subject
  AUD: 3,       // Audience
  EXP: 4,       // Expiration Time
  NBF: 5,       // Not Before
  IAT: 6,       // Issued At
  CTI: 7,       // CWT ID
  CNF: 8,       // Confirmation (RFC 8747)

  // Extended Claims (IANA registry)
  SCOPE: 9,     // Access Token Scope (RFC 9200)
  NONCE: 10,    // Nonce (RFC 8392)

  // UEID and Geographic Claims
  UEID: 256,    // Universal Entity ID (RFC 9334)

  // PSA (Platform Security Architecture) Claims
  PSA_NONCE: 2394,           // PSA Nonce
  PSA_INSTANCE_ID: 2395,     // PSA Instance ID
  PSA_BOOT_SEED: 2396,       // PSA Boot Seed
  PSA_CERT_REF: 2397,        // PSA Certification Reference
  PSA_SOFTWARE_COMPONENTS: 2398, // PSA Software Components
  PSA_SECURITY_LIFECYCLE: 2399,  // PSA Security Lifecycle
  PSA_CLIENT_ID: 2400,       // PSA Client ID
} as const;

/**
 * Type guard to check if a value is a valid COSE algorithm
 */
export function isCoseAlgorithm(alg: number): boolean {
  return Object.values(CoseAlgorithm).includes(alg as any);
}

/**
 * Get algorithm name from identifier
 */
export function getAlgorithmName(alg: number): string {
  const entry = Object.entries(CoseAlgorithm).find(([_, value]) => value === alg);
  return entry ? entry[0] : `Unknown (${alg})`;
}

/**
 * Get header label name from identifier
 */
export function getHeaderLabelName(label: number): string {
  const entry = Object.entries(CoseHeaderLabel).find(([_, value]) => value === label);
  return entry ? entry[0] : `Unknown (${label})`;
}
