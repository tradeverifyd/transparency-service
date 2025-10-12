/**
 * Merkle Tree Database Operations
 * Database operations for managing the Merkle tree structure
 */

import type { Database } from "bun:sqlite";

/**
 * Merkle tree node
 */
export interface MerkleNode {
  id: number;
  leafIndex: number;
  hash: string;
  level: number;
  createdAt: Date;
}

/**
 * Initialize Merkle tree tables
 */
export function initializeMerkleTables(db: Database): void {
  // Create merkle_tree table
  db.run(`
    CREATE TABLE IF NOT EXISTS merkle_tree (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      leaf_index INTEGER NOT NULL,
      hash TEXT NOT NULL,
      level INTEGER NOT NULL DEFAULT 0,
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      UNIQUE(leaf_index, level)
    )
  `);

  db.run("CREATE INDEX IF NOT EXISTS idx_merkle_leaf_index ON merkle_tree(leaf_index)");
  db.run("CREATE INDEX IF NOT EXISTS idx_merkle_level ON merkle_tree(level)");
}

/**
 * Add a leaf to the Merkle tree
 * Returns the leaf index
 */
export async function addLeaf(db: Database, leafHash: Uint8Array): Promise<number> {
  // Ensure tables exist
  initializeMerkleTables(db);

  // Get current tree size
  const treeSize = getTreeSize(db);
  const leafIndex = treeSize;

  // Convert hash to base64
  const hashBase64 = btoa(String.fromCharCode(...leafHash));

  // Insert leaf at level 0
  const stmt = db.prepare(`
    INSERT INTO merkle_tree (leaf_index, hash, level)
    VALUES (?, ?, 0)
  `);

  stmt.run(leafIndex, hashBase64);

  // Update tree size
  updateTreeSize(db, treeSize + 1);

  return leafIndex;
}

/**
 * Get current tree size
 */
export function getTreeSize(db: Database): number {
  // Initialize current_tree_size if it doesn't exist
  db.run(`
    CREATE TABLE IF NOT EXISTS current_tree_size (
      id INTEGER PRIMARY KEY CHECK (id = 1),
      tree_size INTEGER NOT NULL DEFAULT 0,
      last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )
  `);

  db.run("INSERT OR IGNORE INTO current_tree_size (id, tree_size) VALUES (1, 0)");

  const stmt = db.prepare("SELECT tree_size FROM current_tree_size WHERE id = 1");
  const row = stmt.get() as { tree_size: number } | null;

  return row ? row.tree_size : 0;
}

/**
 * Update tree size
 */
function updateTreeSize(db: Database, newSize: number): void {
  const stmt = db.prepare(`
    UPDATE current_tree_size
    SET tree_size = ?, last_updated = CURRENT_TIMESTAMP
    WHERE id = 1
  `);

  stmt.run(newSize);
}

/**
 * Get inclusion proof for a leaf
 * Returns array of sibling hashes from leaf to root
 */
export async function getInclusionProof(
  db: Database,
  leafIndex: number,
  treeSize: number
): Promise<Uint8Array[]> {
  if (leafIndex >= treeSize) {
    throw new Error(`Leaf index ${leafIndex} out of bounds for tree size ${treeSize}`);
  }

  if (treeSize === 0) {
    throw new Error("Cannot generate proof for empty tree");
  }

  if (treeSize === 1) {
    // Single entry tree has empty audit path
    return [];
  }

  const auditPath: Uint8Array[] = [];

  // Build audit path by traversing from leaf to root
  let index = leafIndex;
  let size = treeSize;
  let offset = 0;

  while (size > 1) {
    const k = largestPowerOfTwoLessThan(size);

    if (index < k) {
      // Leaf is in left subtree, need right subtree hash
      const rightSize = size - k;
      if (rightSize > 0) {
        const rightHash = await computeSubtreeHash(db, offset + k, rightSize);
        auditPath.push(rightHash);
      }
      size = k;
    } else {
      // Leaf is in right subtree, need left subtree hash
      const leftHash = await computeSubtreeHash(db, offset, k);
      auditPath.push(leftHash);
      index = index - k;
      offset = offset + k;
      size = size - k;
    }
  }

  // Reverse to get leaf-to-root order
  auditPath.reverse();

  return auditPath;
}

/**
 * Compute hash of a subtree
 */
async function computeSubtreeHash(
  db: Database,
  start: number,
  size: number
): Promise<Uint8Array> {
  if (size === 0) {
    throw new Error("Cannot compute hash of empty subtree");
  }

  if (size === 1) {
    // Single leaf - get from database and hash with 0x00 prefix
    const leaf = await getLeafHash(db, start);
    return await hashLeaf(leaf);
  }

  // Split subtree and recursively compute
  const k = largestPowerOfTwoLessThan(size);

  const leftHash = await computeSubtreeHash(db, start, k);
  const rightSize = size - k;

  if (rightSize === 0) {
    return leftHash;
  }

  const rightHash = await computeSubtreeHash(db, start + k, rightSize);

  // Combine left and right with 0x01 prefix
  return await hashNode(leftHash, rightHash);
}

/**
 * Get leaf hash by index
 */
async function getLeafHash(db: Database, leafIndex: number): Promise<Uint8Array> {
  const stmt = db.prepare(`
    SELECT hash FROM merkle_tree
    WHERE leaf_index = ? AND level = 0
  `);

  const row = stmt.get(leafIndex) as { hash: string } | null;

  if (!row) {
    throw new Error(`Leaf not found at index ${leafIndex}`);
  }

  // Convert base64 back to Uint8Array
  return Uint8Array.from(atob(row.hash), c => c.charCodeAt(0));
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
