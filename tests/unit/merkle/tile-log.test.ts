/**
 * Tile Log Tests
 * Test suite for tile-based Merkle tree operations
 * Tests single leaf append, full tiles, and partial tiles
 */

import { describe, test, expect, beforeEach } from "bun:test";
import * as fs from "fs";

const testDir = "./.test-tile-log";

beforeEach(() => {
  if (fs.existsSync(testDir)) {
    fs.rmSync(testDir, { recursive: true });
  }
  fs.mkdirSync(testDir, { recursive: true });
});

describe("Single Leaf Append", () => {
  test("can append a single leaf to empty tree", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    const leaf = new Uint8Array(32); // SHA-256 hash
    leaf.fill(1);

    const entryId = await tileLog.append(leaf);

    expect(entryId).toBe(0);
    expect(await tileLog.size()).toBe(1);
  });

  test("can append multiple leaves sequentially", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    const leaves: number[] = [];
    for (let i = 0; i < 10; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      const entryId = await tileLog.append(leaf);
      leaves.push(entryId);
    }

    expect(leaves).toEqual([0, 1, 2, 3, 4, 5, 6, 7, 8, 9]);
    expect(await tileLog.size()).toBe(10);
  });

  test("stores entry tile (level entries) correctly", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    const leaf = new Uint8Array(32);
    leaf.fill(42);

    await tileLog.append(leaf);

    // Entry tile should exist at tile/entries/000
    const entryTileExists = await storage.exists("tile/entries/000");
    expect(entryTileExists).toBe(true);
  });

  test("computes tree root after single append", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    const leaf = new Uint8Array(32);
    leaf.fill(1);

    await tileLog.append(leaf);

    const root = await tileLog.root();
    expect(root).toBeInstanceOf(Uint8Array);
    expect(root.length).toBe(32);
  });

  test("rejects invalid leaf size", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    const invalidLeaf = new Uint8Array(16); // Wrong size

    await expect(tileLog.append(invalidLeaf)).rejects.toThrow();
  });
});

describe("Full Tile Creation", () => {
  test("creates full tile after 256 entries", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Append 256 entries
    for (let i = 0; i < 256; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i % 256);
      await tileLog.append(leaf);
    }

    expect(await tileLog.size()).toBe(256);

    // Full entry tile should exist
    const entryTile = await storage.get("tile/entries/000");
    expect(entryTile).not.toBeNull();
    expect(entryTile!.length).toBe(256 * 32); // 256 hashes * 32 bytes
  });

  test("creates full hash tile at level 0 after 256 entries", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Append 256 entries
    for (let i = 0; i < 256; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i % 256);
      await tileLog.append(leaf);
    }

    // Full hash tile should exist at level 0
    const hashTile = await storage.get("tile/0/000");
    expect(hashTile).not.toBeNull();
    expect(hashTile!.length).toBe(256 * 32);
  });

  test("starts new tile after 256 entries", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Append 257 entries (one more than full tile)
    for (let i = 0; i < 257; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i % 256);
      await tileLog.append(leaf);
    }

    expect(await tileLog.size()).toBe(257);

    // First tile should be full
    const firstTile = await storage.get("tile/entries/000");
    expect(firstTile!.length).toBe(256 * 32);

    // Second tile should be partial with 1 entry
    const secondTile = await storage.get("tile/entries/001");
    expect(secondTile).not.toBeNull();
    expect(secondTile!.length).toBe(32); // 1 hash * 32 bytes
  });

  test("creates higher level tiles as tree grows", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Append 512 entries (2 full tiles at level 0)
    for (let i = 0; i < 512; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i % 256);
      await tileLog.append(leaf);
    }

    expect(await tileLog.size()).toBe(512);

    // Level 0 should have 2 tiles
    expect(await storage.exists("tile/0/000")).toBe(true);
    expect(await storage.exists("tile/0/001")).toBe(true);

    // Level 1 should start forming (contains hashes of level 0 tiles)
    expect(await storage.exists("tile/1/000")).toBe(true);
  });
});

describe("Partial Tile Handling", () => {
  test("stores partial entry tile correctly", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Append 128 entries (half a tile)
    for (let i = 0; i < 128; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }

    const entryTile = await storage.get("tile/entries/000");
    expect(entryTile).not.toBeNull();
    expect(entryTile!.length).toBe(128 * 32); // Partial tile
  });

  test("can retrieve leaf from partial tile", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    const testLeaf = new Uint8Array(32);
    testLeaf.fill(99);

    await tileLog.append(testLeaf);

    const retrieved = await tileLog.getLeaf(0);
    expect(retrieved).toEqual(testLeaf);
  });

  test("updates partial tile as more entries are added", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Add 10 entries
    for (let i = 0; i < 10; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }

    const partialTile1 = await storage.get("tile/entries/000");
    expect(partialTile1!.length).toBe(10 * 32);

    // Add 5 more entries
    for (let i = 10; i < 15; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }

    const partialTile2 = await storage.get("tile/entries/000");
    expect(partialTile2!.length).toBe(15 * 32);
  });

  test("computes correct root for partial tile", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Add 7 entries
    for (let i = 0; i < 7; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog.append(leaf);
    }

    const root1 = await tileLog.root();

    // Add 1 more entry
    const leaf = new Uint8Array(32);
    leaf.fill(7);
    await tileLog.append(leaf);

    const root2 = await tileLog.root();

    // Roots should be different
    expect(root1).not.toEqual(root2);
  });

  test("handles boundary between partial and full tile", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Add 255 entries (one less than full)
    for (let i = 0; i < 255; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i % 256);
      await tileLog.append(leaf);
    }

    expect(await tileLog.size()).toBe(255);

    // Add one more to complete the tile
    const finalLeaf = new Uint8Array(32);
    finalLeaf.fill(255);
    await tileLog.append(finalLeaf);

    expect(await tileLog.size()).toBe(256);

    // Tile should now be full
    const fullTile = await storage.get("tile/entries/000");
    expect(fullTile!.length).toBe(256 * 32);
  });
});

describe("Tree State Persistence", () => {
  test("can reload tree state from storage", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);

    // Create tree and add entries
    const tileLog1 = new TileLog(storage);
    for (let i = 0; i < 100; i++) {
      const leaf = new Uint8Array(32);
      leaf.fill(i);
      await tileLog1.append(leaf);
    }

    const size1 = await tileLog1.size();
    const root1 = await tileLog1.root();

    // Create new tree instance with same storage
    const tileLog2 = new TileLog(storage);
    await tileLog2.load();

    const size2 = await tileLog2.size();
    const root2 = await tileLog2.root();

    expect(size2).toBe(size1);
    expect(root2).toEqual(root1);
  });

  test("handles empty tree on first load", async () => {
    const { TileLog } = await import("../../../src/lib/merkle/tile-log.ts");
    const { LocalStorage } = await import("../../../src/lib/storage/local.ts");

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    await tileLog.load();

    expect(await tileLog.size()).toBe(0);
  });
});
