/**
 * COSE Hash Envelope Implementation
 * Streaming hash computation for large files with COSE Sign1 integration
 * Labels: 258 (hash alg), 259 (content type), 260 (location)
 */

import { CoseHashAlgorithm, CoseHeaderLabel, CoseAlgorithm } from "./constants.ts";
import {
  createCoseSign1,
  verifyCoseSign1,
  getProtectedHeaders,
  createProtectedHeaders,
  createCWTClaims,
  type CoseSign1,
  type CoseProtectedHeaders,
  type CWTClaimsSet,
} from "./sign.ts";
import * as fs from "fs";

/**
 * Hash Envelope Structure
 */
export interface HashEnvelope {
  payloadHash: Uint8Array;
  payloadHashAlg: number; // CoseHashAlgorithm value
  preimageContentType?: string;
  payloadLocation?: string;
}

/**
 * Hash Envelope Options
 */
export interface HashEnvelopeOptions {
  contentType?: string;
  location?: string;
  hashAlgorithm?: number; // CoseHashAlgorithm value
}

/**
 * Hash Envelope Verification Result
 */
export interface HashEnvelopeVerificationResult {
  signatureValid: boolean;
  hashValid: boolean;
}

/**
 * Create hash envelope from data
 */
export async function createHashEnvelope(
  data: Uint8Array,
  options: HashEnvelopeOptions = {}
): Promise<HashEnvelope> {
  const hashAlgorithm = options.hashAlgorithm ?? CoseHashAlgorithm.SHA_256;
  const hash = await hashData(data, hashAlgorithm);

  return {
    payloadHash: hash,
    payloadHashAlg: hashAlgorithm,
    preimageContentType: options.contentType,
    payloadLocation: options.location,
  };
}

/**
 * Hash data using specified algorithm
 */
export async function hashData(
  data: Uint8Array,
  algorithm: number = CoseHashAlgorithm.SHA_256
): Promise<Uint8Array> {
  const hashAlgorithm = getWebCryptoHashAlgorithm(algorithm);
  const hashBuffer = await crypto.subtle.digest(hashAlgorithm, data);
  return new Uint8Array(hashBuffer);
}

/**
 * Stream hash from file (for large files)
 * Uses Bun's native streaming I/O
 */
export async function streamHashFromFile(
  filePath: string,
  algorithm: number = CoseHashAlgorithm.SHA_256
): Promise<Uint8Array> {
  const hashAlgorithm = getWebCryptoHashAlgorithm(algorithm);

  // Use Bun's file API for streaming
  const file = Bun.file(filePath);
  const stream = file.stream();

  // Create hash with streaming
  const reader = stream.getReader();
  const chunks: Uint8Array[] = [];
  let totalLength = 0;

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      chunks.push(new Uint8Array(value));
      totalLength += value.length;
    }
  } finally {
    reader.releaseLock();
  }

  // Concatenate chunks and hash
  const concatenated = new Uint8Array(totalLength);
  let offset = 0;
  for (const chunk of chunks) {
    concatenated.set(chunk, offset);
    offset += chunk.length;
  }

  const hashBuffer = await crypto.subtle.digest(hashAlgorithm, concatenated);
  return new Uint8Array(hashBuffer);
}

/**
 * Validate hash envelope against data
 */
export async function validateHashEnvelope(
  envelope: HashEnvelope,
  data: Uint8Array
): Promise<boolean> {
  try {
    const computedHash = await hashData(data, envelope.payloadHashAlg);
    return areEqual(computedHash, envelope.payloadHash);
  } catch (error) {
    return false;
  }
}

/**
 * Sign hash envelope as COSE Sign1
 * Creates statement with hash as payload and envelope params in protected header
 *
 * @param artifact - Original artifact to hash
 * @param options - Hash envelope options (content type, location)
 * @param privateKey - Private key for signing
 * @param cwtClaims - Optional CWT claims (iss, sub, etc.) per RFC 9597
 * @param detached - Whether to create detached signature
 */
export async function signHashEnvelope(
  artifact: Uint8Array,
  options: HashEnvelopeOptions,
  privateKey: CryptoKey,
  cwtClaims?: CWTClaimsSet,
  detached: boolean = false
): Promise<CoseSign1> {
  // Create hash envelope
  const envelope = await createHashEnvelope(artifact, options);

  // Build protected headers with hash envelope labels
  const headersMap = new Map<number, any>();

  // Set algorithm (ES256 for now)
  headersMap.set(CoseHeaderLabel.ALG, CoseAlgorithm.ES256);

  // Set hash envelope labels
  headersMap.set(CoseHeaderLabel.PAYLOAD_HASH_ALG, envelope.payloadHashAlg);

  if (envelope.preimageContentType) {
    headersMap.set(CoseHeaderLabel.PAYLOAD_PREIMAGE_CONTENT_TYPE, envelope.preimageContentType);
  }

  if (envelope.payloadLocation) {
    headersMap.set(CoseHeaderLabel.PAYLOAD_LOCATION, envelope.payloadLocation);
  }

  // Add CWT claims if provided
  if (cwtClaims) {
    headersMap.set(CoseHeaderLabel.CWT_CLAIMS, cwtClaims);
  }

  // Sign with hash as payload
  return await createCoseSign1(
    headersMap,
    envelope.payloadHash,
    privateKey,
    { detached }
  );
}

/**
 * Verify hash envelope in COSE Sign1
 * Verifies both signature and hash
 */
export async function verifyHashEnvelope(
  coseSign1: CoseSign1,
  artifact: Uint8Array,
  publicKey: CryptoKey
): Promise<HashEnvelopeVerificationResult> {
  try {
    // Extract hash envelope parameters
    const params = extractHashEnvelopeParams(coseSign1);

    // Verify signature
    const signatureValid = await verifyCoseSign1(coseSign1, publicKey);

    // Verify hash
    const computedHash = await hashData(artifact, params.payloadHashAlg);
    const expectedHash = coseSign1.payload;

    const hashValid = expectedHash !== null && areEqual(computedHash, expectedHash);

    return {
      signatureValid,
      hashValid,
    };
  } catch (error) {
    return {
      signatureValid: false,
      hashValid: false,
    };
  }
}

/**
 * Extract hash envelope parameters from COSE Sign1 protected headers
 */
export function extractHashEnvelopeParams(coseSign1: CoseSign1): HashEnvelope {
  const headers = getProtectedHeaders(coseSign1);

  const payloadHashAlg = headers.get(CoseHeaderLabel.PAYLOAD_HASH_ALG) as number | undefined;
  const preimageContentType = headers.get(CoseHeaderLabel.PAYLOAD_PREIMAGE_CONTENT_TYPE) as string | undefined;
  const payloadLocation = headers.get(CoseHeaderLabel.PAYLOAD_LOCATION) as string | undefined;

  if (!payloadHashAlg) {
    throw new Error("Missing payload_hash_alg (label 258) in protected headers");
  }

  if (!coseSign1.payload) {
    throw new Error("Missing payload (hash) in COSE Sign1");
  }

  return {
    payloadHash: coseSign1.payload,
    payloadHashAlg,
    preimageContentType,
    payloadLocation,
  };
}

/**
 * Get Web Crypto hash algorithm name from COSE algorithm
 */
function getWebCryptoHashAlgorithm(algorithm: number): string {
  switch (algorithm) {
    case CoseHashAlgorithm.SHA_256:
      return "SHA-256";
    case CoseHashAlgorithm.SHA_384:
      return "SHA-384";
    case CoseHashAlgorithm.SHA_512:
      return "SHA-512";
    default:
      throw new Error(`Unsupported hash algorithm: ${algorithm}`);
  }
}

/**
 * Compare two Uint8Arrays for equality
 */
function areEqual(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) {
    return false;
  }

  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) {
      return false;
    }
  }

  return true;
}
