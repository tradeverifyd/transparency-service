/**
 * Transparency Service HTTP Server
 * Built on Bun's native HTTP server
 */

import { Database } from "bun:sqlite";
import type { Server } from "bun";

/**
 * Server configuration
 */
export interface ServerConfig {
  port: number;
  database: string;
  storage: StorageConfig;
  hostname?: string;
}

/**
 * Storage configuration
 */
export interface StorageConfig {
  type: "local" | "s3" | "azure" | "minio";
  path?: string;
  bucket?: string;
  endpoint?: string;
}

/**
 * Server context (passed to routes)
 */
export interface ServerContext {
  db: Database;
  storage: StorageConfig;
  origin: string;
  serviceKey?: CryptoKey;
  serviceKeyJWK?: any;
}

/**
 * Start the transparency service HTTP server
 */
export async function startServer(config: ServerConfig): Promise<Server> {
  // Initialize database
  const db = new Database(config.database);

  // Initialize storage (placeholder)
  const storage = config.storage;

  // Generate or load service key
  const { generateES256KeyPair } = await import("../lib/cose/signer.ts");
  const { exportPublicKeyToJWK } = await import("../lib/cose/key-material.ts");
  const keyPair = await generateES256KeyPair();
  const jwk = await exportPublicKeyToJWK(keyPair.publicKey);
  jwk.kid = "service-key-1";
  jwk.use = "sig";
  jwk.alg = "ES256";

  // Import route handlers
  const { handleConfig } = await import("./routes/config.ts");
  const { handleHealth } = await import("./routes/health.ts");
  const { handleRegister, handleGetEntry } = await import("./routes/register.ts");
  const { handleGetReceipt } = await import("./routes/receipts.ts");
  const { handleGetTile } = await import("./routes/tiles.ts");
  const { handleGetCheckpoint } = await import("./routes/checkpoint.ts");

  // Create server context (will be updated with actual port after server starts)
  let serverContext: ServerContext;

  // Start Bun HTTP server
  const hostname = config.hostname || "localhost";
  const server = Bun.serve({
    port: config.port,
    hostname,
    async fetch(req) {
      const url = new URL(req.url);

      // Handle CORS preflight
      if (req.method === "OPTIONS") {
        return new Response(null, {
          status: 204,
          headers: {
            "Access-Control-Allow-Origin": "*",
            "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
            "Access-Control-Allow-Headers": "Content-Type",
          },
        });
      }

      // Add CORS headers helper
      const addCorsHeaders = (response: Response) => {
        const headers = new Headers(response.headers);
        headers.set("Access-Control-Allow-Origin", "*");
        return new Response(response.body, {
          status: response.status,
          statusText: response.statusText,
          headers,
        });
      };

      try {
        // Route handling
        if (url.pathname === "/.well-known/scitt-configuration") {
          return addCorsHeaders(await handleConfig(serverContext, "config"));
        }

        if (url.pathname === "/.well-known/jwks.json") {
          return addCorsHeaders(await handleConfig(serverContext, "keys"));
        }

        if (url.pathname === "/health") {
          return addCorsHeaders(await handleHealth(serverContext));
        }

        // Statement registration
        if (url.pathname === "/entries" && req.method === "POST") {
          return addCorsHeaders(await handleRegister(serverContext, req));
        }

        // Receipt resolution (more specific - must come before statement retrieval)
        const receiptMatch = url.pathname.match(/^\/entries\/([^\/]+)\/receipt$/);
        if (receiptMatch && req.method === "GET") {
          const entryId = receiptMatch[1]!;
          return addCorsHeaders(await handleGetReceipt(serverContext, entryId));
        }

        // Statement retrieval
        const entryMatch = url.pathname.match(/^\/entries\/([^\/]+)$/);
        if (entryMatch && req.method === "GET") {
          const entryId = entryMatch[1]!;
          return addCorsHeaders(await handleGetEntry(serverContext, entryId));
        }

        // Checkpoint
        if (url.pathname === "/checkpoint" && req.method === "GET") {
          return addCorsHeaders(await handleGetCheckpoint(serverContext));
        }

        // Tile retrieval - full tile
        const tileMatch = url.pathname.match(/^\/tile\/(\d+)\/(\d+)$/);
        if (tileMatch && req.method === "GET") {
          const level = parseInt(tileMatch[1]!, 10);
          const index = parseInt(tileMatch[2]!, 10);
          return addCorsHeaders(await handleGetTile(serverContext, level, index));
        }

        // Tile retrieval - partial tile
        const partialTileMatch = url.pathname.match(/^\/tile\/(\d+)\/(\d+)\.p\/(\d+)$/);
        if (partialTileMatch && req.method === "GET") {
          const level = parseInt(partialTileMatch[1]!, 10);
          const index = parseInt(partialTileMatch[2]!, 10);
          const width = parseInt(partialTileMatch[3]!, 10);
          return addCorsHeaders(await handleGetTile(serverContext, level, index, { width }));
        }

        // 404 Not Found
        return new Response("Not Found", { status: 404 });
      } catch (error) {
        console.error("Server error:", error);
        return new Response("Internal Server Error", { status: 500 });
      }
    },
  });

  // Create server context with actual port
  const origin = `http://${hostname}:${server.port}`;

  serverContext = {
    db,
    storage,
    origin,
    serviceKey: keyPair.privateKey,
    serviceKeyJWK: jwk,
  };

  console.log(`Transparency service started on ${origin}`);

  return server;
}

