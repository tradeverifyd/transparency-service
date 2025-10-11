/**
 * COSE Signer/Verifier Abstraction
 * Raw signature operations with HSM support pattern
 */

import type { Signer, Verifier } from "../../types/cose.ts";
import { COSEAlgorithm } from "../../types/cose.ts";

/**
 * ES256 Signer
 * Signs data using ECDSA P-256 with SHA-256
 * Supports HSM integration via CryptoKey interface
 */
export class ES256Signer implements Signer {
  readonly algorithm = COSEAlgorithm.ES256;

  constructor(
    private readonly privateKey: CryptoKey,
    public readonly keyId?: Uint8Array
  ) {
    if (privateKey.type !== "private") {
      throw new Error("ES256Signer requires a private key");
    }
    if (privateKey.algorithm.name !== "ECDSA") {
      throw new Error("ES256Signer requires an ECDSA key");
    }
  }

  /**
   * Sign data using ES256 (ECDSA P-256 with SHA-256)
   * Returns raw signature (r || s) as 64 bytes
   */
  async sign(data: Uint8Array): Promise<Uint8Array> {
    // Sign with ECDSA P-256 + SHA-256
    const signature = await crypto.subtle.sign(
      {
        name: "ECDSA",
        hash: "SHA-256",
      },
      this.privateKey,
      data
    );

    const sigBytes = new Uint8Array(signature);

    // Check if signature is already in raw format (Bun returns raw format)
    if (sigBytes.length === 64) {
      return sigBytes;
    }

    // Otherwise convert DER to raw format (for other runtimes)
    return this.derToRaw(sigBytes);
  }

  /**
   * Convert DER-encoded ECDSA signature to raw r||s format
   * DER format: 0x30 [total-length] 0x02 [r-length] [r] 0x02 [s-length] [s]
   * Raw format: [r] [s] (each 32 bytes for P-256)
   */
  private derToRaw(der: Uint8Array): Uint8Array {
    // Parse DER structure
    let offset = 0;

    // Skip SEQUENCE tag (0x30)
    if (der[offset] !== 0x30) {
      throw new Error("Invalid DER signature: missing SEQUENCE tag");
    }
    offset += 1;

    // Skip total length
    const totalLength = der[offset]!;
    offset += 1;

    // Parse r
    if (der[offset] !== 0x02) {
      throw new Error("Invalid DER signature: missing INTEGER tag for r");
    }
    offset += 1;

    const rLength = der[offset]!;
    offset += 1;

    let r = der.slice(offset, offset + rLength);
    offset += rLength;

    // Parse s
    if (der[offset] !== 0x02) {
      throw new Error("Invalid DER signature: missing INTEGER tag for s");
    }
    offset += 1;

    const sLength = der[offset]!;
    offset += 1;

    let s = der.slice(offset, offset + sLength);

    // Remove leading zeros if present (DER encoding may add them)
    while (r.length > 32 && r[0] === 0) {
      r = r.slice(1);
    }
    while (s.length > 32 && s[0] === 0) {
      s = s.slice(1);
    }

    // Pad to 32 bytes if needed
    if (r.length < 32) {
      const padded = new Uint8Array(32);
      padded.set(r, 32 - r.length);
      r = padded;
    }
    if (s.length < 32) {
      const padded = new Uint8Array(32);
      padded.set(s, 32 - s.length);
      s = padded;
    }

    // Concatenate r and s
    const raw = new Uint8Array(64);
    raw.set(r, 0);
    raw.set(s, 32);

    return raw;
  }
}

/**
 * ES256 Verifier
 * Verifies signatures using ECDSA P-256 with SHA-256
 */
export class ES256Verifier implements Verifier {
  readonly algorithm = COSEAlgorithm.ES256;

  constructor(
    private readonly publicKey: CryptoKey,
    public readonly keyId?: Uint8Array
  ) {
    if (publicKey.type !== "public") {
      throw new Error("ES256Verifier requires a public key");
    }
    if (publicKey.algorithm.name !== "ECDSA") {
      throw new Error("ES256Verifier requires an ECDSA key");
    }
  }

  /**
   * Verify signature using ES256 (ECDSA P-256 with SHA-256)
   * Expects raw signature (r || s) as 64 bytes
   */
  async verify(data: Uint8Array, signature: Uint8Array): Promise<boolean> {
    try {
      // Bun accepts raw format directly, try that first
      if (signature.length === 64) {
        try {
          const isValid = await crypto.subtle.verify(
            {
              name: "ECDSA",
              hash: "SHA-256",
            },
            this.publicKey,
            signature,
            data
          );
          return isValid;
        } catch {
          // If raw format fails, try DER format
        }
      }

      // Convert raw signature to DER format for other runtimes
      const derSignature = this.rawToDer(signature);

      // Verify with ECDSA P-256 + SHA-256
      const isValid = await crypto.subtle.verify(
        {
          name: "ECDSA",
          hash: "SHA-256",
        },
        this.publicKey,
        derSignature,
        data
      );

      return isValid;
    } catch (error) {
      // Verification errors (invalid signature format, etc.) return false
      return false;
    }
  }

  /**
   * Convert raw r||s signature to DER format
   * Raw format: [r] [s] (each 32 bytes for P-256)
   * DER format: 0x30 [total-length] 0x02 [r-length] [r] 0x02 [s-length] [s]
   */
  private rawToDer(raw: Uint8Array): Uint8Array {
    if (raw.length !== 64) {
      throw new Error("Invalid raw signature length (expected 64 bytes)");
    }

    let r = raw.slice(0, 32);
    let s = raw.slice(32, 64);

    // Add leading zero if high bit is set (to keep it positive in DER)
    if (r[0]! & 0x80) {
      const padded = new Uint8Array(33);
      padded[0] = 0;
      padded.set(r, 1);
      r = padded;
    }
    if (s[0]! & 0x80) {
      const padded = new Uint8Array(33);
      padded[0] = 0;
      padded.set(s, 1);
      s = padded;
    }

    // Remove leading zeros (except if it would make the number negative)
    while (r.length > 1 && r[0] === 0 && !(r[1]! & 0x80)) {
      r = r.slice(1);
    }
    while (s.length > 1 && s[0] === 0 && !(s[1]! & 0x80)) {
      s = s.slice(1);
    }

    // Build DER structure
    const totalLength = 2 + r.length + 2 + s.length;
    const der = new Uint8Array(2 + totalLength);

    let offset = 0;

    // SEQUENCE tag
    der[offset++] = 0x30;
    der[offset++] = totalLength;

    // r INTEGER
    der[offset++] = 0x02;
    der[offset++] = r.length;
    der.set(r, offset);
    offset += r.length;

    // s INTEGER
    der[offset++] = 0x02;
    der[offset++] = s.length;
    der.set(s, offset);

    return der;
  }
}
