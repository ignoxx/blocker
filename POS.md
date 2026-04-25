# Proof of Stake (PoS) — Tutorial

This tutorial walks you through adding a simple **round-robin Proof of Stake consensus** to the `blocker` blockchain. By the end, your validators will take turns proposing blocks, and all nodes will validate that the proposer had the right to produce a block at that height.

---

## Why Proof of Stake?

Proof of Work (PoW) requires miners to solve a computationally expensive puzzle — find a nonce such that `SHA256(block_header) < target_difficulty`. This is fine for production chains like Bitcoin, but for a learning project it adds complexity without teaching you much about the *consensus* part.

Proof of Stake is simpler: a designated validator is chosen to propose the next block. No mining, no difficulty, no nonce. The "stake" part in our simplified version means we just rotate through a known set of validators.

---

## Design Overview

We use **round-robin proposer selection**:

```
proposer_index = block_height % number_of_validators
```

At height 5 with 3 validators (A, B, C):
- Height 0 → A proposes
- Height 1 → B proposes
- Height 2 → C proposes
- Height 3 → A proposes again
- and so on...

Each node knows the full validator set. When a block arrives, every node checks:
1. Is the block height correct? (`block.Header.Height == chain.Height() + 1`)
2. Is the previous hash correct? (`block.Header.PrevHash == hash_of_current_tip`)
3. Is the signature valid? (signed by the claimed proposer)
4. Is the proposer authorized? (public key is in the validator set)
5. Is it the proposer's turn? (`height % len(validators) == their_index`)

If all checks pass, the block is accepted and gossiped to peers.

---

## Step 1 — Update the Proto Definitions

We need two additions:
1. A new RPC `HandleBlock` so nodes can gossip blocks to each other
2. A `repeated bytes validators` field on `Version` so nodes exchange their validator set during handshake

Edit `proto/types.proto`:

```protobuf
syntax = "proto3";

option go_package = "github.com/ignoxx/blocker/proto";

import "google/protobuf/empty.proto";

service Node {
  rpc Handshake(Version) returns (Version);
  rpc HandleTransaction(Transaction) returns (google.protobuf.Empty);
  rpc HandleBlock(Block) returns (google.protobuf.Empty);      // NEW
}

message Version {
  string version = 1;
  int32 height = 2;
  string listenAddr = 3;
  repeated string peerList = 4;
  repeated bytes validators = 5;                               // NEW
}

message Block {
  Header header = 1;
  repeated Transaction transactions = 2;
  bytes publicKey = 3;
  bytes signature = 4;
}

message Header{
  int32 version = 1;
  int32 height = 2;
  bytes prevHash = 3;
  bytes rootHash = 4;
  int64 timestamp = 5;
}

message TxInput {
  bytes prevTxHash = 1;
  uint32 prevOutIndex = 2;
  bytes publicKey = 3;
  bytes signature = 4;
}

message TxOutput {
  int64 amount = 1;
  bytes address = 2;
}

message Transaction{
  int32 version = 1;
  repeated TxInput inputs = 2;
  repeated TxOutput outputs = 3;
}
```

Then regenerate:

```bash
make proto
```

---

## Step 2 — Create a ValidatorSet

Create a new file `node/validators.go`:

```go
package node

import (
	"github.com/ignoxx/blocker/crypto"
)

type ValidatorSet struct {
	validators []*crypto.PublicKey
}

func NewValidatorSet(validators []*crypto.PublicKey) *ValidatorSet {
	return &ValidatorSet{
		validators: validators,
	}
}

// proposerIndex returns the index of the validator who should propose at this height.
func (s *ValidatorSet) ProposerIndex(height int32) int32 {
	return height % int32(len(s.validators))
}

// GetProposer returns the public key of the validator who should propose at this height.
func (s *ValidatorSet) GetProposer(height int32) *crypto.PublicKey {
	return s.validators[s.ProposerIndex(height)]
}

// Has checks if a public key is in the validator set.
func (s *ValidatorSet) Has(pubKey *crypto.PublicKey) bool {
	for _, v := range s.validators {
		if pubKey.String() == v.String() {
			return true
		}
	}
	return false
}

// IndexOf returns the index of the given validator, or -1 if not found.
func (s *ValidatorSet) IndexOf(pubKey *crypto.PublicKey) int {
	for i, v := range s.validators {
		if pubKey.String() == v.String() {
			return i
		}
	}
	return -1
}

// Len returns the number of validators.
func (s *ValidatorSet) Len() int {
	return len(s.validators)
}
```

---

## Step 3 — Update the Node

Add `Chain` and `ValidatorSet` to the `Node` struct. Edit `node/node.go`:

```go
type Node struct {
	ServerConfig
	lnAddr  string
	height  int32
	version string
	logger  *slog.Logger

	peerLock    sync.RWMutex
	peers       map[proto.NodeClient]*proto.Version
	mempool     *Mempool
	chain       *Chain           // NEW
	validators  *ValidatorSet    // NEW

	proto.UnimplementedNodeServer
}
```

Update `NewNode` to accept these:

```go
func NewNode(cfg ServerConfig, chain *Chain, validators *ValidatorSet) *Node {
	return &Node{
		logger:      slog.Default(),
		peers:       make(map[proto.NodeClient]*proto.Version),
		mempool:     NewMempool(),
		chain:       chain,          // NEW
		validators:  validators,     // NEW
		ServerConfig: cfg,
	}
}
```

---

## Step 4 — Implement Block Proposal in validatorLoop

This is the core of PoS. When the ticker fires, the validator checks if it's their turn. If yes, they create a block, sign it, add it to their own chain, and broadcast it.

Replace the stub in `node/node.go`:

```go
func (n *Node) validatorLoop() {
	n.log("starting validator loop", "pubKey", n.PrivateKey.Public(), "blocktime", blockTime)
	ticker := time.NewTicker(blockTime)
	for {
		<-ticker.C

		height := int32(n.chain.Height() + 1)
		proposer := n.validators.GetProposer(height)

		// Is it my turn?
		if proposer.String() != n.PrivateKey.Public().String() {
			n.log("not my turn to propose", "height", height, "proposer", proposer)
			continue
		}

		txx := n.mempool.Clear()
		n.log("time to create a new block", "height", height, "lenTx", len(txx))

		block, err := n.createBlock(height, txx)
		if err != nil {
			n.log("failed to create block", "err", err)
			continue
		}

		if err := n.chain.AddBlock(block); err != nil {
			n.log("failed to add block to local chain", "err", err)
			continue
		}

		n.log("block created and added", "height", height, "hash", hex.EncodeToString(types.HashBlock(block)))

		go func() {
			if err := n.broadcastBlock(block); err != nil {
				n.log("broadcast block failed", "err", err)
			}
		}()
	}
}

func (n *Node) createBlock(height int32, txx []*proto.Transaction) (*proto.Block, error) {
	currentBlock, err := n.chain.GetBlockByHeight(n.chain.Height())
	if err != nil {
		return nil, fmt.Errorf("failed to get current block: %w", err)
	}

	prevHash := types.HashBlock(currentBlock)

	header := &proto.Header{
		Version:   1,
		Height:    height,
		PrevHash:  prevHash,
		Timestamp: time.Now().UnixNano(),
	}

	block := &proto.Block{
		Header:       header,
		Transactions: txx,
	}

	types.SignBlock(*n.PrivateKey, block)

	return block, nil
}

func (n *Node) broadcastBlock(block *proto.Block) error {
	for peer := range n.peers {
		_, err := peer.HandleBlock(context.Background(), block)
		if err != nil {
			return err
		}
	}
	return nil
}
```

Key points:
- `height = chain.Height() + 1` — this is index balancing in action. Every new block has a height one greater than the current tip
- `prevHash` is pulled from the actual current tip — not random
- The block is signed with the validator's private key
- Only the designated proposer creates the block

---

## Step 5 — Implement HandleBlock RPC

When another node receives a proposed block, they need to validate it and add it to their chain.

Add to `node/node.go`:

```go
func (n *Node) HandleBlock(ctx context.Context, block *proto.Block) (*emptypb.Empty, error) {
	peer, _ := peer.FromContext(ctx)
	hash := hex.EncodeToString(types.HashBlock(block))
	n.log("received block", "from", peer.Addr, "hash", hash, "height", block.Header.Height)

	if err := n.chain.AddBlock(block); err != nil {
		n.log("block validation failed", "err", err)
		return nil, err
	}

	// Remove included transactions from our mempool
	for _, tx := range block.Transactions {
		n.mempool.Clear()
	}

	// Rebroadcast to other peers
	go func() {
		if err := n.broadcastBlock(block); err != nil {
			n.log("broadcast block failed", "err", err)
		}
	}()

	return &emptypb.Empty{}, nil
}
```

**Important:** We remove transactions that were included in the block from the local mempool. The current `mempool.Clear()` removes everything, which is a simplification. A more precise implementation would only remove the transactions that appear in the block.

---

## Step 6 — Validate the Proposer in Chain.ValidateBlock

Edit `node/chain.go`. Update `NewChain` to accept a `ValidatorSet`:

```go
func NewChain(bs BlockStorer, txStore TxStorer, validators *ValidatorSet) *Chain {
	chain := &Chain{
		txStore:    txStore,
		blockStore: bs,
		utxoStore:  NewUTXOStore(),
		headers:    NewHeaderList(),
		validators: validators,
	}

	chain.addBlock(createGenesisBlock())
	return chain
}
```

Add the field to `Chain`:

```go
type Chain struct {
	txStore    TxStorer
	blockStore BlockStorer
	utxoStore  UTXOStorer
	headers    *HeaderList
	validators *ValidatorSet    // NEW
}
```

Update `ValidateBlock`:

```go
func (c *Chain) ValidateBlock(b *proto.Block) error {
	// Verify the block signature
	if !types.VerifyBlock(b) {
		return fmt.Errorf("invalid block: signature verification failed")
	}

	// Validate height (index balancing!)
	expectedHeight := int32(c.Height() + 1)
	if b.Header.Height != expectedHeight {
		return fmt.Errorf("invalid block: expected height %d, got %d", expectedHeight, b.Header.Height)
	}

	// Validate that the signer is an authorized validator
	signerPubKey := crypto.PublicKeyFromBytes(b.PublicKey)
	if !c.validators.Has(&signerPubKey) {
		return fmt.Errorf("invalid block: signer is not a validator")
	}

	// Validate that it's this validator's turn
	expectedProposer := c.validators.GetProposer(b.Header.Height)
	if signerPubKey.String() != expectedProposer.String() {
		return fmt.Errorf("invalid block: wrong proposer at height %d", b.Header.Height)
	}

	// Validate prevHash
	currentBlock, err := c.GetBlockByHeight(c.Height())
	if err != nil {
		return err
	}

	hash := types.HashBlock(currentBlock)
	if !bytes.Equal(hash, b.Header.PrevHash) {
		return fmt.Errorf("invalid block: expected prev hash %s, got %s",
			hex.EncodeToString(hash), hex.EncodeToString(b.Header.PrevHash))
	}

	// Validate transactions
	for _, tx := range b.Transactions {
		if err := c.ValidateTransaction(tx); err != nil {
			return fmt.Errorf("invalid block: %w", err)
		}
	}

	return nil
}
```

The height validation (`b.Header.Height != expectedHeight`) is the index balancing check — see `INDEX_BALANCING.md` for more detail.

---

## Step 7 — Fix the Genesis Block

Update `createGenesisBlock` in `node/chain.go` to explicitly set height to 0:

```go
func createGenesisBlock() *proto.Block {
	privKey, _ := crypto.NewPrivateKeyFromSeedStr(MasterSeed)
	block := &proto.Block{
		Header: &proto.Header{
			Version:   1,
			Height:    0,    // EXPLICIT: genesis block is at height 0
			Timestamp: 0,
		},
	}

	tx := &proto.Transaction{
		Version: 1,
		Inputs:  []*proto.TxInput{},
		Outputs: []*proto.TxOutput{
			{
				Amount:  1_000,
				Address: privKey.Public().Address().Bytes(),
			},
		},
	}

	block.Transactions = append(block.Transactions, tx)
	types.SignBlock(privKey, block)

	return block
}
```

---

## Step 8 — Update ServerConfig and main.go

Update `ServerConfig` — remove `PrivateKey` (move it to `Node` directly or keep it; either works). What matters is passing the chain and validators to the node.

Update `cmd/main.go`:

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/node"
	"github.com/ignoxx/blocker/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Create 3 validator keys
	privKey1, _ := crypto.GeneratePrivateKey()
	privKey2, _ := crypto.GeneratePrivateKey()
	privKey3, _ := crypto.GeneratePrivateKey()

	validators := node.NewValidatorSet([]*crypto.PublicKey{
		privKey1.Public(),
		privKey2.Public(),
		privKey3.Public(),
	})

	// All nodes share the same chain stores for simplicity.
	// In production, each node would have its own persistent store.
	blockStore1 := node.NewMemoryBlockStore()
	txStore1 := node.NewMemoryTxStore()
	chain1 := node.NewChain(blockStore1, txStore1, validators)

	blockStore2 := node.NewMemoryBlockStore()
	txStore2 := node.NewMemoryTxStore()
	chain2 := node.NewChain(blockStore2, txStore2, validators)

	blockStore3 := node.NewMemoryBlockStore()
	txStore3 := node.NewMemoryTxStore()
	chain3 := node.NewChain(blockStore3, txStore3, validators)

	makeNode(":3000", privKey1, chain1, validators)
	time.Sleep(time.Second)
	makeNode(":4000", privKey2, chain2, validators, ":3000")
	time.Sleep(2 * time.Second)
	makeNode(":5001", privKey3, chain3, validators, ":4000")

	select {}
}

func makeNode(lnAddr string, privKey *crypto.PrivateKey, chain *node.Chain, validators *node.ValidatorSet, bootstrapNodes ...string) *node.Node {
	cfg := node.ServerConfig{
		Version:    "blocker-v1",
		ListenAddr: lnAddr,
		PrivateKey: privKey,
	}

	n := node.NewNode(cfg, chain, validators)
	go func() {
		if err := n.Start(bootstrapNodes); err != nil {
			log.Fatal(err)
		}
	}()
	return n
}
```

**Key point:** All three nodes must share the same `ValidatorSet` (same public keys in the same order). Each node only proposes when it's their turn. The other two nodes validate and accept the block.

---

## Step 9 — Propagate Validators During Handshake (Optional Enhancement)

In a real network, validators would be agreed upon via governance or staking. For our learning project, we hardcode them. But if you want to propagate them during handshake, add the validators to the `Version` message:

```go
func (n *Node) getVersion() *proto.Version {
	v := &proto.Version{
		Version:    n.Version,
		Height:     int32(n.chain.Height()),
		ListenAddr: n.ListenAddr,
		PeerList:   n.getPeerList(),
	}

	// Add validator public keys
	for _, pk := range n.validators.validators {
		v.Validators = append(v.Validators, pk.Bytes())
	}

	return v
}
```

And extract them on the receiving end in `Handshake`:

```go
func (n *Node) Handshake(ctx context.Context, v *proto.Version) (*proto.Version, error) {
	c, err := makeNodeClient(v.ListenAddr)
	if err != nil {
		return nil, errors.New("handshake: " + err.Error())
	}

	n.addPeer(c, v)
	n.log("received handshake", "version", v.Version, "addr", v.ListenAddr, "height", v.Height)

	return n.getVersion(), nil
}
```

For now, we skip dynamic validator propagation and hardcode.

---

## How It All Fits Together

```
validatorLoop (every 5s):
    │
    ├── Is it my turn? (height % numValidators == myIndex)
    │       │
    │       ├── NO  → skip, wait for next tick
    │       │
    │       └── YES → create block from mempool
    │                    │
    │                    ├── Set header.Height = chain.Height() + 1
    │                    ├── Set header.PrevHash = hash of current tip
    │                    ├── Sign block with my private key
    │                    ├── Add to local chain
    │                    └── Broadcast to peers via HandleBlock RPC
    │
    └── (other nodes receive block via HandleBlock)
            │
            ├── Validate height, prevHash, signature, proposer
            ├── Add to their chain
            └── Re-broadcast to their peers
```

---

## Testing

Start three nodes and submit transactions:

```bash
# Terminal 1: run the network
make run

# Terminal 2: submit a transaction (or use the HandleTransaction RPC)
# The validator whose turn it is will pick it up and create a block
```

Watch the logs — you'll see each validator taking turns every 5 seconds, with blocks being proposed and validated in round-robin order.

---

## Limitations of This Implementation

This is a **minimal learning implementation**. A production PoS would need:

| Concern | Our simplification | Production approach |
|---------|-------------------|-------------------|
| Validator set | Hardcoded, all nodes know all validators | Dynamic, based on on-chain staking |
| Proposer selection | Round-robin (`height % num`) | Weighted by stake, with randomness |
| Fork resolution | None — single chain assumed | LMD GHOST, Casper FFG, etc. |
| Slashing | No penalties for bad behavior | Slash validators for double-signing, inactivity |
| Block finality | None — last block wins | Finality gadgets (2/3 attestations) |
| Network partitions | No handling | Consensus must handle partitions and recovery |

---

## Summary of Files to Create/Modify

| File | Action |
|------|--------|
| `proto/types.proto` | Add `HandleBlock` RPC, `validators` field on `Version` |
| `node/validators.go` | **NEW** — `ValidatorSet` type |
| `node/node.go` | Add `Chain`, `ValidatorSet` fields; implement `validatorLoop`, `createBlock`, `HandleBlock`, `broadcastBlock` |
| `node/chain.go` | Add `validators` field to `Chain`; add height + proposer validation to `ValidateBlock`; fix genesis block height |
| `cmd/main.go` | Create validator set, chain per node, pass to `NewNode` |