# blocker

custom UTXO blockchain implementation from scratch

## using
- [x] ED25519 crypto for (private)keys
- [x] UTXO model
- [x] Protobuffer encoding
- [x] GRPC transport (gossip, communication between nodes)
- [x] Proof of Stake (PoS) consensus (round-robin, event-driven)
- [x] Minimal wallet CLI (create, balance, transactions)

## cli

```bash
# Create a new wallet
make cli create

# Query balance (use any node's grpc address)
make cli balance -addr <hex-address>
make cli balance -node :4000 -addr 36f8343eaa41afbff509addde0f5cb4a0691295b

# Query transactions
make cli txs -addr <hex-address>
```

## not production-ready (high-level gaps)

this is a learning project. a production blockchain would need at least the following:

### consensus
- **weighted proposer selection** — validators chosen by stake amount, not equal round-robin
- **fork choice rule** — pick the heaviest/best chain when forks occur
- **block finality** — blocks are not final; reorgs can happen at any time
- **slashing** — no economic penalties for double-signing or invalid blocks
- **validator rotation** — validator set is hardcoded; no dynamic join/leave

### networking
- **block sync for late joiners** — new nodes starting behind the tip never catch up
- **out-of-order block buffering** — blocks received too far ahead are dropped, not buffered
- **peer scoring / banning** — no mechanism to disconnect or penalize bad peers
- **connection encryption** — gRPC runs without TLS

### data layer
- **persistent storage** — everything is in-memory; restart loses all data
- **state snapshots / pruning** — chain grows forever, no old state cleanup
- **transaction indexing** — no way to query tx history efficiently
- **merkle proofs** — can't prove inclusion of a tx to a light client

### transactions / vm
- **no smart contracts / VM** — only simple value transfers
- **fixed fees** — no fee market or gas pricing
- **no nonce / replay protection** — same tx can be replayed unless UTXO is spent

### tooling
- **no block explorer** — no way to inspect chain state visually
- **no metrics / monitoring** — no prometheus, health checks, or alerts