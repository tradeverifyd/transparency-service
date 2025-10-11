// Check what our TypeScript implementation generates for oldSize=3, newSize=4

import { generateConsistencyProof } from "./src/lib/merkle/proofs.ts";
import { LocalStorage } from "./src/lib/storage/local.ts";
import { TileLog } from "./src/lib/merkle/tile-log.ts";
import * as fs from "fs";

function bytesToHex(b: Uint8Array): string {
  return Array.from(b).map(x => x.toString(16).padStart(2, '0')).join('');
}

const testDir = "./.test-proof-gen";
if (fs.existsSync(testDir)) {
  fs.rmSync(testDir, { recursive: true });
}
fs.mkdirSync(testDir, { recursive: true });

const storage = new LocalStorage(testDir);
const tileLog = new TileLog(storage);

// Add 4 leaves
for (let i = 0; i < 4; i++) {
  const leaf = new Uint8Array(32).fill(i);
  await tileLog.append(leaf);
}

// Generate consistency proof for oldSize=3, newSize=4
const proof = await generateConsistencyProof(storage, 3, 4);

console.log("TypeScript generated proof for oldSize=3, newSize=4:");
console.log("Proof length:", proof.proof.length);
for (let i = 0; i < proof.proof.length; i++) {
  console.log(`proof[${i}]:`, bytesToHex(proof.proof[i]));
}

// Compare with Go-generated proof
console.log("\nGo-generated proof (from test vectors):");
console.log("proof[0]: acaa04663a8547a2f70c60cc18f9378796b13c4f9a08f70d6adae662365b30c6");

// Clean up
fs.rmSync(testDir, { recursive: true });
