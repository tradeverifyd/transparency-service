import { TileLog } from "./src/lib/merkle/tile-log.ts";
import { generateInclusionProof, verifyInclusionProof } from "./src/lib/merkle/proofs.ts";
import { LocalStorage } from "./src/lib/storage/local.ts";
import * as fs from "fs";

const testDir = "./.test-debug";
if (fs.existsSync(testDir)) {
  fs.rmSync(testDir, { recursive: true });
}
fs.mkdirSync(testDir, { recursive: true });

const storage = new LocalStorage(testDir);
const tileLog = new TileLog(storage);

// Build simple 2-entry tree
const leaf1 = new Uint8Array(32);
leaf1.fill(1);
await tileLog.append(leaf1);

const leaf2 = new Uint8Array(32);
leaf2.fill(2);
await tileLog.append(leaf2);

const root = await tileLog.root();
console.log("Root:", Buffer.from(root).toString("hex").substring(0, 16) + "...");

// Generate proof for first entry
const proof = await generateInclusionProof(storage, 0, 2);
console.log("\nProof for index 0:");
console.log("- leafIndex:", proof.leafIndex);
console.log("- treeSize:", proof.treeSize);
console.log("- auditPath length:", proof.auditPath.length);
if (proof.auditPath.length > 0) {
  console.log("- auditPath[0]:", Buffer.from(proof.auditPath[0]).toString("hex").substring(0, 16) + "...");
}

// Verify
const isValid = await verifyInclusionProof(leaf1, proof, root);
console.log("\nVerification result:", isValid);

// Manual verification
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

const hash1 = await hashLeaf(leaf1);
const hash2 = await hashLeaf(leaf2);
const manualRoot = await hashNode(hash1, hash2);

console.log("\nManual computation:");
console.log("hash(leaf1):", Buffer.from(hash1).toString("hex").substring(0, 16) + "...");
console.log("hash(leaf2):", Buffer.from(hash2).toString("hex").substring(0, 16) + "...");
console.log("Combined root:", Buffer.from(manualRoot).toString("hex").substring(0, 16) + "...");
console.log("Matches tile log root?", Buffer.from(root).equals(Buffer.from(manualRoot)));

// Manual verification of proof
console.log("\nManual proof verification:");
let current = hash1;
console.log("Start with hash(leaf1):", Buffer.from(current).toString("hex").substring(0, 16) + "...");
if (proof.auditPath.length > 0) {
  current = await hashNode(current, proof.auditPath[0]);
  console.log("After combining with audit path:", Buffer.from(current).toString("hex").substring(0, 16) + "...");
}
console.log("Matches root?", Buffer.from(current).equals(Buffer.from(root)));

// Check what's in the tiles
console.log("\nTile contents:");
const entryTile = await storage.get("tile/entries/000");
const hashTile = await storage.get("tile/0/000");

console.log("Entry tile (first 32 bytes):", entryTile ? Buffer.from(entryTile.slice(0, 32)).toString("hex").substring(0, 16) + "..." : "null");
console.log("Hash tile level 0 (first 32 bytes):", hashTile ? Buffer.from(hashTile.slice(0, 32)).toString("hex").substring(0, 16) + "..." : "null");
console.log("Are they equal?", entryTile && hashTile && Buffer.from(entryTile.slice(0, 32)).equals(Buffer.from(hashTile.slice(0, 32))));

// These should be the raw leaves
console.log("Entry tile matches leaf1?", entryTile && Buffer.from(entryTile.slice(0, 32)).equals(Buffer.from(leaf1)));
