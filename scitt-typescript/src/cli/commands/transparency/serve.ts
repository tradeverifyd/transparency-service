/**
 * Transparency Serve Command
 * Starts the transparency service HTTP server
 */

import * as fs from "fs";
import { success, error, info, header, keyValue } from "../../utils/output.ts";
import { loadConfig, type ServiceConfig } from "../../utils/config.ts";
import type { Server } from "bun";

/**
 * Serve command options
 */
export interface ServeOptions {
  port?: number;
  hostname?: string;
  database?: string;
  config?: string;
}

/**
 * Start transparency service
 */
export async function serve(options: ServeOptions = {}): Promise<void> {
  header("Starting Transparency Service");

  try {
    // Load configuration
    const config = loadConfig();

    // Override with CLI options
    if (options.port) config.port = options.port;
    if (options.hostname) config.hostname = options.hostname;
    if (options.database) config.database = options.database;

    // Validate configuration
    validateServeConfig(config);

    // Load service key
    const serviceKey = loadServiceKey(config.serviceKeyPath);

    // Print configuration
    info("Configuration:");
    keyValue("Port", config.port);
    keyValue("Hostname", config.hostname);
    keyValue("Database", config.database);
    keyValue("Storage", config.storage.path || "N/A");
    console.log();

    // Start server
    const { startServer } = await import("../../../service/server.ts");

    const server = await startServer({
      port: config.port,
      hostname: config.hostname,
      database: config.database,
      storage: config.storage,
    });

    const origin = `http://${config.hostname}:${server.port}`;

    success(`Transparency service running on ${origin}`);
    console.log();
    info("Endpoints:");
    keyValue("Health", `${origin}/health`);
    keyValue("Configuration", `${origin}/.well-known/scitt-configuration`);
    keyValue("Service Keys", `${origin}/.well-known/jwks.json`);
    console.log();
    info("Press Ctrl+C to stop");

    // Keep process alive
    await new Promise(() => {});
  } catch (err) {
    error(`Failed to start service: ${err}`);
    process.exit(1);
  }
}

/**
 * Validate serve configuration
 */
function validateServeConfig(config: ServiceConfig): void {
  // Check database exists
  if (!fs.existsSync(config.database)) {
    throw new Error(
      `Database not found: ${config.database}\nRun 'transparency init' first.`
    );
  }

  // Check storage exists
  if (config.storage.type === "local") {
    if (!config.storage.path) {
      throw new Error("Storage path is required for local storage");
    }
    if (!fs.existsSync(config.storage.path)) {
      throw new Error(
        `Storage directory not found: ${config.storage.path}\nRun 'transparency init' first.`
      );
    }
  }

  // Check service key exists
  if (!fs.existsSync(config.serviceKeyPath)) {
    throw new Error(
      `Service key not found: ${config.serviceKeyPath}\nRun 'transparency init' first.`
    );
  }
}

/**
 * Load service key from file
 */
function loadServiceKey(keyPath: string): any {
  try {
    const content = fs.readFileSync(keyPath, "utf-8");
    return JSON.parse(content);
  } catch (err) {
    throw new Error(`Failed to load service key from ${keyPath}: ${err}`);
  }
}
