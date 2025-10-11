import { TileLog } from "./src/lib/merkle/tile-log.ts";
import { generateInclusionProof, verifyInclusionProof } from "./src/lib/merkle/proofs.ts";
import { LocalStorage } from "./src/lib/storage/local.ts";
import * as fs from "fs";

const testDir = "./.test-debug-4";
if (fs.existsSync(testDir)) {
  fs.rmSync(testDir, { recursive: true });
}
fs.mkdirSync(testDir, { recursive: true });

const storage = new LocalStorage(testDir);
const tileLog = new TileLog(storage);

// Helper functions
async function hashLeaf(data: Uint8Array) {
  const prefixed = new Uint8Array(1 + data.length);
  prefixed[0] = 0x00;
  prefixed.set(data, 1);
  return new Uint8Array(await crypto.subtle.digest("SHA-256", prefixed));
}

async function hashNode(left: Uint8Array, right: Uint8Array) {
  const prefixed = new Uint8Array(1 + left.length + right.length);
  prefixed[0] = 0x01;
  prefixed.set(left, 1);
  prefixed.set(right, 1 + left.length);
  return new Uint8Array(await crypto.subtle.digest("SHA-256", prefixed));
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

// Manual computation
const h0 = await hashLeaf(leaves[0]);
const h1 = await hashLeaf(leaves[1]);
const h2 = await hashLeaf(leaves[2]);
const h3 = await hashLeaf(leaves[3]);
const node01 = await hashNode(h0, h1);
const node23 = await hashNode(h2, h3);
const manualRoot = await hashNode(node01, node23);

console.log("Manual root:", Buffer.from(manualRoot).toString("hex").substring(0, 16) + "...");
console.log("TileLog root:", Buffer.from(root).toString("hex").substring(0, 16) + "...");
console.log("Match?", Buffer.from(manualRoot).equals(Buffer.from(root)));

// Test entry 0
console.log("\nTesting entry 0:");
const proof = await generateInclusionProof(storage, 0, 4);
console.log("Audit path:");
for (let i = 0; i < proof.auditPath.length; i++) {
  console.log(`  [${i}]: ${Buffer.from(proof.auditPath[i]).toString("hex").substring(0, 16)}...`);
}

console.log("\nExpected audit path for entry 0:");
console.log(`  [0]: ${Buffer.from(h1).toString("hex").substring(0, 16)}... (sibling h1)`);
console.log(`  [1]: ${Buffer.from(node23).toString("hex").substring(0, 16)}... (uncle node23)`);

// Manual verification
let current = h0;
current = await hashNode(current, proof.auditPath[0]);
console.log("\nAfter step 1:", Buffer.from(current).toString("hex").substring(0, 16) + "...");
console.log("Should be node01:", Buffer.from(node01).toString("hex").substring(0, 16) + "...");

current = await hashNode(current, proof.auditPath[1]);
console.log("After step 2:", Buffer.from(current).toString("hex").substring(0, 16) + "...");
console.log("Should be root:", Buffer.from(root).toString("hex").substring(0, 16) + "...");
