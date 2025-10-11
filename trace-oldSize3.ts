// Trace oldSize=3, newSize=4 (which fails)
// proof = [h3]
// oldRoot = hash(hash(h0,h1), h2)
// newRoot = hash(hash(h0,h1), hash(h2,h3))

console.log("=== FAILING CASE: oldSize=3, newSize=4, proof=[h3] ===\n");

console.log("Call 1: runTreeProof(p=[h3], lo=0, hi=4, n=3, oldRoot=hash(hash(h0,h1),h2))");
console.log("  n=3, hi=4, n != hi");
console.log("  k = largestPowerOfTwoLessThan(4) = 2");
console.log("  n <= lo+k? → 3 <= 0+2 → FALSE");
console.log("  Branch: RIGHT");
console.log("  Recurse: runTreeProof(p=[], lo=2, hi=4, n=3, oldRoot)");
console.log("");

console.log("Call 2: runTreeProof(p=[], lo=2, hi=4, n=3, oldRoot=hash(hash(h0,h1),h2))");
console.log("  n=3, hi=4, n != hi");
console.log("  p.length === 0 → ERROR!");
console.log("");

console.log("But WAIT! Let me check what SHOULD happen:");
console.log("  k = largestPowerOfTwoLessThan(4-2) = largestPowerOfTwoLessThan(2) = 1");
console.log("  n <= lo+k? → 3 <= 2+1 → TRUE");
console.log("  So it SHOULD go LEFT!");
console.log("");

console.log("If we didn't error and continued LEFT:");
console.log("  Call 3: runTreeProof(p=[], lo=2, hi=3, n=3, oldRoot)");
console.log("    n=3, hi=3, n == hi! BASE CASE");
console.log("    lo=2 != 0, so base case 2");
console.log("    p.length should be 1, but we have 0 → ERROR!");
console.log("");

console.log("AH! So the issue is:");
console.log("When we reach the base case at [2,3) with n=3,");
console.log("we need ONE proof element (the hash of this subtree in the new tree).");
console.log("But we already consumed it in Call 1!");
console.log("");

console.log("INSIGHT: The proof should contain MORE elements!");
console.log("Or... the recursion branching is wrong!");
