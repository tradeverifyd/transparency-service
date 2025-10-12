/**
 * Object Storage Interface
 * Abstract interface for object storage backends (MinIO, S3, Azure, local)
 */

/**
 * Storage Interface
 * All storage backends must implement this interface
 */
export interface Storage {
  /**
   * Store data at the given key
   * @param key - Object key (e.g., "tile/0/000" or "receipts/123")
   * @param data - Binary data to store
   */
  put(key: string, data: Uint8Array): Promise<void>;

  /**
   * Retrieve data at the given key
   * @param key - Object key
   * @returns Data if found, null if not found
   */
  get(key: string): Promise<Uint8Array | null>;

  /**
   * Check if object exists at key
   * @param key - Object key
   * @returns True if exists, false otherwise
   */
  exists(key: string): Promise<boolean>;

  /**
   * List all keys with given prefix
   * @param prefix - Key prefix (e.g., "tile/0/" or "receipts/")
   * @returns Array of matching keys
   */
  list(prefix: string): Promise<string[]>;

  /**
   * Delete object at key
   * @param key - Object key
   */
  delete(key: string): Promise<void>;

  /**
   * Get storage backend type
   */
  readonly type: "local" | "minio" | "s3" | "azure";
}

/**
 * Storage Error
 * Base error for storage operations
 */
export class StorageError extends Error {
  constructor(
    message: string,
    public readonly key?: string,
    public readonly cause?: Error
  ) {
    super(message);
    this.name = "StorageError";
  }
}

/**
 * Storage Not Found Error
 * Thrown when object is not found
 */
export class StorageNotFoundError extends StorageError {
  constructor(key: string, cause?: Error) {
    super(`Object not found: ${key}`, key, cause);
    this.name = "StorageNotFoundError";
  }
}

/**
 * Storage Connection Error
 * Thrown when storage backend is unreachable
 */
export class StorageConnectionError extends StorageError {
  constructor(message: string, cause?: Error) {
    super(message, undefined, cause);
    this.name = "StorageConnectionError";
  }
}
