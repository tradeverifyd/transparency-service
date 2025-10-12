# RFC 6962 Consistency Proof Analysis

## Tree Structure

Tree size 3:
```
      MTH(3)
     /      \
  MTH([0,2)) MTH([2,3))
   /    \        |
 h(0)  h(1)    h(2)
```

Actually, with k-based splitting:
- k = largestPowerOfTwo(3) = 2
- Left subtree: [0,2), size=2
- Right subtree: [2,3), size=1

Tree size 3 = hash(MTH([0,2)), MTH([2,3)))
           = hash(hash(h0,h1), h2)

Tree size 4:
- k = largestPowerOfTwo(4) = 2
- Left subtree: [0,2), size=2
- Right subtree: [2,4), size=2

Tree size 4 = hash(MTH([0,2)), MTH([2,4)))
           = hash(hash(h0,h1), hash(h2,h3))

## What should the proof contain?

The old tree root (size 3) = hash(hash(h0,h1), h2)
The new tree root (size 4) = hash(hash(h0,h1), hash(h2,h3))

The proof needs to allow verifier to:
1. Reconstruct the old root
2. Reconstruct the new root

Given:
- oldRoot = hash(hash(h0,h1), h2)  
- The proof should help us compute newRoot

The key insight: hash(h0,h1) is common to both trees.
The difference is:
- Old: hash(COMMON, h2)
- New: hash(COMMON, hash(h2,h3))

So if we know h3, we can:
- Compute hash(h2,h3)
- Combine with COMMON to get new root

But how do we get h2 to reconstruct the old root?

AH! We don't need h2 separately because we're GIVEN oldRoot!

Let me trace what runTreeProof should do...
