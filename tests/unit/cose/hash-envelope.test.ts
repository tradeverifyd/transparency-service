/**
 * COSE Hash Envelope Tests
 * Test suite for hash envelope creation and validation (labels 258, 259, 260)
 */

import { describe, test, expect, beforeEach, afterEach } from "bun:test";
import { CoseHashAlgorithm } from "../../../src/lib/cose/constants.ts";
import * as fs from "fs";
import * as path from "path";

describe("Hash Envelope Creation", () => {
  const testDir = "./.test-hash-envelope";

  beforeEach(() => {
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true });
    }
    fs.mkdirSync(testDir, { recursive: true });
  });

  afterEach(() => {
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true });
    }
  });

  test("can create hash envelope from small data", async () => {
    const { createHashEnvelope } = await import("../../../src/lib/cose/hash-envelope.ts");

    const data = new TextEncoder().encode("Hello, hash envelope!");
    const envelope = await createHashEnvelope(data, {
      contentType: "text/plain",
    });

    expect(envelope.payloadHash).toBeInstanceOf(Uint8Array);
    expect(envelope.payloadHash.length).toBe(32); // SHA-256 = 32 bytes
    expect(envelope.payloadHashAlg).toBe(CoseHashAlgorithm.SHA_256);
    expect(envelope.preimageContentType).toBe("text/plain");
  });

  test("can create hash envelope with payload location", async () => {
    const { createHashEnvelope } = await import("../../../src/lib/cose/hash-envelope.ts");

    const data = new TextEncoder().encode("Test data");
    const envelope = await createHashEnvelope(data, {
      contentType: "application/json",
      location: "https://example.com/data.json",
    });

    expect(envelope.payloadLocation).toBe("https://example.com/data.json");
  });

  test("can stream hash from file", async () => {
    const { streamHashFromFile } = await import("../../../src/lib/cose/hash-envelope.ts");

    // Create test file
    const testFile = path.join(testDir, "test.bin");
    const testData = new Uint8Array(1024);
    for (let i = 0; i < testData.length; i++) {
      testData[i] = i % 256;
    }
    fs.writeFileSync(testFile, testData);

    const hash = await streamHashFromFile(testFile);

    expect(hash).toBeInstanceOf(Uint8Array);
    expect(hash.length).toBe(32); // SHA-256
  });

  test("can stream hash from large file", async () => {
    const { streamHashFromFile } = await import("../../../src/lib/cose/hash-envelope.ts");

    // Create 10MB test file
    const testFile = path.join(testDir, "large.bin");
    const chunkSize = 1024 * 1024; // 1MB chunks
    const file = fs.openSync(testFile, "w");

    for (let i = 0; i < 10; i++) {
      const chunk = new Uint8Array(chunkSize);
      for (let j = 0; j < chunkSize; j++) {
        chunk[j] = (i * chunkSize + j) % 256;
      }
      fs.writeSync(file, chunk);
    }
    fs.closeSync(file);

    const hash = await streamHashFromFile(testFile);

    expect(hash).toBeInstanceOf(Uint8Array);
    expect(hash.length).toBe(32);

    // Verify file size
    const stats = fs.statSync(testFile);
    expect(stats.size).toBe(10 * chunkSize);
  });

  test("streaming hash matches direct hash", async () => {
    const { streamHashFromFile, hashData } = await import("../../../src/lib/cose/hash-envelope.ts");

    // Create test file
    const testFile = path.join(testDir, "test.bin");
    const testData = new Uint8Array(10000);
    for (let i = 0; i < testData.length; i++) {
      testData[i] = i % 256;
    }
    fs.writeFileSync(testFile, testData);

    const streamHash = await streamHashFromFile(testFile);
    const directHash = await hashData(testData);

    expect(streamHash).toEqual(directHash);
  });

  test("can create envelope with custom hash algorithm", async () => {
    const { createHashEnvelope } = await import("../../../src/lib/cose/hash-envelope.ts");

    const data = new TextEncoder().encode("Test");
    const envelope = await createHashEnvelope(data, {
      contentType: "text/plain",
      hashAlgorithm: CoseHashAlgorithm.SHA_256,
    });

    expect(envelope.payloadHashAlg).toBe(CoseHashAlgorithm.SHA_256);
  });

  test("handles empty data", async () => {
    const { createHashEnvelope } = await import("../../../src/lib/cose/hash-envelope.ts");

    const data = new Uint8Array(0);
    const envelope = await createHashEnvelope(data, {
      contentType: "application/octet-stream",
    });

    expect(envelope.payloadHash).toBeInstanceOf(Uint8Array);
    expect(envelope.payloadHash.length).toBe(32);
  });
});

describe("Hash Envelope Validation", () => {
  test("can validate hash envelope", async () => {
    const { createHashEnvelope, validateHashEnvelope } = await import("../../../src/lib/cose/hash-envelope.ts");

    const data = new TextEncoder().encode("Test data");
    const envelope = await createHashEnvelope(data, {
      contentType: "text/plain",
    });

    const isValid = await validateHashEnvelope(envelope, data);

    expect(isValid).toBe(true);
  });

  test("rejects tampered data", async () => {
    const { createHashEnvelope, validateHashEnvelope } = await import("../../../src/lib/cose/hash-envelope.ts");

    const originalData = new TextEncoder().encode("Original data");
    const envelope = await createHashEnvelope(originalData, {
      contentType: "text/plain",
    });

    const tamperedData = new TextEncoder().encode("Tampered data");
    const isValid = await validateHashEnvelope(envelope, tamperedData);

    expect(isValid).toBe(false);
  });

  test("rejects tampered hash", async () => {
    const { createHashEnvelope, validateHashEnvelope } = await import("../../../src/lib/cose/hash-envelope.ts");

    const data = new TextEncoder().encode("Test data");
    const envelope = await createHashEnvelope(data, {
      contentType: "text/plain",
    });

    // Tamper with hash
    envelope.payloadHash[0] ^= 0xFF;

    const isValid = await validateHashEnvelope(envelope, data);

    expect(isValid).toBe(false);
  });

  test("validates algorithm mismatch", async () => {
    const { createHashEnvelope, validateHashEnvelope } = await import("../../../src/lib/cose/hash-envelope.ts");

    const data = new TextEncoder().encode("Test data");
    const envelope = await createHashEnvelope(data, {
      contentType: "text/plain",
    });

    // Change algorithm
    envelope.payloadHashAlg = CoseHashAlgorithm.SHA_384;

    const isValid = await validateHashEnvelope(envelope, data);

    expect(isValid).toBe(false);
  });
});

describe("Hash Envelope with COSE Sign1", () => {
  test("can create signed hash envelope", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");
    const { signHashEnvelope } = await import("../../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();

    const cwtClaims = createCWTClaims({
      iss: "https://example.com/issuer",
    });

    const data = new TextEncoder().encode("Test data");
    const coseSign1 = await signHashEnvelope(
      data,
      {
        contentType: "text/plain",
        location: "https://example.com/data",
      },
      keyPair.privateKey,
      cwtClaims
    );

    expect(coseSign1.protected).toBeInstanceOf(Uint8Array);
    expect(coseSign1.payload).toBeInstanceOf(Uint8Array);
    expect(coseSign1.payload.length).toBe(32); // SHA-256 hash
    expect(coseSign1.signature).toBeInstanceOf(Uint8Array);
  });

  test("can verify signed hash envelope", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");
    const { signHashEnvelope, verifyHashEnvelope } = await import("../../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();

    const cwtClaims = createCWTClaims({
      iss: "https://example.com/issuer",
    });

    const data = new TextEncoder().encode("Test data");
    const coseSign1 = await signHashEnvelope(
      data,
      { contentType: "text/plain" },
      keyPair.privateKey,
      cwtClaims
    );

    const result = await verifyHashEnvelope(coseSign1, data, keyPair.publicKey);

    expect(result.signatureValid).toBe(true);
    expect(result.hashValid).toBe(true);
  });

  test("detects tampered artifact in signed envelope", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");
    const { signHashEnvelope, verifyHashEnvelope } = await import("../../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();

    const cwtClaims = createCWTClaims({
      iss: "https://example.com/issuer",
    });

    const originalData = new TextEncoder().encode("Original data");
    const coseSign1 = await signHashEnvelope(
      originalData,
      { contentType: "text/plain" },
      keyPair.privateKey,
      cwtClaims
    );

    const tamperedData = new TextEncoder().encode("Tampered data");
    const result = await verifyHashEnvelope(coseSign1, tamperedData, keyPair.publicKey);

    expect(result.signatureValid).toBe(true); // Signature is still valid
    expect(result.hashValid).toBe(false); // But hash doesn't match
  });

  test("can extract hash envelope parameters from protected header", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");
    const { signHashEnvelope, extractHashEnvelopeParams } = await import("../../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../../src/lib/cose/sign.ts");
    const { CoseHashAlgorithm } = await import("../../../src/lib/cose/constants.ts");

    const keyPair = await generateES256KeyPair();

    const cwtClaims = createCWTClaims({
      iss: "https://example.com/issuer",
    });

    const data = new TextEncoder().encode("Test data");
    const coseSign1 = await signHashEnvelope(
      data,
      {
        contentType: "application/json",
        location: "https://example.com/data.json",
      },
      keyPair.privateKey,
      cwtClaims
    );

    const params = extractHashEnvelopeParams(coseSign1);

    expect(params.payloadHashAlg).toBe(CoseHashAlgorithm.SHA_256);
    expect(params.preimageContentType).toBe("application/json");
    expect(params.payloadLocation).toBe("https://example.com/data.json");
  });
});
