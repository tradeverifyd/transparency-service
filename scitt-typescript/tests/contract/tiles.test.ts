/**
 * Contract test for tile retrieval endpoints
 * Validates GET /tile/* endpoints per C2SP tile log specification
 */

import { describe, test, expect, beforeAll, afterAll } from "bun:test";

describe("Tile Endpoints", () => {
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

    // Register several test statements to populate tiles
    const { generateES256KeyPair } = await import("../../src/lib/cose/signer.ts");
    const { signHashEnvelope } = await import("../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../src/lib/cose/sign.ts");
    const { encodeCoseSign1 } = await import("../../src/lib/cose/sign.ts");

    const issuerKey = await generateES256KeyPair();

    // Register 5 entries to create tiles
    for (let i = 0; i < 5; i++) {
      const payload = new TextEncoder().encode(`Test artifact ${i}`);
      const cwtClaims = createCWTClaims({
        iss: "https://issuer.example.com",
        sub: `artifact-${i}`,
      });

      const statement = await signHashEnvelope(
        payload,
        { contentType: "text/plain" },
        issuerKey.privateKey,
        cwtClaims
      );

      const statementBytes = encodeCoseSign1(statement);

      await fetch(`${baseUrl}/entries`, {
        method: "POST",
        headers: {
          "Content-Type": "application/cose",
        },
        body: statementBytes,
      });
    }
  });

  afterAll(() => {
    if (server) {
      server.stop();
    }
  });

  test("GET /tile/0/0 returns entry tile for first entries", async () => {
    const response = await fetch(`${baseUrl}/tile/0/0`);

    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toBe("application/octet-stream");

    const data = new Uint8Array(await response.arrayBuffer());
    expect(data.length).toBeGreaterThan(0);

    // Each entry is 32 bytes (SHA-256 hash)
    expect(data.length % 32).toBe(0);
  });

  test("GET /tile/0/999 returns 404 for non-existent tile", async () => {
    const response = await fetch(`${baseUrl}/tile/0/999`);

    expect(response.status).toBe(404);
  });

  test("GET /tile/0/0.p/5 returns partial tile", async () => {
    const response = await fetch(`${baseUrl}/tile/0/0.p/5`);

    // Should either return 200 with partial tile or 404 if not found
    if (response.status === 200) {
      expect(response.headers.get("content-type")).toBe("application/octet-stream");

      const data = new Uint8Array(await response.arrayBuffer());
      expect(data.length).toBeGreaterThan(0);
      expect(data.length).toBeLessThanOrEqual(5 * 32); // Max 5 entries
    } else {
      expect(response.status).toBe(404);
    }
  });

  test("handles CORS for tile endpoints", async () => {
    const response = await fetch(`${baseUrl}/tile/0/0`, {
      method: "OPTIONS",
    });

    expect(response.status).toBe(204);
    expect(response.headers.get("access-control-allow-origin")).toBeDefined();
    expect(response.headers.get("access-control-allow-methods")).toContain("GET");
  });

  test("GET /tile/1/0 returns hash tile at level 1", async () => {
    const response = await fetch(`${baseUrl}/tile/1/0`);

    // Hash tiles may not exist yet if tree is small
    if (response.status === 200) {
      expect(response.headers.get("content-type")).toBe("application/octet-stream");

      const data = new Uint8Array(await response.arrayBuffer());
      expect(data.length).toBeGreaterThan(0);
      expect(data.length % 32).toBe(0);
    } else {
      expect(response.status).toBe(404);
    }
  });

  test("tile data is immutable (idempotent reads)", async () => {
    const response1 = await fetch(`${baseUrl}/tile/0/0`);
    const data1 = new Uint8Array(await response1.arrayBuffer());

    const response2 = await fetch(`${baseUrl}/tile/0/0`);
    const data2 = new Uint8Array(await response2.arrayBuffer());

    // Tiles are immutable - should be identical
    expect(data1).toEqual(data2);
  });
});

describe("Checkpoint Endpoint", () => {
  let server: any;
  let baseUrl: string;

  beforeAll(async () => {
    const { startServer } = await import("../../src/service/server.ts");
    server = await startServer({
      port: 0,
      database: ":memory:",
      storage: { type: "local", path: "./.test-storage" },
    });
    baseUrl = `http://localhost:${server.port}`;

    // Register at least one entry to have a non-empty checkpoint
    const { generateES256KeyPair } = await import("../../src/lib/cose/signer.ts");
    const { signHashEnvelope } = await import("../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../src/lib/cose/sign.ts");
    const { encodeCoseSign1 } = await import("../../src/lib/cose/sign.ts");

    const issuerKey = await generateES256KeyPair();
    const payload = new TextEncoder().encode("Test artifact");
    const cwtClaims = createCWTClaims({
      iss: "https://issuer.example.com",
      sub: "artifact-checkpoint",
    });

    const statement = await signHashEnvelope(
      payload,
      { contentType: "text/plain" },
      issuerKey.privateKey,
      cwtClaims
    );

    const statementBytes = encodeCoseSign1(statement);

    await fetch(`${baseUrl}/entries`, {
      method: "POST",
      headers: {
        "Content-Type": "application/cose",
      },
      body: statementBytes,
    });
  });

  afterAll(() => {
    if (server) {
      server.stop();
    }
  });

  test("GET /checkpoint returns current tree state", async () => {
    const response = await fetch(`${baseUrl}/checkpoint`);

    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toContain("text/plain");

    const checkpoint = await response.text();

    // Checkpoint should be in signed note format
    expect(checkpoint).toContain("—"); // Signature separator
    expect(checkpoint.length).toBeGreaterThan(0);
  });

  test("GET /checkpoint includes origin and tree size", async () => {
    const response = await fetch(`${baseUrl}/checkpoint`);

    expect(response.status).toBe(200);

    const checkpoint = await response.text();
    const lines = checkpoint.split("\n");

    // First line is origin
    expect(lines[0]).toContain("http://localhost");

    // Second line is tree size (should be a number)
    const treeSize = parseInt(lines[1]!, 10);
    expect(treeSize).toBeGreaterThan(0);
  });

  test("handles CORS for checkpoint endpoint", async () => {
    const response = await fetch(`${baseUrl}/checkpoint`, {
      method: "OPTIONS",
    });

    expect(response.status).toBe(204);
    expect(response.headers.get("access-control-allow-origin")).toBeDefined();
    expect(response.headers.get("access-control-allow-methods")).toContain("GET");
  });

  test("checkpoint updates after new registrations", async () => {
    // Get initial checkpoint
    const response1 = await fetch(`${baseUrl}/checkpoint`);
    const checkpoint1 = await response1.text();
    const treeSize1 = parseInt(checkpoint1.split("\n")[1]!, 10);

    // Register another entry
    const { generateES256KeyPair } = await import("../../src/lib/cose/signer.ts");
    const { signHashEnvelope } = await import("../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../src/lib/cose/sign.ts");
    const { encodeCoseSign1 } = await import("../../src/lib/cose/sign.ts");

    const issuerKey = await generateES256KeyPair();
    const payload = new TextEncoder().encode("Another artifact");
    const cwtClaims = createCWTClaims({
      iss: "https://issuer.example.com",
      sub: "artifact-2",
    });

    const statement = await signHashEnvelope(
      payload,
      { contentType: "text/plain" },
      issuerKey.privateKey,
      cwtClaims
    );

    const statementBytes = encodeCoseSign1(statement);

    await fetch(`${baseUrl}/entries`, {
      method: "POST",
      headers: {
        "Content-Type": "application/cose",
      },
      body: statementBytes,
    });

    // Get updated checkpoint
    const response2 = await fetch(`${baseUrl}/checkpoint`);
    const checkpoint2 = await response2.text();
    const treeSize2 = parseInt(checkpoint2.split("\n")[1]!, 10);

    // Tree size should have increased
    expect(treeSize2).toBeGreaterThan(treeSize1);
  });
});
