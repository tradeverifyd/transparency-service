import { TileLog } from "./src/lib/merkle/tile-log.ts";
import { LocalStorage } from "./src/lib/storage/local.ts";
import * as fs from "fs";

function bytesToHex(b: Uint8Array): string {
  return Array.from(b).map(x => x.toString(16).padStart(2, '0')).join('');
}

const testDir = "./.test-check-hashes";
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

// Compute specific hashes
const root2 = await tileLog.rootAtSize(2);
const root3 = await tileLog.rootAtSize(3);
const root4 = await tileLog.rootAtSize(4);

console.log("MTH([0,2)) = hash(h0,h1):", bytesToHex(root2));
console.log("MTH([0,3)):", bytesToHex(root3));
console.log("MTH([0,4)):", bytesToHex(root4));

console.log("\nFrom test vectors:");
console.log("oldSize=2, newSize=4 proof[0]: fc264939b1ac77b06378c5ece54a7b57b6b6c821eb80627bb674d8785c8dc8ca");
console.log("oldSize=3, newSize=4 proof[0]: acaa04663a8547a2f70c60cc18f9378796b13c4f9a08f70d6adae662365b30c6");

// Clean up
fs.rmSync(testDir, { recursive: true });
