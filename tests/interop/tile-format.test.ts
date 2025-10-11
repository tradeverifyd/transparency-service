/**
 * Tile Format Compatibility Test
 *
 * Verifies that our tile storage format is compatible with the
 * C2SP tlog-tiles specification used by Go tlog
 */

import { describe, test, expect } from "bun:test";
import { TileLog } from "../../src/lib/merkle/tile-log.ts";
import { LocalStorage } from "../../src/lib/storage/local.ts";
import {
  entryTileIndexToPath,
  entryIdToTileIndex,
  entryIdToTileOffset,
  TILE_SIZE,
  HASH_SIZE,
} from "../../src/lib/merkle/tile-naming.ts";
import * as fs from "fs";

describe("Tile Format Compatibility", () => {
  const testDir = "./.test-tile-format";

  test("entry tiles use consistent path format", () => {
    // Our format: tile/entries/<index> with partitioning
    expect(entryTileIndexToPath(0)).toBe("tile/entries/000");
    expect(entryTileIndexToPath(1)).toBe("tile/entries/001");
    expect(entryTileIndexToPath(255)).toBe("tile/entries/255");
    // Index 256+ uses partitioning: tile/entries/x001/000
    expect(entryTileIndexToPath(256)).toBe("tile/entries/x001/000");
  });

  test("entry tiles store raw leaf data (not hashed)", async () => {
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true });
    }
    fs.mkdirSync(testDir, { recursive: true });

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Add 3 entries
    const leaves = [
      new Uint8Array(32).fill(0),
      new Uint8Array(32).fill(1),
      new Uint8Array(32).fill(2),
    ];

    for (const leaf of leaves) {
      await tileLog.append(leaf);
    }

    // Read entry tile directly
    const tilePath = entryTileIndexToPath(0);
    const tileData = await storage.get(tilePath);

    expect(tileData).toBeDefined();
    expect(tileData!.length).toBe(3 * HASH_SIZE); // 3 entries × 32 bytes

    // Verify raw leaf data is stored (not hashed)
    for (let i = 0; i < 3; i++) {
      const offset = i * HASH_SIZE;
      const storedLeaf = tileData!.slice(offset, offset + HASH_SIZE);

      expect(storedLeaf).toEqual(leaves[i]);
    }
  });

  test("tile size is 256 entries per tile", () => {
    expect(TILE_SIZE).toBe(256);
  });

  test("hash size is 32 bytes (SHA-256)", () => {
    expect(HASH_SIZE).toBe(32);
  });

  test("tile indexing matches C2SP specification", () => {
    // Entry 0 → Tile 0, Offset 0
    expect(entryIdToTileIndex(0)).toBe(0);
    expect(entryIdToTileOffset(0)).toBe(0);

    // Entry 255 → Tile 0, Offset 255
    expect(entryIdToTileIndex(255)).toBe(0);
    expect(entryIdToTileOffset(255)).toBe(255);

    // Entry 256 → Tile 1, Offset 0
    expect(entryIdToTileIndex(256)).toBe(1);
    expect(entryIdToTileOffset(256)).toBe(0);

    // Entry 1000 → Tile 3, Offset 232
    expect(entryIdToTileIndex(1000)).toBe(3);
    expect(entryIdToTileOffset(1000)).toBe(232);
  });

  test("tiles are written incrementally", async () => {
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true });
    }
    fs.mkdirSync(testDir, { recursive: true });

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Add entries one at a time
    for (let i = 0; i < 10; i++) {
      const leaf = new Uint8Array(32).fill(i);
      await tileLog.append(leaf);

      // Verify tile exists and has correct size
      const tilePath = entryTileIndexToPath(0);
      const tileData = await storage.get(tilePath);

      expect(tileData).toBeDefined();
      expect(tileData!.length).toBe((i + 1) * HASH_SIZE);
    }
  });

  test("tree size tracking is accurate", async () => {
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true });
    }
    fs.mkdirSync(testDir, { recursive: true });

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    expect(await tileLog.size()).toBe(0);

    await tileLog.append(new Uint8Array(32).fill(0));
    expect(await tileLog.size()).toBe(1);

    await tileLog.append(new Uint8Array(32).fill(1));
    expect(await tileLog.size()).toBe(2);

    for (let i = 2; i < 100; i++) {
      await tileLog.append(new Uint8Array(32).fill(i));
    }
    expect(await tileLog.size()).toBe(100);
  });

  test("multiple tiles are created when needed", async () => {
    if (fs.existsSync(testDir)) {
      fs.rmSync(testDir, { recursive: true });
    }
    fs.mkdirSync(testDir, { recursive: true });

    const storage = new LocalStorage(testDir);
    const tileLog = new TileLog(storage);

    // Add 300 entries (spans 2 tiles)
    for (let i = 0; i < 300; i++) {
      await tileLog.append(new Uint8Array(32).fill(i % 256));
    }

    // Verify both tiles exist
    const tile0 = await storage.get(entryTileIndexToPath(0));
    const tile1 = await storage.get(entryTileIndexToPath(1));

    expect(tile0).toBeDefined();
    expect(tile1).toBeDefined();

    // Tile 0 should be full (256 entries)
    expect(tile0!.length).toBe(256 * HASH_SIZE);

    // Tile 1 should have 44 entries (300 - 256)
    expect(tile1!.length).toBe(44 * HASH_SIZE);
  });
});
