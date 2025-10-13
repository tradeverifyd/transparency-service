/**
 * SCRAPI (Supply Chain REST API) Types
 * Type definitions for transparency service API
 */

/**
 * Registration response
 */
export interface RegistrationResponse {
  entry_id: number;
  receipt: Receipt;
}

/**
 * Receipt structure
 */
export interface Receipt {
  tree_size: number;
  leaf_index: number;
  inclusion_proof: string[];
}

/**
 * Error response
 */
export interface ErrorResponse {
  error: string;
  details?: string;
}

/**
 * Registration status
 */
export type RegistrationStatus = "pending" | "accepted" | "registered" | "rejected";

/**
 * Registration status response
 */
export interface RegistrationStatusResponse {
  entry_id: number;
  status: RegistrationStatus;
  receipt?: Receipt;
  error?: string;
}
