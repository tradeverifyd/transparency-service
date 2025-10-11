import { TileLog } from "./src/lib/merkle/tile-log.ts";
import { generateInclusionProof, verifyInclusionProof } from "./src/lib/merkle/proofs.ts";
import { LocalStorage } from "./src/lib/storage/local.ts";
import * as fs from "fs";

const testDir = "./.test-debug-8";
if (fs.existsSync(testDir)) { fs.rmSync(testDir, { recursive: true }); }
fs.mkdirSync(testDir, { recursive: true });

const storage = new LocalStorage(testDir);
const tileLog = new TileLog(storage);

const leaves: Uint8Array[] = [];
for (let i = 0; i < 8; i++) {
  const leaf = new Uint8Array(32); leaf.fill(i);
  leaves.push(leaf); await tileLog.append(leaf);
}

const root = await tileLog.root();

// Test each entry
console.log("Testing size 8 tree:");
for (let i = 0; i < 8; i++) {
  const proof = await generateInclusionProof(storage, i, 8);
  const isValid = await verifyInclusionProof(leaves[i], proof, root);
  console.log("Entry " + i + ": valid=" + isValid + ", auditPath len=" + proof.auditPath.length);
}
