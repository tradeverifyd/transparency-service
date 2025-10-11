/**
 * Local Filesystem Storage Implementation
 * Stores objects in local filesystem for development and testing
 */

import * as fs from "fs";
import * as path from "path";
import type { Storage } from "./interface.ts";
import { StorageError, StorageNotFoundError } from "./interface.ts";

/**
 * Local Filesystem Storage
 * Implements Storage interface using local filesystem
 */
export class LocalStorage implements Storage {
  readonly type = "local" as const;

  /**
   * Create local filesystem storage
   * @param basePath - Base directory for storage
   */
  constructor(private readonly basePath: string) {
    // Ensure base path exists
    if (!fs.existsSync(basePath)) {
      fs.mkdirSync(basePath, { recursive: true });
    }
  }

  /**
   * Get full filesystem path for key
   */
  private getPath(key: string): string {
    return path.join(this.basePath, key);
  }

  /**
   * Store data at key
   */
  async put(key: string, data: Uint8Array): Promise<void> {
    try {
      const filePath = this.getPath(key);
      const dir = path.dirname(filePath);

      // Ensure parent directory exists
      if (!fs.existsSync(dir)) {
        fs.mkdirSync(dir, { recursive: true });
      }

      // Write data
      fs.writeFileSync(filePath, data);
    } catch (error) {
      throw new StorageError(
        `Failed to put object at key: ${key}`,
        key,
        error as Error
      );
    }
  }

  /**
   * Retrieve data at key
   */
  async get(key: string): Promise<Uint8Array | null> {
    try {
      const filePath = this.getPath(key);

      if (!fs.existsSync(filePath)) {
        return null;
      }

      const data = fs.readFileSync(filePath);
      return new Uint8Array(data);
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code === "ENOENT") {
        return null;
      }
      throw new StorageError(
        `Failed to get object at key: ${key}`,
        key,
        error as Error
      );
    }
  }

  /**
   * Check if object exists
   */
  async exists(key: string): Promise<boolean> {
    try {
      const filePath = this.getPath(key);
      return fs.existsSync(filePath);
    } catch (error) {
      return false;
    }
  }

  /**
   * List keys with prefix
   */
  async list(prefix: string): Promise<string[]> {
    try {
      const prefixPath = this.getPath(prefix);
      const baseDir = path.dirname(prefixPath);

      if (!fs.existsSync(baseDir)) {
        return [];
      }

      const keys: string[] = [];

      // Recursive walk
      const walk = (dir: string) => {
        const entries = fs.readdirSync(dir, { withFileTypes: true });

        for (const entry of entries) {
          const fullPath = path.join(dir, entry.name);

          if (entry.isDirectory()) {
            walk(fullPath);
          } else if (entry.isFile()) {
            // Convert absolute path to relative key
            const relativePath = path.relative(this.basePath, fullPath);
            // Normalize to use forward slashes (cross-platform)
            const key = relativePath.split(path.sep).join("/");

            // Check if key starts with prefix
            if (key.startsWith(prefix)) {
              keys.push(key);
            }
          }
        }
      };

      // Start walk from base directory
      walk(this.basePath);

      return keys.sort();
    } catch (error) {
      throw new StorageError(
        `Failed to list objects with prefix: ${prefix}`,
        prefix,
        error as Error
      );
    }
  }

  /**
   * Delete object at key
   */
  async delete(key: string): Promise<void> {
    try {
      const filePath = this.getPath(key);

      if (fs.existsSync(filePath)) {
        fs.unlinkSync(filePath);
      }
    } catch (error) {
      throw new StorageError(
        `Failed to delete object at key: ${key}`,
        key,
        error as Error
      );
    }
  }
}
