import { TileLog } from "./src/lib/merkle/tile-log.ts";
import { LocalStorage } from "./src/lib/storage/local.ts";
import * as fs from "fs";
import {
  entryTileIndexToPath,
  entryIdToTileIndex,
  entryIdToTileOffset,
  HASH_SIZE
} from "./src/lib/merkle/tile-naming.ts";

const testDir = "./.test-trace";
if (fs.existsSync(testDir)) { fs.rmSync(testDir, { recursive: true }); }
fs.mkdirSync(testDir, { recursive: true });

const storage = new LocalStorage(testDir);
const tileLog = new TileLog(storage);

async function hashLeaf(data: Uint8Array) {
  const p = new Uint8Array(1 + data.length);
  p[0] = 0x00; p.set(data, 1);
  return new Uint8Array(await crypto.subtle.digest("SHA-256", p));
}

async function hashNode(left: Uint8Array, right: Uint8Array) {
  const p = new Uint8Array(1 + left.length + right.length);
  p[0] = 0x01; p.set(left, 1); p.set(right, 1 + left.length);
  return new Uint8Array(await crypto.subtle.digest("SHA-256", p));
}

async function getLeafFromStorage(storage: any, entryId: number) {
  const tileIndex = entryIdToTileIndex(entryId);
  const tileOffset = entryIdToTileOffset(entryId);
  const tilePath = entryTileIndexToPath(tileIndex);
  const tileData = await storage.get(tilePath);
  const start = tileOffset * HASH_SIZE;
  return tileData.slice(start, start + HASH_SIZE);
}

function largestPowerOfTwoLessThan(n: number): number {
  let k = 1;
  while (k * 2 < n) {
    k *= 2;
  }
  return k;
}

async function computeSubtreeHash(
  storage: any,
  start: number,
  size: number
): Promise<Uint8Array> {
  console.log(`  computeSubtreeHash(start=${start}, size=${size})`);

  if (size === 1) {
    const leaf = await getLeafFromStorage(storage, start);
    const hash = await hashLeaf(leaf);
    console.log(`    -> single leaf ${start}: ${Buffer.from(hash).toString("hex").substring(0, 16)}`);
    return hash;
  }

  const k = largestPowerOfTwoLessThan(size);
  const leftHash = await computeSubtreeHash(storage, start, k);
  const rightHash = await computeSubtreeHash(storage, start + k, size - k);
  const combined = await hashNode(leftHash, rightHash);
  console.log(`    -> combined: ${Buffer.from(combined).toString("hex").substring(0, 16)}`);
  return combined;
}

async function traceProofGeneration(leafIndex: number, treeSize: number) {
  console.log(`\n=== Generating proof for leaf ${leafIndex} in tree size ${treeSize} ===`);

  const auditPath: Uint8Array[] = [];
  let index = leafIndex;
  let size = treeSize;
  let offset = 0;
  let step = 0;

  while (size > 1) {
    step++;
    const k = largestPowerOfTwoLessThan(size);
    console.log(`\nStep ${step}: index=${index}, size=${size}, offset=${offset}, k=${k}`);

    if (index < k) {
      console.log(`  Leaf in LEFT subtree (index ${index} < k ${k})`);
      const rightSize = size - k;
      console.log(`  Need RIGHT sibling: computeSubtreeHash(${offset + k}, ${rightSize})`);
      const rightHash = await computeSubtreeHash(storage, offset + k, rightSize);
      auditPath.push(rightHash);
      size = k;
      console.log(`  Next: size=${size}, offset=${offset} (unchanged)`);
    } else {
      console.log(`  Leaf in RIGHT subtree (index ${index} >= k ${k})`);
      console.log(`  Need LEFT sibling: computeSubtreeHash(${offset}, ${k})`);
      const leftHash = await computeSubtreeHash(storage, offset, k);
      auditPath.push(leftHash);
      index = index - k;
      offset = offset + k;
      size = size - k;
      console.log(`  Next: index=${index}, size=${size}, offset=${offset}`);
    }
  }

  console.log(`\nAudit path (before reverse): ${auditPath.length} hashes`);
  auditPath.reverse();
  console.log(`Audit path (after reverse): ${auditPath.length} hashes`);

  return auditPath;
}

// Build 4-entry tree
const leaves: Uint8Array[] = [];
for (let i = 0; i < 4; i++) {
  const leaf = new Uint8Array(32);
  leaf.fill(i);
  leaves.push(leaf);
  await tileLog.append(leaf);
}

// Compute expected hashes manually
console.log("\n=== Expected Tree Structure ===");
const h0 = await hashLeaf(leaves[0]);
const h1 = await hashLeaf(leaves[1]);
const h2 = await hashLeaf(leaves[2]);
const h3 = await hashLeaf(leaves[3]);
console.log("h0:", Buffer.from(h0).toString("hex").substring(0, 16));
console.log("h1:", Buffer.from(h1).toString("hex").substring(0, 16));
console.log("h2:", Buffer.from(h2).toString("hex").substring(0, 16));
console.log("h3:", Buffer.from(h3).toString("hex").substring(0, 16));

const node01 = await hashNode(h0, h1);
const node23 = await hashNode(h2, h3);
console.log("node01 (h0+h1):", Buffer.from(node01).toString("hex").substring(0, 16));
console.log("node23 (h2+h3):", Buffer.from(node23).toString("hex").substring(0, 16));

const root = await hashNode(node01, node23);
console.log("root:", Buffer.from(root).toString("hex").substring(0, 16));

// Trace proof generation for index 1 (which fails)
const path1 = await traceProofGeneration(1, 4);

console.log("\n=== Expected audit path for index 1 ===");
console.log("Path should be: [h0, node23]");
console.log("h0:", Buffer.from(h0).toString("hex").substring(0, 16));
console.log("node23:", Buffer.from(node23).toString("hex").substring(0, 16));

console.log("\n=== Actual audit path ===");
console.log("Path[0]:", Buffer.from(path1[0]).toString("hex").substring(0, 16));
console.log("Path[1]:", Buffer.from(path1[1]).toString("hex").substring(0, 16));

console.log("\n=== Verification ===");
console.log("Path[0] == h0?", Buffer.from(path1[0]).equals(Buffer.from(h0)));
console.log("Path[1] == node23?", Buffer.from(path1[1]).equals(Buffer.from(node23)));
