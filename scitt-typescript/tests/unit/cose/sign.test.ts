/**
 * Unit tests for COSE Sign1 signing and verification
 *
 * COSE Sign1 uses integer labels for headers (RFC 9052)
 */

import { describe, test, expect } from "bun:test";
import {
  createCoseSign1,
  verifyCoseSign1,
  createProtectedHeaders,
  createCWTClaims,
} from "../../../src/lib/cose/sign.ts";
import { generateES256KeyPair } from "../../../src/lib/cose/signer.ts";
import { CoseAlgorithm } from "../../../src/lib/cose/constants.ts";

describe("COSE Sign1 with Integer Labels", () => {
  test("createCoseSign1 creates valid COSE Sign1 structure", async () => {
    const keyPair = await generateES256KeyPair();
    const payload = new TextEncoder().encode("Hello, SCITT!");

    const cwtClaims = createCWTClaims({
      iss: "https://example.com/issuer",
      sub: "device-123",
    });

    const protectedHeaders = createProtectedHeaders({
      alg: CoseAlgorithm.ES256,
      kid: "key-1",
      cwtClaims,
    });

    const coseSign1 = await createCoseSign1(protectedHeaders, payload, keyPair.privateKey);

    expect(coseSign1).toBeDefined();
    expect(coseSign1.protected).toBeDefined();
    expect(coseSign1.signature.length).toBe(64);
  });

  test("verifyCoseSign1 verifies valid signature", async () => {
    const keyPair = await generateES256KeyPair();
    const payload = new TextEncoder().encode("Test message");

    const cwtClaims = createCWTClaims({
      iss: "https://example.com/issuer",
    });

    const protectedHeaders = createProtectedHeaders({
      alg: CoseAlgorithm.ES256,
      kid: "key-1",
      cwtClaims,
    });

    const coseSign1 = await createCoseSign1(protectedHeaders, payload, keyPair.privateKey);
    const isValid = await verifyCoseSign1(coseSign1, keyPair.publicKey);
    expect(isValid).toBe(true);
  });

  test("verifyCoseSign1 rejects invalid signature", async () => {
    const keyPair = await generateES256KeyPair();
    const payload = new TextEncoder().encode("Test message");

    const protectedHeaders = createProtectedHeaders({
      alg: CoseAlgorithm.ES256,
      kid: "key-1",
    });

    const coseSign1 = await createCoseSign1(protectedHeaders, payload, keyPair.privateKey);
    coseSign1.signature[0] ^= 0xFF;

    const isValid = await verifyCoseSign1(coseSign1, keyPair.publicKey);
    expect(isValid).toBe(false);
  });

  test("verifyCoseSign1 rejects signature from different key", async () => {
    const keyPair1 = await generateES256KeyPair();
    const keyPair2 = await generateES256KeyPair();
    const payload = new TextEncoder().encode("Test message");

    const protectedHeaders = createProtectedHeaders({
      alg: CoseAlgorithm.ES256,
      kid: "key-1",
    });

    const coseSign1 = await createCoseSign1(protectedHeaders, payload, keyPair1.privateKey);
    const isValid = await verifyCoseSign1(coseSign1, keyPair2.publicKey);
    expect(isValid).toBe(false);
  });

  test("createCoseSign1 supports detached payload mode", async () => {
    const keyPair = await generateES256KeyPair();
    const payload = new TextEncoder().encode("Detached payload");

    const protectedHeaders = createProtectedHeaders({
      alg: CoseAlgorithm.ES256,
      kid: "key-1",
    });

    const coseSign1 = await createCoseSign1(
      protectedHeaders,
      payload,
      keyPair.privateKey,
      { detached: true }
    );

    expect(coseSign1.payload).toBe(null);
    expect(coseSign1.signature).toBeInstanceOf(Uint8Array);
  });

  test("verifyCoseSign1 verifies detached payload signatures", async () => {
    const keyPair = await generateES256KeyPair();
    const payload = new TextEncoder().encode("Detached payload");

    const protectedHeaders = createProtectedHeaders({
      alg: CoseAlgorithm.ES256,
      kid: "key-1",
    });

    const coseSign1 = await createCoseSign1(
      protectedHeaders,
      payload,
      keyPair.privateKey,
      { detached: true }
    );

    const isValid = await verifyCoseSign1(coseSign1, keyPair.publicKey, payload);
    expect(isValid).toBe(true);
  });

  test("createCoseSign1 supports CWT claims in headers (RFC 9597)", async () => {
    const keyPair = await generateES256KeyPair();
    const payload = new TextEncoder().encode("Message with CWT claims");

    const cwtClaims = createCWTClaims({
      iss: "https://issuer.example.com",
      sub: "user@example.com",
      aud: "https://audience.example.com",
      exp: Math.floor(Date.now() / 1000) + 3600,
      iat: Math.floor(Date.now() / 1000),
    });

    const protectedHeaders = createProtectedHeaders({
      alg: CoseAlgorithm.ES256,
      kid: "key-1",
      cwtClaims,
    });

    const coseSign1 = await createCoseSign1(protectedHeaders, payload, keyPair.privateKey);
    const isValid = await verifyCoseSign1(coseSign1, keyPair.publicKey);

    expect(isValid).toBe(true);
    expect(coseSign1.signature).toBeInstanceOf(Uint8Array);
  });
});
