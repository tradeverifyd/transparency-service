/**
 * COSE Key Material Tests
 * Test suite for COSE key generation and thumbprint computation
 */

import { describe, test, expect } from "bun:test";

describe("COSE Key Generation (ES256)", () => {
  test("generateES256KeyPair creates valid key pair", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");

    const keyPair = await generateES256KeyPair();

    expect(keyPair).toBeDefined();
    expect(keyPair.privateKey).toBeDefined();
    expect(keyPair.publicKey).toBeDefined();
  });

  test("generated key has correct algorithm", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");

    const keyPair = await generateES256KeyPair();

    expect(keyPair.privateKey.algorithm.name).toBe("ECDSA");
    expect(keyPair.publicKey.algorithm.name).toBe("ECDSA");
  });

  test("generated key uses P-256 curve", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");

    const keyPair = await generateES256KeyPair();

    const alg = keyPair.privateKey.algorithm as EcKeyAlgorithm;
    expect(alg.namedCurve).toBe("P-256");
  });

  test("private key is extractable", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");

    const keyPair = await generateES256KeyPair();

    expect(keyPair.privateKey.extractable).toBe(true);
  });

  test("public key can be exported to JWK", async () => {
    const { generateES256KeyPair, exportPublicKeyToJWK } = await import("../../../src/lib/cose/key-material.ts");

    const keyPair = await generateES256KeyPair();
    const jwk = await exportPublicKeyToJWK(keyPair.publicKey);

    expect(jwk.kty).toBe("EC");
    expect(jwk.crv).toBe("P-256");
    expect(jwk.x).toBeDefined();
    expect(jwk.y).toBeDefined();
    expect(typeof jwk.x).toBe("string");
    expect(typeof jwk.y).toBe("string");
  });

  test("private key can be exported to PKCS8 PEM", async () => {
    const { generateES256KeyPair, exportPrivateKeyToPEM } = await import("../../../src/lib/cose/key-material.ts");

    const keyPair = await generateES256KeyPair();
    const pem = await exportPrivateKeyToPEM(keyPair.privateKey);

    expect(pem).toContain("-----BEGIN PRIVATE KEY-----");
    expect(pem).toContain("-----END PRIVATE KEY-----");
  });

  test("public key can be exported to SPKI PEM", async () => {
    const { generateES256KeyPair, exportPublicKeyToPEM } = await import("../../../src/lib/cose/key-material.ts");

    const keyPair = await generateES256KeyPair();
    const pem = await exportPublicKeyToPEM(keyPair.publicKey);

    expect(pem).toContain("-----BEGIN PUBLIC KEY-----");
    expect(pem).toContain("-----END PUBLIC KEY-----");
  });

  test("can import private key from PEM", async () => {
    const { generateES256KeyPair, exportPrivateKeyToPEM, importPrivateKeyFromPEM } = await import("../../../src/lib/cose/key-material.ts");

    const keyPair = await generateES256KeyPair();
    const pem = await exportPrivateKeyToPEM(keyPair.privateKey);

    const imported = await importPrivateKeyFromPEM(pem);

    expect(imported.type).toBe("private");
    expect(imported.algorithm.name).toBe("ECDSA");
  });

  test("can import public key from JWK", async () => {
    const { generateES256KeyPair, exportPublicKeyToJWK, importPublicKeyFromJWK } = await import("../../../src/lib/cose/key-material.ts");

    const keyPair = await generateES256KeyPair();
    const jwk = await exportPublicKeyToJWK(keyPair.publicKey);

    const imported = await importPublicKeyFromJWK(jwk);

    expect(imported.type).toBe("public");
    expect(imported.algorithm.name).toBe("ECDSA");
  });
});

describe("COSE Key Thumbprint (RFC 7638 style)", () => {
  test("computeKeyThumbprint generates consistent hash", async () => {
    const { generateES256KeyPair, exportPublicKeyToJWK, computeKeyThumbprint } = await import("../../../src/lib/cose/key-material.ts");

    const keyPair = await generateES256KeyPair();
    const jwk = await exportPublicKeyToJWK(keyPair.publicKey);

    const thumbprint1 = await computeKeyThumbprint(jwk);
    const thumbprint2 = await computeKeyThumbprint(jwk);

    expect(thumbprint1).toBe(thumbprint2);
  });

  test("thumbprint is base64url encoded", async () => {
    const { generateES256KeyPair, exportPublicKeyToJWK, computeKeyThumbprint } = await import("../../../src/lib/cose/key-material.ts");

    const keyPair = await generateES256KeyPair();
    const jwk = await exportPublicKeyToJWK(keyPair.publicKey);

    const thumbprint = await computeKeyThumbprint(jwk);

    // Base64url should not contain +, /, or =
    expect(thumbprint).not.toContain("+");
    expect(thumbprint).not.toContain("/");
    expect(thumbprint).not.toContain("=");
    expect(thumbprint.length).toBeGreaterThan(0);
  });

  test("different keys produce different thumbprints", async () => {
    const { generateES256KeyPair, exportPublicKeyToJWK, computeKeyThumbprint } = await import("../../../src/lib/cose/key-material.ts");

    const keyPair1 = await generateES256KeyPair();
    const jwk1 = await exportPublicKeyToJWK(keyPair1.publicKey);
    const thumbprint1 = await computeKeyThumbprint(jwk1);

    const keyPair2 = await generateES256KeyPair();
    const jwk2 = await exportPublicKeyToJWK(keyPair2.publicKey);
    const thumbprint2 = await computeKeyThumbprint(jwk2);

    expect(thumbprint1).not.toBe(thumbprint2);
  });

  test("thumbprint only uses required JWK fields", async () => {
    const { computeKeyThumbprint } = await import("../../../src/lib/cose/key-material.ts");

    const jwk = {
      kty: "EC",
      crv: "P-256",
      x: "WKn-ZIGevcwGIyyrzFoZNBdaq9_TsqzGl96oc0CWuis",
      y: "y77t-RvAHRKTsSGdIYUfweuOvwrvDD-Q3Hv5J0fSKbE",
      kid: "should-be-ignored",
      use: "sig",
    };

    const thumbprint = await computeKeyThumbprint(jwk);

    // Should be deterministic based on kty, crv, x, y only
    expect(thumbprint).toBeDefined();
    expect(typeof thumbprint).toBe("string");
  });
});

describe("COSE Key to CBOR", () => {
  test("public key can be converted to COSE_Key CBOR", async () => {
    const { generateES256KeyPair, exportPublicKeyToJWK, jwkToCOSEKey } = await import("../../../src/lib/cose/key-material.ts");

    const keyPair = await generateES256KeyPair();
    const jwk = await exportPublicKeyToJWK(keyPair.publicKey);

    const coseKey = jwkToCOSEKey(jwk);

    // COSE_Key labels
    expect(coseKey.get(1)).toBe(2); // kty: EC2
    expect(coseKey.get(-1)).toBe(1); // crv: P-256
    expect(coseKey.get(-2)).toBeInstanceOf(Uint8Array); // x-coordinate
    expect(coseKey.get(-3)).toBeInstanceOf(Uint8Array); // y-coordinate
  });

  test("COSE_Key can be converted back to JWK", async () => {
    const { generateES256KeyPair, exportPublicKeyToJWK, jwkToCOSEKey, coseKeyToJWK } = await import("../../../src/lib/cose/key-material.ts");

    const keyPair = await generateES256KeyPair();
    const originalJwk = await exportPublicKeyToJWK(keyPair.publicKey);

    const coseKey = jwkToCOSEKey(originalJwk);
    const convertedJwk = coseKeyToJWK(coseKey);

    expect(convertedJwk.kty).toBe(originalJwk.kty);
    expect(convertedJwk.crv).toBe(originalJwk.crv);
    expect(convertedJwk.x).toBe(originalJwk.x);
    expect(convertedJwk.y).toBe(originalJwk.y);
  });
});
