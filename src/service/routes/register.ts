/**
 * Statement Registration Endpoint
 * Implements POST /entries per SCRAPI specification
 */

import type { ServerContext } from "../server.ts";
import type { RegistrationResponse, ErrorResponse, Receipt } from "../types/scrapi.ts";
import { decodeCoseSign1, verifyCoseSign1 } from "../../lib/cose/sign.ts";
import { addLeaf, getTreeSize, getInclusionProof } from "../../lib/database/merkle.ts";
import { saveStatement } from "../../lib/database/statements.ts";

/**
 * Hash data using SHA-256
 */
async function hashData(data: Uint8Array): Promise<Uint8Array> {
  const hashBuffer = await crypto.subtle.digest("SHA-256", data);
  return new Uint8Array(hashBuffer);
}

/**
 * Handle statement registration
 */
export async function handleRegister(
  ctx: ServerContext,
  req: Request
): Promise<Response> {
  // Validate content type
  const contentType = req.headers.get("content-type");
  if (contentType !== "application/cose") {
    return errorResponse(415, "Unsupported Media Type", "Expected Content-Type: application/cose");
  }

  try {
    // Read COSE Sign1 statement
    const bodyBuffer = await req.arrayBuffer();
    const statementBytes = new Uint8Array(bodyBuffer);

    // Validate COSE Sign1
    let statement;
    try {
      statement = decodeCoseSign1(statementBytes);
    } catch (error) {
      return errorResponse(400, "Invalid COSE", `Failed to decode COSE Sign1: ${error}`);
    }

    // Compute leaf hash (hash of the entire COSE Sign1)
    const leafHash = await hashData(statementBytes);

    // Add to Merkle tree
    const leafIndex = await addLeaf(ctx.db, leafHash);

    // Generate entry ID (base64url encoded leaf hash)
    const entryId = btoa(String.fromCharCode(...leafHash))
      .replace(/\+/g, "-")
      .replace(/\//g, "_")
      .replace(/=/g, "");

    // Save statement to database
    await saveStatement(ctx.db, entryId, statementBytes, leafHash, leafIndex);

    // Get current tree size
    const treeSize = getTreeSize(ctx.db);

    // Generate inclusion proof
    const proof = await getInclusionProof(ctx.db, leafIndex, treeSize);

    // Create receipt
    const receipt: Receipt = {
      tree_size: treeSize,
      leaf_index: leafIndex,
      inclusion_proof: proof.map((hash) =>
        btoa(String.fromCharCode(...hash))
          .replace(/\+/g, "-")
          .replace(/\//g, "_")
          .replace(/=/g, "")
      ),
    };

    // Create response
    const response: RegistrationResponse = {
      entry_id: entryId,
      receipt,
    };

    return new Response(JSON.stringify(response), {
      status: 201,
      headers: {
        "Content-Type": "application/json",
        Location: `${ctx.origin}/entries/${entryId}`,
      },
    });
  } catch (error) {
    console.error("Registration error:", error);
    return errorResponse(500, "Internal Server Error", String(error));
  }
}

/**
 * Handle statement retrieval
 */
export async function handleGetEntry(
  ctx: ServerContext,
  entryId: string
): Promise<Response> {
  try {
    // Retrieve statement from database
    const { getStatement } = await import("../../lib/database/statements.ts");
    const statement = await getStatement(ctx.db, entryId);

    if (!statement) {
      return errorResponse(404, "Not Found", `Entry ${entryId} not found`);
    }

    return new Response(statement, {
      status: 200,
      headers: {
        "Content-Type": "application/cose",
      },
    });
  } catch (error) {
    console.error("Retrieval error:", error);
    return errorResponse(500, "Internal Server Error", String(error));
  }
}

/**
 * Create error response
 */
function errorResponse(
  status: number,
  error: string,
  details?: string
): Response {
  const body: ErrorResponse = {
    error,
    details,
  };

  return new Response(JSON.stringify(body), {
    status,
    headers: {
      "Content-Type": "application/json",
    },
  });
}
