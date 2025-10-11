# Understanding the `old` parameter in runTreeProof

The `old` parameter represents the Merkle root of the OLD tree at the current recursion level.

## Call 1: runTreeProof(p=[h3], lo=0, hi=4, n=3, oldRoot=MTH(3))

We're verifying tree[0,4) contains tree[0,3) as prefix.
- Old tree [0,3) has root MTH(3) = hash(hash(h0,h1), h2)
- New tree [0,4) has root MTH(4) = hash(hash(h0,h1), hash(h2,h3))

k=2, so:
- Left subtree: [0,2)
- Right subtree: [2,4)

Since n=3 > lo+k=2, the old tree boundary extends into the RIGHT subtree.

When we recurse RIGHT:
- We need to verify subtree [2,4) contains subtree [2,3) as prefix
- The old subtree [2,3) within the new subtree [2,4)
- What's the root of old subtree [2,3)? It's h2!

But we don't have h2 directly. We need to COMPUTE it from MTH(3).

MTH(3) = hash(MTH([0,2)), MTH([2,3)))
       = hash(hash(h0,h1), h2)

If we had MTH([0,2)) = hash(h0,h1), we could extract h2.
But we don't have that either - it's not in the proof!

WAIT! That's the key insight. When we go RIGHT, we consume proof[0] which should be the LEFT subtree hash!

So proof[0] = MTH([0,2)) = hash(h0,h1)

Then we can decompose:
- MTH(3) = hash(proof[0], something)
- Solving: something = h2 = the root of old subtree [2,3)

Let me check the Go code for how it uses proof elements...
