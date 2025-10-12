/**
 * Tile Retrieval Endpoints
 * Implements GET /tile/* endpoints per C2SP tile log specification
 */

import type { ServerContext } from "../server.ts";
import type { ErrorResponse } from "../types/scrapi.ts";

/**
 * Handle tile retrieval
 * Supports both full tiles and partial tiles
 */
export async function handleGetTile(
  ctx: ServerContext,
  level: number,
  index: number,
  partial?: { width: number }
): Promise<Response> {
  try {
    // For now, return a simple implementation that retrieves tile from database
    // In a full implementation, this would fetch from object storage

    const { getTreeSize } = await import("../../lib/database/merkle.ts");
    const treeSize = getTreeSize(ctx.db);

    if (treeSize === 0) {
      return errorResponse(404, "Not Found", "Tree is empty");
    }

    // Calculate if this tile exists
    if (level === 0) {
      // Entry tile - contains leaf hashes
      const tileData = await getEntryTile(ctx.db, index, partial?.width);

      if (!tileData) {
        return errorResponse(404, "Not Found", `Tile ${level}/${index} not found`);
      }

      return new Response(tileData, {
        status: 200,
        headers: {
          "Content-Type": "application/octet-stream",
          "Cache-Control": "public, max-age=31536000, immutable",
        },
      });
    } else {
      // Hash tile - contains internal node hashes
      const tileData = await getHashTile(ctx.db, level, index, partial?.width);

      if (!tileData) {
        return errorResponse(404, "Not Found", `Tile ${level}/${index} not found`);
      }

      return new Response(tileData, {
        status: 200,
        headers: {
          "Content-Type": "application/octet-stream",
          "Cache-Control": "public, max-age=31536000, immutable",
        },
      });
    }
  } catch (error) {
    console.error("Tile retrieval error:", error);
    return errorResponse(500, "Internal Server Error", String(error));
  }
}

/**
 * Get entry tile (level 0)
 * Returns concatenated leaf hashes
 */
async function getEntryTile(
  db: any,
  tileIndex: number,
  width?: number
): Promise<Uint8Array | null> {
  // Each tile contains 256 entries by default (C2SP tile log format)
  const TILE_WIDTH = 256;
  const maxWidth = width || TILE_WIDTH;

  const startIndex = tileIndex * TILE_WIDTH;
  const endIndex = startIndex + maxWidth;

  // Query leaf hashes from merkle_tree table
  const stmt = db.prepare(`
    SELECT hash FROM merkle_tree
    WHERE level = 0 AND leaf_index >= ? AND leaf_index < ?
    ORDER BY leaf_index ASC
  `);

  const rows = stmt.all(startIndex, endIndex) as { hash: string }[];

  if (rows.length === 0) {
    return null;
  }

  // Concatenate all hashes
  const tileData = new Uint8Array(rows.length * 32);

  for (let i = 0; i < rows.length; i++) {
    const hash = Uint8Array.from(atob(rows[i]!.hash), c => c.charCodeAt(0));
    tileData.set(hash, i * 32);
  }

  return tileData;
}

/**
 * Get hash tile (level > 0)
 * Returns internal node hashes
 */
async function getHashTile(
  db: any,
  level: number,
  tileIndex: number,
  width?: number
): Promise<Uint8Array | null> {
  // Hash tiles not yet fully implemented
  // Would require computing internal node hashes
  return null;
}

/**
 * Create error response
 */
function errorResponse(
  status: number,
  error: string,
  details?: string
): Response {
  const body: ErrorResponse = {
    error,
    details,
  };

  return new Response(JSON.stringify(body), {
    status,
    headers: {
      "Content-Type": "application/json",
    },
  });
}
