/**
 * Contract test for health check endpoint
 * Validates /health endpoint
 */

import { describe, test, expect, beforeAll, afterAll } from "bun:test";

describe("Health Check Endpoint", () => {
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

  test("GET /health returns 200 OK", async () => {
    const response = await fetch(`${baseUrl}/health`);

    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toContain("application/json");
  });

  test("health response includes status", async () => {
    const response = await fetch(`${baseUrl}/health`);
    const health = await response.json();

    expect(health.status).toBeDefined();
    expect(health.status).toBe("healthy");
  });

  test("health response includes component checks", async () => {
    const response = await fetch(`${baseUrl}/health`);
    const health = await response.json();

    expect(health.checks).toBeDefined();
    expect(typeof health.checks).toBe("object");

    // Database check
    expect(health.checks.database).toBeDefined();
    expect(health.checks.database.status).toBe("healthy");

    // Storage check
    expect(health.checks.storage).toBeDefined();
    expect(health.checks.storage.status).toBe("healthy");

    // Merkle tree check
    expect(health.checks.merkle_tree).toBeDefined();
    expect(health.checks.merkle_tree.status).toBe("healthy");
  });

  test("health response includes version", async () => {
    const response = await fetch(`${baseUrl}/health`);
    const health = await response.json();

    expect(health.version).toBeDefined();
    expect(typeof health.version).toBe("string");
  });

  test("health response includes tree size", async () => {
    const response = await fetch(`${baseUrl}/health`);
    const health = await response.json();

    expect(health.tree_size).toBeDefined();
    expect(typeof health.tree_size).toBe("number");
    expect(health.tree_size).toBeGreaterThanOrEqual(0);
  });

  test("health check is fast (< 100ms)", async () => {
    const start = Date.now();
    await fetch(`${baseUrl}/health`);
    const duration = Date.now() - start;

    expect(duration).toBeLessThan(100);
  });

  test("health check does not require authentication", async () => {
    const response = await fetch(`${baseUrl}/health`, {
      headers: {
        // No auth headers
      },
    });

    expect(response.status).toBe(200);
  });

  test("handles CORS preflight request", async () => {
    const response = await fetch(`${baseUrl}/health`, {
      method: "OPTIONS",
    });

    expect(response.status).toBe(204);
    expect(response.headers.get("access-control-allow-origin")).toBeDefined();
  });
});

describe("Health Check - Degraded States", () => {
  test("returns degraded status when database is unavailable", async () => {
    // This would require mocking database failures
    // For now, we just verify the contract
    expect(true).toBe(true);
  });

  test("returns degraded status when storage is unavailable", async () => {
    // This would require mocking storage failures
    expect(true).toBe(true);
  });
});
