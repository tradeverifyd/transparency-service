/**
 * Unit tests for statement metadata queries
 *
 * Tests query operations for the statements table including:
 * - Finding statements by issuer (iss)
 * - Finding statements by subject (sub)
 * - Finding statements by content type (cty)
 * - Finding statements by type (typ)
 * - Finding statements by date range
 * - Combined queries (multiple filters)
 * - Getting statement by entry ID
 * - Getting statement by hash
 */

import { describe, test, expect, beforeEach } from "bun:test";
import { Database } from "bun:sqlite";
import * as fs from "fs";
import {
  insertStatement,
  findStatementsByIssuer,
  findStatementsBySubject,
  findStatementsByContentType,
  findStatementsByType,
  findStatementsByDateRange,
  findStatementsBy,
  getStatementByEntryId,
  getStatementByHash,
} from "../../../src/lib/database/statements.ts";

const TEST_DB_PATH = "tests/.test-statements-queries.db";

describe("Statement Queries", () => {
  let db: Database;

  beforeEach(() => {
    // Clean up any existing test database
    if (fs.existsSync(TEST_DB_PATH)) {
      fs.unlinkSync(TEST_DB_PATH);
    }

    // Create fresh database with schema
    db = new Database(TEST_DB_PATH);

    // Create statements table
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

      CREATE INDEX idx_statements_iss ON statements(iss);
      CREATE INDEX idx_statements_sub ON statements(sub);
      CREATE INDEX idx_statements_cty ON statements(cty);
      CREATE INDEX idx_statements_typ ON statements(typ);
      CREATE INDEX idx_statements_registered_at ON statements(registered_at);
      CREATE INDEX idx_statements_hash ON statements(statement_hash);
    `);
  });

  test("insertStatement adds statement with all fields", () => {
    const statement = {
      statement_hash: "abc123",
      iss: "https://example.com/issuer",
      sub: "urn:example:artifact:1",
      cty: "application/vnd.apache.parquet",
      typ: "dataset",
      payload_hash_alg: -16, // SHA-256
      payload_hash: "def456",
      preimage_content_type: "application/vnd.apache.parquet",
      payload_location: "https://example.com/data.parquet",
      tree_size_at_registration: 1,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 0,
    };

    const entryId = insertStatement(db, statement);

    expect(entryId).toBe(1);

    // Verify insertion
    const row = db.query("SELECT * FROM statements WHERE entry_id = ?").get(entryId);
    expect(row).toBeTruthy();
    expect(row.statement_hash).toBe(statement.statement_hash);
    expect(row.iss).toBe(statement.iss);
    expect(row.sub).toBe(statement.sub);
    expect(row.cty).toBe(statement.cty);
    expect(row.typ).toBe(statement.typ);
  });

  test("insertStatement with minimal fields", () => {
    const statement = {
      statement_hash: "minimal123",
      iss: "https://example.com/issuer",
      sub: null,
      cty: null,
      typ: null,
      payload_hash_alg: -16,
      payload_hash: "hash789",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 1,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 0,
    };

    const entryId = insertStatement(db, statement);

    expect(entryId).toBeGreaterThan(0);

    const row = db.query("SELECT * FROM statements WHERE entry_id = ?").get(entryId);
    expect(row.iss).toBe(statement.iss);
    expect(row.sub).toBe(null);
    expect(row.cty).toBe(null);
    expect(row.typ).toBe(null);
  });

  test("findStatementsByIssuer returns matching statements", () => {
    // Insert test data
    insertStatement(db, {
      statement_hash: "hash1",
      iss: "https://issuer1.com",
      sub: null,
      cty: null,
      typ: null,
      payload_hash_alg: -16,
      payload_hash: "ph1",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 1,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 0,
    });

    insertStatement(db, {
      statement_hash: "hash2",
      iss: "https://issuer2.com",
      sub: null,
      cty: null,
      typ: null,
      payload_hash_alg: -16,
      payload_hash: "ph2",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 2,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 1,
    });

    insertStatement(db, {
      statement_hash: "hash3",
      iss: "https://issuer1.com",
      sub: null,
      cty: null,
      typ: null,
      payload_hash_alg: -16,
      payload_hash: "ph3",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 3,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 2,
    });

    const results = findStatementsByIssuer(db, "https://issuer1.com");

    expect(results).toHaveLength(2);
    expect(results[0].statement_hash).toBe("hash1");
    expect(results[1].statement_hash).toBe("hash3");
  });

  test("findStatementsBySubject returns matching statements", () => {
    insertStatement(db, {
      statement_hash: "hash1",
      iss: "https://issuer.com",
      sub: "urn:subject:1",
      cty: null,
      typ: null,
      payload_hash_alg: -16,
      payload_hash: "ph1",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 1,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 0,
    });

    insertStatement(db, {
      statement_hash: "hash2",
      iss: "https://issuer.com",
      sub: "urn:subject:2",
      cty: null,
      typ: null,
      payload_hash_alg: -16,
      payload_hash: "ph2",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 2,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 1,
    });

    const results = findStatementsBySubject(db, "urn:subject:1");

    expect(results).toHaveLength(1);
    expect(results[0].sub).toBe("urn:subject:1");
  });

  test("findStatementsByContentType returns matching statements", () => {
    insertStatement(db, {
      statement_hash: "hash1",
      iss: "https://issuer.com",
      sub: null,
      cty: "application/json",
      typ: null,
      payload_hash_alg: -16,
      payload_hash: "ph1",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 1,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 0,
    });

    insertStatement(db, {
      statement_hash: "hash2",
      iss: "https://issuer.com",
      sub: null,
      cty: "application/vnd.apache.parquet",
      typ: null,
      payload_hash_alg: -16,
      payload_hash: "ph2",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 2,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 1,
    });

    const results = findStatementsByContentType(db, "application/vnd.apache.parquet");

    expect(results).toHaveLength(1);
    expect(results[0].cty).toBe("application/vnd.apache.parquet");
  });

  test("findStatementsByType returns matching statements", () => {
    insertStatement(db, {
      statement_hash: "hash1",
      iss: "https://issuer.com",
      sub: null,
      cty: null,
      typ: "dataset",
      payload_hash_alg: -16,
      payload_hash: "ph1",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 1,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 0,
    });

    insertStatement(db, {
      statement_hash: "hash2",
      iss: "https://issuer.com",
      sub: null,
      cty: null,
      typ: "model",
      payload_hash_alg: -16,
      payload_hash: "ph2",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 2,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 1,
    });

    const results = findStatementsByType(db, "dataset");

    expect(results).toHaveLength(1);
    expect(results[0].typ).toBe("dataset");
  });

  test("findStatementsByDateRange returns statements in range", () => {
    // Insert statements with specific timestamps
    db.exec(`
      INSERT INTO statements (statement_hash, iss, sub, cty, typ, payload_hash_alg, payload_hash, registered_at, tree_size_at_registration, entry_tile_key, entry_tile_offset)
      VALUES
        ('hash1', 'https://issuer.com', NULL, NULL, NULL, -16, 'ph1', '2025-01-15 10:00:00', 1, 'tile/entries/000', 0),
        ('hash2', 'https://issuer.com', NULL, NULL, NULL, -16, 'ph2', '2025-01-20 10:00:00', 2, 'tile/entries/000', 1),
        ('hash3', 'https://issuer.com', NULL, NULL, NULL, -16, 'ph3', '2025-01-25 10:00:00', 3, 'tile/entries/000', 2),
        ('hash4', 'https://issuer.com', NULL, NULL, NULL, -16, 'ph4', '2025-02-05 10:00:00', 4, 'tile/entries/000', 3)
    `);

    const results = findStatementsByDateRange(
      db,
      "2025-01-18 00:00:00",
      "2025-01-31 23:59:59"
    );

    expect(results).toHaveLength(2);
    // Results are ordered DESC (newest first)
    expect(results.map(r => r.statement_hash)).toEqual(["hash3", "hash2"]);
  });

  test("findStatementsBy supports combined filters", () => {
    insertStatement(db, {
      statement_hash: "hash1",
      iss: "https://issuer1.com",
      sub: "urn:artifact:1",
      cty: "application/json",
      typ: "dataset",
      payload_hash_alg: -16,
      payload_hash: "ph1",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 1,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 0,
    });

    insertStatement(db, {
      statement_hash: "hash2",
      iss: "https://issuer1.com",
      sub: "urn:artifact:2",
      cty: "application/json",
      typ: "model",
      payload_hash_alg: -16,
      payload_hash: "ph2",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 2,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 1,
    });

    insertStatement(db, {
      statement_hash: "hash3",
      iss: "https://issuer2.com",
      sub: "urn:artifact:3",
      cty: "application/json",
      typ: "dataset",
      payload_hash_alg: -16,
      payload_hash: "ph3",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 3,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 2,
    });

    // Query: issuer1 + cty=json + typ=dataset
    const results = findStatementsBy(db, {
      iss: "https://issuer1.com",
      cty: "application/json",
      typ: "dataset",
    });

    expect(results).toHaveLength(1);
    expect(results[0].statement_hash).toBe("hash1");
  });

  test("getStatementByEntryId returns correct statement", () => {
    const entryId = insertStatement(db, {
      statement_hash: "hash1",
      iss: "https://issuer.com",
      sub: null,
      cty: null,
      typ: null,
      payload_hash_alg: -16,
      payload_hash: "ph1",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 1,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 0,
    });

    const statement = getStatementByEntryId(db, entryId);

    expect(statement).toBeTruthy();
    expect(statement.entry_id).toBe(entryId);
    expect(statement.statement_hash).toBe("hash1");
  });

  test("getStatementByEntryId returns null for non-existent ID", () => {
    const statement = getStatementByEntryId(db, 999);
    expect(statement).toBe(null);
  });

  test("getStatementByHash returns correct statement", () => {
    insertStatement(db, {
      statement_hash: "unique-hash-123",
      iss: "https://issuer.com",
      sub: null,
      cty: null,
      typ: null,
      payload_hash_alg: -16,
      payload_hash: "ph1",
      preimage_content_type: null,
      payload_location: null,
      tree_size_at_registration: 1,
      entry_tile_key: "tile/entries/000",
      entry_tile_offset: 0,
    });

    const statement = getStatementByHash(db, "unique-hash-123");

    expect(statement).toBeTruthy();
    expect(statement.statement_hash).toBe("unique-hash-123");
  });

  test("getStatementByHash returns null for non-existent hash", () => {
    const statement = getStatementByHash(db, "non-existent-hash");
    expect(statement).toBe(null);
  });

  test("empty result sets return empty arrays", () => {
    expect(findStatementsByIssuer(db, "https://no-such-issuer.com")).toEqual([]);
    expect(findStatementsBySubject(db, "urn:no-such:subject")).toEqual([]);
    expect(findStatementsByContentType(db, "application/non-existent")).toEqual([]);
    expect(findStatementsByType(db, "non-existent-type")).toEqual([]);
  });
});
