/**
 * Checkpoint (Signed Tree Head) Management
 *
 * A checkpoint is a signed commitment to the current state of the Merkle tree.
 * It contains:
 * - Tree size (number of entries)
 * - Root hash (Merkle tree root)
 * - Timestamp
 * - Origin (transparency service URL)
 * - Signature (signed by transparency service)
 *
 * Format follows the "signed note" format used by transparency logs.
 */

import { ES256Signer, ES256Verifier } from "../cose/signer.ts";

/**
 * Checkpoint structure
 */
export interface Checkpoint {
  treeSize: number;
  rootHash: Uint8Array;
  timestamp: number;
  origin: string;
  signature: Uint8Array;
}

/**
 * Create a checkpoint (signed tree head)
 *
 * @param treeSize - Number of entries in the tree
 * @param rootHash - Root hash of the Merkle tree (32 bytes)
 * @param privateKey - Private key for signing
 * @param origin - Transparency service origin URL
 * @returns Checkpoint with signature
 */
export async function createCheckpoint(
  treeSize: number,
  rootHash: Uint8Array,
  privateKey: CryptoKey,
  origin: string
): Promise<Checkpoint> {
  // Validate inputs
  if (rootHash.length !== 32) {
    throw new Error(`Root hash must be 32 bytes (SHA-256), got ${rootHash.length}`);
  }

  // Validate origin URL
  try {
    new URL(origin);
  } catch (error) {
    throw new Error(`Invalid origin URL: ${origin}`);
  }

  const timestamp = Date.now();

  // Create the data to be signed
  const dataToSign = encodeCheckpointData(treeSize, rootHash, timestamp, origin);

  // Sign with ES256
  const signer = new ES256Signer(privateKey);
  const signature = await signer.sign(dataToSign);

  return {
    treeSize,
    rootHash,
    timestamp,
    origin,
    signature,
  };
}

/**
 * Verify a checkpoint signature
 *
 * @param checkpoint - Checkpoint to verify
 * @param publicKey - Public key for verification
 * @returns True if signature is valid
 */
export async function verifyCheckpoint(
  checkpoint: Checkpoint,
  publicKey: CryptoKey
): Promise<boolean> {
  // Reconstruct the signed data
  const dataToSign = encodeCheckpointData(
    checkpoint.treeSize,
    checkpoint.rootHash,
    checkpoint.timestamp,
    checkpoint.origin
  );

  // Verify signature
  const verifier = new ES256Verifier(publicKey);
  return await verifier.verify(dataToSign, checkpoint.signature);
}

/**
 * Encode checkpoint to signed note format (text)
 *
 * Format (per RFC 6962 - root hash must be hex-encoded):
 * ```
 * <origin>
 * <tree-size>
 * <root-hash-hex>
 * <timestamp>
 *
 * — <origin> <signature-base64>
 * ```
 *
 * @param checkpoint - Checkpoint to encode
 * @returns Signed note string
 */
export function encodeCheckpoint(checkpoint: Checkpoint): string {
  // RFC 6962 requires hex encoding for merkle tree hashes
  const rootHashHex = Array.from(checkpoint.rootHash)
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
  const signatureBase64 = btoa(String.fromCharCode(...checkpoint.signature));

  const lines = [
    checkpoint.origin,
    checkpoint.treeSize.toString(),
    rootHashHex,
    checkpoint.timestamp.toString(),
    "",
    `— ${checkpoint.origin} ${signatureBase64}`,
  ];

  return lines.join("\n");
}

/**
 * Decode checkpoint from signed note format
 *
 * @param encoded - Signed note string
 * @returns Checkpoint
 */
export function decodeCheckpoint(encoded: string): Checkpoint {
  const lines = encoded.trim().split("\n");

  if (lines.length < 6) {
    throw new Error("Invalid checkpoint format: too few lines");
  }

  const origin = lines[0]!;
  const treeSize = parseInt(lines[1]!, 10);
  const rootHashStr = lines[2]!;
  const timestamp = parseInt(lines[3]!, 10);

  // Parse signature line: "— <origin> <signature>"
  const signatureLine = lines[5]!;
  const signatureMatch = signatureLine.match(/^— .+ (.+)$/);
  if (!signatureMatch) {
    throw new Error("Invalid checkpoint format: signature line malformed");
  }
  const signatureBase64 = signatureMatch[1]!;

  // Decode root hash - try hex first (RFC 6962 compliant), then base64 for backwards compatibility
  let rootHash: Uint8Array;

  // Check if it's hex (only contains 0-9, a-f, A-F)
  if (/^[0-9a-fA-F]+$/.test(rootHashStr) && rootHashStr.length === 64) {
    // Decode hex
    rootHash = new Uint8Array(32);
    for (let i = 0; i < 32; i++) {
      rootHash[i] = parseInt(rootHashStr.substr(i * 2, 2), 16);
    }
  } else {
    // Try base64 for backwards compatibility
    try {
      rootHash = Uint8Array.from(atob(rootHashStr), c => c.charCodeAt(0));
    } catch (error) {
      throw new Error(`Invalid root hash encoding (expected hex or base64): ${error}`);
    }
  }

  const signature = Uint8Array.from(atob(signatureBase64), c => c.charCodeAt(0));

  if (rootHash.length !== 32) {
    throw new Error(`Invalid root hash length: ${rootHash.length}`);
  }

  return {
    treeSize,
    rootHash,
    timestamp,
    origin,
    signature,
  };
}

/**
 * Encode checkpoint data for signing
 *
 * The signed data is a CBOR-encoded array:
 * [tree_size, root_hash, timestamp, origin]
 */
function encodeCheckpointData(
  treeSize: number,
  rootHash: Uint8Array,
  timestamp: number,
  origin: string
): Uint8Array {
  // Simple binary encoding: tree_size (8 bytes) + root_hash (32 bytes) + timestamp (8 bytes) + origin (variable)
  const encoder = new TextEncoder();
  const originBytes = encoder.encode(origin);

  const buffer = new Uint8Array(8 + 32 + 8 + originBytes.length);
  const view = new DataView(buffer.buffer);

  // Write tree size (64-bit big-endian)
  view.setBigUint64(0, BigInt(treeSize), false);

  // Write root hash
  buffer.set(rootHash, 8);

  // Write timestamp (64-bit big-endian)
  view.setBigUint64(40, BigInt(timestamp), false);

  // Write origin
  buffer.set(originBytes, 48);

  return buffer;
}
