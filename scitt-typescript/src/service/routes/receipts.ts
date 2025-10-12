/**
 * Receipt Resolution Endpoint
 * Implements GET /entries/{entry_id}/receipt per SCRAPI specification
 */

import type { ServerContext } from "../server.ts";
import type { Receipt, ErrorResponse } from "../types/scrapi.ts";
import { getTreeSize, getInclusionProof } from "../../lib/database/merkle.ts";

/**
 * Handle receipt resolution
 */
export async function handleGetReceipt(
  ctx: ServerContext,
  entryId: string
): Promise<Response> {
  try {
    // Check if entry exists in statement_blobs
    const { getStatement } = await import("../../lib/database/statements.ts");
    const statement = await getStatement(ctx.db, entryId);

    if (!statement) {
      return errorResponse(404, "Not Found", `Entry ${entryId} not found`);
    }

    // Get leaf hash and index for this entry
    const leafInfo = await getLeafInfo(ctx.db, entryId);

    if (!leafInfo) {
      return errorResponse(404, "Not Found", `Receipt not found for entry ${entryId}`);
    }

    // Get current tree size
    const treeSize = getTreeSize(ctx.db);

    // Generate inclusion proof
    const proof = await getInclusionProof(ctx.db, leafInfo.leafIndex, treeSize);

    // Create receipt
    const receipt: Receipt = {
      tree_size: treeSize,
      leaf_index: leafInfo.leafIndex,
      inclusion_proof: proof.map((hash) =>
        btoa(String.fromCharCode(...hash))
          .replace(/\+/g, "-")
          .replace(/\//g, "_")
          .replace(/=/g, "")
      ),
    };

    return new Response(JSON.stringify(receipt), {
      status: 200,
      headers: {
        "Content-Type": "application/json",
      },
    });
  } catch (error) {
    console.error("Receipt resolution error:", error);
    return errorResponse(500, "Internal Server Error", String(error));
  }
}

/**
 * Get leaf information for an entry
 */
async function getLeafInfo(
  db: any,
  entryId: string
): Promise<{ leafIndex: number; leafHash: Uint8Array } | null> {
  const stmt = db.prepare(`
    SELECT leaf_index, leaf_hash FROM statement_blobs WHERE entry_id = ?
  `);

  const row = stmt.get(entryId) as { leaf_index: number; leaf_hash: string } | null;

  if (!row) {
    return null;
  }

  // Convert base64 back to Uint8Array
  const leafHash = Uint8Array.from(atob(row.leaf_hash), c => c.charCodeAt(0));

  return {
    leafIndex: row.leaf_index,
    leafHash,
  };
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
