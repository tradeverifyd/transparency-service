/**
 * COSE (CBOR Object Signing and Encryption) Type Definitions
 *
 * Based on:
 * - RFC 8152: COSE (CBOR Object Signing and Encryption)
 * - RFC 9052: COSE Updated
 * - draft-ietf-cose-hash-envelope: COSE Hash Envelope
 * - draft-ietf-cose-merkle-tree-proofs: COSE Merkle Tree Proofs
 */

/**
 * COSE Algorithm Identifiers
 * https://www.iana.org/assignments/cose/cose.xhtml#algorithms
 */
export enum COSEAlgorithm {
  ES256 = -7,        // ECDSA w/ SHA-256
  ES384 = -35,       // ECDSA w/ SHA-384
  ES512 = -36,       // ECDSA w/ SHA-512
  EdDSA = -8,        // EdDSA
  SHA_256 = -16,     // SHA-256 Hash
  SHA_384 = -43,     // SHA-384 Hash
  SHA_512 = -44,     // SHA-512 Hash
}

/**
 * COSE Key Type
 * https://www.iana.org/assignments/cose/cose.xhtml#key-type
 */
export enum COSEKeyType {
  OKP = 1,    // Octet Key Pair (Ed25519, Ed448)
  EC2 = 2,    // Elliptic Curve Keys w/ x- and y-coordinate pair
  Symmetric = 4,
}

/**
 * COSE Elliptic Curves
 * https://www.iana.org/assignments/cose/cose.xhtml#elliptic-curves
 */
export enum COSECurve {
  P_256 = 1,      // NIST P-256 (secp256r1)
  P_384 = 2,      // NIST P-384 (secp384r1)
  P_521 = 3,      // NIST P-521 (secp521r1)
  Ed25519 = 6,    // Ed25519 for use w/ EdDSA
  Ed448 = 7,      // Ed448 for use w/ EdDSA
}

/**
 * COSE Key Common Parameters
 * https://www.iana.org/assignments/cose/cose.xhtml#key-common-parameters
 */
export interface COSEKeyCommonParameters {
  1?: COSEKeyType;        // kty: Key Type
  2?: Uint8Array;         // kid: Key ID
  3?: COSEAlgorithm;      // alg: Algorithm
  4?: number[];           // key_ops: Key Operations
  5?: Uint8Array;         // Base IV
}

/**
 * COSE EC2 Key Parameters
 * For Elliptic Curve keys with x- and y-coordinates
 */
export interface COSEEC2Key extends COSEKeyCommonParameters {
  1: COSEKeyType.EC2;     // kty: must be EC2
  "-1": COSECurve;        // crv: Curve
  "-2": Uint8Array;       // x: x-coordinate
  "-3"?: Uint8Array;      // y: y-coordinate (optional for compressed)
  "-4"?: Uint8Array;      // d: private key (optional)
}

/**
 * COSE OKP Key Parameters
 * For Octet Key Pair (Ed25519, Ed448)
 */
export interface COSEOKPKey extends COSEKeyCommonParameters {
  1: COSEKeyType.OKP;     // kty: must be OKP
  "-1": COSECurve;        // crv: Curve
  "-2": Uint8Array;       // x: public key
  "-4"?: Uint8Array;      // d: private key (optional)
}

/**
 * COSE Key (union type)
 */
export type COSEKey = COSEEC2Key | COSEOKPKey;

/**
 * COSE Protected Header Map
 * CBOR-encoded map that is integrity protected
 */
export interface COSEProtectedHeader {
  1?: COSEAlgorithm;      // alg: Algorithm
  3?: string | number;    // content_type
  4?: Uint8Array;         // kid: Key ID
  5?: Uint8Array;         // IV
  [key: string]: unknown; // Additional header parameters
}

/**
 * COSE Unprotected Header Map
 * Map that is not integrity protected
 */
export interface COSEUnprotectedHeader {
  [key: string]: unknown; // Header parameters
}

/**
 * COSE Sign1 Structure
 * Single signer digital signature
 *
 * COSE_Sign1 = [
 *   protected: bstr,      // CBOR-encoded protected header
 *   unprotected: {},      // Unprotected header map
 *   payload: bstr / nil,  // Payload or detached (nil)
 *   signature: bstr       // Signature bytes
 * ]
 */
export interface COSESign1 {
  protected: Uint8Array;              // CBOR-encoded protected header
  unprotected: COSEUnprotectedHeader; // Unprotected header map
  payload: Uint8Array | null;         // Payload (or null for detached)
  signature: Uint8Array;              // Signature bytes
}

/**
 * COSE Hash Envelope Parameters
 * draft-ietf-cose-hash-envelope
 */
export enum COSEHashEnvelopeLabel {
  PAYLOAD_HASH_ALG = 258,          // Algorithm for payload hash
  PAYLOAD_PREIMAGE_CONTENT_TYPE = 259, // Content type of preimage
  PAYLOAD_LOCATION = 260,          // Location of payload
}

/**
 * COSE Hash Envelope Protected Header
 * Extends standard protected header with hash envelope labels
 */
export interface COSEHashEnvelopeProtectedHeader extends COSEProtectedHeader {
  258: COSEAlgorithm;     // payload_hash_alg: Hash algorithm
  259?: string;           // preimage_content_type: Content type
  260?: string;           // payload_location: URL or reference
}

/**
 * COSE Merkle Tree Proof Parameters
 * draft-ietf-cose-merkle-tree-proofs
 */
export enum COSEMerkleTreeProofLabel {
  RECEIPTS = 394,         // Receipts header (array of receipts)
  VDS = 395,              // Verifiable Data Structure algorithm
  VDP = 396,              // Verifiable Data Structure Parameters
}

/**
 * Verifiable Data Structure Algorithm
 */
export type VDSAlgorithm = "RFC9162_SHA256"; // C2SP tlog-tiles with SHA-256

/**
 * Verifiable Data Structure Parameters (VDP)
 * Inclusion proof parameters
 */
export interface InclusionProofVDP {
  tree_size: number;           // Tree size at time of proof
  leaf_index: number;          // Index of leaf in tree (0-based)
  inclusion_path: Uint8Array[]; // Merkle path hashes
}

/**
 * COSE Receipt Structure
 * Contains Merkle inclusion proof
 */
export interface COSEReceipt {
  protected: {
    395: VDSAlgorithm;          // vds: RFC9162_SHA256
    396: InclusionProofVDP;     // vdp: Inclusion proof
  };
  payload: null;                // Always detached
  signature: Uint8Array;        // Transparency service signature
}

/**
 * COSE Sign1 with Receipts
 * Statement with attached receipt
 */
export interface COSESign1WithReceipt extends COSESign1 {
  unprotected: {
    394: COSEReceipt[];         // receipts: Array of receipts
  };
}

/**
 * Signer Interface
 * Abstract interface for signing operations (supports HSM)
 */
export interface Signer {
  /**
   * Sign data using the configured key
   * @param data - Data to sign
   * @returns Signature bytes
   */
  sign(data: Uint8Array): Promise<Uint8Array>;

  /**
   * Get the algorithm identifier
   */
  algorithm: COSEAlgorithm;

  /**
   * Get the key ID (optional)
   */
  keyId?: Uint8Array;
}

/**
 * Verifier Interface
 * Abstract interface for verification operations
 */
export interface Verifier {
  /**
   * Verify signature over data
   * @param data - Data that was signed
   * @param signature - Signature to verify
   * @returns True if signature is valid
   */
  verify(data: Uint8Array, signature: Uint8Array): Promise<boolean>;

  /**
   * Get the algorithm identifier
   */
  algorithm: COSEAlgorithm;

  /**
   * Get the key ID (optional)
   */
  keyId?: Uint8Array;
}
