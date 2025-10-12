/**
 * Configuration and Keys Endpoints
 * /.well-known/scitt-configuration
 * /.well-known/jwks.json
 */

import type { ServerContext } from "../server.ts";

/**
 * Handle configuration endpoints
 */
export async function handleConfig(
  ctx: ServerContext,
  type: "config" | "keys"
): Promise<Response> {
  if (type === "config") {
    return handleScittConfiguration(ctx);
  } else {
    return handleJwks(ctx);
  }
}

/**
 * Handle /.well-known/scitt-configuration
 * Returns service configuration per SCITT specification
 */
async function handleScittConfiguration(ctx: ServerContext): Promise<Response> {
  const config = {
    issuer: ctx.origin,
    registration_endpoint: `${ctx.origin}/entries`,
    jwks_uri: `${ctx.origin}/.well-known/jwks.json`,
    tile_endpoint: `${ctx.origin}/tile`,
    checkpoint_endpoint: `${ctx.origin}/checkpoint`,
    supported_algorithms: ["ES256"],
    service_documentation: "https://github.com/transparency-dev/transparency-service",
  };

  return new Response(JSON.stringify(config, null, 2), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
      "Cache-Control": "public, max-age=3600",
    },
  });
}

/**
 * Handle /.well-known/jwks.json
 * Returns service public keys in JWK Set format
 */
async function handleJwks(ctx: ServerContext): Promise<Response> {
  const jwks = {
    keys: [ctx.serviceKeyJWK],
  };

  return new Response(JSON.stringify(jwks, null, 2), {
    status: 200,
    headers: {
      "Content-Type": "application/json",
      "Cache-Control": "public, max-age=3600",
    },
  });
}
