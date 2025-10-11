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

// Build 4-entry tree
const leaves: Uint8Array[] = [];
for (let i = 0; i < 4; i++) {
  const leaf = new Uint8Array(32);
  leaf.fill(i);
  leaves.push(leaf);
  await tileLog.append(leaf);
}

const root = await tileLog.root();
console.log("Root:", Buffer.from(root).toString("hex").substring(0, 16) + "...");

// Test each entry
for (let i = 0; i < 4; i++) {
  const proof = await generateInclusionProof(storage, i, 4);
  const isValid = await verifyInclusionProof(leaves[i], proof, root);
  console.log(`Entry ${i}: valid=${isValid}, auditPath length=${proof.auditPath.length}`);
}
