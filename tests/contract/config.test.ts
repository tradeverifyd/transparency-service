/**
 * Contract test for transparency configuration endpoint
 * Validates /.well-known/scitt-configuration endpoint
 */

import { describe, test, expect, beforeAll, afterAll } from "bun:test";

describe("Transparency Configuration Endpoint", () => {
  let server: any;
  let baseUrl: string;

  beforeAll(async () => {
    // Start test server
    const { startServer } = await import("../../src/service/server.ts");
    server = await startServer({
      port: 0, // Random port
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

  test("GET /.well-known/scitt-configuration returns valid configuration", async () => {
    const response = await fetch(`${baseUrl}/.well-known/scitt-configuration`);

    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toContain("application/json");

    const config = await response.json();

    // Required fields per SCITT specification
    expect(config.issuer).toBeDefined();
    expect(typeof config.issuer).toBe("string");

    expect(config.registration_endpoint).toBeDefined();
    expect(typeof config.registration_endpoint).toBe("string");
    expect(config.registration_endpoint).toContain("/entries");

    expect(config.jwks_uri).toBeDefined();
    expect(typeof config.jwks_uri).toBe("string");

    // Optional fields
    if (config.service_documentation) {
      expect(typeof config.service_documentation).toBe("string");
    }
  });

  test("configuration issuer matches service origin", async () => {
    const response = await fetch(`${baseUrl}/.well-known/scitt-configuration`);
    const config = await response.json();

    expect(config.issuer).toBe(baseUrl);
  });

  test("configuration includes tile endpoints", async () => {
    const response = await fetch(`${baseUrl}/.well-known/scitt-configuration`);
    const config = await response.json();

    // Should include information about tile access
    expect(config.tile_endpoint).toBeDefined();
    expect(config.tile_endpoint).toContain("/tile");
  });

  test("configuration includes checkpoint endpoint", async () => {
    const response = await fetch(`${baseUrl}/.well-known/scitt-configuration`);
    const config = await response.json();

    expect(config.checkpoint_endpoint).toBeDefined();
    expect(config.checkpoint_endpoint).toBe(`${baseUrl}/checkpoint`);
  });

  test("configuration includes supported algorithms", async () => {
    const response = await fetch(`${baseUrl}/.well-known/scitt-configuration`);
    const config = await response.json();

    expect(config.supported_algorithms).toBeDefined();
    expect(Array.isArray(config.supported_algorithms)).toBe(true);
    expect(config.supported_algorithms).toContain("ES256");
  });

  test("handles CORS preflight request", async () => {
    const response = await fetch(`${baseUrl}/.well-known/scitt-configuration`, {
      method: "OPTIONS",
    });

    expect(response.status).toBe(204);
    expect(response.headers.get("access-control-allow-origin")).toBeDefined();
  });
});
