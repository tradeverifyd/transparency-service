/**
 * Configuration Type Definitions
 * Application configuration and environment settings
 */

/**
 * Storage Backend Type
 */
export type StorageBackend = "local" | "minio" | "s3" | "azure";

/**
 * Storage Configuration
 * Configuration for object storage backend
 */
export interface StorageConfig {
  backend: StorageBackend;
  // Local filesystem storage
  local_path?: string;
  // MinIO/S3-compatible storage
  endpoint?: string;
  access_key?: string;
  secret_key?: string;
  bucket?: string;
  region?: string;
  force_path_style?: boolean; // For MinIO compatibility
  // Azure Blob storage
  azure_account?: string;
  azure_key?: string;
  azure_container?: string;
}

/**
 * Database Configuration
 * SQLite database settings
 */
export interface DatabaseConfig {
  path: string;              // Path to SQLite database file
  enable_wal?: boolean;      // Enable Write-Ahead Logging (default: true)
  busy_timeout?: number;     // Busy timeout in ms (default: 5000)
}

/**
 * Service Configuration
 * HTTP service settings
 */
export interface ServiceConfig {
  port: number;              // HTTP port (default: 3000)
  host: string;              // Bind address (default: "0.0.0.0")
  service_url: string;       // Public service URL
  cors_origins?: string[];   // Allowed CORS origins
  request_timeout?: number;  // Request timeout in ms (default: 30000)
}

/**
 * Key Configuration
 * Transparency service signing keys
 */
export interface KeyConfig {
  kid: string;               // Key ID
  algorithm: string;         // Algorithm ("ES256", "EdDSA")
  private_key_path: string;  // Path to private key (PEM format)
  public_key_path?: string;  // Path to public key (JWK format)
}

/**
 * Issuer Configuration
 * Issuer identity and signing key
 */
export interface IssuerConfig {
  iss: string;               // Issuer URL
  kid: string;               // Key ID
  algorithm: string;         // Algorithm ("ES256", "EdDSA")
  private_key_path: string;  // Path to private key (PEM format)
  public_key_path?: string;  // Path to public key (JWK format)
}

/**
 * Transparency Service Configuration
 * Complete configuration for transparency service
 */
export interface TransparencyServiceConfig {
  database: DatabaseConfig;
  storage: StorageConfig;
  service: ServiceConfig;
  keys: KeyConfig[];
  checkpoint_frequency?: number; // Checkpoint every N entries (default: 1000)
  tile_height?: number;          // Tile height parameter (not used in C2SP, for future)
}

/**
 * CLI Configuration
 * User-specific CLI settings
 */
export interface CLIConfig {
  default_service_url?: string;  // Default transparency service URL
  cache_dir?: string;            // Cache directory for keys and receipts
  output_format?: "json" | "text"; // Output format preference
}

/**
 * Environment Variables
 * Mapped environment variables
 */
export interface EnvironmentVariables {
  // Database
  TRANSPARENCY_DATABASE_PATH?: string;
  // Storage
  TRANSPARENCY_STORAGE_BACKEND?: StorageBackend;
  TRANSPARENCY_STORAGE_LOCAL_PATH?: string;
  TRANSPARENCY_STORAGE_ENDPOINT?: string;
  TRANSPARENCY_STORAGE_ACCESS_KEY?: string;
  TRANSPARENCY_STORAGE_SECRET_KEY?: string;
  TRANSPARENCY_STORAGE_BUCKET?: string;
  TRANSPARENCY_STORAGE_REGION?: string;
  // Service
  TRANSPARENCY_SERVICE_PORT?: string;
  TRANSPARENCY_SERVICE_HOST?: string;
  TRANSPARENCY_SERVICE_URL?: string;
  // Keys
  TRANSPARENCY_KEY_PATH?: string;
  TRANSPARENCY_KEY_KID?: string;
  // Logging
  LOG_LEVEL?: "debug" | "info" | "warn" | "error";
  LOG_FORMAT?: "json" | "pretty";
}

/**
 * Validation Result
 * Result of configuration validation
 */
export interface ValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
}

/**
 * Configuration Loader Interface
 * Abstract interface for loading configuration
 */
export interface ConfigLoader {
  /**
   * Load configuration from environment, files, or defaults
   */
  load(): Promise<TransparencyServiceConfig>;

  /**
   * Validate configuration
   */
  validate(config: TransparencyServiceConfig): ValidationResult;

  /**
   * Save configuration to file
   */
  save(config: TransparencyServiceConfig, path: string): Promise<void>;
}
