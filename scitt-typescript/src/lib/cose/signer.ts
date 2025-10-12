/**
 * Signer/Verifier abstraction for raw signature operations
 *
 * This module provides an abstraction layer for signature operations
 * that can support HSM integration in the future while using Web Crypto API now.
 *
 * Supports ES256 (ECDSA P-256 + SHA-256) as required by SCITT/COSE specifications.
 */

/**
 * Signer interface for creating signatures
 */
export interface Signer {
  sign(data: Uint8Array): Promise<Uint8Array>;
}

/**
 * Verifier interface for validating signatures
 */
export interface Verifier {
  verify(data: Uint8Array, signature: Uint8Array): Promise<boolean>;
}

/**
 * Generate an ES256 (ECDSA P-256) key pair using Web Crypto API
 */
export async function generateES256KeyPair(): Promise<CryptoKeyPair> {
  return await crypto.subtle.generateKey(
    {
      name: "ECDSA",
      namedCurve: "P-256",
    },
    true, // extractable
    ["sign", "verify"]
  );
}

/**
 * ES256 Signer implementation using Web Crypto API
 */
export class ES256Signer implements Signer {
  constructor(private privateKey: CryptoKey) {
    if (privateKey.type !== "private") {
      throw new Error("ES256Signer requires a private key");
    }
  }

  async sign(data: Uint8Array): Promise<Uint8Array> {
    const signature = await crypto.subtle.sign(
      {
        name: "ECDSA",
        hash: "SHA-256",
      },
      this.privateKey,
      data
    );

    return new Uint8Array(signature);
  }
}

/**
 * ES256 Verifier implementation using Web Crypto API
 */
export class ES256Verifier implements Verifier {
  constructor(private publicKey: CryptoKey) {
    if (publicKey.type !== "public") {
      throw new Error("ES256Verifier requires a public key");
    }
  }

  async verify(data: Uint8Array, signature: Uint8Array): Promise<boolean> {
    try {
      return await crypto.subtle.verify(
        {
          name: "ECDSA",
          hash: "SHA-256",
        },
        this.publicKey,
        signature,
        data
      );
    } catch (error) {
      // Invalid signature format or other errors
      return false;
    }
  }
}
