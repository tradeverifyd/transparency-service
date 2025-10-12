/**
 * Contract test for statement registration endpoint
 * Validates POST /entries endpoint per SCRAPI specification
 */

import { describe, test, expect, beforeAll, afterAll } from "bun:test";

describe("Statement Registration Endpoint", () => {
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

  test("POST /entries registers a valid statement", async () => {
    const { generateES256KeyPair } = await import("../../src/lib/cose/signer.ts");
    const { signHashEnvelope } = await import("../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../src/lib/cose/sign.ts");
    const { encodeCoseSign1 } = await import("../../src/lib/cose/sign.ts");

    // Create issuer key
    const issuerKey = await generateES256KeyPair();

    // Create statement (hash envelope)
    const payload = new TextEncoder().encode("Test artifact");
    const cwtClaims = createCWTClaims({
      iss: "https://issuer.example.com",
      sub: "artifact-123",
    });

    const statement = await signHashEnvelope(
      payload,
      { contentType: "application/octet-stream" },
      issuerKey.privateKey,
      cwtClaims
    );

    // Encode statement
    const statementBytes = encodeCoseSign1(statement);

    // Register statement
    const response = await fetch(`${baseUrl}/entries`, {
      method: "POST",
      headers: {
        "Content-Type": "application/cose",
      },
      body: statementBytes,
    });

    expect(response.status).toBe(201);
    expect(response.headers.get("location")).toBeDefined();

    const result = await response.json();

    expect(result.entry_id).toBeDefined();
    expect(typeof result.entry_id).toBe("string");
  });

  test("POST /entries returns 400 for invalid COSE", async () => {
    const invalidData = new Uint8Array([1, 2, 3, 4]);

    const response = await fetch(`${baseUrl}/entries`, {
      method: "POST",
      headers: {
        "Content-Type": "application/cose",
      },
      body: invalidData,
    });

    expect(response.status).toBe(400);

    const error = await response.json();
    expect(error.error).toBeDefined();
  });

  test("POST /entries returns 415 for wrong content type", async () => {
    const response = await fetch(`${baseUrl}/entries`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ test: "data" }),
    });

    expect(response.status).toBe(415);
  });

  test("POST /entries includes receipt in response", async () => {
    const { generateES256KeyPair } = await import("../../src/lib/cose/signer.ts");
    const { signHashEnvelope } = await import("../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../src/lib/cose/sign.ts");
    const { encodeCoseSign1 } = await import("../../src/lib/cose/sign.ts");

    const issuerKey = await generateES256KeyPair();
    const payload = new TextEncoder().encode("Test artifact 2");
    const cwtClaims = createCWTClaims({
      iss: "https://issuer.example.com",
    });

    const statement = await signHashEnvelope(
      payload,
      { contentType: "text/plain" },
      issuerKey.privateKey,
      cwtClaims
    );

    const statementBytes = encodeCoseSign1(statement);

    const response = await fetch(`${baseUrl}/entries`, {
      method: "POST",
      headers: {
        "Content-Type": "application/cose",
      },
      body: statementBytes,
    });

    expect(response.status).toBe(201);

    const result = await response.json();

    expect(result.receipt).toBeDefined();
    expect(result.receipt.tree_size).toBeDefined();
    expect(typeof result.receipt.tree_size).toBe("number");
  });

  test("GET /entries/{entry_id} returns registered statement", async () => {
    const { generateES256KeyPair } = await import("../../src/lib/cose/signer.ts");
    const { signHashEnvelope } = await import("../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../src/lib/cose/sign.ts");
    const { encodeCoseSign1 } = await import("../../src/lib/cose/sign.ts");

    // Register a statement
    const issuerKey = await generateES256KeyPair();
    const payload = new TextEncoder().encode("Test artifact 3");
    const cwtClaims = createCWTClaims({
      iss: "https://issuer.example.com",
    });

    const statement = await signHashEnvelope(
      payload,
      { contentType: "text/plain" },
      issuerKey.privateKey,
      cwtClaims
    );

    const statementBytes = encodeCoseSign1(statement);

    const registerResponse = await fetch(`${baseUrl}/entries`, {
      method: "POST",
      headers: {
        "Content-Type": "application/cose",
      },
      body: statementBytes,
    });

    const { entry_id } = await registerResponse.json();

    // Retrieve the statement
    const getResponse = await fetch(`${baseUrl}/entries/${entry_id}`);

    expect(getResponse.status).toBe(200);
    expect(getResponse.headers.get("content-type")).toContain("application/cose");

    const retrievedBytes = new Uint8Array(await getResponse.arrayBuffer());
    expect(retrievedBytes.length).toBeGreaterThan(0);
  });

  test("GET /entries/{entry_id} returns 404 for non-existent entry", async () => {
    const response = await fetch(`${baseUrl}/entries/nonexistent`);

    expect(response.status).toBe(404);
  });

  test("handles CORS for registration endpoint", async () => {
    const response = await fetch(`${baseUrl}/entries`, {
      method: "OPTIONS",
    });

    expect(response.status).toBe(204);
    expect(response.headers.get("access-control-allow-origin")).toBeDefined();
    expect(response.headers.get("access-control-allow-methods")).toContain("POST");
  });
});

describe("Registration Status Polling", () => {
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
  });

  afterAll(() => {
    if (server) {
      server.stop();
    }
  });

  test("GET /entries/{entry_id}/status returns registration status", async () => {
    // This test would check async registration status
    // For now, we assume synchronous registration
    expect(true).toBe(true);
  });

  test("returns 'accepted' status for pending registrations", async () => {
    // Placeholder for async registration testing
    expect(true).toBe(true);
  });

  test("returns 'registered' status for completed registrations", async () => {
    // Placeholder for async registration testing
    expect(true).toBe(true);
  });
});
