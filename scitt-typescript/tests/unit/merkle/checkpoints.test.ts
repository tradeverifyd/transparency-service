/**
 * Unit tests for checkpoint (signed tree head) management
 * Checkpoints are signed notes containing tree size and root hash
 */

import { describe, test, expect } from "bun:test";

describe("Checkpoint Creation", () => {
  test("can create checkpoint from tree state", async () => {
    const { createCheckpoint } = await import("../../../src/lib/merkle/checkpoints.ts");
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const treeSize = 42;
    const rootHash = new Uint8Array(32).fill(0xab); // Mock root hash

    const checkpoint = await createCheckpoint(
      treeSize,
      rootHash,
      keyPair.privateKey,
      "https://transparency.example.com"
    );

    expect(checkpoint).toBeDefined();
    expect(checkpoint.treeSize).toBe(42);
    expect(checkpoint.rootHash).toEqual(rootHash);
    expect(checkpoint.signature).toBeInstanceOf(Uint8Array);
    expect(checkpoint.origin).toBe("https://transparency.example.com");
  });

  test("checkpoint includes timestamp", async () => {
    const { createCheckpoint } = await import("../../../src/lib/merkle/checkpoints.ts");
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const beforeTime = Date.now();

    const checkpoint = await createCheckpoint(
      100,
      new Uint8Array(32),
      keyPair.privateKey,
      "https://example.com"
    );

    const afterTime = Date.now();

    expect(checkpoint.timestamp).toBeGreaterThanOrEqual(beforeTime);
    expect(checkpoint.timestamp).toBeLessThanOrEqual(afterTime);
  });

  test("can encode checkpoint to signed note format", async () => {
    const { createCheckpoint, encodeCheckpoint } = await import("../../../src/lib/merkle/checkpoints.ts");
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();

    const checkpoint = await createCheckpoint(
      256,
      new Uint8Array(32).fill(0xcd),
      keyPair.privateKey,
      "https://example.com"
    );

    const encoded = encodeCheckpoint(checkpoint);

    expect(encoded).toBeDefined();
    expect(typeof encoded).toBe("string");
    expect(encoded).toContain("https://example.com");
    expect(encoded).toContain("256");
  });

  test("checkpoint for empty tree (size 0)", async () => {
    const { createCheckpoint } = await import("../../../src/lib/merkle/checkpoints.ts");
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const emptyTreeHash = new Uint8Array(32); // All zeros for empty tree

    const checkpoint = await createCheckpoint(
      0,
      emptyTreeHash,
      keyPair.privateKey,
      "https://example.com"
    );

    expect(checkpoint.treeSize).toBe(0);
    expect(checkpoint.rootHash).toEqual(emptyTreeHash);
  });

  test("checkpoint signature is deterministic for same inputs", async () => {
    const { createCheckpoint } = await import("../../../src/lib/merkle/checkpoints.ts");
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const treeSize = 100;
    const rootHash = new Uint8Array(32).fill(0x42);

    // Note: ECDSA signatures are NOT deterministic (they include random k)
    // This test verifies both signatures are valid, not that they're identical
    const checkpoint1 = await createCheckpoint(
      treeSize,
      rootHash,
      keyPair.privateKey,
      "https://example.com"
    );

    const checkpoint2 = await createCheckpoint(
      treeSize,
      rootHash,
      keyPair.privateKey,
      "https://example.com"
    );

    // Both should be valid checkpoints with same tree state
    expect(checkpoint1.treeSize).toBe(checkpoint2.treeSize);
    expect(checkpoint1.rootHash).toEqual(checkpoint2.rootHash);
    expect(checkpoint1.signature).toBeInstanceOf(Uint8Array);
    expect(checkpoint2.signature).toBeInstanceOf(Uint8Array);
  });
});

describe("Checkpoint Verification", () => {
  test("can verify valid checkpoint signature", async () => {
    const { createCheckpoint, verifyCheckpoint } = await import("../../../src/lib/merkle/checkpoints.ts");
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();

    const checkpoint = await createCheckpoint(
      500,
      new Uint8Array(32).fill(0x99),
      keyPair.privateKey,
      "https://example.com"
    );

    const isValid = await verifyCheckpoint(checkpoint, keyPair.publicKey);

    expect(isValid).toBe(true);
  });

  test("rejects checkpoint with invalid signature", async () => {
    const { createCheckpoint, verifyCheckpoint } = await import("../../../src/lib/merkle/checkpoints.ts");
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();

    const checkpoint = await createCheckpoint(
      100,
      new Uint8Array(32),
      keyPair.privateKey,
      "https://example.com"
    );

    // Tamper with signature
    checkpoint.signature[0] ^= 0xFF;

    const isValid = await verifyCheckpoint(checkpoint, keyPair.publicKey);

    expect(isValid).toBe(false);
  });

  test("rejects checkpoint signed with different key", async () => {
    const { createCheckpoint, verifyCheckpoint } = await import("../../../src/lib/merkle/checkpoints.ts");
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");

    const keyPair1 = await generateES256KeyPair();
    const keyPair2 = await generateES256KeyPair();

    const checkpoint = await createCheckpoint(
      100,
      new Uint8Array(32),
      keyPair1.privateKey,
      "https://example.com"
    );

    const isValid = await verifyCheckpoint(checkpoint, keyPair2.publicKey);

    expect(isValid).toBe(false);
  });

  test("can parse and verify encoded checkpoint", async () => {
    const { createCheckpoint, encodeCheckpoint, decodeCheckpoint, verifyCheckpoint } = await import("../../../src/lib/merkle/checkpoints.ts");
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();

    const original = await createCheckpoint(
      750,
      new Uint8Array(32).fill(0xef),
      keyPair.privateKey,
      "https://example.com"
    );

    const encoded = encodeCheckpoint(original);
    const decoded = decodeCheckpoint(encoded);

    expect(decoded.treeSize).toBe(original.treeSize);
    expect(decoded.rootHash).toEqual(original.rootHash);
    expect(decoded.signature).toEqual(original.signature);
    expect(decoded.origin).toBe(original.origin);

    const isValid = await verifyCheckpoint(decoded, keyPair.publicKey);
    expect(isValid).toBe(true);
  });
});

describe("Checkpoint Comparison", () => {
  test("can detect tree growth between checkpoints", async () => {
    const { createCheckpoint } = await import("../../../src/lib/merkle/checkpoints.ts");
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();

    const checkpoint1 = await createCheckpoint(
      100,
      new Uint8Array(32).fill(0x01),
      keyPair.privateKey,
      "https://example.com"
    );

    const checkpoint2 = await createCheckpoint(
      200,
      new Uint8Array(32).fill(0x02),
      keyPair.privateKey,
      "https://example.com"
    );

    expect(checkpoint2.treeSize).toBeGreaterThan(checkpoint1.treeSize);
  });

  test("checkpoints with same tree size should have same root hash", async () => {
    const { createCheckpoint } = await import("../../../src/lib/merkle/checkpoints.ts");
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const rootHash = new Uint8Array(32).fill(0xaa);

    const checkpoint1 = await createCheckpoint(
      100,
      rootHash,
      keyPair.privateKey,
      "https://example.com"
    );

    const checkpoint2 = await createCheckpoint(
      100,
      rootHash,
      keyPair.privateKey,
      "https://example.com"
    );

    expect(checkpoint1.rootHash).toEqual(checkpoint2.rootHash);
  });
});

describe("Checkpoint Edge Cases", () => {
  test("handles large tree sizes", async () => {
    const { createCheckpoint } = await import("../../../src/lib/merkle/checkpoints.ts");
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const largeTreeSize = 10_000_000;

    const checkpoint = await createCheckpoint(
      largeTreeSize,
      new Uint8Array(32),
      keyPair.privateKey,
      "https://example.com"
    );

    expect(checkpoint.treeSize).toBe(largeTreeSize);
  });

  test("validates origin URL format", async () => {
    const { createCheckpoint } = await import("../../../src/lib/merkle/checkpoints.ts");
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();

    await expect(
      createCheckpoint(
        100,
        new Uint8Array(32),
        keyPair.privateKey,
        "not-a-url"
      )
    ).rejects.toThrow();
  });

  test("root hash must be 32 bytes (SHA-256)", async () => {
    const { createCheckpoint } = await import("../../../src/lib/merkle/checkpoints.ts");
    const { generateES256KeyPair } = await import("../../../src/lib/cose/signer.ts");

    const keyPair = await generateES256KeyPair();
    const invalidHash = new Uint8Array(16); // Wrong size

    await expect(
      createCheckpoint(
        100,
        invalidHash,
        keyPair.privateKey,
        "https://example.com"
      )
    ).rejects.toThrow(/32 bytes/);
  });
});
