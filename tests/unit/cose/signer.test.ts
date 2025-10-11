/**
 * Unit tests for Signer/Verifier abstraction
 *
 * Tests raw signature creation and verification using the Signer/Verifier pattern.
 * This abstraction supports HSM integration in the future while using Web Crypto API for now.
 */

import { describe, test, expect } from "bun:test";
import {
  Signer,
  Verifier,
  ES256Signer,
  ES256Verifier,
  generateES256KeyPair,
} from "../../../src/lib/cose/signer.ts";

describe("ES256 Signer/Verifier", () => {
  test("generateES256KeyPair creates valid key pair", async () => {
    const keyPair = await generateES256KeyPair();

    expect(keyPair.privateKey).toBeDefined();
    expect(keyPair.publicKey).toBeDefined();
    expect(keyPair.privateKey.type).toBe("private");
    expect(keyPair.publicKey.type).toBe("public");
  });

  test("ES256Signer signs data and produces signature", async () => {
    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);

    const data = new TextEncoder().encode("Hello, COSE!");
    const signature = await signer.sign(data);

    expect(signature).toBeInstanceOf(Uint8Array);
    expect(signature.length).toBeGreaterThan(0);
    expect(signature.length).toBe(64); // ES256 = 64 bytes
  });

  test("ES256Verifier verifies valid signature", async () => {
    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const verifier = new ES256Verifier(keyPair.publicKey);

    const data = new TextEncoder().encode("Test message");
    const signature = await signer.sign(data);

    const isValid = await verifier.verify(data, signature);
    expect(isValid).toBe(true);
  });

  test("ES256Verifier rejects invalid signature", async () => {
    const keyPair = await generateES256KeyPair();
    const verifier = new ES256Verifier(keyPair.publicKey);

    const data = new TextEncoder().encode("Test message");
    const invalidSignature = new Uint8Array(64);

    const isValid = await verifier.verify(data, invalidSignature);
    expect(isValid).toBe(false);
  });

  test("ES256Verifier rejects signature from different key", async () => {
    const keyPair1 = await generateES256KeyPair();
    const keyPair2 = await generateES256KeyPair();

    const signer1 = new ES256Signer(keyPair1.privateKey);
    const verifier2 = new ES256Verifier(keyPair2.publicKey);

    const data = new TextEncoder().encode("Test message");
    const signature = await signer1.sign(data);

    const isValid = await verifier2.verify(data, signature);
    expect(isValid).toBe(false);
  });

  test("ES256Verifier rejects modified data", async () => {
    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const verifier = new ES256Verifier(keyPair.publicKey);

    const originalData = new TextEncoder().encode("Original message");
    const modifiedData = new TextEncoder().encode("Modified message");
    const signature = await signer.sign(originalData);

    const isValid = await verifier.verify(modifiedData, signature);
    expect(isValid).toBe(false);
  });
});
