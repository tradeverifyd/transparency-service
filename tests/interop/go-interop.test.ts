/**
 * Go tlog Interoperability Tests
 *
 * These tests verify that our TypeScript implementation produces results
 * compatible with the canonical Go implementation from golang.org/x/mod/sumdb/tlog
 */

import { describe, test, expect, beforeAll } from "bun:test";
import { TileLog } from "../../src/lib/merkle/tile-log.ts";
import {
  generateInclusionProof,
  verifyInclusionProof,
  verifyConsistencyProof,
} from "../../src/lib/merkle/proofs.ts";
import { LocalStorage } from "../../src/lib/storage/local.ts";
import * as fs from "fs";
import * as path from "path";

interface GoTestCase {
  treeSize: number;
  leaves: string[];
  root: string;
  inclusionProofs: Array<{
    leafIndex: number;
    auditPath: string[];
  }>;
  consistencyProofs: Array<{
    oldSize: number;
    newSize: number;
    oldRoot: string;
    newRoot: string;
    proof: string[];
  }>;
}

function hexToBytes(hex: string): Uint8Array {
  const bytes = new Uint8Array(hex.length / 2);
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.substr(i, 2), 16);
  }
  return bytes;
}

function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

describe("Go tlog Interoperability", () => {
  // Test various tree sizes covering:
  // - Small trees: 2, 4, 7, 8, 16, 31, 32
  // - Approaching tile boundary: 63, 64, 127, 128
  const treeSizes = [2, 4, 7, 8, 16, 31, 32, 63, 64, 127, 128];

  for (const size of treeSizes) {
    describe(`Tree size ${size}`, () => {
      let testCase: GoTestCase;
      let storage: LocalStorage;
      let tileLog: TileLog;
      const testDir = `tests/.test-go-interop-${size}`;

      beforeAll(async () => {
        // Load Go-generated test vectors
        const vectorPath = path.join(__dirname, "test-vectors", `tlog-size-${size}.json`);
        const vectorData = fs.readFileSync(vectorPath, "utf-8");
        testCase = JSON.parse(vectorData);

        // Setup TypeScript implementation
        if (fs.existsSync(testDir)) {
          fs.rmSync(testDir, { recursive: true });
        }
        fs.mkdirSync(testDir, { recursive: true });

        storage = new LocalStorage(testDir);
        tileLog = new TileLog(storage);

        // Append all leaves
        for (const leafHex of testCase.leaves) {
          const leaf = hexToBytes(leafHex);
          await tileLog.append(leaf);
        }
      });

      test("computes same root hash as Go", async () => {
        const tsRoot = await tileLog.root();
        const tsRootHex = bytesToHex(tsRoot);

        expect(tsRootHex).toBe(testCase.root);
      });

      test("generates compatible inclusion proofs", async () => {
        for (let i = 0; i < testCase.treeSize; i++) {
          const tsProof = await generateInclusionProof(storage, i, testCase.treeSize);
          const goProof = testCase.inclusionProofs[i];

          // Check proof structure
          expect(tsProof.leafIndex).toBe(goProof.leafIndex);
          expect(tsProof.treeSize).toBe(testCase.treeSize);
          expect(tsProof.auditPath.length).toBe(goProof.auditPath.length);

          // Check each hash in audit path matches
          for (let j = 0; j < tsProof.auditPath.length; j++) {
            const tsHashHex = bytesToHex(tsProof.auditPath[j]);
            expect(tsHashHex).toBe(goProof.auditPath[j]);
          }
        }
      });

      test("verifies Go-generated proofs", async () => {
        const root = hexToBytes(testCase.root);

        for (const goProof of testCase.inclusionProofs) {
          const leaf = hexToBytes(testCase.leaves[goProof.leafIndex]);
          const auditPath = goProof.auditPath.map(hexToBytes);

          const isValid = await verifyInclusionProof(
            leaf,
            {
              leafIndex: goProof.leafIndex,
              treeSize: testCase.treeSize,
              auditPath,
            },
            root
          );

          expect(isValid).toBe(true);
        }
      });

      test("Go proofs verify with TypeScript root", async () => {
        const tsRoot = await tileLog.root();

        for (const goProof of testCase.inclusionProofs) {
          const leaf = hexToBytes(testCase.leaves[goProof.leafIndex]);
          const auditPath = goProof.auditPath.map(hexToBytes);

          const isValid = await verifyInclusionProof(
            leaf,
            {
              leafIndex: goProof.leafIndex,
              treeSize: testCase.treeSize,
              auditPath,
            },
            tsRoot
          );

          expect(isValid).toBe(true);
        }
      });

      test("verifies Go-generated consistency proofs", async () => {
        for (const goProof of testCase.consistencyProofs) {
          const oldRoot = hexToBytes(goProof.oldRoot);
          const newRoot = hexToBytes(goProof.newRoot);
          const proof = goProof.proof.map(hexToBytes);

          const isValid = await verifyConsistencyProof(
            {
              oldSize: goProof.oldSize,
              newSize: goProof.newSize,
              proof,
            },
            oldRoot,
            newRoot
          );

          if (!isValid) {
            console.log(`Failed: oldSize=${goProof.oldSize}, newSize=${goProof.newSize}`);
          }
          expect(isValid).toBe(true);
        }
      });
    });
  }
});
