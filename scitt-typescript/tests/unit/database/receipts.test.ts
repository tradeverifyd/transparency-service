/**
 * Unit tests for receipt storage and retrieval
 *
 * Tests receipt operations including:
 * - Inserting receipts
 * - Retrieving receipts by entry ID
 * - Retrieving receipts by hash
 * - Getting storage keys for receipts
 * - Handling missing receipts
 */

import { describe, test, expect, beforeEach } from "bun:test";
import { Database } from "bun:sqlite";
import * as fs from "fs";
import {
  insertReceipt,
  getReceiptByEntryId,
  getReceiptByHash,
  getReceiptStorageKey,
  hasReceipt,
} from "../../../src/lib/database/receipts.ts";

const TEST_DB_PATH = "tests/.test-receipts.db";

describe("Receipt Storage", () => {
  let db: Database;

  beforeEach(() => {
    // Clean up any existing test database
    if (fs.existsSync(TEST_DB_PATH)) {
      fs.unlinkSync(TEST_DB_PATH);
    }

    // Create fresh database with schema
    db = new Database(TEST_DB_PATH);

    // Enable foreign key constraints
    db.exec("PRAGMA foreign_keys = ON");

    // Create tables
    db.exec(`
      CREATE TABLE statements (
        entry_id INTEGER PRIMARY KEY AUTOINCREMENT,
        statement_hash TEXT UNIQUE NOT NULL,
        iss TEXT NOT NULL,
        sub TEXT,
        cty TEXT,
        typ TEXT,
        payload_hash_alg INTEGER NOT NULL,
        payload_hash TEXT NOT NULL,
        preimage_content_type TEXT,
        payload_location TEXT,
        registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        tree_size_at_registration INTEGER NOT NULL,
        entry_tile_key TEXT NOT NULL,
        entry_tile_offset INTEGER NOT NULL
      );

      CREATE TABLE receipts (
        entry_id INTEGER PRIMARY KEY,
        receipt_hash TEXT UNIQUE NOT NULL,
        storage_key TEXT UNIQUE NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        tree_size INTEGER NOT NULL,
        leaf_index INTEGER NOT NULL,
        FOREIGN KEY (entry_id) REFERENCES statements(entry_id)
      );
    `);

    // Insert a test statement for foreign key constraint
    db.exec(`
      INSERT INTO statements (entry_id, statement_hash, iss, sub, cty, typ, payload_hash_alg, payload_hash, tree_size_at_registration, entry_tile_key, entry_tile_offset)
      VALUES (1, 'stmt-hash-1', 'https://issuer.com', NULL, NULL, NULL, -16, 'payload-hash-1', 1, 'tile/entries/000', 0)
    `);
  });

  test("insertReceipt adds receipt with all fields", () => {
    const receipt = {
      entry_id: 1,
      receipt_hash: "receipt-hash-abc",
      storage_key: "receipts/1",
      tree_size: 10,
      leaf_index: 0,
    };

    insertReceipt(db, receipt);

    // Verify insertion
    const row = db.query("SELECT * FROM receipts WHERE entry_id = ?").get(1);
    expect(row).toBeTruthy();
    expect(row.entry_id).toBe(1);
    expect(row.receipt_hash).toBe("receipt-hash-abc");
    expect(row.storage_key).toBe("receipts/1");
    expect(row.tree_size).toBe(10);
    expect(row.leaf_index).toBe(0);
  });

  test("insertReceipt generates correct storage key format", () => {
    const receipt = {
      entry_id: 1,
      receipt_hash: "hash-123",
      storage_key: "receipts/1",
      tree_size: 5,
      leaf_index: 0,
    };

    insertReceipt(db, receipt);

    const row = db.query("SELECT storage_key FROM receipts WHERE entry_id = ?").get(1);
    expect(row.storage_key).toMatch(/^receipts\/\d+$/);
  });

  test("getReceiptByEntryId returns correct receipt", () => {
    insertReceipt(db, {
      entry_id: 1,
      receipt_hash: "hash-1",
      storage_key: "receipts/1",
      tree_size: 10,
      leaf_index: 0,
    });

    const receipt = getReceiptByEntryId(db, 1);

    expect(receipt).toBeTruthy();
    expect(receipt.entry_id).toBe(1);
    expect(receipt.receipt_hash).toBe("hash-1");
    expect(receipt.storage_key).toBe("receipts/1");
    expect(receipt.tree_size).toBe(10);
    expect(receipt.leaf_index).toBe(0);
  });

  test("getReceiptByEntryId returns null for non-existent ID", () => {
    const receipt = getReceiptByEntryId(db, 999);
    expect(receipt).toBe(null);
  });

  test("getReceiptByHash returns correct receipt", () => {
    insertReceipt(db, {
      entry_id: 1,
      receipt_hash: "unique-receipt-hash",
      storage_key: "receipts/1",
      tree_size: 10,
      leaf_index: 0,
    });

    const receipt = getReceiptByHash(db, "unique-receipt-hash");

    expect(receipt).toBeTruthy();
    expect(receipt.receipt_hash).toBe("unique-receipt-hash");
    expect(receipt.entry_id).toBe(1);
  });

  test("getReceiptByHash returns null for non-existent hash", () => {
    const receipt = getReceiptByHash(db, "non-existent-hash");
    expect(receipt).toBe(null);
  });

  test("getReceiptStorageKey returns correct key", () => {
    insertReceipt(db, {
      entry_id: 1,
      receipt_hash: "hash-1",
      storage_key: "receipts/1",
      tree_size: 10,
      leaf_index: 0,
    });

    const storageKey = getReceiptStorageKey(db, 1);

    expect(storageKey).toBe("receipts/1");
  });

  test("getReceiptStorageKey returns null for non-existent entry", () => {
    const storageKey = getReceiptStorageKey(db, 999);
    expect(storageKey).toBe(null);
  });

  test("hasReceipt returns true when receipt exists", () => {
    insertReceipt(db, {
      entry_id: 1,
      receipt_hash: "hash-1",
      storage_key: "receipts/1",
      tree_size: 10,
      leaf_index: 0,
    });

    expect(hasReceipt(db, 1)).toBe(true);
  });

  test("hasReceipt returns false when receipt does not exist", () => {
    expect(hasReceipt(db, 999)).toBe(false);
  });

  test("receipt_hash must be unique", () => {
    insertReceipt(db, {
      entry_id: 1,
      receipt_hash: "duplicate-hash",
      storage_key: "receipts/1",
      tree_size: 10,
      leaf_index: 0,
    });

    // Insert another statement for second receipt
    db.exec(`
      INSERT INTO statements (entry_id, statement_hash, iss, sub, cty, typ, payload_hash_alg, payload_hash, tree_size_at_registration, entry_tile_key, entry_tile_offset)
      VALUES (2, 'stmt-hash-2', 'https://issuer.com', NULL, NULL, NULL, -16, 'payload-hash-2', 2, 'tile/entries/000', 1)
    `);

    // Attempting to insert receipt with duplicate hash should throw
    expect(() => {
      insertReceipt(db, {
        entry_id: 2,
        receipt_hash: "duplicate-hash",
        storage_key: "receipts/2",
        tree_size: 10,
        leaf_index: 1,
      });
    }).toThrow();
  });

  test("storage_key must be unique", () => {
    insertReceipt(db, {
      entry_id: 1,
      receipt_hash: "hash-1",
      storage_key: "receipts/duplicate",
      tree_size: 10,
      leaf_index: 0,
    });

    // Insert another statement
    db.exec(`
      INSERT INTO statements (entry_id, statement_hash, iss, sub, cty, typ, payload_hash_alg, payload_hash, tree_size_at_registration, entry_tile_key, entry_tile_offset)
      VALUES (2, 'stmt-hash-2', 'https://issuer.com', NULL, NULL, NULL, -16, 'payload-hash-2', 2, 'tile/entries/000', 1)
    `);

    // Attempting to insert receipt with duplicate storage_key should throw
    expect(() => {
      insertReceipt(db, {
        entry_id: 2,
        receipt_hash: "hash-2",
        storage_key: "receipts/duplicate",
        tree_size: 10,
        leaf_index: 1,
      });
    }).toThrow();
  });

  test("receipt requires valid entry_id (foreign key)", () => {
    // Attempting to insert receipt with non-existent entry_id should throw
    expect(() => {
      insertReceipt(db, {
        entry_id: 999,
        receipt_hash: "hash-1",
        storage_key: "receipts/999",
        tree_size: 10,
        leaf_index: 0,
      });
    }).toThrow();
  });

  test("multiple receipts can be stored", () => {
    // Insert more statements
    db.exec(`
      INSERT INTO statements (entry_id, statement_hash, iss, sub, cty, typ, payload_hash_alg, payload_hash, tree_size_at_registration, entry_tile_key, entry_tile_offset)
      VALUES
        (2, 'stmt-hash-2', 'https://issuer.com', NULL, NULL, NULL, -16, 'payload-hash-2', 2, 'tile/entries/000', 1),
        (3, 'stmt-hash-3', 'https://issuer.com', NULL, NULL, NULL, -16, 'payload-hash-3', 3, 'tile/entries/000', 2)
    `);

    insertReceipt(db, {
      entry_id: 1,
      receipt_hash: "hash-1",
      storage_key: "receipts/1",
      tree_size: 10,
      leaf_index: 0,
    });

    insertReceipt(db, {
      entry_id: 2,
      receipt_hash: "hash-2",
      storage_key: "receipts/2",
      tree_size: 11,
      leaf_index: 1,
    });

    insertReceipt(db, {
      entry_id: 3,
      receipt_hash: "hash-3",
      storage_key: "receipts/3",
      tree_size: 12,
      leaf_index: 2,
    });

    const count = db.query("SELECT COUNT(*) as count FROM receipts").get();
    expect(count.count).toBe(3);
  });

  test("leaf_index tracks position in log", () => {
    // Insert more statements
    db.exec(`
      INSERT INTO statements (entry_id, statement_hash, iss, sub, cty, typ, payload_hash_alg, payload_hash, tree_size_at_registration, entry_tile_key, entry_tile_offset)
      VALUES
        (2, 'stmt-hash-2', 'https://issuer.com', NULL, NULL, NULL, -16, 'payload-hash-2', 2, 'tile/entries/000', 1),
        (3, 'stmt-hash-3', 'https://issuer.com', NULL, NULL, NULL, -16, 'payload-hash-3', 3, 'tile/entries/000', 2)
    `);

    insertReceipt(db, {
      entry_id: 1,
      receipt_hash: "hash-1",
      storage_key: "receipts/1",
      tree_size: 10,
      leaf_index: 0,
    });

    insertReceipt(db, {
      entry_id: 2,
      receipt_hash: "hash-2",
      storage_key: "receipts/2",
      tree_size: 11,
      leaf_index: 1,
    });

    insertReceipt(db, {
      entry_id: 3,
      receipt_hash: "hash-3",
      storage_key: "receipts/3",
      tree_size: 12,
      leaf_index: 2,
    });

    const receipt1 = getReceiptByEntryId(db, 1);
    const receipt2 = getReceiptByEntryId(db, 2);
    const receipt3 = getReceiptByEntryId(db, 3);

    expect(receipt1.leaf_index).toBe(0);
    expect(receipt2.leaf_index).toBe(1);
    expect(receipt3.leaf_index).toBe(2);
  });
});
