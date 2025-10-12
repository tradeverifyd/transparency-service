/**
 * Health Check Endpoint
 * GET /health
 */

import type { ServerContext } from "../server.ts";

/**
 * Health check response
 */
interface HealthResponse {
  status: "healthy" | "degraded" | "unhealthy";
  version: string;
  tree_size: number;
  checks: {
    database: HealthCheck;
    storage: HealthCheck;
    merkle_tree: HealthCheck;
  };
}

/**
 * Individual health check
 */
interface HealthCheck {
  status: "healthy" | "degraded" | "unhealthy";
  message?: string;
}

/**
 * Handle /health endpoint
 */
export async function handleHealth(ctx: ServerContext): Promise<Response> {
  const checks = {
    database: await checkDatabase(ctx),
    storage: await checkStorage(ctx),
    merkle_tree: await checkMerkleTree(ctx),
  };

  // Determine overall status
  const statuses = Object.values(checks).map((c) => c.status);
  let overallStatus: "healthy" | "degraded" | "unhealthy" = "healthy";

  if (statuses.includes("unhealthy")) {
    overallStatus = "unhealthy";
  } else if (statuses.includes("degraded")) {
    overallStatus = "degraded";
  }

  // Get tree size
  const treeSize = getTreeSize(ctx);

  const health: HealthResponse = {
    status: overallStatus,
    version: "0.1.0",
    tree_size: treeSize,
    checks,
  };

  const statusCode = overallStatus === "healthy" ? 200 : 503;

  return new Response(JSON.stringify(health, null, 2), {
    status: statusCode,
    headers: {
      "Content-Type": "application/json",
      "Cache-Control": "no-cache",
    },
  });
}

/**
 * Check database health
 */
async function checkDatabase(ctx: ServerContext): Promise<HealthCheck> {
  try {
    // Try a simple query
    const result = ctx.db.query("SELECT 1 as test").get();

    if (result && (result as any).test === 1) {
      return { status: "healthy" };
    } else {
      return {
        status: "unhealthy",
        message: "Database query returned unexpected result",
      };
    }
  } catch (error) {
    return {
      status: "unhealthy",
      message: `Database error: ${error}`,
    };
  }
}

/**
 * Check storage health
 */
async function checkStorage(ctx: ServerContext): Promise<HealthCheck> {
  try {
    // For local storage, just check if path is accessible
    if (ctx.storage.type === "local") {
      // Placeholder - would check if storage path is writable
      return { status: "healthy" };
    }

    // For cloud storage, would check connectivity
    return { status: "healthy" };
  } catch (error) {
    return {
      status: "unhealthy",
      message: `Storage error: ${error}`,
    };
  }
}

/**
 * Check Merkle tree health
 */
async function checkMerkleTree(ctx: ServerContext): Promise<HealthCheck> {
  try {
    // Check if we can query tree state
    const treeSize = getTreeSize(ctx);

    if (treeSize >= 0) {
      return { status: "healthy" };
    } else {
      return {
        status: "unhealthy",
        message: "Invalid tree size",
      };
    }
  } catch (error) {
    return {
      status: "unhealthy",
      message: `Merkle tree error: ${error}`,
    };
  }
}

/**
 * Get current tree size
 */
function getTreeSize(ctx: ServerContext): number {
  try {
    // Try to get tree size from database
    const result = ctx.db
      .query("SELECT COALESCE(MAX(entry_id), 0) as size FROM statements")
      .get();

    return (result as any)?.size || 0;
  } catch (error) {
    // If statements table doesn't exist yet, return 0
    return 0;
  }
}
