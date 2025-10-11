/**
 * Merkle Proof Tests
 * Test suite for RFC 6962 inclusion and consistency proofs
 */

import { describe, test, expect, beforeEach } from "bun:test";
import * as fs from "fs";

const testDir = "./.test-proofs";

beforeEach(() => {
  if (fs.existsSync(testDir)) {
    fs.rmSync(testDir, { recursive: true });
  }
  fs.mkdirSync(testDir, { recursive: true });
});

describe("Inclusion Proof Generation", () => {
  test("generates inclusion proof for single-entry tree", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateInclusionProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    const leaf = new Uint8Array(32);
    leaf.fill(1);
    await tileLog.append(leaf);

    const proof = await generateInclusionProof(storage, 0, 1);

    expect(proof.leafIndex).toBe(0);
    expect(proof.treeSize).toBe(1);
    expect(proof.auditPath).toEqual([]); // Single entry has empty audit path
  });

  test("generates inclusion proof for first entry in 2-entry tree", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateInclusionProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    const leaf1 = new Uint8Array(32);
    leaf1.fill(1);
    await tileLog.append(leaf1);

    const leaf2 = new Uint8Array(32);
    leaf2.fill(2);
    await tileLog.append(leaf2);

    const proof = await generateInclusionProof(storage, 0, 2);

    expect(proof.leafIndex).toBe(0);
    expect(proof.treeSize).toBe(2);
    expect(proof.auditPath.length).toBe(1); // One sibling hash
  });

  test("generates inclusion proof for entry in larger tree", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateInclusionProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Build tree with 8 entries
    for (let i = 0; i < 8; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }

    const proof = await generateInclusionProof(storage, 3, 8);

    expect(proof.leafIndex).toBe(3);
    expect(proof.treeSize).toBe(8);
    expect(proof.auditPath.length).toBe(3); // log2(8) = 3
  });

  test("generates different proofs for different indices", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateInclusionProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Build tree with 4 entries
    for (let i = 0; i < 4; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }

    const proof0 = await generateInclusionProof(storage, 0, 4);
    const proof1 = await generateInclusionProof(storage, 1, 4);

    expect(proof0.auditPath).not.toEqual(proof1.auditPath);
  });

  test("rejects invalid leaf index", async () => {
    const { generateInclusionProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);

    await expect(generateInclusionProof(storage, 5, 3)).rejects.toThrow();
  });

  test("generates proof for entry in full tile", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateInclusionProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Build tree with 256 entries (full tile)
    for (let i = 0; i < 256; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i % 256);
      await tileLog.append(leaf);
    }

    const proof = await generateInclusionProof(storage, 100, 256);

    expect(proof.leafIndex).toBe(100);
    expect(proof.treeSize).toBe(256);
    expect(proof.auditPath.length).toBeGreaterThan(0);
  });
});

describe("Inclusion Proof Verification", () => {
  test("verifies valid inclusion proof for single entry", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateInclusionProof, verifyInclusionProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    const leaf = new Uint8Array(32);
    leaf.fill(1);
    await tileLog.append(leaf);

    const root = await tileLog.root();
    const proof = await generateInclusionProof(storage, 0, 1);

    const isValid = await verifyInclusionProof(leaf, proof, root);

    expect(isValid).toBe(true);
  });

  test("verifies valid inclusion proof for entry in larger tree", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateInclusionProof, verifyInclusionProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Build tree with 8 entries
    const leaves: Uint8Array[] = [];
    for (let i = 0; i < 8; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      leaves.push(leaf);
      await tileLog.append(leaf);
    }

    const root = await tileLog.root();
    const proof = await generateInclusionProof(storage, 3, 8);

    const isValid = await verifyInclusionProof(leaves[3], proof, root);

    expect(isValid).toBe(true);
  });

  test("rejects proof with wrong leaf data", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateInclusionProof, verifyInclusionProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Build tree
    for (let i = 0; i < 4; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }

    const root = await tileLog.root();
    const proof = await generateInclusionProof(storage, 2, 4);

    const wrongLeaf = new Uint8Array(32);
    wrongLeaf.fill(99);

    const isValid = await verifyInclusionProof(wrongLeaf, proof, root);

    expect(isValid).toBe(false);
  });

  test("rejects proof with tampered audit path", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateInclusionProof, verifyInclusionProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    const leaves: Uint8Array[] = [];
    for (let i = 0; i < 8; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      leaves.push(leaf);
      await tileLog.append(leaf);
    }

    const root = await tileLog.root();
    const proof = await generateInclusionProof(storage, 3, 8);

    // Tamper with audit path
    if (proof.auditPath.length > 0) {
      proof.auditPath[0][0] ^= 0xFF;
    }

    const isValid = await verifyInclusionProof(leaves[3], proof, root);

    expect(isValid).toBe(false);
  });

  test("verifies all entries in tree have valid proofs", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateInclusionProof, verifyInclusionProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    const leaves: Uint8Array[] = [];
    for (let i = 0; i < 7; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      leaves.push(leaf);
      await tileLog.append(leaf);
    }

    const root = await tileLog.root();

    // Verify all entries
    for (let i = 0; i < 7; i++) {
      const proof = await generateInclusionProof(storage, i, 7);
      const isValid = await verifyInclusionProof(leaves[i], proof, root);
      expect(isValid).toBe(true);
    }
  });
});

describe("Consistency Proof Generation", () => {
  test("generates consistency proof for same tree size", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateConsistencyProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    for (let i = 0; i < 4; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }

    const proof = await generateConsistencyProof(storage, 4, 4);

    expect(proof.oldSize).toBe(4);
    expect(proof.newSize).toBe(4);
    expect(proof.proof).toEqual([]); // Same size = empty proof
  });

  test("generates consistency proof for growing tree", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateConsistencyProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    for (let i = 0; i < 8; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }

    const proof = await generateConsistencyProof(storage, 4, 8);

    expect(proof.oldSize).toBe(4);
    expect(proof.newSize).toBe(8);
    expect(proof.proof.length).toBeGreaterThan(0);
  });

  test("rejects invalid tree sizes", async () => {
    const { generateConsistencyProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);

    // oldSize > newSize
    await expect(generateConsistencyProof(storage, 8, 4)).rejects.toThrow();

    // newSize = 0
    await expect(generateConsistencyProof(storage, 0, 0)).rejects.toThrow();
  });

  test("generates proof for size 1 to larger tree", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateConsistencyProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    for (let i = 0; i < 8; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }

    const proof = await generateConsistencyProof(storage, 1, 8);

    expect(proof.oldSize).toBe(1);
    expect(proof.newSize).toBe(8);
    expect(proof.proof).toBeDefined();
  });
});

describe("Consistency Proof Verification", () => {
  test("verifies valid consistency proof for growing tree", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateConsistencyProof, verifyConsistencyProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Build initial tree
    for (let i = 0; i < 4; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }
    const oldRoot = await tileLog.root();

    // Grow tree
    for (let i = 4; i < 8; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }
    const newRoot = await tileLog.root();

    const proof = await generateConsistencyProof(storage, 4, 8);
    const isValid = await verifyConsistencyProof(proof, oldRoot, newRoot);

    expect(isValid).toBe(true);
  });

  test("verifies consistency for same tree size", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateConsistencyProof, verifyConsistencyProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    for (let i = 0; i < 4; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }
    const root = await tileLog.root();

    const proof = await generateConsistencyProof(storage, 4, 4);
    const isValid = await verifyConsistencyProof(proof, root, root);

    expect(isValid).toBe(true);
  });

  test("rejects proof with tampered audit path", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateConsistencyProof, verifyConsistencyProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    for (let i = 0; i < 4; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }
    const oldRoot = await tileLog.root();

    for (let i = 4; i < 8; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }
    const newRoot = await tileLog.root();

    const proof = await generateConsistencyProof(storage, 4, 8);

    // Tamper with proof
    if (proof.proof.length > 0) {
      proof.proof[0][0] ^= 0xFF;
    }

    const isValid = await verifyConsistencyProof(proof, oldRoot, newRoot);

    expect(isValid).toBe(false);
  });

  test("verifies multiple consistency proofs as tree grows", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { generateConsistencyProof, verifyConsistencyProof } = await import("../../../src/lib/merkle/proofs.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    const roots: Uint8Array[] = [];

    // Build tree incrementally and collect roots
    for (let size = 1; size <= 8; size++) {
      const leaf = new Uint8Array(32);
      leaf.fill(size - 1);
      await tileLog.append(leaf);
      roots.push(await tileLog.root());
    }

    // Verify consistency between all pairs
    for (let oldSize = 1; oldSize <= 8; oldSize++) {
      for (let newSize = oldSize; newSize <= 8; newSize++) {
        const proof = await generateConsistencyProof(storage, oldSize, newSize);
        const isValid = await verifyConsistencyProof(
          proof,
          roots[oldSize - 1],
          roots[newSize - 1]
        );
        expect(isValid).toBe(true);
      }
    }
  });
});
