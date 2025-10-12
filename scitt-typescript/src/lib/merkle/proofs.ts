/**
 * Merkle Proof Generation and Verification
 * RFC 6962 compliant inclusion and consistency proofs
 */

import type { Storage } from "../storage/interface.ts";
import {
  HASH_SIZE,
  entryTileIndexToPath,
  entryIdToTileIndex,
  entryIdToTileOffset,
} from "./tile-naming.ts";

/**
 * Inclusion proof structure
 * Proves that a leaf is included in a tree of a given size
 */
export interface InclusionProof {
  leafIndex: number;
  treeSize: number;
  auditPath: Uint8Array[]; // Hashes from leaf to root
}

/**
 * Consistency proof structure
 * Proves that an older tree is a prefix of a newer tree
 */
export interface ConsistencyProof {
  oldSize: number;
  newSize: number;
  proof: Uint8Array[]; // Hashes proving consistency
}

/**
 * Generate RFC 6962 inclusion proof
 * Proves that leaf at leafIndex is included in tree of treeSize
 */
export async function generateInclusionProof(
  storage: Storage,
  leafIndex: number,
  treeSize: number
): Promise<InclusionProof> {
  if (leafIndex >= treeSize) {
    throw new Error(`Leaf index ${leafIndex} out of bounds for tree size ${treeSize}`);
  }

  if (treeSize === 0) {
    throw new Error("Cannot generate proof for empty tree");
  }

  if (treeSize === 1) {
    // Single entry tree has empty audit path
    return {
      leafIndex,
      treeSize,
      auditPath: [],
    };
  }

  const auditPath: Uint8Array[] = [];

  // Build audit path by traversing from root to leaf
  // Collect siblings as we go, then reverse at the end
  let index = leafIndex;
  let size = treeSize;
  let offset = 0; // Track our position in the original tree

  while (size > 1) {
    // Find sibling node
    const k = largestPowerOfTwoLessThan(size);

    if (index < k) {
      // Leaf is in left subtree, need right subtree hash
      const rightSize = size - k;
      if (rightSize > 0) {
        const rightHash = await computeSubtreeHash(storage, offset + k, rightSize);
        auditPath.push(rightHash);
      }
      // Continue in left subtree
      size = k;
      // offset stays the same
    } else {
      // Leaf is in right subtree, need left subtree hash
      const leftHash = await computeSubtreeHash(storage, offset, k);
      auditPath.push(leftHash);
      // Continue in right subtree
      index = index - k;
      offset = offset + k;
      size = size - k;
    }
  }

  // Reverse to get leaf-to-root order
  auditPath.reverse();

  return {
    leafIndex,
    treeSize,
    auditPath,
  };
}

/**
 * Verify RFC 6962 inclusion proof
 * Note: leaf should be the raw leaf data (will be hashed with 0x00 prefix)
 */
export async function verifyInclusionProof(
  leaf: Uint8Array,
  proof: InclusionProof,
  root: Uint8Array
): Promise<boolean> {
  try {
    if (proof.treeSize === 0) {
      return false;
    }

    if (proof.treeSize === 1) {
      // Single entry - hash leaf and compare
      const leafHash = await hashLeaf(leaf);
      return areEqual(leafHash, root);
    }

    // Start with leaf hash (RFC 6962: 0x00 || leaf)
    let currentHash = await hashLeaf(leaf);

    // Verify by building up from leaf to root, mirroring generation logic
    // The audit path contains siblings encountered during root-to-leaf traversal,
    // now reversed to leaf-to-root order.

    // We need to reconstruct the tree traversal to know which subtree we're in at each level
    const treeStates: Array<{ index: number; size: number }> = [];

    // First, recreate the traversal path from root to leaf
    let idx = proof.leafIndex;
    let sz = proof.treeSize;
    while (sz > 1) {
      treeStates.push({ index: idx, size: sz });
      const k = largestPowerOfTwoLessThan(sz);
      if (idx < k) {
        sz = k;
      } else {
        idx = idx - k;
        sz = sz - k;
      }
    }

    // Now verify: at each level (bottom to top), combine with the sibling
    for (let i = treeStates.length - 1; i >= 0; i--) {
      const state = treeStates[i];
      const sibling = proof.auditPath[treeStates.length - 1 - i];
      const k = largestPowerOfTwoLessThan(state.size);

      if (state.index < k) {
        // In left subtree, sibling is on right
        currentHash = await hashNode(currentHash, sibling);
      } else {
        // In right subtree, sibling is on left
        currentHash = await hashNode(sibling, currentHash);
      }
    }

    return areEqual(currentHash, root);
  } catch (error) {
    return false;
  }
}

/**
 * Generate RFC 6962 consistency proof
 * Proves that tree of oldSize is a prefix of tree of newSize
 */
export async function generateConsistencyProof(
  storage: Storage,
  oldSize: number,
  newSize: number
): Promise<ConsistencyProof> {
  if (oldSize > newSize) {
    throw new Error(`Old size ${oldSize} cannot be greater than new size ${newSize}`);
  }

  if (newSize === 0) {
    throw new Error("Cannot generate proof for empty tree");
  }

  if (oldSize === newSize) {
    // Same tree - empty proof
    return {
      oldSize,
      newSize,
      proof: [],
    };
  }

  if (oldSize === 0) {
    // No old tree - return empty proof
    return {
      oldSize,
      newSize,
      proof: [],
    };
  }

  const proof: Uint8Array[] = [];

  // Generate proof nodes
  await consistencyProofHelper(storage, oldSize, newSize, 0, newSize, proof);

  return {
    oldSize,
    newSize,
    proof,
  };
}

/**
 * Recursive helper for consistency proof generation
 * Matches Go's treeProof algorithm exactly
 */
async function consistencyProofHelper(
  storage: Storage,
  oldSize: number,
  newSize: number,
  lo: number,
  hi: number,
  proof: Uint8Array[]
): Promise<void> {
  // Validate invariant
  if (!(lo < oldSize && oldSize <= hi)) {
    throw new Error(`Invalid range in consistencyProofHelper: lo=${lo}, n=${oldSize}, hi=${hi}`);
  }

  // Base case: reached the exact old tree boundary
  if (oldSize === hi) {
    if (lo === 0) {
      // Root of old tree - nothing to add
      return;
    }
    // Add the hash of this subtree
    const hash = await computeSubtreeHash(storage, lo, hi - lo);
    proof.push(hash);
    return;
  }

  // Find split point
  const k = largestPowerOfTwoLessThan(hi - lo);

  if (oldSize <= lo + k) {
    // Old tree ends in left subtree
    // Recurse left, then add right subtree hash
    await consistencyProofHelper(storage, oldSize, newSize, lo, lo + k, proof);
    const rightHash = await computeSubtreeHash(storage, lo + k, hi - (lo + k));
    proof.push(rightHash);
  } else {
    // Old tree extends into right subtree
    // Recurse right FIRST, then add left subtree hash AT THE END
    // This matches Go's: append(p, th) where p is from right recursion
    const leftHash = await computeSubtreeHash(storage, lo, k);
    await consistencyProofHelper(storage, oldSize, newSize, lo + k, hi, proof);
    proof.push(leftHash); // Add left hash AFTER right recursion
  }
}

/**
 * Verify RFC 6962 consistency proof
 */
export async function verifyConsistencyProof(
  proof: ConsistencyProof,
  oldRoot: Uint8Array,
  newRoot: Uint8Array
): Promise<boolean> {
  try {
    if (proof.newSize < 1 || proof.oldSize < 1 || proof.oldSize > proof.newSize) {
      return false;
    }

    if (proof.oldSize === proof.newSize) {
      // Same tree - roots must match and proof must be empty
      return areEqual(oldRoot, newRoot) && proof.proof.length === 0;
    }

    // Use runTreeProof to compute both old and new roots
    const [computedOldRoot, computedNewRoot] = await runTreeProof(
      proof.proof,
      0,
      proof.newSize,
      proof.oldSize,
      oldRoot,
      false // set to true for debugging
    );

    return areEqual(computedOldRoot, oldRoot) && areEqual(computedNewRoot, newRoot);
  } catch (error) {
    return false;
  }
}

/**
 * Recursive tree proof verification (based on Go's runTreeProof)
 * Returns [oldHash, newHash] computed from the proof
 */
async function runTreeProof(
  p: Uint8Array[],
  lo: number,
  hi: number,
  n: number,
  oldRoot: Uint8Array,
  debug: boolean = false
): Promise<[Uint8Array, Uint8Array]> {
  if (debug) {
    console.log(`runTreeProof: lo=${lo}, hi=${hi}, n=${n}, p.length=${p.length}`);
  }

  // Validate range
  if (!(lo < n && n <= hi)) {
    throw new Error(`Invalid range in runTreeProof: lo=${lo}, n=${n}, hi=${hi}`);
  }

  // Reached common ground - both trees are identical up to n
  if (n === hi) {
    if (lo === 0) {
      // Root of old tree
      if (p.length !== 0) {
        throw new Error("Proof too long");
      }
      if (debug) console.log("  Base case: n==hi, lo==0, returning oldRoot");
      return [oldRoot, oldRoot];
    }
    // Subtree root
    if (p.length !== 1) {
      throw new Error(`Proof length mismatch: expected 1, got ${p.length}`);
    }
    if (debug) console.log("  Base case: n==hi, lo!=0, returning p[0]");
    return [p[0], p[0]];
  }

  if (p.length === 0) {
    throw new Error("Proof too short");
  }

  // Determine subtree size
  const k = largestPowerOfTwoLessThan(hi - lo);
  if (debug) console.log(`  k=${k}, checking if n=${n} <= lo+k=${lo + k}`);

  if (n <= lo + k) {
    // Old tree ends in left subtree
    if (debug) console.log("  Branch: left subtree");
    const [oh, th] = await runTreeProof(p.slice(0, -1), lo, lo + k, n, oldRoot, debug);
    // New tree includes right subtree
    const newHash = await hashNode(th, p[p.length - 1]);
    return [oh, newHash];
  } else {
    // Old tree spans into right subtree
    if (debug) console.log("  Branch: right subtree");
    const [oh, th] = await runTreeProof(p.slice(0, -1), lo + k, hi, n, oldRoot, debug);
    // Both old and new trees include left subtree
    const oldHash = await hashNode(p[p.length - 1], oh);
    const newHash = await hashNode(p[p.length - 1], th);
    return [oldHash, newHash];
  }
}

/**
 * Compute hash of a subtree
 * Range: [start, start+size)
 *
 * Note: This computes the MTH (Merkle Tree Hash) for a subtree.
 * TileLog stores raw leaf data, so we need to hash it properly.
 */
async function computeSubtreeHash(
  storage: Storage,
  start: number,
  size: number
): Promise<Uint8Array> {
  if (size === 0) {
    throw new Error("Cannot compute hash of empty subtree");
  }

  if (size === 1) {
    // Single leaf - get from storage and hash with 0x00 prefix
    const leaf = await getLeafFromStorage(storage, start);
    return await hashLeaf(leaf);
  }

  // Split subtree and recursively compute
  // k is the largest power of 2 less than size
  const k = largestPowerOfTwoLessThan(size);

  const leftHash = await computeSubtreeHash(storage, start, k);
  const rightSize = size - k;

  if (rightSize === 0) {
    return leftHash;
  }

  const rightHash = await computeSubtreeHash(storage, start + k, rightSize);

  // Combine left and right with 0x01 prefix
  return await hashNode(leftHash, rightHash);
}

/**
 * Get leaf from storage by entry ID
 */
async function getLeafFromStorage(storage: Storage, entryId: number): Promise<Uint8Array> {
  const tileIndex = entryIdToTileIndex(entryId);
  const tileOffset = entryIdToTileOffset(entryId);

  const tilePath = entryTileIndexToPath(tileIndex);
  const tileData = await storage.get(tilePath);

  if (!tileData) {
    throw new Error(`Entry tile not found: ${tilePath}`);
  }

  const start = tileOffset * HASH_SIZE;
  const end = start + HASH_SIZE;

  return tileData.slice(start, end);
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

/**
 * Compare two Uint8Arrays for equality
 */
function areEqual(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) {
    return false;
  }

  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) {
      return false;
    }
  }

  return true;
}
