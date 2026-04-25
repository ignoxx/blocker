# Index Balancing

## What Is It?

"Index balancing" refers to the invariant that **every block's height must equal its position in the chain**. If the last block is at height 5, the next block *must* have height 6. Not 7, not a random number — exactly 6.

In other words:

```
block.Header.Height == chain.Height() + 1
```

This sounds obvious, but the current codebase has **three bugs** that violate this invariant.

---

## Why It Matters

Without index balancing, your blockchain breaks in several ways:

### 1. Chain integrity
If block heights can be arbitrary, you lose the ability to identify blocks by height. `GetBlockByHeight(5)` becomes meaningless if multiple blocks exist at height 5 or if heights skip numbers.

### 2. Proposer selection (PoS)
In Proof of Stake, the proposer is determined by height:

```go
proposerIndex := height % len(validators)
```

If heights are wrong, the wrong validator proposes, and consensus breaks immediately.

### 3. Chain reorgs and forks
When two nodes compare their chains, they use height to determine which chain is "ahead." If heights are inconsistent, nodes can't agree on which chain is canonical.

---

## The Three Bugs

### Bug 1: `util.RandomBlock()` sets random height

```go
// util/random.go — current code
func RandomBlock() *proto.Block {
    header := &proto.Header{
        Version:   1,
        Height:    int32(rand.Intn(1000)),  // BUG: random height!
        PrevHash:  RandomHash(),
        RootHash:  RandomHash(),
        Timestamp: time.Now().UnixNano(),
    }
    return &proto.Block{Header: header}
}
```

This function is used in tests to create blocks. The `rand.Intn(1000)` means blocks get heights like 47, 831, 12 — completely detached from their actual position in the chain. It also generates a random `PrevHash`, which would fail the prevHash validation.

**Fix:** Accept height and prevHash as parameters:

```go
func RandomBlock(height int32, prevHash []byte) *proto.Block {
    header := &proto.Header{
        Version:   1,
        Height:    height,
        PrevHash:  prevHash,
        Timestamp: time.Now().UnixNano(),
    }
    return &proto.Block{Header: header}
}
```

Then update `chain_test.go`:

```go
func randomBlock(t *testing.T, chain *Chain) *proto.Block {
    privKey, _ := crypto.GeneratePrivateKey()
    prevBlock, err := chain.GetBlockByHeight(chain.Height())
    require.Nil(t, err)
    block := util.RandomBlock(int32(chain.Height()+1), types.HashBlock(prevBlock))
    types.SignBlock(privKey, block)
    return block
}
```

### Bug 2: Genesis block height is implicit

```go
// node/chain.go — current code
func createGenesisBlock() *proto.Block {
    privKey, _ := crypto.NewPrivateKeyFromSeedStr(MasterSeed)
    block := &proto.Block{
        Header: &proto.Header{
            Version: 1,
            // Height is not set — relies on protobuf default of 0
        },
    }
    // ...
}
```

The proto3 default for `int32` is `0`, which *happens* to be correct for a genesis block. But relying on implicit defaults is fragile. If someone changes the proto definition or if the default changes, this silently breaks.

**Fix:** Set the height explicitly:

```go
func createGenesisBlock() *proto.Block {
    privKey, _ := crypto.NewPrivateKeyFromSeedStr(MasterSeed)
    block := &proto.Block{
        Header: &proto.Header{
            Version:   1,
            Height:    0,          // EXPLICIT: genesis is height 0
            Timestamp: 0,          // Genesis has no meaningful timestamp
        },
    }
    // ...
}
```

### Bug 3: `ValidateBlock` doesn't check height

This is the most critical bug. The current validation function checks signature and prevHash, but never verifies the height:

```go
// node/chain.go — current code
func (c *Chain) ValidateBlock(b *proto.Block) error {
    if !types.VerifyBlock(b) {
        return fmt.Errorf("invalid block: signature verification failed")
    }

    currentBlock, err := c.GetBlockByHeight(c.Height())
    if err != nil {
        return err
    }

    hash := types.HashBlock(currentBlock)
    if !bytes.Equal(hash, b.Header.PrevHash) {
        return fmt.Errorf("invalid block: expected prev hash %s, got %s", ...)
    }

    // ... transaction validation ...

    return nil  // BUG: height is never checked!
}
```

An attacker could submit a block at height 99999 or height -5 and it would pass validation (assuming valid signature and prevHash).

**Fix:** Add a height check before the prevHash check:

```go
func (c *Chain) ValidateBlock(b *proto.Block) error {
    if !types.VerifyBlock(b) {
        return fmt.Errorf("invalid block: signature verification failed")
    }

    // INDEX BALANCING: height must be exactly one more than current tip
    expectedHeight := int32(c.Height() + 1)
    if b.Header.Height != expectedHeight {
        return fmt.Errorf("invalid block: expected height %d, got %d",
            expectedHeight, b.Header.Height)
    }

    currentBlock, err := c.GetBlockByHeight(c.Height())
    // ... rest of validation ...
}
```

---

## The Full Picture

Here's the complete flow with index balancing enforced:

```
Genesis Block (height 0)
    │
    ▼
Block 1 (height 1, prevHash = hash of genesis)
    │
    ▼
Block 2 (height 2, prevHash = hash of Block 1)
    │
    ▼
Block 3 (height 3, prevHash = hash of Block 2)
    ...
```

Each block's height is deterministic:
- **Who proposes it?** → `validators[height % len(validators)]`
- **What is its prevHash?** → hash of the block at `height - 1`
- **What height does it have?** → `chain.Height() + 1`

There is no ambiguity. No gaps. No duplicates. That's index balancing.

---

## The `HeaderList` — Height vs. Len vs. Index

There's a subtle distinction in `node/chain.go` that's worth understanding:

```go
type HeaderList struct {
    headers []*proto.Header
}

func (list *HeaderList) Len() int {
    return len(list.headers)
}

func (list *HeaderList) Height() int {
    return len(list.headers) - 1
}
```

- `Len()` = number of headers stored (1-based count)
- `Height()` = index of the latest header (0-based)

If you have the genesis block (height 0) and block 1:
- `Len()` returns 2
- `Height()` returns 1

This means:
- `headers[0]` → genesis (height 0)
- `headers[1]` → block 1 (height 1)
- `headers[height]` → the block at that height

`Height()` will always equal `Len() - 1` because the genesis block is stored at index 0. This is correct and consistent — just be aware of the off-by-one when converting between count and index.

---

## Quick Reference: Fixes Summary

| Where | What | Fix |
|-------|------|-----|
| `util/random.go` | `RandomBlock()` uses random height & prevHash | Accept `height` and `prevHash` as params |
| `node/chain.go` | `createGenesisBlock()` doesn't set height explicitly | Set `Height: 0` |
| `node/chain.go` | `ValidateBlock()` doesn't check height | Add `b.Header.Height != int32(c.Height()+1)` check |
| `node/node.go` | `validatorLoop()` doesn't create blocks | Set height correctly when creating blocks (see POS.md Step 4) |