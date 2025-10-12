/**
 * Database Schema and Migrations
 * SQLite schema for transparency service metadata
 */

import type { Database } from "bun:sqlite";

/**
 * Initialize database schema
 * Creates all tables, indexes, and initial data
 */
export async function initializeSchema(db: Database): Promise<void> {
  // Enable foreign keys
  db.run("PRAGMA foreign_keys = ON");

  // Schema versioning
  db.run(`
    CREATE TABLE IF NOT EXISTS schema_version (
      version TEXT PRIMARY KEY,
      applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )
  `);

  // Check if schema is already initialized
  const currentVersion = db.query("SELECT version FROM schema_version ORDER BY applied_at DESC LIMIT 1").get() as { version: string } | null;

  if (currentVersion?.version === "1.0.0") {
    // Schema already initialized
    return;
  }

  // Statements table: Metadata for registered signed statements
  db.run(`
    CREATE TABLE IF NOT EXISTS statements (
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
    )
  `);

  // Indexes for efficient querying
  db.run("CREATE INDEX IF NOT EXISTS idx_statements_iss ON statements(iss)");
  db.run("CREATE INDEX IF NOT EXISTS idx_statements_sub ON statements(sub)");
  db.run("CREATE INDEX IF NOT EXISTS idx_statements_cty ON statements(cty)");
  db.run("CREATE INDEX IF NOT EXISTS idx_statements_typ ON statements(typ)");
  db.run("CREATE INDEX IF NOT EXISTS idx_statements_registered_at ON statements(registered_at)");
  db.run("CREATE INDEX IF NOT EXISTS idx_statements_hash ON statements(statement_hash)");

  // Receipts table: Pointers to receipt objects in storage
  db.run(`
    CREATE TABLE IF NOT EXISTS receipts (
      entry_id INTEGER PRIMARY KEY,
      receipt_hash TEXT UNIQUE NOT NULL,
      storage_key TEXT UNIQUE NOT NULL,
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

      tree_size INTEGER NOT NULL,
      leaf_index INTEGER NOT NULL,

      FOREIGN KEY (entry_id) REFERENCES statements(entry_id)
    )
  `);

  // Tiles table: Merkle tree tile metadata
  db.run(`
    CREATE TABLE IF NOT EXISTS tiles (
      tile_id INTEGER PRIMARY KEY AUTOINCREMENT,
      level INTEGER NOT NULL,
      tile_index INTEGER NOT NULL,

      storage_key TEXT UNIQUE NOT NULL,

      is_partial BOOLEAN DEFAULT FALSE,
      width INTEGER,

      tile_hash TEXT NOT NULL,

      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

      UNIQUE(level, tile_index)
    )
  `);

  // Index for tile lookups
  db.run("CREATE INDEX IF NOT EXISTS idx_tiles_level_index ON tiles(level, tile_index)");

  // Tree state: Current Merkle tree state
  db.run(`
    CREATE TABLE IF NOT EXISTS tree_state (
      tree_size INTEGER PRIMARY KEY,
      root_hash TEXT NOT NULL,
      checkpoint_storage_key TEXT NOT NULL,
      checkpoint_signed_note TEXT NOT NULL,
      updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )
  `);

  // Current tree size (singleton table)
  db.run(`
    CREATE TABLE IF NOT EXISTS current_tree_size (
      id INTEGER PRIMARY KEY CHECK (id = 1),
      tree_size INTEGER NOT NULL DEFAULT 0,
      last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )
  `);

  // Initialize with tree size 0
  db.run("INSERT OR IGNORE INTO current_tree_size (id, tree_size) VALUES (1, 0)");

  // Service configuration
  db.run(`
    CREATE TABLE IF NOT EXISTS service_config (
      key TEXT PRIMARY KEY,
      value TEXT NOT NULL,
      updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )
  `);

  // Initialize service config
  const configInsert = db.prepare(`
    INSERT OR IGNORE INTO service_config (key, value) VALUES (?, ?)
  `);

  configInsert.run("service_url", "http://localhost:3000");
  configInsert.run("tile_height", "8");
  configInsert.run("checkpoint_frequency", "1000");
  configInsert.run("hash_algorithm", "-16");
  configInsert.run("signature_algorithm", "-7");

  // Service keys: Transparency service signing keys
  db.run(`
    CREATE TABLE IF NOT EXISTS service_keys (
      kid TEXT PRIMARY KEY,
      public_key_jwk TEXT NOT NULL,
      private_key_pem TEXT NOT NULL,
      algorithm TEXT NOT NULL,
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
      active BOOLEAN DEFAULT TRUE
    )
  `);

  // Mark schema as initialized
  db.run("INSERT INTO schema_version (version) VALUES ('1.0.0')");
}

/**
 * Enable Write-Ahead Logging (WAL) mode
 * Improves concurrent read/write performance
 */
export async function enableWAL(db: Database): Promise<void> {
  db.run("PRAGMA journal_mode = WAL");
  db.run("PRAGMA synchronous = NORMAL");
  db.run("PRAGMA busy_timeout = 5000");
}

/**
 * Database connection options
 */
export interface DatabaseOptions {
  path: string;
  enableWAL?: boolean;
  busyTimeout?: number;
}

/**
 * Open database connection with options
 */
export async function openDatabase(options: DatabaseOptions): Promise<Database> {
  const { Database: SQLiteDatabase } = await import("bun:sqlite");

  const db = new SQLiteDatabase(options.path, {
    create: true,
    readwrite: true,
  });

  // Initialize schema
  await initializeSchema(db);

  // Enable WAL mode if requested (default: true)
  if (options.enableWAL !== false) {
    await enableWAL(db);
  }

  // Set busy timeout
  if (options.busyTimeout) {
    db.run(`PRAGMA busy_timeout = ${options.busyTimeout}`);
  }

  return db;
}

/**
 * Close database connection
 */
export function closeDatabase(db: Database): void {
  db.close();
}
