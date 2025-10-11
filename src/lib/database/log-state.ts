/**
 * Merkle tree state management
 *
 * Provides operations for managing the Merkle tree state including:
 * - Getting and updating current tree size
 * - Recording tree state at specific sizes
 * - Retrieving historical tree states
 * - Managing checkpoint references
 */

import { Database } from "bun:sqlite";

export interface TreeState {
  tree_size: number;
  root_hash: string;
  checkpoint_storage_key: string;
  checkpoint_signed_note: string;
  updated_at?: string;
}

/**
 * Get the current tree size
 */
export function getCurrentTreeSize(db: Database): number {
  const stmt = db.prepare("SELECT tree_size FROM current_tree_size WHERE id = 1");
  const result = stmt.get();

  return result ? (result as any).tree_size : 0;
}

/**
 * Update the current tree size
 */
export function updateTreeSize(db: Database, newSize: number): void {
  const stmt = db.prepare(`
    UPDATE current_tree_size
    SET tree_size = ?, last_updated = CURRENT_TIMESTAMP
    WHERE id = 1
  `);

  stmt.run(newSize);
}

/**
 * Record tree state at a specific size (for checkpoints)
 */
export function recordTreeState(db: Database, state: TreeState): void {
  const stmt = db.prepare(`
    INSERT INTO tree_state (
      tree_size, root_hash, checkpoint_storage_key, checkpoint_signed_note
    ) VALUES (?, ?, ?, ?)
  `);

  stmt.run(
    state.tree_size,
    state.root_hash,
    state.checkpoint_storage_key,
    state.checkpoint_signed_note
  );
}

/**
 * Get tree state for a specific size
 */
export function getTreeState(db: Database, treeSize: number): TreeState | null {
  const stmt = db.prepare("SELECT * FROM tree_state WHERE tree_size = ?");
  const result = stmt.get(treeSize);

  return result ? (result as TreeState) : null;
}

/**
 * Get tree state history (most recent first)
 */
export function getTreeStateHistory(db: Database, limit?: number): TreeState[] {
  let query = "SELECT * FROM tree_state ORDER BY tree_size DESC";

  if (limit !== undefined) {
    query += ` LIMIT ${limit}`;
  }

  const stmt = db.prepare(query);
  return stmt.all() as TreeState[];
}

/**
 * Get the latest checkpoint (most recent tree state)
 */
export function getLatestCheckpoint(db: Database): TreeState | null {
  const stmt = db.prepare(`
    SELECT * FROM tree_state
    ORDER BY tree_size DESC
    LIMIT 1
  `);

  const result = stmt.get();
  return result ? (result as TreeState) : null;
}
