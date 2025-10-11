/**
 * Generate test vectors using TypeScript implementation
 * for cross-validation with Go
 */

import { TileLog } from "../../src/lib/merkle/tile-log.ts";
import { generateInclusionProof, generateConsistencyProof } from "../../src/lib/merkle/proofs.ts";
import { LocalStorage } from "../../src/lib/storage/local.ts";
import * as fs from "fs";
import * as path from "path";

interface TestCase {
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

function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

async function generateVectors(size: number): Promise<TestCase> {
  const testDir = `./.test-ts-vectors-${size}`;
  if (fs.existsSync(testDir)) {
    fs.rmSync(testDir, { recursive: true });
  }
  fs.mkdirSync(testDir, { recursive: true });

  const storage = new LocalStorage(testDir);
  const tileLog = new TileLog(storage);

  const testCase: TestCase = {
    treeSize: size,
    leaves: [],
    root: "",
    inclusionProofs: [],
    consistencyProofs: [],
  };

  // Create leaves (32 bytes filled with index value)
  const leaves: Uint8Array[] = [];
  for (let i = 0; i < size; i++) {
    const leaf = new Uint8Array(32);
    leaf.fill(i);
    leaves.push(leaf);
    testCase.leaves.push(bytesToHex(leaf));
    await tileLog.append(leaf);
  }

  // Get root
  const root = await tileLog.root();
  testCase.root = bytesToHex(root);

  // Store intermediate roots for consistency proofs
  const roots: Uint8Array[] = [];
  for (let i = 1; i <= size; i++) {
    // Rebuild tree up to size i
    const tempDir = `./.test-ts-temp-${size}-${i}`;
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
    fs.mkdirSync(tempDir, { recursive: true });

    const tempStorage = new LocalStorage(tempDir);
    const tempLog = new TileLog(tempStorage);

    for (let j = 0; j < i; j++) {
      await tempLog.append(leaves[j]);
    }

    roots.push(await tempLog.root());
    fs.rmSync(tempDir, { recursive: true });
  }

  // Generate inclusion proofs
  for (let i = 0; i < size; i++) {
    const proof = await generateInclusionProof(storage, i, size);

    testCase.inclusionProofs.push({
      leafIndex: proof.leafIndex,
      auditPath: proof.auditPath.map(bytesToHex),
    });
  }

  // Generate consistency proofs
  for (let oldSize = 1; oldSize < size; oldSize++) {
    const proof = await generateConsistencyProof(storage, oldSize, size);

    testCase.consistencyProofs.push({
      oldSize: proof.oldSize,
      newSize: proof.newSize,
      oldRoot: bytesToHex(roots[oldSize - 1]),
      newRoot: bytesToHex(roots[size - 1]),
      proof: proof.proof.map(bytesToHex),
    });
  }

  // Clean up
  fs.rmSync(testDir, { recursive: true });

  return testCase;
}

async function main() {
  // Test various tree sizes covering:
  // - Small trees: 2, 4, 7, 8, 16, 31, 32
  // - Approaching tile boundary: 63, 64, 127, 128
  const treeSizes = [2, 4, 7, 8, 16, 31, 32, 63, 64, 127, 128];
  const outputDir = path.join(__dirname, "ts-test-vectors");

  if (fs.existsSync(outputDir)) {
    fs.rmSync(outputDir, { recursive: true });
  }
  fs.mkdirSync(outputDir, { recursive: true });

  for (const size of treeSizes) {
    console.log(`Generating test vectors for tree size ${size}...`);
    const testCase = await generateVectors(size);

    const filename = path.join(outputDir, `tlog-size-${size}.json`);
    fs.writeFileSync(filename, JSON.stringify(testCase, null, 2));

    console.log(`  ✓ Generated ${testCase.inclusionProofs.length} inclusion proofs`);
    console.log(`  ✓ Generated ${testCase.consistencyProofs.length} consistency proofs`);
  }

  console.log("\n✅ All TypeScript test vectors generated successfully!");
  console.log(`   Output directory: ${outputDir}`);
}

await main();
