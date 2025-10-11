function largestPowerOfTwoLessThan(n: number): number {
  let k = 1;
  while (k * 2 < n) {
    k *= 2;
  }
  return k;
}

console.log("largestPowerOfTwoLessThan(4) =", largestPowerOfTwoLessThan(4));
console.log("largestPowerOfTwoLessThan(2) =", largestPowerOfTwoLessThan(2));
console.log("largestPowerOfTwoLessThan(3) =", largestPowerOfTwoLessThan(3));

console.log("\nFor oldSize=3, newSize=4:");
console.log("Call 1: lo=0, hi=4, n=3");
console.log("  k = largestPowerOfTwoLessThan(4-0) =", largestPowerOfTwoLessThan(4));
console.log("  n <= lo+k? → 3 <= 0+2 =", 3 <= 0+2);
console.log("  Branch: RIGHT");
console.log("\nCall 2: lo=2, hi=4, n=3");
console.log("  k = largestPowerOfTwoLessThan(4-2) =", largestPowerOfTwoLessThan(2));
console.log("  n <= lo+k? → 3 <= 2+1 =", 3 <= 2+1);
console.log("  Branch:", 3 <= 2+1 ? "LEFT" : "RIGHT");
