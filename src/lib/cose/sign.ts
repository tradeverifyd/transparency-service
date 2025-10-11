/**
 * COSE Sign1 Implementation
 * Single signer digital signature with CBOR encoding
 */

import type { COSESign1, COSEProtectedHeader, COSEUnprotectedHeader, Signer, Verifier } from "../../types/cose.ts";
import { encode as cborEncode, decode as cborDecode } from "cbor-x";

/**
 * Sign data using COSE Sign1
 *
 * @param payload - Data to sign
 * @param protectedHeaders - Protected header parameters (will be CBOR-encoded)
 * @param signer - Signer instance
 * @param unprotectedHeaders - Unprotected header parameters (optional)
 * @param detached - If true, payload will be null in result (detached signature)
 * @returns COSE Sign1 structure
 */
export async function signCOSESign1(
  payload: Uint8Array,
  protectedHeaders: Partial<COSEProtectedHeader>,
  signer: Signer,
  unprotectedHeaders: COSEUnprotectedHeader = {},
  detached: boolean = false
): Promise<COSESign1> {
  // Encode protected headers as CBOR
  const protectedEncoded = new Uint8Array(cborEncode(protectedHeaders));

  // Create Sig_structure for signing
  // Sig_structure = [
  //   context = "Signature1",
  //   body_protected = protected headers (encoded),
  //   external_aad = empty,
  //   payload = payload
  // ]
  const sigStructure = [
    "Signature1",
    protectedEncoded,
    new Uint8Array(0), // external_aad (empty)
    payload,
  ];

  const sigStructureEncoded = new Uint8Array(cborEncode(sigStructure));

  // Sign the Sig_structure
  const signature = await signer.sign(sigStructureEncoded);

  // Return COSE Sign1
  return {
    protected: protectedEncoded,
    unprotected: unprotectedHeaders,
    payload: detached ? null : payload,
    signature,
  };
}

/**
 * Verify COSE Sign1 signature
 *
 * @param coseSign1 - COSE Sign1 structure
 * @param verifier - Verifier instance
 * @param externalPayload - External payload (for detached signatures)
 * @returns True if signature is valid
 */
export async function verifyCOSESign1(
  coseSign1: COSESign1,
  verifier: Verifier,
  externalPayload?: Uint8Array
): Promise<boolean> {
  try {
    // Determine payload (embedded or external)
    const payload = coseSign1.payload !== null ? coseSign1.payload : externalPayload;

    if (!payload) {
      throw new Error("Payload required for verification (detached signature needs external payload)");
    }

    // Reconstruct Sig_structure
    const sigStructure = [
      "Signature1",
      coseSign1.protected,
      new Uint8Array(0), // external_aad (empty)
      payload,
    ];

    const sigStructureEncoded = new Uint8Array(cborEncode(sigStructure));

    // Verify signature
    return await verifier.verify(sigStructureEncoded, coseSign1.signature);
  } catch (error) {
    return false;
  }
}

/**
 * Encode COSE Sign1 to CBOR array format
 * COSE_Sign1 = [
 *   protected: bstr,
 *   unprotected: {},
 *   payload: bstr / nil,
 *   signature: bstr
 * ]
 */
export function encodeCOSESign1(coseSign1: COSESign1): Uint8Array {
  const array = [
    coseSign1.protected,
    coseSign1.unprotected,
    coseSign1.payload,
    coseSign1.signature,
  ];

  return new Uint8Array(cborEncode(array));
}

/**
 * Decode COSE Sign1 from CBOR array format
 */
export function decodeCOSESign1(encoded: Uint8Array): COSESign1 {
  const array = cborDecode(encoded) as [Uint8Array, COSEUnprotectedHeader, Uint8Array | null, Uint8Array];

  if (!Array.isArray(array) || array.length !== 4) {
    throw new Error("Invalid COSE Sign1 structure");
  }

  return {
    protected: array[0],
    unprotected: array[1],
    payload: array[2],
    signature: array[3],
  };
}

/**
 * Decode protected headers from COSE Sign1
 */
export function decodeProtectedHeaders(coseSign1: COSESign1): Partial<COSEProtectedHeader> {
  return cborDecode(coseSign1.protected) as Partial<COSEProtectedHeader>;
}
