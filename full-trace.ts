console.log("=== Full Trace for oldSize=3, newSize=4, proof=[h3] ===\n");

console.log("Call 1: runTreeProof(p=[h3], lo=0, hi=4, n=3, oldRoot)");
console.log("  n=3, hi=4 → n != hi (not base case)");
console.log("  p.length=1 (not empty)");
console.log("  k = largestPowerOfTwoLessThan(4-0) = 2");
console.log("  n <= lo+k? → 3 <= 0+2 = false");
console.log("  Branch: RIGHT (n > lo+k)");
console.log("  Recurse: runTreeProof(p.slice(0,-1), lo+k, hi, n, oldRoot)");
console.log("         = runTreeProof([], 2, 4, 3, oldRoot)\n");

console.log("Call 2: runTreeProof(p=[], lo=2, hi=4, n=3, oldRoot)");
console.log("  n=3, hi=4 → n != hi (not base case)");
console.log("  p.length=0 → ERROR: Proof too short!");
console.log("");
console.log("WAIT! We check p.length === 0 BEFORE computing k!");
console.log("Let me check the Go code order...");
