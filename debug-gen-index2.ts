import { TileLog } from "./src/lib/merkle/tile-log.ts";
import { LocalStorage } from "./src/lib/storage/local.ts";
import * as fs from "fs";
import {
  entryTileIndexToPath,
  entryIdToTileIndex,
  entryIdToTileOffset,
  HASH_SIZE
} from "./src/lib/merkle/tile-naming.ts";

const testDir = "./.test-gen-index2";
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

// Build 4-entry tree
const leaves: Uint8Array[] = [];
for (let i = 0; i < 4; i++) {
  const leaf = new Uint8Array(32);
  leaf.fill(i);
  leaves.push(leaf);
  await tileLog.append(leaf);
}

// Expected structure
const h0 = await hashLeaf(leaves[0]);
const h1 = await hashLeaf(leaves[1]);
const h2 = await hashLeaf(leaves[2]);
const h3 = await hashLeaf(leaves[3]);
const node01 = await hashNode(h0, h1);
const node23 = await hashNode(h2, h3);
const root = await hashNode(node01, node23);

console.log("Expected tree structure:");
console.log("h0:", Buffer.from(h0).toString("hex").substring(0, 16));
console.log("h1:", Buffer.from(h1).toString("hex").substring(0, 16));
console.log("h2:", Buffer.from(h2).toString("hex").substring(0, 16));
console.log("h3:", Buffer.from(h3).toString("hex").substring(0, 16));
console.log("node01:", Buffer.from(node01).toString("hex").substring(0, 16));
console.log("node23:", Buffer.from(node23).toString("hex").substring(0, 16));
console.log("root:", Buffer.from(root).toString("hex").substring(0, 16));

console.log("\nFor index 2 (h2), the audit path should be:");
console.log("1. Sibling at leaf level: h3");
console.log("2. Sibling at next level: node01");
console.log("Expected audit path: [h3, node01]");

console.log("\nVerification should be:");
console.log("1. Start with h2");
console.log("2. h2 is left child of (h2, h3), so: hash(h2, h3) = node23");
console.log("3. node23 is right child of (node01, node23), so: hash(node01, node23) = root");

// Import and test actual generation
import { generateInclusionProof } from "./src/lib/merkle/proofs.ts";
const proof2 = await generateInclusionProof(storage, 2, 4);

console.log("\nActual audit path:");
console.log("Path[0]:", Buffer.from(proof2.auditPath[0]).toString("hex").substring(0, 16));
console.log("Path[1]:", Buffer.from(proof2.auditPath[1]).toString("hex").substring(0, 16));

console.log("\nComparison:");
console.log("Path[0] == h3?", Buffer.from(proof2.auditPath[0]).equals(Buffer.from(h3)));
console.log("Path[1] == node01?", Buffer.from(proof2.auditPath[1]).equals(Buffer.from(node01)));
