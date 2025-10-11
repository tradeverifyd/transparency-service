import { TileLog } from "./src/lib/merkle/tile-log.ts";
import { generateInclusionProof, verifyInclusionProof } from "./src/lib/merkle/proofs.ts";
import { LocalStorage } from "./src/lib/storage/local.ts";
import * as fs from "fs";

for (const treeSize of [2, 4, 8]) {
  const testDir = "./.test-size-" + treeSize;
  if (fs.existsSync(testDir)) { fs.rmSync(testDir, { recursive: true }); }
  fs.mkdirSync(testDir, { recursive: true });

  const storage = new LocalStorage(testDir);
  const tileLog = new TileLog(storage);

  const leaves: Uint8Array[] = [];
  for (let i = 0; i < treeSize; i++) {
    const leaf = new Uint8Array(32); leaf.fill(i);
    leaves.push(leaf); await tileLog.append(leaf);
  }

  const root = await tileLog.root();
  
  console.log("\nTree size " + treeSize + ":");
  let passCount = 0;
  for (let i = 0; i < treeSize; i++) {
    const proof = await generateInclusionProof(storage, i, treeSize);
    const isValid = await verifyInclusionProof(leaves[i], proof, root);
    if (isValid) passCount++;
    if (!isValid) console.log("  Entry " + i + ": FAIL");
  }
  console.log("  Result: " + passCount + "/" + treeSize + " valid");
}
