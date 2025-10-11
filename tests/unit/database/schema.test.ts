/**
 * Database Schema Tests
 * Test suite for SQLite schema and migrations
 */

import { describe, test, expect, beforeEach, afterEach } from "bun:test";
import { Database } from "bun:sqlite";
import * as fs from "fs";

describe("Database Schema", () => {
  const testDbPath = "./.test-db.sqlite";
  let db: Database;

  beforeEach(() => {
    // Clean up test database
    if (fs.existsSync(testDbPath)) {
      fs.unlinkSync(testDbPath);
    }
  });

  afterEach(() => {
    if (db) {
      db.close();
    }
    if (fs.existsSync(testDbPath)) {
      fs.unlinkSync(testDbPath);
    }
  });

  test("initializeSchema creates all required tables", async () => {
    db = new Database(testDbPath);

    // Import and run schema initialization
    const { initializeSchema } = await import("../../../src/lib/database/schema.ts");
    await initializeSchema(db);

    // Check that all tables exist
    const tables = db.query("SELECT name FROM sqlite_master WHERE type='table'").all() as Array<{ name: string }>;
    const tableNames = tables.map(t => t.name);

    expect(tableNames).toContain("schema_version");
    expect(tableNames).toContain("statements");
    expect(tableNames).toContain("receipts");
    expect(tableNames).toContain("tiles");
    expect(tableNames).toContain("tree_state");
    expect(tableNames).toContain("current_tree_size");
    expect(tableNames).toContain("service_config");
    expect(tableNames).toContain("service_keys");
  });

  test("schema_version table has initial version", async () => {
    db = new Database(testDbPath);
    const { initializeSchema } = await import("../../../src/lib/database/schema.ts");
    await initializeSchema(db);

    const version = db.query("SELECT version FROM schema_version ORDER BY applied_at DESC LIMIT 1").get() as { version: string } | null;

    expect(version).not.toBeNull();
    expect(version?.version).toBe("1.0.0");
  });

  test("statements table has correct schema", async () => {
    db = new Database(testDbPath);
    const { initializeSchema } = await import("../../../src/lib/database/schema.ts");
    await initializeSchema(db);

    const columns = db.query("PRAGMA table_info(statements)").all() as Array<{ name: string; type: string }>;
    const columnNames = columns.map(c => c.name);

    expect(columnNames).toContain("entry_id");
    expect(columnNames).toContain("statement_hash");
    expect(columnNames).toContain("iss");
    expect(columnNames).toContain("sub");
    expect(columnNames).toContain("cty");
    expect(columnNames).toContain("payload_hash_alg");
    expect(columnNames).toContain("payload_hash");
    expect(columnNames).toContain("preimage_content_type");
    expect(columnNames).toContain("payload_location");
    expect(columnNames).toContain("registered_at");
    expect(columnNames).toContain("tree_size_at_registration");
    expect(columnNames).toContain("entry_tile_key");
    expect(columnNames).toContain("entry_tile_offset");
  });

  test("statements table has required indexes", async () => {
    db = new Database(testDbPath);
    const { initializeSchema } = await import("../../../src/lib/database/schema.ts");
    await initializeSchema(db);

    const indexes = db.query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='statements'").all() as Array<{ name: string }>;
    const indexNames = indexes.map(i => i.name);

    expect(indexNames).toContain("idx_statements_iss");
    expect(indexNames).toContain("idx_statements_sub");
    expect(indexNames).toContain("idx_statements_cty");
    expect(indexNames).toContain("idx_statements_hash");
  });

  test("receipts table has correct schema", async () => {
    db = new Database(testDbPath);
    const { initializeSchema } = await import("../../../src/lib/database/schema.ts");
    await initializeSchema(db);

    const columns = db.query("PRAGMA table_info(receipts)").all() as Array<{ name: string }>;
    const columnNames = columns.map(c => c.name);

    expect(columnNames).toContain("entry_id");
    expect(columnNames).toContain("receipt_hash");
    expect(columnNames).toContain("storage_key");
    expect(columnNames).toContain("tree_size");
    expect(columnNames).toContain("leaf_index");
  });

  test("tiles table has correct schema", async () => {
    db = new Database(testDbPath);
    const { initializeSchema } = await import("../../../src/lib/database/schema.ts");
    await initializeSchema(db);

    const columns = db.query("PRAGMA table_info(tiles)").all() as Array<{ name: string }>;
    const columnNames = columns.map(c => c.name);

    expect(columnNames).toContain("tile_id");
    expect(columnNames).toContain("level");
    expect(columnNames).toContain("tile_index");
    expect(columnNames).toContain("storage_key");
    expect(columnNames).toContain("is_partial");
    expect(columnNames).toContain("width");
    expect(columnNames).toContain("tile_hash");
  });

  test("current_tree_size table initializes with zero", async () => {
    db = new Database(testDbPath);
    const { initializeSchema } = await import("../../../src/lib/database/schema.ts");
    await initializeSchema(db);

    const row = db.query("SELECT tree_size FROM current_tree_size WHERE id = 1").get() as { tree_size: number } | null;

    expect(row).not.toBeNull();
    expect(row?.tree_size).toBe(0);
  });

  test("service_config table has initial configuration", async () => {
    db = new Database(testDbPath);
    const { initializeSchema } = await import("../../../src/lib/database/schema.ts");
    await initializeSchema(db);

    const configs = db.query("SELECT key, value FROM service_config").all() as Array<{ key: string; value: string }>;
    const configMap = new Map(configs.map(c => [c.key, c.value]));

    expect(configMap.has("service_url")).toBe(true);
    expect(configMap.has("tile_height")).toBe(true);
    expect(configMap.has("hash_algorithm")).toBe(true);
    expect(configMap.has("signature_algorithm")).toBe(true);
  });

  test("WAL mode can be enabled", async () => {
    db = new Database(testDbPath);
    const { initializeSchema, enableWAL } = await import("../../../src/lib/database/schema.ts");
    await initializeSchema(db);
    await enableWAL(db);

    const result = db.query("PRAGMA journal_mode").get() as { journal_mode: string } | null;

    expect(result?.journal_mode).toBe("wal");
  });

  test("statement_hash uniqueness is enforced", async () => {
    db = new Database(testDbPath);
    const { initializeSchema } = await import("../../../src/lib/database/schema.ts");
    await initializeSchema(db);

    // Insert first statement
    db.run(`
      INSERT INTO statements (statement_hash, iss, payload_hash_alg, payload_hash, tree_size_at_registration, entry_tile_key, entry_tile_offset)
      VALUES (?, ?, ?, ?, ?, ?, ?)
    `, ["hash123", "https://example.com", -16, "abc123", 1, "tile/entries/000", 0]);

    // Try to insert duplicate - should fail
    expect(() => {
      db.run(`
        INSERT INTO statements (statement_hash, iss, payload_hash_alg, payload_hash, tree_size_at_registration, entry_tile_key, entry_tile_offset)
        VALUES (?, ?, ?, ?, ?, ?, ?)
      `, ["hash123", "https://example.com", -16, "abc123", 1, "tile/entries/000", 1]);
    }).toThrow();
  });
});
