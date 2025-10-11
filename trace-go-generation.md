# Trace Go's treeProofIndex for oldSize=3, newSize=4

## Call 1: treeProofIndex(lo=0, hi=4, n=3, need=[])
- n=3, hi=4, n != hi (not base case)
- k = maxpow2(4-0) = 2
- n <= lo+k? → 3 <= 0+2 → FALSE
- Go RIGHT: recurse for right subtree, then add left subtree
- need = treeProofIndex(lo+k=2, hi=4, n=3, need)
- need = subTreeIndex(lo=0, hi=2, need)

## Call 2 (RIGHT): treeProofIndex(lo=2, hi=4, n=3, need=[])
- n=3, hi=4, n != hi (not base case)
- k = maxpow2(4-2) = maxpow2(2) = 1
- n <= lo+k? → 3 <= 2+1 → TRUE
- Go LEFT: recurse for left subtree, then add right subtree
- need = treeProofIndex(lo=2, hi=2+1=3, n=3, need)
- need = subTreeIndex(lo+k=3, hi=4, need)

## Call 3 (LEFT of Call 2): treeProofIndex(lo=2, hi=3, n=3, need=[])
- n=3, hi=3, n == hi! Base case!
- lo=2 != 0, so: return subTreeIndex(lo=2, hi=3, need)

## subTreeIndex calls:
1. subTreeIndex(lo=2, hi=3, need=[]) - returns need + indexes for subtree [2,3)
2. subTreeIndex(lo=3, hi=4, need=...) - returns need + indexes for subtree [3,4)
3. subTreeIndex(lo=0, hi=2, need=...) - returns need + indexes for subtree [0,2)

## Order of additions:
1. First added: subtree [2,3) = just position 2
2. Then added: subtree [3,4) = just position 3
3. Then added: subtree [0,2) = positions 0,1

Wait, that would give us [h2, h3, ...]. But our proof is [h3].

Let me re-check the Go code structure...
