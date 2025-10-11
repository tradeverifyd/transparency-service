/**
 * COSE Signer/Verifier Tests
 * Test suite for raw signature operations (HSM-ready pattern)
 */

import { describe, test, expect } from "bun:test";

describe("ES256 Signer", () => {
  test("can create signer from private key", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);

    expect(signer).toBeDefined();
    expect(signer.algorithm).toBe(-7); // ES256
  });

  test("can sign data", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);

    const data = new TextEncoder().encode("Hello, COSE!");
    const signature = await signer.sign(data);

    expect(signature).toBeInstanceOf(Uint8Array);
    expect(signature.length).toBeGreaterThan(0);
  });

  test("signature length is correct for ES256", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);

    const data = new TextEncoder().encode("Test data");
    const signature = await signer.sign(data);

    // ES256 signature is 64 bytes (r + s, each 32 bytes)
    expect(signature.length).toBe(64);
  });

  test("different data produces different signatures", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);

    const data1 = new TextEncoder().encode("Message 1");
    const data2 = new TextEncoder().encode("Message 2");

    const sig1 = await signer.sign(data1);
    const sig2 = await signer.sign(data2);

    expect(sig1).not.toEqual(sig2);
  });

  test("can set optional key ID", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const kid = new TextEncoder().encode("my-key-id");
    const signer = new ES256Signer(keyPair.privateKey, kid);

    expect(signer.keyId).toEqual(kid);
  });
});

describe("ES256 Verifier", () => {
  test("can create verifier from public key", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Verifier } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const verifier = new ES256Verifier(keyPair.publicKey);

    expect(verifier).toBeDefined();
    expect(verifier.algorithm).toBe(-7); // ES256
  });

  test("can verify valid signature", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const verifier = new ES256Verifier(keyPair.publicKey);

    const data = new TextEncoder().encode("Hello, COSE!");
    const signature = await signer.sign(data);

    const isValid = await verifier.verify(data, signature);

    expect(isValid).toBe(true);
  });

  test("rejects signature with wrong data", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const verifier = new ES256Verifier(keyPair.publicKey);

    const originalData = new TextEncoder().encode("Original message");
    const tamperedData = new TextEncoder().encode("Tampered message");

    const signature = await signer.sign(originalData);
    const isValid = await verifier.verify(tamperedData, signature);

    expect(isValid).toBe(false);
  });

  test("rejects signature from different key", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");

    const keyPair1 = await generateES256KeyPair();
    const keyPair2 = await generateES256KeyPair();

    const signer = new ES256Signer(keyPair1.privateKey);
    const verifier = new ES256Verifier(keyPair2.publicKey);

    const data = new TextEncoder().encode("Test data");
    const signature = await signer.sign(data);

    const isValid = await verifier.verify(data, signature);

    expect(isValid).toBe(false);
  });

  test("rejects tampered signature", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const verifier = new ES256Verifier(keyPair.publicKey);

    const data = new TextEncoder().encode("Test data");
    const signature = await signer.sign(data);

    // Tamper with signature
    signature[0] ^= 0xFF;

    const isValid = await verifier.verify(data, signature);

    expect(isValid).toBe(false);
  });

  test("verifier works with different key ID", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const kid1 = new TextEncoder().encode("key-1");
    const kid2 = new TextEncoder().encode("key-2");

    const signer = new ES256Signer(keyPair.privateKey, kid1);
    const verifier = new ES256Verifier(keyPair.publicKey, kid2);

    const data = new TextEncoder().encode("Test data");
    const signature = await signer.sign(data);

    // Verification should still work (kid is metadata, not cryptographic)
    const isValid = await verifier.verify(data, signature);

    expect(isValid).toBe(true);
  });
});

describe("Signer/Verifier Round-trip", () => {
  test("multiple sign/verify cycles work correctly", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const verifier = new ES256Verifier(keyPair.publicKey);

    for (let i = 0; i < 10; i++) {
      const data = new TextEncoder().encode(`Message ${i}`);
      const signature = await signer.sign(data);
      const isValid = await verifier.verify(data, signature);

      expect(isValid).toBe(true);
    }
  });

  test("handles large data correctly", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const verifier = new ES256Verifier(keyPair.publicKey);

    // 1MB of data
    const largeData = new Uint8Array(1024 * 1024);
    for (let i = 0; i < largeData.length; i++) {
      largeData[i] = i % 256;
    }

    const signature = await signer.sign(largeData);
    const isValid = await verifier.verify(largeData, signature);

    expect(isValid).toBe(true);
  });

  test("handles empty data correctly", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const verifier = new ES256Verifier(keyPair.publicKey);

    const emptyData = new Uint8Array(0);
    const signature = await signer.sign(emptyData);
    const isValid = await verifier.verify(emptyData, signature);

    expect(isValid).toBe(true);
  });
});
