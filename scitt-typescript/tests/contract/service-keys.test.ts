/**
 * Contract test for service keys endpoint
 * Validates /.well-known/jwks.json endpoint
 */

import { describe, test, expect, beforeAll, afterAll } from "bun:test";

describe("Service Keys Endpoint", () => {
  let server: any;
  let baseUrl: string;

  beforeAll(async () => {
    // Start test server
    const { startServer } = await import("../../src/service/server.ts");
    server = await startServer({
      port: 0,
      database: ":memory:",
      storage: { type: "local", path: "./.test-storage" },
    });
    baseUrl = `http://localhost:${server.port}`;
  });

  afterAll(() => {
    if (server) {
      server.stop();
    }
  });

  test("GET /.well-known/jwks.json returns valid JWKS", async () => {
    const response = await fetch(`${baseUrl}/.well-known/jwks.json`);

    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toContain("application/json");

    const jwks = await response.json();

    expect(jwks.keys).toBeDefined();
    expect(Array.isArray(jwks.keys)).toBe(true);
    expect(jwks.keys.length).toBeGreaterThan(0);
  });

  test("service keys include required JWK fields", async () => {
    const response = await fetch(`${baseUrl}/.well-known/jwks.json`);
    const jwks = await response.json();

    const key = jwks.keys[0];

    // Required JWK fields
    expect(key.kty).toBe("EC");
    expect(key.crv).toBe("P-256");
    expect(key.x).toBeDefined();
    expect(key.y).toBeDefined();

    // Key ID
    expect(key.kid).toBeDefined();
    expect(typeof key.kid).toBe("string");

    // Use (signature)
    expect(key.use).toBe("sig");

    // Algorithm
    expect(key.alg).toBe("ES256");
  });

  test("service keys are for verification only (no private key)", async () => {
    const response = await fetch(`${baseUrl}/.well-known/jwks.json`);
    const jwks = await response.json();

    for (const key of jwks.keys) {
      expect(key.d).toBeUndefined(); // No private key component
    }
  });

  test("service keys have consistent kid", async () => {
    const response = await fetch(`${baseUrl}/.well-known/jwks.json`);
    const jwks = await response.json();

    const kids = jwks.keys.map((k: any) => k.kid);
    const uniqueKids = new Set(kids);

    // Each key should have a unique kid
    expect(uniqueKids.size).toBe(kids.length);
  });

  test("handles CORS preflight request", async () => {
    const response = await fetch(`${baseUrl}/.well-known/jwks.json`, {
      method: "OPTIONS",
    });

    expect(response.status).toBe(204);
    expect(response.headers.get("access-control-allow-origin")).toBeDefined();
  });

  test("keys are cacheable", async () => {
    const response = await fetch(`${baseUrl}/.well-known/jwks.json`);

    const cacheControl = response.headers.get("cache-control");
    expect(cacheControl).toBeDefined();
    expect(cacheControl).toContain("max-age");
  });
});
