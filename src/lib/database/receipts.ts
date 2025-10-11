/**
 * Receipt storage and retrieval
 *
 * Provides operations for the receipts table including:
 * - Insertion of new receipts
 * - Retrieving receipts by entry ID
 * - Retrieving receipts by hash
 * - Getting storage keys for receipts
 * - Checking if receipts exist
 */

import { Database } from "bun:sqlite";

export interface Receipt {
  entry_id: number;
  receipt_hash: string;
  storage_key: string;
  created_at?: string;
  tree_size: number;
  leaf_index: number;
}

/**
 * Insert a new receipt into the database
 */
export function insertReceipt(db: Database, receipt: Receipt): void {
  const stmt = db.prepare(`
    INSERT INTO receipts (
      entry_id, receipt_hash, storage_key, tree_size, leaf_index
    ) VALUES (?, ?, ?, ?, ?)
  `);

  stmt.run(
    receipt.entry_id,
    receipt.receipt_hash,
    receipt.storage_key,
    receipt.tree_size,
    receipt.leaf_index
  );
}

/**
 * Get receipt by entry ID
 */
export function getReceiptByEntryId(db: Database, entryId: number): Receipt | null {
  const stmt = db.prepare("SELECT * FROM receipts WHERE entry_id = ?");
  const result = stmt.get(entryId);

  return result ? (result as Receipt) : null;
}

/**
 * Get receipt by receipt hash
 */
export function getReceiptByHash(db: Database, hash: string): Receipt | null {
  const stmt = db.prepare("SELECT * FROM receipts WHERE receipt_hash = ?");
  const result = stmt.get(hash);

  return result ? (result as Receipt) : null;
}

/**
 * Get storage key for a receipt
 */
export function getReceiptStorageKey(db: Database, entryId: number): string | null {
  const stmt = db.prepare("SELECT storage_key FROM receipts WHERE entry_id = ?");
  const result = stmt.get(entryId);

  return result ? (result as any).storage_key : null;
}

/**
 * Check if a receipt exists for an entry ID
 */
export function hasReceipt(db: Database, entryId: number): boolean {
  const stmt = db.prepare("SELECT 1 FROM receipts WHERE entry_id = ? LIMIT 1");
  const result = stmt.get(entryId);

  return result !== null;
}
