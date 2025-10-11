/**
 * COSE Hash Envelope Implementation
 * Streaming hash computation for large files with COSE Sign1 integration
 * Labels: 258 (hash alg), 259 (content type), 260 (location)
 */

import { COSEAlgorithm, COSEHashEnvelopeLabel } from "../../types/cose.ts";
import type { COSESign1, COSEHashEnvelopeProtectedHeader, Signer, Verifier } from "../../types/cose.ts";
import { signCOSESign1, verifyCOSESign1, decodeProtectedHeaders } from "./sign.ts";
import * as fs from "fs";

/**
 * Hash Envelope Structure
 */
export interface HashEnvelope {
  payloadHash: Uint8Array;
  payloadHashAlg: COSEAlgorithm;
  preimageContentType?: string;
  payloadLocation?: string;
}

/**
 * Hash Envelope Options
 */
export interface HashEnvelopeOptions {
  contentType?: string;
  location?: string;
  hashAlgorithm?: COSEAlgorithm;
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
  const hashAlgorithm = options.hashAlgorithm ?? COSEAlgorithm.SHA_256;
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
  algorithm: COSEAlgorithm = COSEAlgorithm.SHA_256
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
  algorithm: COSEAlgorithm = COSEAlgorithm.SHA_256
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
 */
export async function signHashEnvelope(
  artifact: Uint8Array,
  options: HashEnvelopeOptions,
  signer: Signer,
  additionalHeaders: Record<string, unknown> = {},
  detached: boolean = false
): Promise<COSESign1> {
  // Create hash envelope
  const envelope = await createHashEnvelope(artifact, options);

  // Build protected headers with hash envelope labels
  const protectedHeaders: Partial<COSEHashEnvelopeProtectedHeader> = {
    alg: signer.algorithm,
    258: envelope.payloadHashAlg, // payload_hash_alg
    ...additionalHeaders,
  };

  if (envelope.preimageContentType) {
    protectedHeaders[259] = envelope.preimageContentType; // preimage_content_type
  }

  if (envelope.payloadLocation) {
    protectedHeaders[260] = envelope.payloadLocation; // payload_location
  }

  // Sign with hash as payload
  return await signCOSESign1(
    envelope.payloadHash,
    protectedHeaders,
    signer,
    {},
    detached
  );
}

/**
 * Verify hash envelope in COSE Sign1
 * Verifies both signature and hash
 */
export async function verifyHashEnvelope(
  coseSign1: COSESign1,
  artifact: Uint8Array,
  verifier: Verifier
): Promise<HashEnvelopeVerificationResult> {
  try {
    // Extract hash envelope parameters
    const params = extractHashEnvelopeParams(coseSign1);

    // Verify signature
    const signatureValid = await verifyCOSESign1(coseSign1, verifier);

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
export function extractHashEnvelopeParams(coseSign1: COSESign1): HashEnvelope {
  const headers = decodeProtectedHeaders(coseSign1);

  const payloadHashAlg = headers[258] as COSEAlgorithm | undefined;
  const preimageContentType = headers[259] as string | undefined;
  const payloadLocation = headers[260] as string | undefined;

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
function getWebCryptoHashAlgorithm(algorithm: COSEAlgorithm): string {
  switch (algorithm) {
    case COSEAlgorithm.SHA_256:
      return "SHA-256";
    case COSEAlgorithm.SHA_384:
      return "SHA-384";
    case COSEAlgorithm.SHA_512:
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
