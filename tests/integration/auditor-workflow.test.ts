/**
 * Integration test for auditor workflow
 * Tests consistency checking and log integrity verification
 */

import { describe, test, expect, beforeAll, afterAll } from "bun:test";

describe("Auditor Workflow - Consistency Verification", () => {
  let server: any;
  let baseUrl: string;
  let checkpoint1: string;
  let checkpoint2: string;
  let treeSize1: number;
  let treeSize2: number;

  beforeAll(async () => {
    // Start test server
    const { startServer } = await import("../../src/service/server.ts");
    server = await startServer({
      port: 0,
      database: ":memory:",
      storage: { type: "local", path: "./.test-storage" },
    });
    baseUrl = `http://localhost:${server.port}`;

    // Register initial statements and get first checkpoint
    const { generateES256KeyPair } = await import("../../src/lib/cose/signer.ts");
    const { signHashEnvelope } = await import("../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../src/lib/cose/sign.ts");
    const { encodeCoseSign1 } = await import("../../src/lib/cose/sign.ts");

    const issuerKey = await generateES256KeyPair();

    // Register 3 statements
    for (let i = 0; i < 3; i++) {
      const payload = new TextEncoder().encode(`Audit test artifact ${i}`);
      const cwtClaims = createCWTClaims({
        iss: "https://auditor-test.example.com",
        sub: `audit-artifact-${i}`,
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
        headers: { "Content-Type": "application/cose" },
        body: statementBytes,
      });
    }

    // Get first checkpoint
    const cp1Response = await fetch(`${baseUrl}/checkpoint`);
    checkpoint1 = await cp1Response.text();
    treeSize1 = parseInt(checkpoint1.split("\n")[1]!, 10);

    // Register 2 more statements
    for (let i = 3; i < 5; i++) {
      const payload = new TextEncoder().encode(`Audit test artifact ${i}`);
      const cwtClaims = createCWTClaims({
        iss: "https://auditor-test.example.com",
        sub: `audit-artifact-${i}`,
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
        headers: { "Content-Type": "application/cose" },
        body: statementBytes,
      });
    }

    // Get second checkpoint
    const cp2Response = await fetch(`${baseUrl}/checkpoint`);
    checkpoint2 = await cp2Response.text();
    treeSize2 = parseInt(checkpoint2.split("\n")[1]!, 10);
  });

  afterAll(() => {
    if (server) {
      server.stop();
    }
  });

  test("checkpoints show increasing tree size", () => {
    expect(treeSize1).toBe(3);
    expect(treeSize2).toBe(5);
    expect(treeSize2).toBeGreaterThan(treeSize1);
  });

  test("checkpoints have valid signed note format", () => {
    // First checkpoint
    const lines1 = checkpoint1.split("\n");
    expect(lines1.length).toBeGreaterThanOrEqual(6);
    expect(lines1[0]).toContain("http://localhost"); // origin
    expect(lines1[1]).toBe("3"); // tree size
    expect(lines1[5]).toMatch(/^— /); // signature line

    // Second checkpoint
    const lines2 = checkpoint2.split("\n");
    expect(lines2.length).toBeGreaterThanOrEqual(6);
    expect(lines2[0]).toContain("http://localhost");
    expect(lines2[1]).toBe("5");
    expect(lines2[5]).toMatch(/^— /);
  });

  test("checkpoints have different root hashes", () => {
    const rootHash1 = checkpoint1.split("\n")[2]!;
    const rootHash2 = checkpoint2.split("\n")[2]!;

    expect(rootHash1).toBeDefined();
    expect(rootHash2).toBeDefined();
    expect(rootHash1).not.toBe(rootHash2);
  });

  test("checkpoints have monotonically increasing timestamps", () => {
    const timestamp1 = parseInt(checkpoint1.split("\n")[3]!, 10);
    const timestamp2 = parseInt(checkpoint2.split("\n")[3]!, 10);

    expect(timestamp1).toBeGreaterThan(0);
    expect(timestamp2).toBeGreaterThan(0);
    expect(timestamp2).toBeGreaterThanOrEqual(timestamp1);
  });

  test("checkpoint signatures are present and non-empty", () => {
    const sig1Match = checkpoint1.split("\n")[5]!.match(/^— .+ (.+)$/);
    const sig2Match = checkpoint2.split("\n")[5]!.match(/^— .+ (.+)$/);

    expect(sig1Match).toBeTruthy();
    expect(sig2Match).toBeTruthy();
    expect(sig1Match![1]).toBeDefined();
    expect(sig2Match![1]).toBeDefined();
    expect(sig1Match![1]!.length).toBeGreaterThan(0);
    expect(sig2Match![1]!.length).toBeGreaterThan(0);
  });
});

describe("Auditor Workflow - Log Integrity", () => {
  let server: any;
  let baseUrl: string;
  let entryIds: string[] = [];

  beforeAll(async () => {
    const { startServer } = await import("../../src/service/server.ts");
    server = await startServer({
      port: 0,
      database: ":memory:",
      storage: { type: "local", path: "./.test-storage" },
    });
    baseUrl = `http://localhost:${server.port}`;

    // Register multiple statements
    const { generateES256KeyPair } = await import("../../src/lib/cose/signer.ts");
    const { signHashEnvelope } = await import("../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../src/lib/cose/sign.ts");
    const { encodeCoseSign1 } = await import("../../src/lib/cose/sign.ts");

    const issuerKey = await generateES256KeyPair();

    for (let i = 0; i < 10; i++) {
      const payload = new TextEncoder().encode(`Integrity test ${i}`);
      const cwtClaims = createCWTClaims({
        iss: "https://integrity-test.example.com",
        sub: `integrity-${i}`,
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
        headers: { "Content-Type": "application/cose" },
        body: statementBytes,
      });

      const result = await response.json();
      entryIds.push(result.entry_id);
    }
  });

  afterAll(() => {
    if (server) {
      server.stop();
    }
  });

  test("all registered statements can be retrieved", async () => {
    for (const entryId of entryIds) {
      const response = await fetch(`${baseUrl}/entries/${entryId}`);
      expect(response.status).toBe(200);
      expect(response.headers.get("content-type")).toContain("application/cose");

      const data = new Uint8Array(await response.arrayBuffer());
      expect(data.length).toBeGreaterThan(0);
    }
  });

  test("all registered statements have receipts", async () => {
    for (const entryId of entryIds) {
      const response = await fetch(`${baseUrl}/entries/${entryId}/receipt`);
      expect(response.status).toBe(200);

      const receipt = await response.json();
      expect(receipt.tree_size).toBeDefined();
      expect(receipt.leaf_index).toBeDefined();
      expect(receipt.inclusion_proof).toBeDefined();
    }
  });

  test("receipt leaf indices are sequential", async () => {
    const leafIndices: number[] = [];

    for (const entryId of entryIds) {
      const response = await fetch(`${baseUrl}/entries/${entryId}/receipt`);
      const receipt = await response.json();
      leafIndices.push(receipt.leaf_index);
    }

    // Check that leaf indices are 0, 1, 2, 3, ..., 9
    for (let i = 0; i < leafIndices.length; i++) {
      expect(leafIndices[i]).toBe(i);
    }
  });

  test("all receipts reference same final tree size", async () => {
    const treeSizes: number[] = [];

    for (const entryId of entryIds) {
      const response = await fetch(`${baseUrl}/entries/${entryId}/receipt`);
      const receipt = await response.json();
      treeSizes.push(receipt.tree_size);
    }

    // All receipts should reference the current tree size (10)
    for (const size of treeSizes) {
      expect(size).toBe(10);
    }
  });

  test("checkpoint reflects all registered entries", async () => {
    const response = await fetch(`${baseUrl}/checkpoint`);
    const checkpoint = await response.text();
    const treeSize = parseInt(checkpoint.split("\n")[1]!, 10);

    expect(treeSize).toBe(entryIds.length);
  });
});

describe("Auditor Workflow - Append-Only Verification", () => {
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

  test("tree grows monotonically with new registrations", async () => {
    const { generateES256KeyPair } = await import("../../src/lib/cose/signer.ts");
    const { signHashEnvelope } = await import("../../src/lib/cose/hash-envelope.ts");
    const { createCWTClaims } = await import("../../src/lib/cose/sign.ts");
    const { encodeCoseSign1 } = await import("../../src/lib/cose/sign.ts");

    const issuerKey = await generateES256KeyPair();
    const treeSizes: number[] = [];

    // Register statements and track tree growth
    for (let i = 0; i < 5; i++) {
      const payload = new TextEncoder().encode(`Growth test ${i}`);
      const cwtClaims = createCWTClaims({
        iss: "https://growth-test.example.com",
        sub: `growth-${i}`,
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
        headers: { "Content-Type": "application/cose" },
        body: statementBytes,
      });

      // Get checkpoint after each registration
      const cpResponse = await fetch(`${baseUrl}/checkpoint`);
      const checkpoint = await cpResponse.text();
      const treeSize = parseInt(checkpoint.split("\n")[1]!, 10);
      treeSizes.push(treeSize);
    }

    // Verify monotonic growth
    for (let i = 1; i < treeSizes.length; i++) {
      expect(treeSizes[i]).toBe(treeSizes[i - 1]! + 1);
    }

    // Final tree size should be 5
    expect(treeSizes[treeSizes.length - 1]).toBe(5);
  });
});
