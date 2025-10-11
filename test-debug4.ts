import { TileLog } from "./src/lib/merkle/tile-log.ts";
import { generateInclusionProof } from "./src/lib/merkle/proofs.ts";
import { LocalStorage } from "./src/lib/storage/local.ts";
import * as fs from "fs";

const testDir = "./.test-debug-4";
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

const leaves: Uint8Array[] = [];
for (let i = 0; i < 4; i++) {
  const leaf = new Uint8Array(32); leaf.fill(i);
  leaves.push(leaf); await tileLog.append(leaf);
}

const root = await tileLog.root();

const h0 = await hashLeaf(leaves[0]);
const h1 = await hashLeaf(leaves[1]);
const h2 = await hashLeaf(leaves[2]);
const h3 = await hashLeaf(leaves[3]);
const node01 = await hashNode(h0, h1);
const node23 = await hashNode(h2, h3);
const manualRoot = await hashNode(node01, node23);

console.log("Roots match?", Buffer.from(manualRoot).equals(Buffer.from(root)));

const proof = await generateInclusionProof(storage, 0, 4);
console.log("Audit path length:", proof.auditPath.length);
console.log("Path[0] == h1?", Buffer.from(proof.auditPath[0]).equals(Buffer.from(h1)));
console.log("Path[1] == node23?", Buffer.from(proof.auditPath[1]).equals(Buffer.from(node23)));

let current = h0;
current = await hashNode(current, proof.auditPath[0]);
console.log("After step 1 == node01?", Buffer.from(current).equals(Buffer.from(node01)));
current = await hashNode(current, proof.auditPath[1]);
console.log("After step 2 == root?", Buffer.from(current).equals(Buffer.from(root)));

// Check what computeSubtreeHash actually returns
import { entryTileIndexToPath, entryIdToTileIndex, entryIdToTileOffset, HASH_SIZE } from "./src/lib/merkle/tile-naming.ts";

async function getLeafFromStorage(storage: any, entryId: number) {
  const tileIndex = entryIdToTileIndex(entryId);
  const tileOffset = entryIdToTileOffset(entryId);
  const tilePath = entryTileIndexToPath(tileIndex);
  const tileData = await storage.get(tilePath);
  const start = tileOffset * HASH_SIZE;
  return tileData.slice(start, start + HASH_SIZE);
}

console.log("\nChecking stored leaves:");
for (let i = 0; i < 4; i++) {
  const stored = await getLeafFromStorage(storage, i);
  console.log("Stored leaf " + i + " matches input?", Buffer.from(stored).equals(Buffer.from(leaves[i])));
}

console.log("\nAudit path contents:");
console.log("Path[0]:", Buffer.from(proof.auditPath[0]).toString("hex").substring(0, 32));
console.log("Expected h1:", Buffer.from(h1).toString("hex").substring(0, 32));
console.log("Path[1]:", Buffer.from(proof.auditPath[1]).toString("hex").substring(0, 32));
console.log("Expected node23:", Buffer.from(node23).toString("hex").substring(0, 32));
