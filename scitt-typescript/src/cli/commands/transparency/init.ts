/**
 * Transparency Init Command
 * Initializes a new transparency service with database, keys, and storage
 */

import { Database } from "bun:sqlite";
import * as fs from "fs";
import * as path from "path";
import { success, error, info, header, keyValue, progress, progressDone } from "../../utils/output.ts";
import { saveConfig, DEFAULT_CONFIG, type ServiceConfig } from "../../utils/config.ts";

/**
 * Init command options
 */
export interface InitOptions {
  database?: string;
  storage?: string;
  port?: number;
  force?: boolean;
}

/**
 * Initialize transparency service
 */
export async function init(options: InitOptions = {}): Promise<void> {
  header("Initializing Transparency Service");

  try {
    // Step 1: Create configuration
    progress("Creating configuration");
    const config = createConfig(options);
    saveConfig(config);
    progressDone();

    // Step 2: Initialize database
    progress("Initializing database");
    await initDatabase(config.database, options.force);
    progressDone();

    // Step 3: Create storage directory
    progress("Creating storage directory");
    await initStorage(config.storage.path!, options.force);
    progressDone();

    // Step 4: Generate service key
    progress("Generating service key");
    await generateServiceKey(config.serviceKeyPath, options.force);
    progressDone();

    // Print summary
    console.log();
    success("Transparency service initialized successfully!");
    console.log();
    info("Configuration:");
    keyValue("Database", config.database);
    keyValue("Storage", config.storage.path || "N/A");
    keyValue("Port", config.port);
    keyValue("Service Key", config.serviceKeyPath);
    console.log();
    info("Next steps:");
    console.log("  1. Start the service: transparency serve");
    console.log("  2. Check health: curl http://localhost:3000/health");
    console.log();
  } catch (err) {
    progressDone(false);
    error(`Initialization failed: ${err}`);
    process.exit(1);
  }
}

/**
 * Create configuration
 */
function createConfig(options: InitOptions): ServiceConfig {
  const config = { ...DEFAULT_CONFIG };

  if (options.database) {
    config.database = options.database;
  }

  if (options.storage) {
    config.storage.path = options.storage;
  }

  if (options.port) {
    config.port = options.port;
  }

  return config;
}

/**
 * Initialize database
 */
async function initDatabase(dbPath: string, force: boolean = false): Promise<void> {
  // Check if database exists
  if (fs.existsSync(dbPath) && !force) {
    throw new Error(`Database already exists: ${dbPath}. Use --force to overwrite.`);
  }

  // Create database directory if needed
  const dbDir = path.dirname(dbPath);
  if (!fs.existsSync(dbDir)) {
    fs.mkdirSync(dbDir, { recursive: true });
  }

  // Create database
  const db = new Database(dbPath);

  // Create schema
  const { initializeSchema } = await import("../../../lib/database/schema.ts");
  await initializeSchema(db);

  db.close();
}

/**
 * Initialize storage
 */
async function initStorage(storagePath: string, force: boolean = false): Promise<void> {
  if (fs.existsSync(storagePath)) {
    if (!force) {
      // Storage directory exists, just verify it's accessible
      return;
    }
  }

  // Create storage directory
  fs.mkdirSync(storagePath, { recursive: true });

  // Create subdirectories
  const subdirs = ["tile", "receipts", "checkpoint"];
  for (const subdir of subdirs) {
    const subdirPath = path.join(storagePath, subdir);
    if (!fs.existsSync(subdirPath)) {
      fs.mkdirSync(subdirPath, { recursive: true });
    }
  }
}

/**
 * Generate service key
 */
async function generateServiceKey(keyPath: string, force: boolean = false): Promise<void> {
  // Check if key exists
  if (fs.existsSync(keyPath) && !force) {
    throw new Error(`Service key already exists: ${keyPath}. Use --force to overwrite.`);
  }

  // Generate ES256 key pair
  const { generateES256KeyPair } = await import("../../../lib/cose/signer.ts");
  const { exportPublicKeyToJWK, exportPrivateKeyToPEM } = await import("../../../lib/cose/key-material.ts");

  const keyPair = await generateES256KeyPair();

  // Export keys
  const publicJWK = await exportPublicKeyToJWK(keyPair.publicKey);
  const privatePEM = await exportPrivateKeyToPEM(keyPair.privateKey);

  // Add key metadata
  publicJWK.kid = "service-key-1";
  publicJWK.use = "sig";
  publicJWK.alg = "ES256";

  // Save keys
  const keyData = {
    publicKey: publicJWK,
    privateKey: privatePEM,
    created: new Date().toISOString(),
  };

  // Create key directory if needed
  const keyDir = path.dirname(keyPath);
  if (!fs.existsSync(keyDir)) {
    fs.mkdirSync(keyDir, { recursive: true });
  }

  fs.writeFileSync(keyPath, JSON.stringify(keyData, null, 2), "utf-8");

  // Set restrictive permissions (Unix-like systems)
  try {
    fs.chmodSync(keyPath, 0o600);
  } catch (err) {
    // Ignore on Windows
  }
}
