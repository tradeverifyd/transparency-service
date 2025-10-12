/**
 * Issuer Resolver Tests
 * Test suite for issuer key discovery (single choke point)
 */

import { describe, test, expect, beforeEach, afterEach } from "bun:test";

// Mock HTTP server for testing
let mockServer: ReturnType<typeof Bun.serve> | null = null;
let mockPort = 0;

describe("Issuer Resolution", () => {
  afterEach(() => {
    if (mockServer) {
      mockServer.stop();
      mockServer = null;
    }
  });

  test("can resolve issuer public key from URL", async () => {
    const { generateES256KeyPair, exportPublicKeyToJWK } = await import("../../../src/lib/cose/key-material.ts");
    const { resolveIssuer } = await import("../../../src/lib/cose/issuer-resolver.ts");

    // Generate test key
    const keyPair = await generateES256KeyPair();
    const jwk = await exportPublicKeyToJWK(keyPair.publicKey);
    jwk.kid = "test-key-1";

    // Start mock server
    mockServer = Bun.serve({
      port: 0,
      fetch(req) {
        const url = new URL(req.url);
        if (url.pathname === "/.well-known/jwks.json") {
          return new Response(JSON.stringify({ keys: [jwk] }), {
            headers: { "Content-Type": "application/json" },
          });
        }
        return new Response("Not Found", { status: 404 });
      },
    });
    mockPort = mockServer.port;

    const issuerUrl = `http://localhost:${mockPort}`;
    const publicKey = await resolveIssuer(issuerUrl);

    expect(publicKey).toBeDefined();
    expect(publicKey.type).toBe("public");
  });

  test("can resolve specific key by kid", async () => {
    const { generateES256KeyPair, exportPublicKeyToJWK } = await import("../../../src/lib/cose/key-material.ts");
    const { resolveKeyByKid } = await import("../../../src/lib/cose/issuer-resolver.ts");

    // Generate test keys
    const keyPair1 = await generateES256KeyPair();
    const jwk1 = await exportPublicKeyToJWK(keyPair1.publicKey);
    jwk1.kid = "key-1";

    const keyPair2 = await generateES256KeyPair();
    const jwk2 = await exportPublicKeyToJWK(keyPair2.publicKey);
    jwk2.kid = "key-2";

    // Start mock server
    mockServer = Bun.serve({
      port: 0,
      fetch(req) {
        const url = new URL(req.url);
        if (url.pathname === "/.well-known/jwks.json") {
          return new Response(JSON.stringify({ keys: [jwk1, jwk2] }), {
            headers: { "Content-Type": "application/json" },
          });
        }
        return new Response("Not Found", { status: 404 });
      },
    });
    mockPort = mockServer.port;

    const issuerUrl = `http://localhost:${mockPort}`;
    const publicKey = await resolveKeyByKid(issuerUrl, "key-2");

    expect(publicKey).toBeDefined();
    expect(publicKey.type).toBe("public");
  });

  test("throws error when issuer URL is unreachable", async () => {
    const { resolveIssuer } = await import("../../../src/lib/cose/issuer-resolver.ts");

    const unreachableUrl = "http://localhost:9999";

    await expect(resolveIssuer(unreachableUrl)).rejects.toThrow();
  });

  test("throws error when kid is not found", async () => {
    const { generateES256KeyPair, exportPublicKeyToJWK } = await import("../../../src/lib/cose/key-material.ts");
    const { resolveKeyByKid } = await import("../../../src/lib/cose/issuer-resolver.ts");

    // Generate test key
    const keyPair = await generateES256KeyPair();
    const jwk = await exportPublicKeyToJWK(keyPair.publicKey);
    jwk.kid = "existing-key";

    // Start mock server
    mockServer = Bun.serve({
      port: 0,
      fetch(req) {
        const url = new URL(req.url);
        if (url.pathname === "/.well-known/jwks.json") {
          return new Response(JSON.stringify({ keys: [jwk] }), {
            headers: { "Content-Type": "application/json" },
          });
        }
        return new Response("Not Found", { status: 404 });
      },
    });
    mockPort = mockServer.port;

    const issuerUrl = `http://localhost:${mockPort}`;

    await expect(resolveKeyByKid(issuerUrl, "non-existent-key")).rejects.toThrow();
  });

  test("caches issuer keys to avoid repeated fetches", async () => {
    const { generateES256KeyPair, exportPublicKeyToJWK } = await import("../../../src/lib/cose/key-material.ts");
    const { resolveIssuer, clearCache } = await import("../../../src/lib/cose/issuer-resolver.ts");

    // Clear cache before test
    clearCache();

    // Generate test key
    const keyPair = await generateES256KeyPair();
    const jwk = await exportPublicKeyToJWK(keyPair.publicKey);
    jwk.kid = "test-key";

    let fetchCount = 0;

    // Start mock server
    mockServer = Bun.serve({
      port: 0,
      fetch(req) {
        const url = new URL(req.url);
        if (url.pathname === "/.well-known/jwks.json") {
          fetchCount++;
          return new Response(JSON.stringify({ keys: [jwk] }), {
            headers: { "Content-Type": "application/json" },
          });
        }
        return new Response("Not Found", { status: 404 });
      },
    });
    mockPort = mockServer.port;

    const issuerUrl = `http://localhost:${mockPort}`;

    // First resolution
    await resolveIssuer(issuerUrl);
    expect(fetchCount).toBe(1);

    // Second resolution (should use cache)
    await resolveIssuer(issuerUrl);
    expect(fetchCount).toBe(1); // Should still be 1

    // Third resolution (should use cache)
    await resolveIssuer(issuerUrl);
    expect(fetchCount).toBe(1); // Should still be 1
  });

  test("supports custom .well-known paths", async () => {
    const { generateES256KeyPair, exportPublicKeyToJWK } = await import("../../../src/lib/cose/key-material.ts");
    const { resolveIssuer } = await import("../../../src/lib/cose/issuer-resolver.ts");

    // Generate test key
    const keyPair = await generateES256KeyPair();
    const jwk = await exportPublicKeyToJWK(keyPair.publicKey);
    jwk.kid = "test-key";

    // Start mock server with custom path
    mockServer = Bun.serve({
      port: 0,
      fetch(req) {
        const url = new URL(req.url);
        if (url.pathname === "/.well-known/custom-keys") {
          return new Response(JSON.stringify({ keys: [jwk] }), {
            headers: { "Content-Type": "application/json" },
          });
        }
        return new Response("Not Found", { status: 404 });
      },
    });
    mockPort = mockServer.port;

    const issuerUrl = `http://localhost:${mockPort}`;
    const publicKey = await resolveIssuer(issuerUrl, "/.well-known/custom-keys");

    expect(publicKey).toBeDefined();
  });

  test("validates JWK structure", async () => {
    const { resolveIssuer } = await import("../../../src/lib/cose/issuer-resolver.ts");

    // Start mock server with invalid JWK
    mockServer = Bun.serve({
      port: 0,
      fetch(req) {
        const url = new URL(req.url);
        if (url.pathname === "/.well-known/jwks.json") {
          return new Response(JSON.stringify({ keys: [{ invalid: "jwk" }] }), {
            headers: { "Content-Type": "application/json" },
          });
        }
        return new Response("Not Found", { status: 404 });
      },
    });
    mockPort = mockServer.port;

    const issuerUrl = `http://localhost:${mockPort}`;

    await expect(resolveIssuer(issuerUrl)).rejects.toThrow();
  });
});

describe("Key Cache Management", () => {
  test("can clear cache", async () => {
    const { clearCache, getCacheStats } = await import("../../../src/lib/cose/issuer-resolver.ts");

    clearCache();
    const stats = getCacheStats();

    expect(stats.size).toBe(0);
  });

  test("cache respects TTL", async () => {
    const { generateES256KeyPair, exportPublicKeyToJWK } = await import("../../../src/lib/cose/key-material.ts");
    const { resolveIssuer, clearCache } = await import("../../../src/lib/cose/issuer-resolver.ts");

    clearCache();

    // Generate test key
    const keyPair = await generateES256KeyPair();
    const jwk = await exportPublicKeyToJWK(keyPair.publicKey);
    jwk.kid = "test-key";

    let fetchCount = 0;

    // Start mock server
    mockServer = Bun.serve({
      port: 0,
      fetch(req) {
        const url = new URL(req.url);
        if (url.pathname === "/.well-known/jwks.json") {
          fetchCount++;
          return new Response(JSON.stringify({ keys: [jwk] }), {
            headers: { "Content-Type": "application/json" },
          });
        }
        return new Response("Not Found", { status: 404 });
      },
    });
    mockPort = mockServer.port;

    const issuerUrl = `http://localhost:${mockPort}`;

    // First resolution
    await resolveIssuer(issuerUrl, undefined, { ttl: 100 }); // 100ms TTL
    expect(fetchCount).toBe(1);

    // Wait for TTL to expire
    await new Promise((resolve) => setTimeout(resolve, 150));

    // Second resolution (should fetch again)
    await resolveIssuer(issuerUrl, undefined, { ttl: 100 });
    expect(fetchCount).toBe(2);
  });
});
