/**
 * SCITT (Supply Chain Integrity, Transparency and Trust) Type Definitions
 *
 * Based on:
 * - draft-ietf-scitt-architecture: SCITT Architecture
 * - draft-ietf-scitt-scrapi: SCITT SCRAPI (Supply Chain Repository API)
 */

import type { COSESign1, COSESign1WithReceipt } from "./cose.ts";

/**
 * Statement
 * A signed assertion about an artifact or event
 * Encoded as COSE Sign1 with hash envelope
 */
export type Statement = COSESign1;

/**
 * Transparent Statement
 * Statement with attached receipt proving registration
 */
export type TransparentStatement = COSESign1WithReceipt;

/**
 * Statement Metadata
 * Extracted metadata from statement protected header and payload
 */
export interface StatementMetadata {
  iss: string;                    // Issuer URL
  sub?: string;                   // Subject identifier
  cty?: string;                   // Content type
  typ?: string;                   // Type
  payload_hash_alg: number;       // Hash algorithm (-16 for SHA-256)
  payload_hash: string;           // Hex-encoded hash of artifact
  preimage_content_type?: string; // Content type of original artifact
  payload_location?: string;      // Optional location URL
}

/**
 * Receipt
 * Proof of statement registration in transparency log
 * Contains Merkle inclusion proof and service signature
 */
export interface Receipt {
  entry_id: number;               // Entry ID in log (0-indexed)
  tree_size: number;              // Tree size at time of issuance
  leaf_index: number;             // Position in log
  inclusion_path: Uint8Array[];   // Merkle path hashes
  signature: Uint8Array;          // Service signature over proof
}

/**
 * Registration Policy
 * Determines who can register statements
 */
export type RegistrationPolicy = "open" | "restricted" | "closed";

/**
 * Service Configuration
 * Transparency service capabilities and policies
 * Returned from /.well-known/scitt-configuration
 */
export interface ServiceConfiguration {
  service_url: string;                    // Base URL of service
  supported_algorithms: number[];         // COSE algorithm identifiers
  registration_policy: RegistrationPolicy; // Who can register
  tile_height?: number;                   // Tile height (for compatibility)
  checkpoint_frequency?: number;          // Checkpoint every N entries
}

/**
 * Service Public Keys
 * Public keys for verifying receipts and checkpoints
 * Returned from /.well-known/scitt-keys
 */
export interface ServiceKeys {
  keys: Array<{
    kty: string;        // Key type ("EC", "OKP")
    crv: string;        // Curve ("P-256", "Ed25519")
    x: string;          // x-coordinate (base64url)
    y?: string;         // y-coordinate for EC keys (base64url)
    kid: string;        // Key ID
    alg: string;        // Algorithm ("ES256", "EdDSA")
  }>;
}

/**
 * Registration Request
 * POST /entries request body
 * Content-Type: application/cose
 */
export type RegistrationRequest = Uint8Array; // CBOR-encoded COSE Sign1

/**
 * Registration Response
 * Synchronous registration (201)
 */
export interface RegistrationResponseSync {
  status: 201;
  receipt: Uint8Array;  // CBOR-encoded receipt
  location: string;     // URL to retrieve statement
}

/**
 * Registration Response
 * Asynchronous registration (303)
 */
export interface RegistrationResponseAsync {
  status: 303;
  location: string;     // URL to poll for status
}

/**
 * Registration Status Response
 * GET /entries/{entry-id} response
 */
export interface RegistrationStatusResponse {
  status: 200 | 302;
  receipt?: Uint8Array;  // Receipt if status is 200
  location?: string;     // Polling URL if status is 302
}

/**
 * Entry Status
 * Status of a registered statement
 */
export type EntryStatus = "pending" | "registered" | "failed";

/**
 * Log Entry
 * Internal representation of registered statement
 */
export interface LogEntry {
  entry_id: number;               // Entry ID (0-indexed)
  statement_hash: string;         // SHA-256 of statement
  metadata: StatementMetadata;    // Extracted metadata
  registered_at: Date;            // Registration timestamp
  tree_size_at_registration: number; // Tree size when registered
  entry_tile_key: string;         // Object storage key for entry tile
  entry_tile_offset: number;      // Position within tile (0-255)
}

/**
 * Checkpoint (Signed Tree Head)
 * Commitment to current state of log
 * Format: C2SP signed note
 */
export interface Checkpoint {
  origin: string;          // Log origin (e.g., "transparency.example.com/log")
  tree_size: number;       // Number of entries in log
  root_hash: string;       // Base64-encoded Merkle root hash
  timestamp?: number;      // Optional UNIX timestamp
  signature: string;       // Base64-encoded signature
  signature_name: string;  // Name in signature line (e.g., "transparency.example.com")
}

/**
 * Tile
 * Merkle tree tile in C2SP format
 */
export interface Tile {
  level: number;           // Tile level (0 = leaves)
  index: number;           // Tile index at this level
  is_partial: boolean;     // True if partial tile (< 256 hashes)
  width?: number;          // Width if partial (1-255)
  data: Uint8Array;        // Tile content (256 * 32 bytes or less)
  storage_key: string;     // Object storage key
}

/**
 * Tile Path
 * Parsed C2SP tile path
 */
export interface TilePath {
  level: number;           // Tile level
  index: number;           // Tile index
  is_partial: boolean;     // True if path ends with .p/{W}
  width?: number;          // Width if partial
  path_string: string;     // Full path (e.g., "tile/0/x001/x234/067.p/128")
}

/**
 * Query Parameters
 * For searching log entries
 */
export interface QueryParameters {
  iss?: string;                    // Filter by issuer
  sub?: string;                    // Filter by subject
  cty?: string;                    // Filter by content type
  typ?: string;                    // Filter by type
  registered_after?: string;       // ISO 8601 date
  registered_before?: string;      // ISO 8601 date
  limit?: number;                  // Max results (default: 100)
  offset?: number;                 // Pagination offset
}

/**
 * Query Result
 * Result of log query
 */
export interface QueryResult {
  entries: LogEntry[];     // Matching entries
  total: number;           // Total matches
  limit: number;           // Applied limit
  offset: number;          // Applied offset
}

/**
 * Health Status
 * Service health check response
 */
export interface HealthStatus {
  status: "healthy" | "degraded" | "unhealthy";
  components: {
    database: ComponentHealth;
    object_storage: ComponentHealth;
    service: ComponentHealth;
  };
}

/**
 * Component Health
 * Health status of individual component
 */
export interface ComponentHealth {
  status: "up" | "down";
  message?: string;
}

/**
 * Problem Details
 * RFC 7807 Problem Details for HTTP APIs
 * Encoded as CBOR for SCITT
 */
export interface ProblemDetails {
  type: string;            // URI reference identifying problem type
  title: string;           // Short human-readable summary
  status: number;          // HTTP status code
  detail?: string;         // Human-readable explanation
  instance?: string;       // URI reference for this occurrence
}
