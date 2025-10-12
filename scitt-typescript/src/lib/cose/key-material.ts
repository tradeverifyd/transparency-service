/**
 * COSE Key Material
 * Key generation, import/export, and thumbprint computation
 */

import { COSEAlgorithm, COSEKeyType, COSECurve } from "../../types/cose.ts";

/**
 * JWK (JSON Web Key) structure for ES256 keys
 */
export interface JWK {
  kty: string;
  crv: string;
  x: string;
  y?: string;
  d?: string;
  kid?: string;
  alg?: string;
  use?: string;
}

/**
 * Generate ES256 (ECDSA P-256 with SHA-256) key pair
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
 * Export public key to JWK format
 */
export async function exportPublicKeyToJWK(publicKey: CryptoKey): Promise<JWK> {
  const jwk = await crypto.subtle.exportKey("jwk", publicKey);
  return jwk as JWK;
}

/**
 * Export private key to PEM format (PKCS#8)
 */
export async function exportPrivateKeyToPEM(privateKey: CryptoKey): Promise<string> {
  const exported = await crypto.subtle.exportKey("pkcs8", privateKey);
  const exportedAsString = arrayBufferToBase64(exported);
  const pemKey = `-----BEGIN PRIVATE KEY-----\n${chunkString(exportedAsString, 64)}\n-----END PRIVATE KEY-----\n`;
  return pemKey;
}

/**
 * Export public key to PEM format (SPKI)
 */
export async function exportPublicKeyToPEM(publicKey: CryptoKey): Promise<string> {
  const exported = await crypto.subtle.exportKey("spki", publicKey);
  const exportedAsString = arrayBufferToBase64(exported);
  const pemKey = `-----BEGIN PUBLIC KEY-----\n${chunkString(exportedAsString, 64)}\n-----END PUBLIC KEY-----\n`;
  return pemKey;
}

/**
 * Import private key from PEM format (PKCS#8)
 */
export async function importPrivateKeyFromPEM(pem: string): Promise<CryptoKey> {
  // Remove PEM headers and whitespace
  const pemContents = pem
    .replace(/-----BEGIN PRIVATE KEY-----/, "")
    .replace(/-----END PRIVATE KEY-----/, "")
    .replace(/\s/g, "");

  const binaryDer = base64ToArrayBuffer(pemContents);

  return await crypto.subtle.importKey(
    "pkcs8",
    binaryDer,
    {
      name: "ECDSA",
      namedCurve: "P-256",
    },
    true,
    ["sign"]
  );
}

/**
 * Import public key from JWK format
 */
export async function importPublicKeyFromJWK(jwk: JWK): Promise<CryptoKey> {
  return await crypto.subtle.importKey(
    "jwk",
    jwk,
    {
      name: "ECDSA",
      namedCurve: "P-256",
    },
    true,
    ["verify"]
  );
}

/**
 * Compute JWK thumbprint (RFC 7638 style)
 * Uses SHA-256 hash of required JWK fields in lexicographic order
 */
export async function computeKeyThumbprint(jwk: JWK): Promise<string> {
  // Required fields for EC keys in lexicographic order
  const requiredFields: Record<string, unknown> = {
    crv: jwk.crv,
    kty: jwk.kty,
    x: jwk.x,
  };

  // Include y for EC keys
  if (jwk.y) {
    requiredFields.y = jwk.y;
  }

  // Serialize to JSON (maintains lexicographic order)
  const jsonString = JSON.stringify(requiredFields);

  // Compute SHA-256 hash
  const encoder = new TextEncoder();
  const data = encoder.encode(jsonString);
  const hashBuffer = await crypto.subtle.digest("SHA-256", data);

  // Encode as base64url
  return arrayBufferToBase64Url(hashBuffer);
}

/**
 * Convert JWK to COSE_Key format (CBOR map)
 */
export function jwkToCOSEKey(jwk: JWK): Map<number, unknown> {
  const coseKey = new Map<number, unknown>();

  // kty (1): Key Type
  coseKey.set(1, COSEKeyType.EC2);

  // alg (3): Algorithm (optional)
  if (jwk.alg === "ES256") {
    coseKey.set(3, COSEAlgorithm.ES256);
  }

  // kid (2): Key ID (optional)
  if (jwk.kid) {
    const encoder = new TextEncoder();
    coseKey.set(2, encoder.encode(jwk.kid));
  }

  // crv (-1): Curve
  if (jwk.crv === "P-256") {
    coseKey.set(-1, COSECurve.P_256);
  }

  // x (-2): x-coordinate
  if (jwk.x) {
    coseKey.set(-2, base64UrlToUint8Array(jwk.x));
  }

  // y (-3): y-coordinate
  if (jwk.y) {
    coseKey.set(-3, base64UrlToUint8Array(jwk.y));
  }

  // d (-4): private key (optional, not included for public keys)
  if (jwk.d) {
    coseKey.set(-4, base64UrlToUint8Array(jwk.d));
  }

  return coseKey;
}

/**
 * Convert COSE_Key to JWK format
 */
export function coseKeyToJWK(coseKey: Map<number, unknown>): JWK {
  const jwk: JWK = {
    kty: "EC",
    crv: "P-256",
    x: "",
    y: "",
  };

  // crv (-1): Curve
  const crv = coseKey.get(-1);
  if (crv === COSECurve.P_256) {
    jwk.crv = "P-256";
  }

  // x (-2): x-coordinate
  const x = coseKey.get(-2);
  if (x instanceof Uint8Array) {
    jwk.x = uint8ArrayToBase64Url(x);
  }

  // y (-3): y-coordinate
  const y = coseKey.get(-3);
  if (y instanceof Uint8Array) {
    jwk.y = uint8ArrayToBase64Url(y);
  }

  // kid (2): Key ID
  const kid = coseKey.get(2);
  if (kid instanceof Uint8Array) {
    const decoder = new TextDecoder();
    jwk.kid = decoder.decode(kid);
  }

  // alg (3): Algorithm
  const alg = coseKey.get(3);
  if (alg === COSEAlgorithm.ES256) {
    jwk.alg = "ES256";
  }

  return jwk;
}

// Utility functions

function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]!);
  }
  return btoa(binary);
}

function base64ToArrayBuffer(base64: string): ArrayBuffer {
  const binaryString = atob(base64);
  const bytes = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }
  return bytes.buffer;
}

function arrayBufferToBase64Url(buffer: ArrayBuffer): string {
  const base64 = arrayBufferToBase64(buffer);
  return base64
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

function base64UrlToUint8Array(base64url: string): Uint8Array {
  // Convert base64url to base64
  const base64 = base64url
    .replace(/-/g, "+")
    .replace(/_/g, "/");

  // Add padding if necessary
  const padding = (4 - (base64.length % 4)) % 4;
  const padded = base64 + "=".repeat(padding);

  return new Uint8Array(base64ToArrayBuffer(padded));
}

function uint8ArrayToBase64Url(bytes: Uint8Array): string {
  return arrayBufferToBase64Url(bytes.buffer);
}

function chunkString(str: string, size: number): string {
  const chunks: string[] = [];
  for (let i = 0; i < str.length; i += size) {
    chunks.push(str.substring(i, i + size));
  }
  return chunks.join("\n");
}
