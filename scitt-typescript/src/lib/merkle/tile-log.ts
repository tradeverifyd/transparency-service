/**
 * Tile-Based Merkle Tree Implementation
 * RFC 6962 compliant Merkle tree with C2SP tlog-tiles storage format
 */

import type { Storage } from "../storage/interface.ts";
import {
  TILE_SIZE,
  HASH_SIZE,
  tileIndexToPath,
  entryTileIndexToPath,
  entryIdToTileIndex,
  entryIdToTileOffset,
} from "./tile-naming.ts";

/**
 * Tile Log State
 * Stored in object storage for persistence
 */
interface TileLogState {
  size: number; // Number of leaves in the tree
  root: Uint8Array | null; // Current tree root hash
}

/**
 * Tile-based Merkle tree
 * Implements RFC 6962 Merkle tree with C2SP tlog-tiles storage
 */
export class TileLog {
  private storage: Storage;
  private state: TileLogState;

  constructor(storage: Storage) {
    this.storage = storage;
    this.state = {
      size: 0,
      root: null,
    };
  }

  /**
   * Load tree state from storage
   * Called on initialization to restore tree from persistent storage
   */
  async load(): Promise<void> {
    try {
      const stateData = await this.storage.get(".tree-state");
      if (stateData) {
        const decoded = JSON.parse(new TextDecoder().decode(stateData));
        this.state.size = decoded.size;
        this.state.root = decoded.root ? new Uint8Array(decoded.root) : null;
      }
    } catch (error) {
      // Empty tree - no state file exists yet
      this.state = { size: 0, root: null };
    }
  }

  /**
   * Save tree state to storage
   */
  private async saveState(): Promise<void> {
    const encoded = JSON.stringify({
      size: this.state.size,
      root: this.state.root ? Array.from(this.state.root) : null,
    });
    await this.storage.put(".tree-state", new TextEncoder().encode(encoded));
  }

  /**
   * Append a leaf to the tree
   * Returns the entry ID of the appended leaf
   */
  async append(leaf: Uint8Array): Promise<number> {
    if (leaf.length !== HASH_SIZE) {
      throw new Error(`Invalid leaf size: expected ${HASH_SIZE} bytes, got ${leaf.length}`);
    }

    const entryId = this.state.size;

    // Store in entry tile
    await this.appendToEntryTile(entryId, leaf);

    // Update hash tiles
    await this.updateHashTiles(entryId, leaf);

    // Increment size and recompute root
    this.state.size++;
    this.state.root = await this.computeRoot();

    // Persist state
    await this.saveState();

    return entryId;
  }

  /**
   * Get the current tree size (number of leaves)
   */
  async size(): Promise<number> {
    return this.state.size;
  }

  /**
   * Get the current tree root hash
   */
  async root(): Promise<Uint8Array> {
    if (this.state.root === null) {
      throw new Error("Cannot get root of empty tree");
    }
    return this.state.root;
  }

  /**
   * Get a leaf by entry ID
   */
  async getLeaf(entryId: number): Promise<Uint8Array> {
    if (entryId >= this.state.size) {
      throw new Error(`Entry ID ${entryId} out of bounds (size: ${this.state.size})`);
    }

    const tileIndex = entryIdToTileIndex(entryId);
    const tileOffset = entryIdToTileOffset(entryId);

    const tilePath = entryTileIndexToPath(tileIndex);
    const tileData = await this.storage.get(tilePath);

    if (!tileData) {
      throw new Error(`Entry tile not found: ${tilePath}`);
    }

    // Extract the specific hash from the tile
    const start = tileOffset * HASH_SIZE;
    const end = start + HASH_SIZE;

    return tileData.slice(start, end);
  }

  /**
   * Append leaf to entry tile
   * Entry tiles store the raw leaf data
   */
  private async appendToEntryTile(entryId: number, leaf: Uint8Array): Promise<void> {
    const tileIndex = entryIdToTileIndex(entryId);
    const tilePath = entryTileIndexToPath(tileIndex);

    // Read existing tile (if any)
    const existingTile = await this.storage.get(tilePath);
    const currentSize = existingTile ? existingTile.length / HASH_SIZE : 0;

    // Append new leaf
    const newTile = new Uint8Array((currentSize + 1) * HASH_SIZE);
    if (existingTile) {
      newTile.set(existingTile, 0);
    }
    newTile.set(leaf, currentSize * HASH_SIZE);

    // Write updated tile
    await this.storage.put(tilePath, newTile);
  }

  /**
   * Update hash tiles after appending a leaf
   * Hash tiles store the Merkle tree structure
   */
  private async updateHashTiles(entryId: number, leaf: Uint8Array): Promise<void> {
    // Level 0: Store leaf hash in tile
    await this.appendToHashTile(0, entryId, leaf);

    // Propagate up the tree if we completed a tile
    const tileOffset = entryIdToTileOffset(entryId);
    if (tileOffset === TILE_SIZE - 1) {
      // We just completed a tile - compute its hash and propagate
      await this.propagateTileHash(0, entryIdToTileIndex(entryId));
    }
  }

  /**
   * Append hash to a tile at a specific level
   */
  private async appendToHashTile(level: number, entryId: number, hash: Uint8Array): Promise<void> {
    const tileIndex = entryIdToTileIndex(entryId);
    const tilePath = tileIndexToPath(level, tileIndex);

    // Read existing tile (if any)
    const existingTile = await this.storage.get(tilePath);
    const currentSize = existingTile ? existingTile.length / HASH_SIZE : 0;

    // Append new hash
    const newTile = new Uint8Array((currentSize + 1) * HASH_SIZE);
    if (existingTile) {
      newTile.set(existingTile, 0);
    }
    newTile.set(hash, currentSize * HASH_SIZE);

    // Write updated tile
    await this.storage.put(tilePath, newTile);
  }

  /**
   * Propagate completed tile hash to the next level
   */
  private async propagateTileHash(level: number, tileIndex: number): Promise<void> {
    // Read the completed tile
    const tilePath = tileIndexToPath(level, tileIndex);
    const tileData = await this.storage.get(tilePath);

    if (!tileData) {
      throw new Error(`Tile not found: ${tilePath}`);
    }

    // Compute hash of the tile
    const tileHash = await this.hashData(tileData);

    // Store in next level
    const nextLevel = level + 1;
    const nextEntryId = tileIndex; // Tile index becomes entry ID at next level

    await this.appendToHashTile(nextLevel, nextEntryId, tileHash);

    // Recursively propagate if we completed a tile at the next level
    const nextTileOffset = entryIdToTileOffset(nextEntryId);
    if (nextTileOffset === TILE_SIZE - 1) {
      await this.propagateTileHash(nextLevel, entryIdToTileIndex(nextEntryId));
    }
  }

  /**
   * Compute the current tree root
   * RFC 6962 MTH (Merkle Tree Hash) computation
   */
  private async computeRoot(): Promise<Uint8Array> {
    if (this.state.size === 0) {
      throw new Error("Cannot compute root of empty tree");
    }

    if (this.state.size === 1) {
      // Root is just the single leaf (RFC 6962: MTH({d[0]}) = SHA-256(0x00 || d[0]))
      const leaf = await this.getLeaf(0);
      return await this.hashLeaf(leaf);
    }

    // For larger trees, compute from tiles
    return await this.computeRootFromTiles();
  }

  /**
   * Compute root from tile structure
   * Combines full and partial tiles at each level
   */
  private async computeRootFromTiles(): Promise<Uint8Array> {
    let currentLevel = 0;
    let currentSize = this.state.size;

    // Find the highest level with data
    while (currentSize > 1) {
      const numTiles = Math.ceil(currentSize / TILE_SIZE);

      if (numTiles === 1) {
        // Single partial tile at this level - hash it to get root
        const tilePath = tileIndexToPath(currentLevel, 0);
        const tileData = await this.storage.get(tilePath);

        if (!tileData) {
          throw new Error(`Tile not found: ${tilePath}`);
        }

        // Hash all entries in the tile to get the root
        return await this.hashTileToRoot(tileData, currentSize);
      }

      // Move up one level
      currentLevel++;
      currentSize = numTiles;
    }

    // Single tile at current level
    const tilePath = tileIndexToPath(currentLevel, 0);
    const tileData = await this.storage.get(tilePath);

    if (!tileData) {
      throw new Error(`Root tile not found: ${tilePath}`);
    }

    return tileData.slice(0, HASH_SIZE);
  }

  /**
   * Hash a tile to compute root
   * Combines hashes using RFC 6962 algorithm
   * Note: Tile contains raw leaf data, so we need to hash each leaf first
   */
  private async hashTileToRoot(tileData: Uint8Array, numHashes: number): Promise<Uint8Array> {
    if (numHashes === 0) {
      throw new Error("Cannot hash empty tile");
    }

    if (numHashes === 1) {
      // Single leaf - hash it with 0x00 prefix
      const leaf = tileData.slice(0, HASH_SIZE);
      return await this.hashLeaf(leaf);
    }

    // Build a binary tree from the leaves in the tile
    // First, hash each leaf with 0x00 prefix
    const hashes: Uint8Array[] = [];
    for (let i = 0; i < numHashes; i++) {
      const start = i * HASH_SIZE;
      const leaf = tileData.slice(start, start + HASH_SIZE);
      const leafHash = await this.hashLeaf(leaf);
      hashes.push(leafHash);
    }

    // Recursively combine hashes
    return await this.mergeHashes(hashes);
  }

  /**
   * Recursively merge hashes into a single root
   * RFC 6962: MTH(D[n]) = SHA-256(0x01 || MTH(D[0:k]) || MTH(D[k:n]))
   * where k is the largest power of 2 less than n
   */
  private async mergeHashes(hashes: Uint8Array[]): Promise<Uint8Array> {
    const n = hashes.length;

    if (n === 0) {
      throw new Error("Cannot merge empty hash list");
    }

    if (n === 1) {
      return hashes[0];
    }

    if (n === 2) {
      // Base case: combine two hashes
      return await this.hashNode(hashes[0], hashes[1]);
    }

    // Find k: largest power of 2 less than n
    const k = this.largestPowerOfTwoLessThan(n);

    const left = hashes.slice(0, k);
    const right = hashes.slice(k);

    const leftHash = await this.mergeHashes(left);
    const rightHash = await this.mergeHashes(right);

    return await this.hashNode(leftHash, rightHash);
  }

  /**
   * Find largest power of 2 strictly less than n
   * For RFC 6962 MTH algorithm
   */
  private largestPowerOfTwoLessThan(n: number): number {
    let k = 1;
    while (k * 2 < n) {
      k *= 2;
    }
    return k;
  }

  /**
   * Hash a leaf with RFC 6962 prefix (0x00)
   */
  private async hashLeaf(leaf: Uint8Array): Promise<Uint8Array> {
    const data = new Uint8Array(1 + leaf.length);
    data[0] = 0x00; // RFC 6962 leaf prefix
    data.set(leaf, 1);
    return await this.hashData(data);
  }

  /**
   * Hash an internal node with RFC 6962 prefix (0x01)
   */
  private async hashNode(left: Uint8Array, right: Uint8Array): Promise<Uint8Array> {
    const data = new Uint8Array(1 + left.length + right.length);
    data[0] = 0x01; // RFC 6962 node prefix
    data.set(left, 1);
    data.set(right, 1 + left.length);
    return await this.hashData(data);
  }

  /**
   * Hash data using SHA-256
   */
  private async hashData(data: Uint8Array): Promise<Uint8Array> {
    const hashBuffer = await crypto.subtle.digest("SHA-256", data);
    return new Uint8Array(hashBuffer);
  }
}
