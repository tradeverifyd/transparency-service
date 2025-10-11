import { TileLog } from "./src/lib/merkle/tile-log.ts";
import { generateConsistencyProof, verifyConsistencyProof } from "./src/lib/merkle/proofs.ts";
import { LocalStorage } from "./src/lib/storage/local.ts";
import * as fs from "fs";

const testDir = "./.test-all-pairs";
if (fs.existsSync(testDir)) {
  fs.rmSync(testDir, { recursive: true });
}
fs.mkdirSync(testDir, { recursive: true });

const storage = new LocalStorage(testDir);
const tileLog = new TileLog(storage);

const roots: Uint8Array[] = [];

// Build tree incrementally
for (let size = 1; size <= 8; size++) {
  const leaf = new Uint8Array(32);
  leaf.fill(size - 1);
  await tileLog.append(leaf);
  roots.push(await tileLog.root());
}

// Verify all pairs
let failCount = 0;
for (let oldSize = 1; oldSize <= 8; oldSize++) {
  for (let newSize = oldSize; newSize <= 8; newSize++) {
    const proof = await generateConsistencyProof(storage, oldSize, newSize);
    const isValid = await verifyConsistencyProof(
      proof,
      roots[oldSize - 1],
      roots[newSize - 1]
    );
    
    if (!isValid) {
      console.log(`FAIL: oldSize=${oldSize}, newSize=${newSize}, proofLen=${proof.proof.length}`);
      failCount++;
    }
  }
}

console.log(`\nTotal failures: ${failCount}/64 pairs`);

// Clean up
fs.rmSync(testDir, { recursive: true });
