import { TileLog } from "./src/lib/merkle/tile-log.ts";
import { LocalStorage } from "./src/lib/storage/local.ts";
import * as fs from "fs";

const testDir = "./.test-verify";
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

function largestPowerOfTwoLessThan(n: number): number {
  let k = 1;
  while (k * 2 < n) {
    k *= 2;
  }
  return k;
}

async function traceVerification(
  leaf: Uint8Array,
  leafIndex: number,
  treeSize: number,
  auditPath: Uint8Array[],
  root: Uint8Array
) {
  console.log(`\n=== Verifying proof for leaf ${leafIndex} in tree size ${treeSize} ===`);
  console.log(`Audit path length: ${auditPath.length}`);
  console.log(`Expected root: ${Buffer.from(root).toString("hex").substring(0, 16)}`);

  let currentHash = await hashLeaf(leaf);
  console.log(`\nStarting with leaf hash: ${Buffer.from(currentHash).toString("hex").substring(0, 16)}`);

  let index = leafIndex;
  let size = treeSize;
  let pathIndex = 0;
  let step = 0;

  while (size > 1) {
    step++;
    const k = largestPowerOfTwoLessThan(size);
    console.log(`\nStep ${step}: index=${index}, size=${size}, k=${k}, pathIndex=${pathIndex}`);

    if (pathIndex >= auditPath.length) {
      console.log("  ERROR: pathIndex out of bounds!");
      return false;
    }

    const sibling = auditPath[pathIndex];
    console.log(`  Sibling: ${Buffer.from(sibling).toString("hex").substring(0, 16)}`);

    if (index < k) {
      console.log(`  Leaf in LEFT subtree (index ${index} < k ${k})`);
      console.log(`  Computing: hashNode(current, sibling)`);
      currentHash = await hashNode(currentHash, sibling);
      size = k;
      console.log(`  Result: ${Buffer.from(currentHash).toString("hex").substring(0, 16)}`);
      console.log(`  Next: size=${size}, index=${index}`);
    } else {
      console.log(`  Leaf in RIGHT subtree (index ${index} >= k ${k})`);
      console.log(`  Computing: hashNode(sibling, current)`);
      currentHash = await hashNode(sibling, currentHash);
      index = index - k;
      size = size - k;
      console.log(`  Result: ${Buffer.from(currentHash).toString("hex").substring(0, 16)}`);
      console.log(`  Next: size=${size}, index=${index}`);
    }

    pathIndex++;
  }

  console.log(`\nFinal computed root: ${Buffer.from(currentHash).toString("hex").substring(0, 16)}`);
  console.log(`Expected root:       ${Buffer.from(root).toString("hex").substring(0, 16)}`);
  const matches = Buffer.from(currentHash).equals(Buffer.from(root));
  console.log(`Matches: ${matches}`);

  return matches;
}

// Build 4-entry tree
const leaves: Uint8Array[] = [];
for (let i = 0; i < 4; i++) {
  const leaf = new Uint8Array(32);
  leaf.fill(i);
  leaves.push(leaf);
  await tileLog.append(leaf);
}

const root = await tileLog.root();

// Compute expected structure
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

const manualRoot = await hashNode(node01, node23);
console.log("root:", Buffer.from(manualRoot).toString("hex").substring(0, 16));

// Test verification for index 1 with correct audit path [h0, node23]
const auditPath1 = [h0, node23];
await traceVerification(leaves[1], 1, 4, auditPath1, root);
