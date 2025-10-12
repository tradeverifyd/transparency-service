/**
 * Checkpoint Endpoint
 * Implements GET /checkpoint for signed tree head retrieval
 */

import type { ServerContext } from "../server.ts";
import { getTreeSize } from "../../lib/database/merkle.ts";
import { createCheckpoint, encodeCheckpoint } from "../../lib/merkle/checkpoints.ts";

/**
 * Handle checkpoint retrieval
 * Returns current tree state as a signed note
 */
export async function handleGetCheckpoint(ctx: ServerContext): Promise<Response> {
  try {
    // Get current tree size
    const treeSize = getTreeSize(ctx.db);

    if (treeSize === 0) {
      // Empty tree - return a minimal checkpoint
      const emptyCheckpoint = {
        origin: ctx.origin,
        treeSize: 0,
        rootHash: new Uint8Array(32), // Zero hash
        timestamp: Date.now(),
        signature: new Uint8Array(64), // Empty signature
      };

      const encoded = encodeCheckpoint(emptyCheckpoint);

      return new Response(encoded, {
        status: 200,
        headers: {
          "Content-Type": "text/plain; charset=utf-8",
          "Cache-Control": "no-cache",
        },
      });
    }

    // Compute current root hash
    const rootHash = await computeRootHash(ctx.db, treeSize);

    // Create checkpoint
    const checkpoint = await createCheckpoint(
      treeSize,
      rootHash,
      ctx.serviceKey!,
      ctx.origin
    );

    // Encode as signed note
    const encoded = encodeCheckpoint(checkpoint);

    return new Response(encoded, {
      status: 200,
      headers: {
        "Content-Type": "text/plain; charset=utf-8",
        "Cache-Control": "no-cache",
      },
    });
  } catch (error) {
    console.error("Checkpoint error:", error);
    return new Response("Internal Server Error", { status: 500 });
  }
}

/**
 * Compute root hash for current tree
 */
async function computeRootHash(db: any, treeSize: number): Promise<Uint8Array> {
  if (treeSize === 0) {
    throw new Error("Cannot compute root hash of empty tree");
  }

  if (treeSize === 1) {
    // Single leaf - get and hash it
    const stmt = db.prepare(`
      SELECT hash FROM merkle_tree WHERE level = 0 AND leaf_index = 0
    `);
    const row = stmt.get() as { hash: string } | null;

    if (!row) {
      throw new Error("Leaf 0 not found");
    }

    const leafHash = Uint8Array.from(atob(row.hash), c => c.charCodeAt(0));

    // Hash with RFC 6962 leaf prefix (0x00)
    return await hashLeaf(leafHash);
  }

  // Recursively compute root hash
  return await computeSubtreeHash(db, 0, treeSize);
}

/**
 * Compute hash of a subtree
 */
async function computeSubtreeHash(
  db: any,
  start: number,
  size: number
): Promise<Uint8Array> {
  if (size === 0) {
    throw new Error("Cannot compute hash of empty subtree");
  }

  if (size === 1) {
    // Single leaf
    const stmt = db.prepare(`
      SELECT hash FROM merkle_tree WHERE level = 0 AND leaf_index = ?
    `);
    const row = stmt.get(start) as { hash: string } | null;

    if (!row) {
      throw new Error(`Leaf ${start} not found`);
    }

    const leafHash = Uint8Array.from(atob(row.hash), c => c.charCodeAt(0));
    return await hashLeaf(leafHash);
  }

  // Split and recurse
  const k = largestPowerOfTwoLessThan(size);

  const leftHash = await computeSubtreeHash(db, start, k);
  const rightSize = size - k;

  if (rightSize === 0) {
    return leftHash;
  }

  const rightHash = await computeSubtreeHash(db, start + k, rightSize);

  return await hashNode(leftHash, rightHash);
}

/**
 * Find largest power of 2 strictly less than n
 */
function largestPowerOfTwoLessThan(n: number): number {
  let k = 1;
  while (k * 2 < n) {
    k *= 2;
  }
  return k;
}

/**
 * Hash a leaf with RFC 6962 prefix (0x00)
 */
async function hashLeaf(leaf: Uint8Array): Promise<Uint8Array> {
  const data = new Uint8Array(1 + leaf.length);
  data[0] = 0x00; // RFC 6962 leaf prefix
  data.set(leaf, 1);
  return await hashData(data);
}

/**
 * Hash an internal node with RFC 6962 prefix (0x01)
 */
async function hashNode(left: Uint8Array, right: Uint8Array): Promise<Uint8Array> {
  const data = new Uint8Array(1 + left.length + right.length);
  data[0] = 0x01; // RFC 6962 node prefix
  data.set(left, 1);
  data.set(right, 1 + left.length);
  return await hashData(data);
}

/**
 * Hash data using SHA-256
 */
async function hashData(data: Uint8Array): Promise<Uint8Array> {
  const hashBuffer = await crypto.subtle.digest("SHA-256", data);
  return new Uint8Array(hashBuffer);
}
