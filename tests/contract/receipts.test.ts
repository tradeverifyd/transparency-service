/**
 * Contract test for receipt resolution endpoint
 * Validates GET /entries/{entry_id}/receipt endpoint per SCRAPI specification
 */

import { describe, test, expect, beforeAll, afterAll } from "bun:test";

describe("Receipt Resolution Endpoint", () => {
  let server: any;
  let baseUrl: string;
  let testEntryId: string;

  beforeAll(async () => {
    // Start test server
    const { startServer } = await import("../../src/service/server.ts");
    server = await startServer({
      port: 0,
      database: ":memory:",
      storage: { type: "local", path: "./.test-storage" },
    });
    baseUrl = `http://localhost:${server.port}`;

    // Register a test statement to get an entry ID
    const { generateES256KeyPair } = await import("../../src/lib/cose/signer.ts");
    const { signHashEnvelope } = await import("../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../src/lib/cose/sign.ts");
    const { encodeCoseSign1 } = await import("../../src/lib/cose/sign.ts");

    const issuerKey = await generateES256KeyPair();
    const payload = new TextEncoder().encode("Test artifact for receipts");
    const cwtClaims = createCWTClaims({
      iss: "https://issuer.example.com",
      sub: "artifact-receipts-test",
    });

    const statement = await signHashEnvelope(
      payload,
      { contentType: "application/octet-stream" },
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

    if (response.status !== 201) {
      throw new Error(`Registration failed with status ${response.status}`);
    }

    const result = await response.json();
    testEntryId = result.entry_id;

    if (!testEntryId) {
      throw new Error("No entry_id returned from registration");
    }
  });

  afterAll(() => {
    if (server) {
      server.stop();
    }
  });

  test("GET /entries/{entry_id}/receipt returns receipt with inclusion proof", async () => {
    const response = await fetch(`${baseUrl}/entries/${testEntryId}/receipt`);

    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toContain("application/json");

    const receipt = await response.json();

    expect(receipt.tree_size).toBeDefined();
    expect(typeof receipt.tree_size).toBe("number");
    expect(receipt.tree_size).toBeGreaterThan(0);

    expect(receipt.leaf_index).toBeDefined();
    expect(typeof receipt.leaf_index).toBe("number");

    expect(receipt.inclusion_proof).toBeDefined();
    expect(Array.isArray(receipt.inclusion_proof)).toBe(true);
  });

  test("GET /entries/{entry_id}/receipt returns 404 for non-existent entry", async () => {
    const response = await fetch(`${baseUrl}/entries/nonexistent/receipt`);

    expect(response.status).toBe(404);

    const error = await response.json();
    expect(error.error).toBeDefined();
  });

  test("GET /entries/{entry_id}/receipt includes tree size and leaf index", async () => {
    const response = await fetch(`${baseUrl}/entries/${testEntryId}/receipt`);

    expect(response.status).toBe(200);

    const receipt = await response.json();

    // Tree size should be at least 1
    expect(receipt.tree_size).toBeGreaterThanOrEqual(1);

    // Leaf index should be valid (0 or greater, less than tree size)
    expect(receipt.leaf_index).toBeGreaterThanOrEqual(0);
    expect(receipt.leaf_index).toBeLessThan(receipt.tree_size);
  });

  test("GET /entries/{entry_id}/receipt has valid inclusion proof structure", async () => {
    const response = await fetch(`${baseUrl}/entries/${testEntryId}/receipt`);

    expect(response.status).toBe(200);

    const receipt = await response.json();

    // Inclusion proof should be an array of base64url strings
    expect(Array.isArray(receipt.inclusion_proof)).toBe(true);

    for (const hash of receipt.inclusion_proof) {
      expect(typeof hash).toBe("string");
      // Base64url should not contain +, /, or =
      expect(hash).not.toContain("+");
      expect(hash).not.toContain("/");
      expect(hash).not.toContain("=");
    }
  });

  test("handles CORS for receipt endpoint", async () => {
    const response = await fetch(`${baseUrl}/entries/${testEntryId}/receipt`, {
      method: "OPTIONS",
    });

    expect(response.status).toBe(204);
    expect(response.headers.get("access-control-allow-origin")).toBeDefined();
    expect(response.headers.get("access-control-allow-methods")).toContain("GET");
  });

  test("GET /entries/{entry_id}/receipt is idempotent", async () => {
    // Fetch receipt twice
    const response1 = await fetch(`${baseUrl}/entries/${testEntryId}/receipt`);
    const receipt1 = await response1.json();

    const response2 = await fetch(`${baseUrl}/entries/${testEntryId}/receipt`);
    const receipt2 = await response2.json();

    // Results should be identical
    expect(receipt1.tree_size).toBe(receipt2.tree_size);
    expect(receipt1.leaf_index).toBe(receipt2.leaf_index);
    expect(receipt1.inclusion_proof).toEqual(receipt2.inclusion_proof);
  });
});

describe("Receipt Content Validation", () => {
  let server: any;
  let baseUrl: string;
  let testEntryId: string;
  let testReceipt: any;

  beforeAll(async () => {
    const { startServer } = await import("../../src/service/server.ts");
    server = await startServer({
      port: 0,
      database: ":memory:",
      storage: { type: "local", path: "./.test-storage" },
    });
    baseUrl = `http://localhost:${server.port}`;

    // Register multiple statements to create a tree with proofs
    const { generateES256KeyPair } = await import("../../src/lib/cose/signer.ts");
    const { signHashEnvelope } = await import("../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../src/lib/cose/sign.ts");
    const { encodeCoseSign1 } = await import("../../src/lib/cose/sign.ts");

    const issuerKey = await generateES256KeyPair();

    // Register multiple entries
    for (let i = 0; i < 3; i++) {
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

      const response = await fetch(`${baseUrl}/entries`, {
        method: "POST",
        headers: {
          "Content-Type": "application/cose",
        },
        body: statementBytes,
      });

      const result = await response.json();

      // Save first entry for testing
      if (i === 0) {
        testEntryId = result.entry_id;
        testReceipt = result.receipt;
      }
    }
  });

  afterAll(() => {
    if (server) {
      server.stop();
    }
  });

  test("receipt from registration matches receipt from resolution", async () => {
    const response = await fetch(`${baseUrl}/entries/${testEntryId}/receipt`);
    const resolvedReceipt = await response.json();

    // Registration receipt and resolved receipt should match
    expect(resolvedReceipt.tree_size).toBeGreaterThanOrEqual(testReceipt.tree_size);
    expect(resolvedReceipt.leaf_index).toBe(testReceipt.leaf_index);
  });

  test("receipt inclusion proof can be used for verification", async () => {
    const response = await fetch(`${baseUrl}/entries/${testEntryId}/receipt`);
    const receipt = await response.json();

    // The inclusion proof should have expected structure
    // For tree size > 1, we should have some proof nodes
    if (receipt.tree_size > 1) {
      expect(receipt.inclusion_proof.length).toBeGreaterThan(0);
    } else {
      // Single entry tree has empty proof
      expect(receipt.inclusion_proof.length).toBe(0);
    }
  });
});
