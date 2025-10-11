// Trace through consistency proof GENERATION for oldSize=3, newSize=4

function largestPowerOfTwoLessThan(n: number): number {
  let k = 1;
  while (k * 2 < n) {
    k *= 2;
  }
  return k;
}

console.log("consistencyProofHelper(oldSize=3, newSize=4, start=0, end=4, proof=[])");
console.log("  size = 4");
console.log("  start=0 < oldSize=3, so not all new");
console.log("  end=4 > oldSize=3, so not all old");
console.log("  Spans boundary! Recurse both sides.");
console.log("  k = largestPowerOfTwoLessThan(4) = 2");
console.log("  mid = 0 + 2 = 2");
console.log("");

console.log("Call LEFT: consistencyProofHelper(oldSize=3, newSize=4, start=0, end=2, proof=[])");
console.log("  size = 2");
console.log("  start=0 < oldSize=3");
console.log("  end=2 <= oldSize=3, so entirely old!");
console.log("  Skip (don't add to proof)");
console.log("");

console.log("Call RIGHT: consistencyProofHelper(oldSize=3, newSize=4, start=2, end=4, proof=[])");
console.log("  size = 2");
console.log("  start=2 < oldSize=3, so not all new");
console.log("  end=4 > oldSize=3, so not all old");
console.log("  Spans boundary! Recurse both sides.");
console.log("  k = largestPowerOfTwoLessThan(2) = 1");
console.log("  mid = 2 + 1 = 3");
console.log("");

console.log("Call LEFT: consistencyProofHelper(oldSize=3, newSize=4, start=2, end=3, proof=[])");
console.log("  size = 1");
console.log("  start=2 < oldSize=3");
console.log("  end=3 <= oldSize=3, so entirely old!");
console.log("  Skip (don't add to proof)");
console.log("");

console.log("Call RIGHT: consistencyProofHelper(oldSize=3, newSize=4, start=3, end=4, proof=[])");
console.log("  size = 1");
console.log("  start=3 >= oldSize=3, so entirely new!");
console.log("  Add hash of subtree [3,4) to proof");
console.log("  That's h3!");
console.log("");

console.log("Final proof: [h3]");
