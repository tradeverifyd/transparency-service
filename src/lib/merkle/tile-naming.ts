/**
 * Tile Naming Utilities
 * Implementation of C2SP tlog-tiles naming convention
 * Specification: https://c2sp.org/tlog-tiles
 * Format: tile/<L>/<N>[.p/<W>]
 */

/**
 * C2SP Constants
 */
export const TILE_SIZE = 256; // Hashes per tile
export const HASH_SIZE = 32; // SHA-256 hash size in bytes
export const FULL_TILE_BYTES = TILE_SIZE * HASH_SIZE; // 8192 bytes

/**
 * Parsed tile path components
 */
export interface ParsedTilePath {
  level: number;
  index: number;
  isPartial: boolean;
  width?: number;
}

/**
 * Parsed entry tile path components
 */
export interface ParsedEntryTilePath {
  index: number;
  isPartial: boolean;
  width?: number;
}

/**
 * Generate tile path from level and index
 * Format: tile/<L>/<N>[.p/<W>]
 *
 * For indices >= 256, uses x-prefixed segments:
 * Index 1234067 → tile/0/x001/x234/067
 *
 * @param level - Tree level (0 = leaf level)
 * @param index - Tile index at that level
 * @param width - Optional width for partial tiles (1-255)
 * @returns Tile path string
 */
export function tileIndexToPath(level: number, index: number, width?: number): string {
  if (width !== undefined && (width < 1 || width > 255)) {
    throw new Error(`Invalid partial tile width: ${width}. Must be between 1 and 255.`);
  }

  const indexPath = formatIndexPath(index);
  const basePath = `tile/${level}/${indexPath}`;

  if (width !== undefined) {
    return `${basePath}.p/${width}`;
  }

  return basePath;
}

/**
 * Parse tile path into components
 *
 * @param path - Tile path string (e.g., "tile/0/x001/x234/067.p/128")
 * @returns Parsed tile path components
 */
export function parseTilePath(path: string): ParsedTilePath {
  // Match format: tile/<level>/<segments...>[.p/<width>]
  const partialMatch = path.match(/^tile\/(\d+)\/(.+?)\.p\/(\d+)$/);
  if (partialMatch) {
    const [, levelStr, indexPath, widthStr] = partialMatch;
    return {
      level: parseInt(levelStr, 10),
      index: parseIndexPath(indexPath),
      isPartial: true,
      width: parseInt(widthStr, 10),
    };
  }

  const fullMatch = path.match(/^tile\/(\d+)\/(.+)$/);
  if (fullMatch) {
    const [, levelStr, indexPath] = fullMatch;
    return {
      level: parseInt(levelStr, 10),
      index: parseIndexPath(indexPath),
      isPartial: false,
    };
  }

  throw new Error(`Invalid tile path format: ${path}`);
}

/**
 * Generate entry tile path
 * Format: tile/entries/<N>[.p/<W>]
 *
 * @param index - Entry tile index
 * @param width - Optional width for partial tiles (1-255)
 * @returns Entry tile path string
 */
export function entryTileIndexToPath(index: number, width?: number): string {
  if (width !== undefined && (width < 1 || width > 255)) {
    throw new Error(`Invalid partial tile width: ${width}. Must be between 1 and 255.`);
  }

  const indexPath = formatIndexPath(index);
  const basePath = `tile/entries/${indexPath}`;

  if (width !== undefined) {
    return `${basePath}.p/${width}`;
  }

  return basePath;
}

/**
 * Parse entry tile path into components
 *
 * @param path - Entry tile path string
 * @returns Parsed entry tile path components
 */
export function parseEntryTilePath(path: string): ParsedEntryTilePath {
  // Match format: tile/entries/<segments...>[.p/<width>]
  const partialMatch = path.match(/^tile\/entries\/(.+?)\.p\/(\d+)$/);
  if (partialMatch) {
    const [, indexPath, widthStr] = partialMatch;
    return {
      index: parseIndexPath(indexPath),
      isPartial: true,
      width: parseInt(widthStr, 10),
    };
  }

  const fullMatch = path.match(/^tile\/entries\/(.+)$/);
  if (fullMatch) {
    const [, indexPath] = fullMatch;
    return {
      index: parseIndexPath(indexPath),
      isPartial: false,
    };
  }

  throw new Error(`Invalid entry tile path format: ${path}`);
}

/**
 * Calculate tile index from entry ID
 * Entry ID 0-255 → tile 0, 256-511 → tile 1, etc.
 *
 * @param entryId - Entry ID (0-based)
 * @returns Tile index
 */
export function entryIdToTileIndex(entryId: number): number {
  return Math.floor(entryId / TILE_SIZE);
}

/**
 * Calculate tile offset from entry ID
 * Entry ID modulo tile size
 *
 * @param entryId - Entry ID (0-based)
 * @returns Tile offset (0-255)
 */
export function entryIdToTileOffset(entryId: number): number {
  return entryId % TILE_SIZE;
}

/**
 * Calculate entry ID from tile coordinates
 * Reverse of entryIdToTileIndex/entryIdToTileOffset
 *
 * @param tileIndex - Tile index
 * @param tileOffset - Tile offset (0-255)
 * @returns Entry ID
 */
export function tileCoordinatesToEntryId(tileIndex: number, tileOffset: number): number {
  return tileIndex * TILE_SIZE + tileOffset;
}

/**
 * Format index as path segments
 * C2SP tlog-tiles uses a hybrid approach:
 * - Index 0-255: Simple 3-digit format "042"
 * - Index 256-65535: Base-256 encoding "x001/000"
 * - Index >= 65536: Decimal digit grouping "x001/x234/067"
 *
 * @param index - Tile index
 * @returns Path string
 */
function formatIndexPath(index: number): string {
  if (index < 256) {
    // Simple 3-digit format with leading zeros
    return index.toString().padStart(3, "0");
  }

  if (index < 65536) {
    // Base-256 encoding for indices 256-65535
    const digits: number[] = [];
    let remaining = index;

    while (remaining >= 256) {
      digits.unshift(remaining % 256);
      remaining = Math.floor(remaining / 256);
    }
    digits.unshift(remaining);

    return digits
      .map((digit, i) => {
        const formatted = digit.toString().padStart(3, "0");
        return i < digits.length - 1 ? `x${formatted}` : formatted;
      })
      .join("/");
  }

  // For indices >= 65536, use decimal digit grouping
  // Split the decimal representation into 3-digit groups
  const indexStr = index.toString();
  const paddedLength = Math.ceil(indexStr.length / 3) * 3;
  const paddedIndex = indexStr.padStart(paddedLength, "0");

  const segments: string[] = [];
  for (let i = 0; i < paddedIndex.length; i += 3) {
    segments.push(paddedIndex.slice(i, i + 3));
  }

  // Add x prefix to all but the last segment
  return segments
    .map((s, i) => (i < segments.length - 1 ? `x${s}` : s))
    .join("/");
}

/**
 * Parse index path segments into index number
 * Reverse of formatIndexPath
 * Handles both base-256 and decimal grouping formats
 *
 * @param indexPath - Path string (e.g., "x001/x234/067" or "x001/000")
 * @returns Tile index
 */
function parseIndexPath(indexPath: string): number {
  const segments = indexPath.split("/");

  if (segments.length === 1) {
    // Simple 3-digit format (0-255)
    return parseInt(segments[0], 10);
  }

  // Strip x prefixes and parse values
  const values = segments.map(s => parseInt(s.startsWith("x") ? s.slice(1) : s, 10));

  // Try base-256 first (for indices 256-65535)
  let base256Result = 0;
  for (let i = 0; i < values.length; i++) {
    base256Result = base256Result * 256 + values[i];
  }

  // If base-256 result is < 65536, it's valid base-256 encoding
  if (base256Result < 65536) {
    return base256Result;
  }

  // Otherwise, it's decimal grouping (e.g., "x001/x234/067" → 1234067)
  // Concatenate all segments as decimal digits
  return parseInt(values.map(v => v.toString().padStart(3, "0")).join(""), 10);
}
