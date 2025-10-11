import { verifyConsistencyProof } from "./src/lib/merkle/proofs.ts";

// Test case: oldSize=3, newSize=4
// Old tree [0,1,2]: ba8d94b7fbcecae7b81c4c80574fe24734a6917bf9c1ecd66ff3e0c34ead4620
// New tree [0,1,2,3]: fdea52008cdae79fa8bf806261959e23f5e11681646a2fa2bc9b5e56b32030a2
// Proof: [acaa04663a8547a2f70c60cc18f9378796b13c4f9a08f70d6adae662365b30c6] (hash of leaf 3)

function hexToBytes(hex: string): Uint8Array {
  const bytes = new Uint8Array(hex.length / 2);
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.substr(i, 2), 16);
  }
  return bytes;
}

const oldRoot = hexToBytes("ba8d94b7fbcecae7b81c4c80574fe24734a6917bf9c1ecd66ff3e0c34ead4620");
const newRoot = hexToBytes("fdea52008cdae79fa8bf806261959e23f5e11681646a2fa2bc9b5e56b32030a2");
const proof = [hexToBytes("acaa04663a8547a2f70c60cc18f9378796b13c4f9a08f70d6adae662365b30c6")];

console.log("Testing oldSize=3, newSize=4");
console.log("oldRoot:", oldRoot.slice(0, 8));
console.log("newRoot:", newRoot.slice(0, 8));
console.log("proof length:", proof.length);

try {
  const result = await verifyConsistencyProof(
    {
      oldSize: 3,
      newSize: 4,
      proof,
    },
    oldRoot,
    newRoot
  );

  console.log("\nVerification result:", result);
} catch (e) {
  console.error("Error:", e);
}
