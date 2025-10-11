/**
 * COSE Sign1 Tests
 * Test suite for COSE Sign1 signing and verification
 */

import { describe, test, expect } from "bun:test";
import { COSEAlgorithm } from "../../../src/types/cose.ts";

describe("COSE Sign1 Signing", () => {
  test("can create COSE Sign1 structure", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer } = await import("../../../src/lib/cose/signer.ts");
    const { signCOSESign1 } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);

    const payload = new TextEncoder().encode("Hello, COSE Sign1!");
    const protectedHeaders = { alg: COSEAlgorithm.ES256 };

    const coseSign1 = await signCOSESign1(payload, protectedHeaders, signer);

    expect(coseSign1.protected).toBeInstanceOf(Uint8Array);
    expect(coseSign1.unprotected).toBeDefined();
    expect(coseSign1.payload).toEqual(payload);
    expect(coseSign1.signature).toBeInstanceOf(Uint8Array);
    expect(coseSign1.signature.length).toBe(64); // ES256 signature
  });

  test("protected header is CBOR-encoded", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer } = await import("../../../src/lib/cose/signer.ts");
    const { signCOSESign1 } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);

    const payload = new TextEncoder().encode("Test");
    const protectedHeaders = { alg: COSEAlgorithm.ES256 };

    const coseSign1 = await signCOSESign1(payload, protectedHeaders, signer);

    // Protected header should be non-empty CBOR
    expect(coseSign1.protected.length).toBeGreaterThan(0);
  });

  test("can include custom headers", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer } = await import("../../../src/lib/cose/signer.ts");
    const { signCOSESign1 } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();
    const kid = new TextEncoder().encode("my-key-1");
    const signer = new ES256Signer(keyPair.privateKey, kid);

    const payload = new TextEncoder().encode("Test");
    const protectedHeaders = {
      alg: COSEAlgorithm.ES256,
      kid: kid,
      cty: "text/plain",
    };

    const coseSign1 = await signCOSESign1(payload, protectedHeaders, signer);

    expect(coseSign1.protected.length).toBeGreaterThan(0);
    expect(coseSign1.signature).toBeDefined();
  });

  test("can sign with detached payload", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer } = await import("../../../src/lib/cose/signer.ts");
    const { signCOSESign1 } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);

    const payload = new TextEncoder().encode("Detached payload");
    const protectedHeaders = { alg: COSEAlgorithm.ES256 };

    const coseSign1 = await signCOSESign1(
      payload,
      protectedHeaders,
      signer,
      {}, // unprotected
      true // detached
    );

    expect(coseSign1.payload).toBeNull();
    expect(coseSign1.signature).toBeInstanceOf(Uint8Array);
  });

  test("different payloads produce different signatures", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer } = await import("../../../src/lib/cose/signer.ts");
    const { signCOSESign1 } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const protectedHeaders = { alg: COSEAlgorithm.ES256 };

    const payload1 = new TextEncoder().encode("Message 1");
    const payload2 = new TextEncoder().encode("Message 2");

    const sign1_1 = await signCOSESign1(payload1, protectedHeaders, signer);
    const sign1_2 = await signCOSESign1(payload2, protectedHeaders, signer);

    expect(sign1_1.signature).not.toEqual(sign1_2.signature);
  });

  test("can encode to CBOR array format", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer } = await import("../../../src/lib/cose/signer.ts");
    const { signCOSESign1, encodeCOSESign1 } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);

    const payload = new TextEncoder().encode("Test");
    const protectedHeaders = { alg: COSEAlgorithm.ES256 };

    const coseSign1 = await signCOSESign1(payload, protectedHeaders, signer);
    const encoded = encodeCOSESign1(coseSign1);

    expect(encoded).toBeInstanceOf(Uint8Array);
    expect(encoded.length).toBeGreaterThan(0);
  });
});

describe("COSE Sign1 Verification", () => {
  test("can verify valid COSE Sign1", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");
    const { signCOSESign1, verifyCOSESign1 } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const verifier = new ES256Verifier(keyPair.publicKey);

    const payload = new TextEncoder().encode("Hello, COSE!");
    const protectedHeaders = { alg: COSEAlgorithm.ES256 };

    const coseSign1 = await signCOSESign1(payload, protectedHeaders, signer);
    const isValid = await verifyCOSESign1(coseSign1, verifier);

    expect(isValid).toBe(true);
  });

  test("rejects tampered payload", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");
    const { signCOSESign1, verifyCOSESign1 } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const verifier = new ES256Verifier(keyPair.publicKey);

    const payload = new TextEncoder().encode("Original message");
    const protectedHeaders = { alg: COSEAlgorithm.ES256 };

    const coseSign1 = await signCOSESign1(payload, protectedHeaders, signer);

    // Tamper with payload
    coseSign1.payload = new TextEncoder().encode("Tampered message");

    const isValid = await verifyCOSESign1(coseSign1, verifier);

    expect(isValid).toBe(false);
  });

  test("rejects tampered signature", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");
    const { signCOSESign1, verifyCOSESign1 } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const verifier = new ES256Verifier(keyPair.publicKey);

    const payload = new TextEncoder().encode("Test message");
    const protectedHeaders = { alg: COSEAlgorithm.ES256 };

    const coseSign1 = await signCOSESign1(payload, protectedHeaders, signer);

    // Tamper with signature
    coseSign1.signature[0] ^= 0xFF;

    const isValid = await verifyCOSESign1(coseSign1, verifier);

    expect(isValid).toBe(false);
  });

  test("rejects signature from wrong key", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");
    const { signCOSESign1, verifyCOSESign1 } = await import("../../../src/lib/cose/sign.ts");

    const keyPair1 = await generateES256KeyPair();
    const keyPair2 = await generateES256KeyPair();

    const signer = new ES256Signer(keyPair1.privateKey);
    const verifier = new ES256Verifier(keyPair2.publicKey);

    const payload = new TextEncoder().encode("Test message");
    const protectedHeaders = { alg: COSEAlgorithm.ES256 };

    const coseSign1 = await signCOSESign1(payload, protectedHeaders, signer);
    const isValid = await verifyCOSESign1(coseSign1, verifier);

    expect(isValid).toBe(false);
  });

  test("can verify detached payload", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");
    const { signCOSESign1, verifyCOSESign1 } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const verifier = new ES256Verifier(keyPair.publicKey);

    const payload = new TextEncoder().encode("Detached payload");
    const protectedHeaders = { alg: COSEAlgorithm.ES256 };

    const coseSign1 = await signCOSESign1(
      payload,
      protectedHeaders,
      signer,
      {},
      true // detached
    );

    expect(coseSign1.payload).toBeNull();

    // Verify with external payload
    const isValid = await verifyCOSESign1(coseSign1, verifier, payload);

    expect(isValid).toBe(true);
  });

  test("can decode from CBOR and verify", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");
    const { signCOSESign1, encodeCOSESign1, decodeCOSESign1, verifyCOSESign1 } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const verifier = new ES256Verifier(keyPair.publicKey);

    const payload = new TextEncoder().encode("Test");
    const protectedHeaders = { alg: COSEAlgorithm.ES256 };

    const coseSign1 = await signCOSESign1(payload, protectedHeaders, signer);
    const encoded = encodeCOSESign1(coseSign1);
    const decoded = decodeCOSESign1(encoded);

    const isValid = await verifyCOSESign1(decoded, verifier);

    expect(isValid).toBe(true);
  });
});

describe("COSE Sign1 Round-trip", () => {
  test("encode and decode preserves structure", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer } = await import("../../../src/lib/cose/signer.ts");
    const { signCOSESign1, encodeCOSESign1, decodeCOSESign1 } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);

    const payload = new TextEncoder().encode("Round-trip test");
    const protectedHeaders = { alg: COSEAlgorithm.ES256 };

    const original = await signCOSESign1(payload, protectedHeaders, signer);
    const encoded = encodeCOSESign1(original);
    const decoded = decodeCOSESign1(encoded);

    expect(decoded.protected).toEqual(original.protected);
    expect(decoded.payload).toEqual(original.payload);
    expect(decoded.signature).toEqual(original.signature);
  });

  test("handles large payloads correctly", async () => {
    const { generateES256KeyPair } = await import("../../../src/lib/cose/key-material.ts");
    const { ES256Signer, ES256Verifier } = await import("../../../src/lib/cose/signer.ts");
    const { signCOSESign1, verifyCOSESign1 } = await import("../../../src/lib/cose/sign.ts");

    const keyPair = await generateES256KeyPair();
    const signer = new ES256Signer(keyPair.privateKey);
    const verifier = new ES256Verifier(keyPair.publicKey);

    // 100KB payload
    const largePayload = new Uint8Array(100 * 1024);
    for (let i = 0; i < largePayload.length; i++) {
      largePayload[i] = i % 256;
    }

    const protectedHeaders = { alg: COSEAlgorithm.ES256 };
    const coseSign1 = await signCOSESign1(largePayload, protectedHeaders, signer);
    const isValid = await verifyCOSESign1(coseSign1, verifier);

    expect(isValid).toBe(true);
  });
});
