/**
 * Unit tests for Merkle tree state management
 *
 * Tests log state operations including:
 * - Getting current tree size
 * - Updating tree size
 * - Recording tree state at specific sizes
 * - Getting tree state history
 * - Managing checkpoint references
 */

import { describe, test, expect, beforeEach } from "bun:test";
import { Database } from "bun:sqlite";
import * as fs from "fs";
import {
  getCurrentTreeSize,
  updateTreeSize,
  recordTreeState,
  getTreeState,
  getTreeStateHistory,
  getLatestCheckpoint,
} from "../../../src/lib/database/log-state.ts";

const TEST_DB_PATH = "tests/.test-log-state.db";

describe("Log State Management", () => {
  let db: Database;

  beforeEach(() => {
    // Clean up any existing test database
    if (fs.existsSync(TEST_DB_PATH)) {
      fs.unlinkSync(TEST_DB_PATH);
    }

    // Create fresh database with schema
    db = new Database(TEST_DB_PATH);

    // Create tables
    db.exec(`
      CREATE TABLE current_tree_size (
        id INTEGER PRIMARY KEY CHECK (id = 1),
        tree_size INTEGER NOT NULL DEFAULT 0,
        last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
      );

      INSERT INTO current_tree_size (id, tree_size) VALUES (1, 0);

      CREATE TABLE tree_state (
        tree_size INTEGER PRIMARY KEY,
        root_hash TEXT NOT NULL,
        checkpoint_storage_key TEXT NOT NULL,
        checkpoint_signed_note TEXT NOT NULL,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
      );
    `);
  });

  test("getCurrentTreeSize returns initial size of 0", () => {
    const size = getCurrentTreeSize(db);
    expect(size).toBe(0);
  });

  test("updateTreeSize increments tree size", () => {
    expect(getCurrentTreeSize(db)).toBe(0);

    updateTreeSize(db, 1);
    expect(getCurrentTreeSize(db)).toBe(1);

    updateTreeSize(db, 2);
    expect(getCurrentTreeSize(db)).toBe(2);

    updateTreeSize(db, 10);
    expect(getCurrentTreeSize(db)).toBe(10);
  });

  test("updateTreeSize updates last_updated timestamp", () => {
    // Get initial timestamp
    updateTreeSize(db, 1);
    const before = db.query(
      "SELECT last_updated FROM current_tree_size WHERE id = 1"
    ).get();

    // Wait to ensure timestamp changes (SQLite CURRENT_TIMESTAMP is second precision)
    Bun.sleepSync(1100); // 1.1 seconds to ensure change

    updateTreeSize(db, 5);

    const after = db.query(
      "SELECT last_updated FROM current_tree_size WHERE id = 1"
    ).get();

    expect(after.last_updated).not.toBe(before.last_updated);
  });

  test("recordTreeState stores tree state with checkpoint", () => {
    const state = {
      tree_size: 10,
      root_hash: "abc123def456",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: "example.com/log\n10\nabc123def456\n\n— example.com signature",
    };

    recordTreeState(db, state);

    const row = db.query("SELECT * FROM tree_state WHERE tree_size = ?").get(10);
    expect(row).toBeTruthy();
    expect(row.tree_size).toBe(10);
    expect(row.root_hash).toBe("abc123def456");
    expect(row.checkpoint_storage_key).toBe("checkpoint");
    expect(row.checkpoint_signed_note).toContain("example.com/log");
  });

  test("getTreeState retrieves specific tree state", () => {
    recordTreeState(db, {
      tree_size: 5,
      root_hash: "hash-5",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: "note-5",
    });

    recordTreeState(db, {
      tree_size: 10,
      root_hash: "hash-10",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: "note-10",
    });

    const state5 = getTreeState(db, 5);
    expect(state5).toBeTruthy();
    expect(state5.tree_size).toBe(5);
    expect(state5.root_hash).toBe("hash-5");

    const state10 = getTreeState(db, 10);
    expect(state10).toBeTruthy();
    expect(state10.tree_size).toBe(10);
    expect(state10.root_hash).toBe("hash-10");
  });

  test("getTreeState returns null for non-existent size", () => {
    const state = getTreeState(db, 999);
    expect(state).toBe(null);
  });

  test("getTreeStateHistory returns states in descending order", () => {
    recordTreeState(db, {
      tree_size: 1,
      root_hash: "hash-1",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: "note-1",
    });

    recordTreeState(db, {
      tree_size: 5,
      root_hash: "hash-5",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: "note-5",
    });

    recordTreeState(db, {
      tree_size: 10,
      root_hash: "hash-10",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: "note-10",
    });

    const history = getTreeStateHistory(db);

    expect(history).toHaveLength(3);
    expect(history[0].tree_size).toBe(10);
    expect(history[1].tree_size).toBe(5);
    expect(history[2].tree_size).toBe(1);
  });

  test("getTreeStateHistory with limit", () => {
    for (let i = 1; i <= 10; i++) {
      recordTreeState(db, {
        tree_size: i,
        root_hash: `hash-${i}`,
        checkpoint_storage_key: "checkpoint",
        checkpoint_signed_note: `note-${i}`,
      });
    }

    const history = getTreeStateHistory(db, 3);

    expect(history).toHaveLength(3);
    expect(history[0].tree_size).toBe(10);
    expect(history[1].tree_size).toBe(9);
    expect(history[2].tree_size).toBe(8);
  });

  test("getLatestCheckpoint returns most recent tree state", () => {
    recordTreeState(db, {
      tree_size: 1,
      root_hash: "hash-1",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: "note-1",
    });

    recordTreeState(db, {
      tree_size: 5,
      root_hash: "hash-5",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: "note-5",
    });

    recordTreeState(db, {
      tree_size: 10,
      root_hash: "hash-10",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: "note-10",
    });

    const latest = getLatestCheckpoint(db);

    expect(latest).toBeTruthy();
    expect(latest.tree_size).toBe(10);
    expect(latest.root_hash).toBe("hash-10");
    expect(latest.checkpoint_signed_note).toBe("note-10");
  });

  test("getLatestCheckpoint returns null when no checkpoints exist", () => {
    const latest = getLatestCheckpoint(db);
    expect(latest).toBe(null);
  });

  test("tree_size is primary key (unique)", () => {
    recordTreeState(db, {
      tree_size: 10,
      root_hash: "hash-1",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: "note-1",
    });

    // Attempting to record state with same tree_size should throw
    expect(() => {
      recordTreeState(db, {
        tree_size: 10,
        root_hash: "hash-2",
        checkpoint_storage_key: "checkpoint",
        checkpoint_signed_note: "note-2",
      });
    }).toThrow();
  });

  test("updateTreeSize only allows single row (singleton)", () => {
    // The CHECK constraint ensures only id=1 can exist
    expect(() => {
      db.exec("INSERT INTO current_tree_size (id, tree_size) VALUES (2, 100)");
    }).toThrow();
  });

  test("tree state tracks append-only log growth", () => {
    // Simulate log growing
    updateTreeSize(db, 1);
    recordTreeState(db, {
      tree_size: 1,
      root_hash: "root-1",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: "checkpoint-1",
    });

    updateTreeSize(db, 5);
    recordTreeState(db, {
      tree_size: 5,
      root_hash: "root-5",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: "checkpoint-5",
    });

    updateTreeSize(db, 10);
    recordTreeState(db, {
      tree_size: 10,
      root_hash: "root-10",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: "checkpoint-10",
    });

    // Verify current size
    expect(getCurrentTreeSize(db)).toBe(10);

    // Verify we can retrieve any historical state
    expect(getTreeState(db, 1).root_hash).toBe("root-1");
    expect(getTreeState(db, 5).root_hash).toBe("root-5");
    expect(getTreeState(db, 10).root_hash).toBe("root-10");
  });

  test("checkpoint_signed_note stores full signed note content", () => {
    const signedNote = `transparency.example.com/log
100
abc123def456789
timestamp 1633024800

— transparency.example.com wXyz123abc...`;

    recordTreeState(db, {
      tree_size: 100,
      root_hash: "abc123def456789",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: signedNote,
    });

    const state = getTreeState(db, 100);
    expect(state.checkpoint_signed_note).toBe(signedNote);
    expect(state.checkpoint_signed_note).toContain("transparency.example.com/log");
    expect(state.checkpoint_signed_note).toContain("— transparency.example.com");
  });

  test("empty history returns empty array", () => {
    const history = getTreeStateHistory(db);
    expect(history).toEqual([]);
  });

  test("concurrent tree size updates maintain consistency", () => {
    // Simulate multiple registrations
    for (let i = 1; i <= 100; i++) {
      updateTreeSize(db, i);
    }

    expect(getCurrentTreeSize(db)).toBe(100);
  });

  test("tree state updated_at is automatically set", () => {
    recordTreeState(db, {
      tree_size: 10,
      root_hash: "hash-10",
      checkpoint_storage_key: "checkpoint",
      checkpoint_signed_note: "note-10",
    });

    const state = getTreeState(db, 10);
    expect(state.updated_at).toBeTruthy();
    expect(typeof state.updated_at).toBe("string");
  });
});
