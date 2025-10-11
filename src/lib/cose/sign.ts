/**
 * COSE Sign1 signing and verification
 *
 * COSE_Sign1 is a single-signer signature format defined in RFC 9052.
 * Structure: [protected, unprotected, payload, signature]
 *
 * This implementation uses the Signer/Verifier abstraction for actual
 * signature operations, supporting ES256 (ECDSA P-256 + SHA-256).
 */

import { encode as cborEncode, decode as cborDecode } from "cbor-x";
import { ES256Signer, ES256Verifier } from "./signer.ts";
import { CoseHeaderLabel, CWTClaim } from "./constants.ts";

/**
 * COSE protected headers map (integer labels to values)
 * Headers are stored as Map<number, any> for proper CBOR encoding
 */
export type CoseProtectedHeaders = Map<number, any>;

/**
 * CWT Claims map (integer claim keys to values)
 * Used in CWT Claims header (label 15) per RFC 9597
 */
export type CWTClaimsSet = Map<number, any>;

/**
 * Helper to create CWT claims set
 */
export function createCWTClaims(claims: {
  iss?: string;
  sub?: string;
  aud?: string;
  exp?: number;
  nbf?: number;
  iat?: number;
  cti?: Uint8Array;
  cnf?: any;
  scope?: string;
  nonce?: Uint8Array;
  [key: string]: any;
}): CWTClaimsSet {
  const map = new Map<number, any>();

  if (claims.iss !== undefined) {
    map.set(CWTClaim.ISS, claims.iss);
  }
  if (claims.sub !== undefined) {
    map.set(CWTClaim.SUB, claims.sub);
  }
  if (claims.aud !== undefined) {
    map.set(CWTClaim.AUD, claims.aud);
  }
  if (claims.exp !== undefined) {
    map.set(CWTClaim.EXP, claims.exp);
  }
  if (claims.nbf !== undefined) {
    map.set(CWTClaim.NBF, claims.nbf);
  }
  if (claims.iat !== undefined) {
    map.set(CWTClaim.IAT, claims.iat);
  }
  if (claims.cti !== undefined) {
    map.set(CWTClaim.CTI, claims.cti);
  }
  if (claims.cnf !== undefined) {
    map.set(CWTClaim.CNF, claims.cnf);
  }
  if (claims.scope !== undefined) {
    map.set(CWTClaim.SCOPE, claims.scope);
  }
  if (claims.nonce !== undefined) {
    map.set(CWTClaim.NONCE, claims.nonce);
  }

  return map;
}

/**
 * Helper to create protected headers map
 *
 * Note: Per RFC 9597, issuer (iss) and subject (sub) claims should be
 * placed inside the CWT Claims header (label 15) using createCWTClaims(),
 * not as separate SCITT headers (labels 391/392).
 */
export function createProtectedHeaders(headers: {
  alg: number;
  kid?: string | Uint8Array;
  cty?: string;
  typ?: string;
  cwtClaims?: CWTClaimsSet;
  [key: string]: any;
}): CoseProtectedHeaders {
  const map = new Map<number, any>();

  map.set(CoseHeaderLabel.ALG, headers.alg);

  if (headers.kid !== undefined) {
    map.set(CoseHeaderLabel.KID, headers.kid);
  }
  if (headers.cty !== undefined) {
    map.set(CoseHeaderLabel.CTY, headers.cty);
  }
  if (headers.typ !== undefined) {
    map.set(CoseHeaderLabel.TYP, headers.typ);
  }
  if (headers.cwtClaims !== undefined) {
    map.set(CoseHeaderLabel.CWT_CLAIMS, headers.cwtClaims);
  }

  return map;
}

/**
 * COSE Sign1 structure
 */
export interface CoseSign1 {
  protected: Uint8Array; // CBOR-encoded protected headers
  unprotected: Map<any, any>; // Unprotected headers
  payload: Uint8Array | null; // Payload (null for detached)
  signature: Uint8Array; // Signature bytes
}

/**
 * Options for creating COSE Sign1
 */
export interface CoseSign1Options {
  detached?: boolean; // If true, payload is null in structure
}

/**
 * Create a COSE Sign1 structure and sign it
 *
 * @param protectedHeaders - Protected headers to include
 * @param payload - Payload to sign
 * @param privateKey - Private key for signing
 * @param options - Additional options (e.g., detached mode)
 * @returns COSE Sign1 structure with signature
 */
export async function createCoseSign1(
  protectedHeaders: CoseProtectedHeaders,
  payload: Uint8Array,
  privateKey: CryptoKey,
  options: CoseSign1Options = {}
): Promise<CoseSign1> {
  // Encode protected headers
  const protectedEncoded = cborEncode(protectedHeaders);

  // Create Sig_structure for signing
  // Sig_structure = [
  //   context = "Signature1",
  //   body_protected,
  //   external_aad = h'',
  //   payload
  // ]
  // Note: Use Buffer.from() to avoid CBOR tags (d840) that cbor-x adds to Uint8Array
  const sigStructure = [
    "Signature1",
    Buffer.from(protectedEncoded),
    Buffer.from([]), // empty external AAD
    Buffer.from(payload),
  ];

  const toBeSigned = cborEncode(sigStructure);

  // Sign using ES256Signer
  const signer = new ES256Signer(privateKey);
  const signature = await signer.sign(toBeSigned);

  return {
    protected: protectedEncoded,
    unprotected: new Map(),
    payload: options.detached ? null : payload,
    signature,
  };
}

/**
 * Verify a COSE Sign1 signature
 *
 * @param coseSign1 - COSE Sign1 structure to verify
 * @param publicKey - Public key for verification
 * @param externalPayload - External payload (for detached mode)
 * @returns True if signature is valid
 */
export async function verifyCoseSign1(
  coseSign1: CoseSign1,
  publicKey: CryptoKey,
  externalPayload?: Uint8Array
): Promise<boolean> {
  // Determine payload to verify
  const payload = coseSign1.payload !== null ? coseSign1.payload : externalPayload;

  if (!payload) {
    throw new Error("No payload available for verification (detached mode requires externalPayload)");
  }

  // Reconstruct Sig_structure
  // Note: Use Buffer.from() to avoid CBOR tags (d840) that cbor-x adds to Uint8Array
  const sigStructure = [
    "Signature1",
    Buffer.from(coseSign1.protected),
    Buffer.from([]), // empty external AAD
    Buffer.from(payload),
  ];

  const toBeSigned = cborEncode(sigStructure);

  // Verify using ES256Verifier
  const verifier = new ES256Verifier(publicKey);
  return await verifier.verify(toBeSigned, coseSign1.signature);
}

/**
 * Parse protected headers from COSE Sign1
 *
 * @param coseSign1 - COSE Sign1 structure
 * @returns Decoded protected headers
 */
export function getProtectedHeaders(coseSign1: CoseSign1): CoseProtectedHeaders {
  return cborDecode(coseSign1.protected);
}

/**
 * Encode COSE Sign1 to CBOR bytes
 *
 * @param coseSign1 - COSE Sign1 structure
 * @returns CBOR-encoded bytes
 */
export function encodeCoseSign1(coseSign1: CoseSign1): Uint8Array {
  // COSE_Sign1 = [
  //   protected: bstr,
  //   unprotected: {},
  //   payload: bstr / nil,
  //   signature: bstr
  // ]
  const coseArray = [
    coseSign1.protected,
    coseSign1.unprotected,
    coseSign1.payload,
    coseSign1.signature,
  ];

  return cborEncode(coseArray);
}

/**
 * Decode CBOR bytes to COSE Sign1 structure
 *
 * Handles both tagged (tag 18) and untagged COSE_Sign1 structures.
 * RFC 9052 defines tag 18 for COSE_Sign1.
 *
 * @param encoded - CBOR-encoded bytes
 * @returns COSE Sign1 structure
 */
export function decodeCoseSign1(encoded: Uint8Array): CoseSign1 {
  let decoded = cborDecode(encoded);

  // Handle CBOR tag 18 (COSE_Sign1 tag from RFC 9052)
  if (decoded && typeof decoded === 'object' && 'tag' in decoded && decoded.tag === 18) {
    decoded = decoded.value;
  }

  if (!Array.isArray(decoded) || decoded.length !== 4) {
    throw new Error("Invalid COSE Sign1 structure");
  }

  return {
    protected: decoded[0],
    unprotected: decoded[1] instanceof Map ? decoded[1] : new Map(),
    payload: decoded[2],
    signature: decoded[3],
  };
}
