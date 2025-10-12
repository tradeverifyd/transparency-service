/**
 * CLI Configuration Utilities
 * Handles configuration file management for the transparency service
 */

import * as fs from "fs";
import * as path from "path";

/**
 * Service configuration
 */
export interface ServiceConfig {
  port: number;
  hostname: string;
  database: string;
  storage: StorageConfig;
  serviceKeyPath: string;
}

/**
 * Storage configuration
 */
export interface StorageConfig {
  type: "local" | "s3" | "azure" | "minio";
  path?: string;
  bucket?: string;
  endpoint?: string;
  region?: string;
}

/**
 * Default configuration
 */
export const DEFAULT_CONFIG: ServiceConfig = {
  port: 3000,
  hostname: "localhost",
  database: "./transparency.db",
  storage: {
    type: "local",
    path: "./storage",
  },
  serviceKeyPath: "./service-key.json",
};

/**
 * Get configuration directory
 */
export function getConfigDir(): string {
  const home = process.env.HOME || process.env.USERPROFILE || ".";
  return path.join(home, ".transparency-service");
}

/**
 * Get configuration file path
 */
export function getConfigPath(): string {
  return path.join(getConfigDir(), "config.json");
}

/**
 * Load configuration from file
 * Falls back to default if file doesn't exist
 */
export function loadConfig(): ServiceConfig {
  const configPath = getConfigPath();

  if (!fs.existsSync(configPath)) {
    return { ...DEFAULT_CONFIG };
  }

  try {
    const content = fs.readFileSync(configPath, "utf-8");
    const config = JSON.parse(content);
    return { ...DEFAULT_CONFIG, ...config };
  } catch (error) {
    console.error(`Error loading config from ${configPath}:`, error);
    return { ...DEFAULT_CONFIG };
  }
}

/**
 * Save configuration to file
 */
export function saveConfig(config: ServiceConfig): void {
  const configDir = getConfigDir();
  const configPath = getConfigPath();

  // Create config directory if it doesn't exist
  if (!fs.existsSync(configDir)) {
    fs.mkdirSync(configDir, { recursive: true });
  }

  // Write config file
  fs.writeFileSync(configPath, JSON.stringify(config, null, 2), "utf-8");
}

/**
 * Merge configuration with CLI options
 */
export function mergeConfig(
  base: ServiceConfig,
  options: Partial<ServiceConfig>
): ServiceConfig {
  return {
    ...base,
    ...options,
    storage: {
      ...base.storage,
      ...(options.storage || {}),
    },
  };
}

/**
 * Validate configuration
 */
export function validateConfig(config: ServiceConfig): string[] {
  const errors: string[] = [];

  if (config.port < 1 || config.port > 65535) {
    errors.push(`Invalid port: ${config.port} (must be 1-65535)`);
  }

  if (!config.hostname) {
    errors.push("Hostname is required");
  }

  if (!config.database) {
    errors.push("Database path is required");
  }

  if (config.storage.type === "local" && !config.storage.path) {
    errors.push("Storage path is required for local storage");
  }

  return errors;
}
