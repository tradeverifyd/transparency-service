// Trace oldSize=2, newSize=4 (which works)
// proof = [fc264939...] = hash(h2,h3)

console.log("=== WORKING CASE: oldSize=2, newSize=4, proof=[hash(h2,h3)] ===\n");

console.log("Call 1: runTreeProof(p=[hash(h2,h3)], lo=0, hi=4, n=2, oldRoot=hash(h0,h1))");
console.log("  n=2, hi=4, n != hi");
console.log("  k = largestPowerOfTwoLessThan(4) = 2");
console.log("  n <= lo+k? → 2 <= 0+2 → TRUE");
console.log("  Branch: LEFT");
console.log("  Recurse: runTreeProof(p=[], lo=0, hi=2, n=2, oldRoot)");
console.log("");

console.log("Call 2: runTreeProof(p=[], lo=0, hi=2, n=2, oldRoot=hash(h0,h1))");
console.log("  n=2, hi=2, n == hi! BASE CASE");
console.log("  lo=0, so base case 1");
console.log("  p.length should be 0 ✓");
console.log("  Return [oldRoot, oldRoot] = [hash(h0,h1), hash(h0,h1)]");
console.log("");

console.log("Back to Call 1:");
console.log("  Received [oh=hash(h0,h1), th=hash(h0,h1)]");
console.log("  newHash = hashNode(th, p[0]) = hashNode(hash(h0,h1), hash(h2,h3))");
console.log("  Return [hash(h0,h1), hash(hash(h0,h1), hash(h2,h3))]");
console.log("  That's [oldRoot2, newRoot4] ✓✓✓");
