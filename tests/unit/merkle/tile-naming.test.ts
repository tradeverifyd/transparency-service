/**
 * Tile Naming Tests
 * Test suite for C2SP tlog-tiles naming convention
 * Format: tile/<L>/<N>[.p/<W>]
 */

import { describe, test, expect } from "bun:test";

describe("Tile Path Generation", () => {
  test("generates path for level 0, index 0", async () => {
    const { tileIndexToPath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const path = tileIndexToPath(0, 0);

    expect(path).toBe("tile/0/000");
  });

  test("generates path for level 0, index 255", async () => {
    const { tileIndexToPath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const path = tileIndexToPath(0, 255);

    expect(path).toBe("tile/0/255");
  });

  test("generates path for level 0, index 256", async () => {
    const { tileIndexToPath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const path = tileIndexToPath(0, 256);

    expect(path).toBe("tile/0/x001/000");
  });

  test("generates path for large index with multiple segments", async () => {
    const { tileIndexToPath } = await import("../../../src/lib/merkle/tile-naming.ts");

    // Index 1234067 → x001/x234/067
    const path = tileIndexToPath(0, 1234067);

    expect(path).toBe("tile/0/x001/x234/067");
  });

  test("generates path for level 1", async () => {
    const { tileIndexToPath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const path = tileIndexToPath(1, 100);

    expect(path).toBe("tile/1/100");
  });

  test("generates path for higher levels", async () => {
    const { tileIndexToPath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const path = tileIndexToPath(5, 42);

    expect(path).toBe("tile/5/042");
  });

  test("generates partial tile path", async () => {
    const { tileIndexToPath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const path = tileIndexToPath(0, 0, 128);

    expect(path).toBe("tile/0/000.p/128");
  });

  test("generates partial tile path with large index", async () => {
    const { tileIndexToPath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const path = tileIndexToPath(0, 1234067, 64);

    expect(path).toBe("tile/0/x001/x234/067.p/64");
  });

  test("rejects invalid width for partial tile", async () => {
    const { tileIndexToPath } = await import("../../../src/lib/merkle/tile-naming.ts");

    expect(() => tileIndexToPath(0, 0, 0)).toThrow();
    expect(() => tileIndexToPath(0, 0, 256)).toThrow();
    expect(() => tileIndexToPath(0, 0, 300)).toThrow();
  });
});

describe("Tile Path Parsing", () => {
  test("parses simple full tile path", async () => {
    const { parseTilePath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const parsed = parseTilePath("tile/0/000");

    expect(parsed.level).toBe(0);
    expect(parsed.index).toBe(0);
    expect(parsed.isPartial).toBe(false);
    expect(parsed.width).toBeUndefined();
  });

  test("parses full tile with large index", async () => {
    const { parseTilePath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const parsed = parseTilePath("tile/0/x001/x234/067");

    expect(parsed.level).toBe(0);
    expect(parsed.index).toBe(1234067);
    expect(parsed.isPartial).toBe(false);
  });

  test("parses partial tile path", async () => {
    const { parseTilePath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const parsed = parseTilePath("tile/0/000.p/128");

    expect(parsed.level).toBe(0);
    expect(parsed.index).toBe(0);
    expect(parsed.isPartial).toBe(true);
    expect(parsed.width).toBe(128);
  });

  test("parses partial tile with large index", async () => {
    const { parseTilePath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const parsed = parseTilePath("tile/0/x001/x234/067.p/64");

    expect(parsed.level).toBe(0);
    expect(parsed.index).toBe(1234067);
    expect(parsed.isPartial).toBe(true);
    expect(parsed.width).toBe(64);
  });

  test("parses higher level tiles", async () => {
    const { parseTilePath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const parsed = parseTilePath("tile/5/042");

    expect(parsed.level).toBe(5);
    expect(parsed.index).toBe(42);
  });

  test("throws on invalid tile path format", async () => {
    const { parseTilePath } = await import("../../../src/lib/merkle/tile-naming.ts");

    expect(() => parseTilePath("invalid/path")).toThrow();
    expect(() => parseTilePath("tile/0")).toThrow();
    expect(() => parseTilePath("tile/abc/000")).toThrow();
  });
});

describe("Entry Tile Paths", () => {
  test("generates entry tile path for index 0", async () => {
    const { entryTileIndexToPath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const path = entryTileIndexToPath(0);

    expect(path).toBe("tile/entries/000");
  });

  test("generates entry tile path for large index", async () => {
    const { entryTileIndexToPath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const path = entryTileIndexToPath(1234067);

    expect(path).toBe("tile/entries/x001/x234/067");
  });

  test("generates partial entry tile path", async () => {
    const { entryTileIndexToPath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const path = entryTileIndexToPath(0, 128);

    expect(path).toBe("tile/entries/000.p/128");
  });

  test("parses entry tile path", async () => {
    const { parseEntryTilePath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const parsed = parseEntryTilePath("tile/entries/x001/x234/067");

    expect(parsed.index).toBe(1234067);
    expect(parsed.isPartial).toBe(false);
  });

  test("parses partial entry tile path", async () => {
    const { parseEntryTilePath } = await import("../../../src/lib/merkle/tile-naming.ts");

    const parsed = parseEntryTilePath("tile/entries/000.p/64");

    expect(parsed.index).toBe(0);
    expect(parsed.isPartial).toBe(true);
    expect(parsed.width).toBe(64);
  });
});

describe("Tile Index Calculations", () => {
  test("calculates tile index for entry ID", async () => {
    const { entryIdToTileIndex } = await import("../../../src/lib/merkle/tile-naming.ts");

    expect(entryIdToTileIndex(0)).toBe(0);
    expect(entryIdToTileIndex(255)).toBe(0);
    expect(entryIdToTileIndex(256)).toBe(1);
    expect(entryIdToTileIndex(511)).toBe(1);
    expect(entryIdToTileIndex(512)).toBe(2);
    expect(entryIdToTileIndex(1000)).toBe(3); // 1000 / 256 = 3
  });

  test("calculates tile offset for entry ID", async () => {
    const { entryIdToTileOffset } = await import("../../../src/lib/merkle/tile-naming.ts");

    expect(entryIdToTileOffset(0)).toBe(0);
    expect(entryIdToTileOffset(255)).toBe(255);
    expect(entryIdToTileOffset(256)).toBe(0);
    expect(entryIdToTileOffset(257)).toBe(1);
    expect(entryIdToTileOffset(1000)).toBe(232); // 1000 % 256 = 232
  });

  test("round-trip: entry ID → tile index/offset → entry ID", async () => {
    const { entryIdToTileIndex, entryIdToTileOffset, tileCoordinatesToEntryId } = await import("../../../src/lib/merkle/tile-naming.ts");

    for (const entryId of [0, 100, 255, 256, 500, 1000, 10000]) {
      const tileIndex = entryIdToTileIndex(entryId);
      const tileOffset = entryIdToTileOffset(entryId);
      const reconstructed = tileCoordinatesToEntryId(tileIndex, tileOffset);

      expect(reconstructed).toBe(entryId);
    }
  });
});

describe("C2SP Specification Compliance", () => {
  test("tile size is 256 hashes", async () => {
    const { TILE_SIZE } = await import("../../../src/lib/merkle/tile-naming.ts");

    expect(TILE_SIZE).toBe(256);
  });

  test("hash size is 32 bytes (SHA-256)", async () => {
    const { HASH_SIZE } = await import("../../../src/lib/merkle/tile-naming.ts");

    expect(HASH_SIZE).toBe(32);
  });

  test("full tile is 8192 bytes (256 * 32)", async () => {
    const { TILE_SIZE, HASH_SIZE, FULL_TILE_BYTES } = await import("../../../src/lib/merkle/tile-naming.ts");

    expect(FULL_TILE_BYTES).toBe(TILE_SIZE * HASH_SIZE);
    expect(FULL_TILE_BYTES).toBe(8192);
  });
});
